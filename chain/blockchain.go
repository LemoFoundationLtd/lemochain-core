package chain

import (
	"bytes"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/flag"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/subscribe"
	"github.com/LemoFoundationLtd/lemochain-core/network"
	db "github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type BlockChain struct {
	chainID      uint16
	flags        flag.CmdFlags
	db           db.ChainDB
	am           *account.Manager
	dm           *deputynode.Manager
	currentBlock atomic.Value // latest block in current chain
	stableBlock  atomic.Value // latest stable block in current chain
	genesisBlock *types.Block // genesis block

	chainForksHead map[common.Hash]*types.Block // total latest header of different fork chain
	chainForksLock sync.Mutex
	mux            sync.Mutex
	engine         Engine       // consensus engine
	processor      *TxProcessor // state processor
	running        int32

	RecvBlockFeed subscribe.Feed

	setStableMux sync.Mutex

	quitCh chan struct{}
}

func NewBlockChain(chainID uint16, engine Engine, dm *deputynode.Manager, db db.ChainDB, flags flag.CmdFlags) (bc *BlockChain, err error) {
	bc = &BlockChain{
		chainID: chainID,
		db:      db,
		dm:      dm,
		// newBlockCh:     newBlockCh,
		flags:          flags,
		engine:         engine,
		chainForksHead: make(map[common.Hash]*types.Block, 16),
		quitCh:         make(chan struct{}),
	}
	bc.genesisBlock, err = bc.db.GetBlockByHeight(0)
	if err != nil {
		return nil, ErrNoGenesis
	}
	if err := bc.loadLastState(); err != nil {
		return nil, err
	}
	bc.processor = NewTxProcessor(bc)
	return bc, nil
}

func (bc *BlockChain) AccountManager() *account.Manager {
	return bc.am
}

func (bc *BlockChain) DeputyManager() *deputynode.Manager {
	return bc.dm
}

// Lock call by miner
func (bc *BlockChain) Lock() *sync.Mutex {
	return &bc.mux
}

// loadLastState load latest state in starting
func (bc *BlockChain) loadLastState() error {
	block, err := bc.db.LoadLatestBlock()
	if err != nil {
		log.Errorf("Can't load last state: %v", err)
		return err
	}
	bc.currentBlock.Store(block)
	bc.stableBlock.Store(block)
	bc.am = account.NewManager(block.Hash(), bc.db)
	return nil
}

// Genesis genesis block
func (bc *BlockChain) Genesis() *types.Block {
	return bc.genesisBlock
}

// Founder is LEMO holder in the genesis block
func (bc *BlockChain) Founder() common.Address {
	return bc.Genesis().DeputyNodes[0].MinerAddress
}

// ChainID
func (bc *BlockChain) ChainID() uint16 {
	return bc.chainID
}

func (bc *BlockChain) TxProcessor() *TxProcessor {
	return bc.processor
}

func (bc *BlockChain) Flags() flag.CmdFlags {
	return bc.flags
}

// HasBlock has special block in local
func (bc *BlockChain) HasBlock(hash common.Hash) bool {
	if ok, _ := bc.db.IsExistByHash(hash); ok {
		return true
	}
	return false
}

func (bc *BlockChain) GetBlockByHeight(height uint32) *types.Block {
	currentBlock := bc.currentBlock.Load().(*types.Block)
	currentBlockHeight := currentBlock.Height()
	stableBlockHeight := bc.stableBlock.Load().(*types.Block).Height()
	var block *types.Block
	var err error

	// TODO 如果没有分叉，非稳定块能否直接用hash读？结合数据库实现考虑如何优化
	if height <= stableBlockHeight {
		block, err = bc.db.GetBlockByHeight(height)
		if err != nil {
			log.Error("load stable parent block fail", "height", height, "err", err)
			panic(err)
		}
	} else if height <= currentBlockHeight {
		// These part of chain may be forked, so iterate the blocks one by one
		block, err = FindParentByHeight(height, currentBlock, bc.db)
		if err != nil {
			panic(err)
		}
	}
	return block
}

func (bc *BlockChain) GetBlockByHash(hash common.Hash) *types.Block {
	block, err := bc.db.GetBlockByHash(hash)
	if err != nil {
		log.Debug("load block fail", "hash", hash.Hex(), "err", err)
		return nil
	}
	return block
}

// CurrentBlock get latest current block
func (bc *BlockChain) CurrentBlock() *types.Block {
	return bc.currentBlock.Load().(*types.Block)
}

// StableBlock get latest stable block
func (bc *BlockChain) StableBlock() *types.Block {
	return bc.stableBlock.Load().(*types.Block)
}

// SetMinedBlock 挖到新块
func (bc *BlockChain) SetMinedBlock(block *types.Block) error {
	sb := bc.stableBlock.Load().(*types.Block)
	if sb.Height() >= block.Height() {
		log.Debug("mine a block, but height is smaller than stable block")
		return nil
	}
	if ok, _ := bc.db.IsExistByHash(block.ParentHash()); !ok {
		return ErrParentNotExist
	}

	// save
	if err := bc.db.SetBlock(block.Hash(), block); err != nil {
		log.Errorf("can't insert block to cache. height:%d hash:%s", block.Height(), block.Hash().Hex())
		return ErrSaveBlock
	}
	log.Debugf("Insert mined block to db, height: %d, hash: %s, time: %d, parent: %s", block.Height(), block.Hash().Prefix(), block.Time(), block.ParentHash().Prefix())
	err := bc.AccountManager().Save(block.Hash())
	if err != nil {
		log.Error("save account error!", "hash", block.Hash().Hex(), "err", err)
		return ErrSaveAccount
	}
	bc.currentBlock.Store(block)
	delete(bc.chainForksHead, block.ParentHash())
	bc.chainForksHead[block.Hash()] = block

	// check consensus
	nodeCount := bc.dm.GetDeputiesCount(block.Height())
	if nodeCount == 1 {
		_ = bc.SetStableBlock(block.Hash(), block.Height())
	}
	bc.updateDeputyNodes(block)

	// notify
	go func() {
		subscribe.Send(subscribe.NewMinedBlock, block)
		bc.confirmBlock(block.Hash(), block.Height())
		bc.newBlockNotify(block)
	}()
	return nil
}

// updateDeputyNodes update deputy nodes map
func (bc *BlockChain) updateDeputyNodes(block *types.Block) {
	if deputynode.IsSnapshotBlock(block.Height()) {
		bc.dm.SaveSnapshot(block.Height(), block.DeputyNodes)
		log.Debugf("add new term deputy nodes: %v", block.DeputyNodes)
	}
}

// newBlockNotify
func (bc *BlockChain) newBlockNotify(block *types.Block) {
	bc.RecvBlockFeed.Send(block)
}

// InsertChain insert block of non-self to chain
func (bc *BlockChain) InsertChain(rawBlock *types.Block, isSynchronising bool) (err error) {
	bc.mux.Lock()
	defer bc.mux.Unlock()

	hash := rawBlock.Hash()
	parentHash := rawBlock.ParentHash()
	oldCurrentBlock := bc.currentBlock.Load().(*types.Block)
	oldCurrentHash := oldCurrentBlock.Hash()

	block, err := bc.VerifyAndSeal(rawBlock)
	if err != nil {
		log.Errorf("block verify failed: %v", err)
		return ErrVerifyBlockFailed
	}

	// save
	if err = bc.db.SetBlock(hash, block); err != nil {
		log.Errorf("can't insert block to cache. height:%d hash:%s", block.Height(), hash.Prefix())
		return ErrSaveBlock
	}
	log.Infof("Insert block to chain. height: %d. hash: %s. time: %d. parent: %s", block.Height(), block.Hash().Prefix(), block.Time(), block.ParentHash().Prefix())
	if err := bc.AccountManager().Save(hash); err != nil {
		log.Error("save account error!", "height", block.Height(), "hash", hash.Prefix(), "err", err)
		return err
	}
	// check confirm packages
	nodeCount := bc.dm.GetDeputiesCount(block.Height())
	if nodeCount < 3 {
		// TODO 异常情况也会进入defer
		// Two deputy nodes scene: One node mined a block and broadcasted it. Then it means two confirms after the receiver one's verification
		defer func() {
			_ = bc.SetStableBlock(hash, block.Height())
			bc.updateDeputyNodes(block)
		}()
	} else {
		minCount := int(math.Ceil(float64(nodeCount) * 2.0 / 3.0))
		if len(block.Confirms) >= minCount {
			defer func() {
				_ = bc.SetStableBlock(hash, block.Height())
				bc.updateDeputyNodes(block)
			}()
		}
	}

	broadcastConfirm := false
	bc.chainForksLock.Lock()
	defer func() {
		bc.chainForksLock.Unlock()
		if broadcastConfirm && bc.dm.IsSelfDeputyNode(block.Height()) {
			// only broadcast confirm info within 3 minutes
			currentTime := time.Now().Unix()
			if currentTime-int64(block.Time()) < 3*60 {
				go func() {
					bc.confirmBlock(block.Hash(), block.Height())
				}()
			}
		}
		// for debug
		b := bc.currentBlock.Load().(*types.Block)
		log.Debugf("current block: %d, %s, parent: %s", b.Height(), b.Hash().String()[:16], b.ParentHash().String()[:16])
	}()

	// normal, in same chain
	if parentHash == oldCurrentHash {
		bc.currentBlock.Store(block)
		delete(bc.chainForksHead, oldCurrentHash) // remove old record from fork container
		bc.chainForksHead[hash] = block           // record new fork
		bc.newBlockNotify(block)
		broadcastConfirm = true
		return nil
	}

	// fork
	fork, err := bc.needFork(block)
	if err != nil {
		log.Errorf("InsertChain: needFork failed: %v", err)
		// TODO should we panic here? Keep running will lead to wrong confirmation
		return nil
	}
	if fork {
		if block.Height() > oldCurrentBlock.Height() {
			// TODO 如果先切到较短分支，再切到较长分支，就可能对同一高度确认两次
			broadcastConfirm = true
		}
		bc.currentBlock.Store(block)
		log.Warnf("chain forked-1! current block: height(%d), hash(%s)", block.Height(), block.Hash().Prefix())
	} else {
		log.Debugf("not update current block. block: %d - %s, parent: %s, current: %d - %s",
			block.Height(), block.Hash().Prefix(), block.ParentHash().Prefix(), oldCurrentBlock.Height(), oldCurrentBlock.Hash().Prefix())
	}
	// update fork's head
	if _, ok := bc.chainForksHead[parentHash]; ok {
		delete(bc.chainForksHead, parentHash)
	}
	bc.chainForksHead[hash] = block
	bc.newBlockNotify(block)
	return nil
}

// needFork
func (bc *BlockChain) needFork(block *types.Block) (bool, error) {
	cBlock := bc.currentBlock.Load().(*types.Block)
	sBlock := bc.stableBlock.Load().(*types.Block)

	cParent, parent, err := FindFirstForkBlocks(cBlock, block, bc.db)
	if err != nil {
		log.Debug("needFork: find first fork parent blocks fail")
		return false, err
	}
	// ancestor's height can't less than stable block's height
	if parent.Height() <= sBlock.Height() {
		log.Debug("needFork: fork block is stable", "ancestorHeight", parent.Height(), "stableHeight", sBlock.Height())
		return false, nil
	}

	// 1. choose the one has smaller time
	if cParent.Time() > parent.Time() {
		return true, nil
	}
	// 2. if times are equal, choose the one has smaller hash in dictionary order
	hash := parent.Hash()
	cHash := cParent.Hash()
	if cParent.Time() == parent.Time() && bytes.Compare(cHash[:], hash[:]) > 0 {
		return true, nil
	}
	return false, nil
}

// SetStableBlock
func (bc *BlockChain) SetStableBlock(hash common.Hash, height uint32) error {
	bc.setStableMux.Lock()
	defer bc.setStableMux.Unlock()

	block := bc.GetBlockByHash(hash)
	if block == nil {
		log.Warnf("SetStableBlock: block not exist. height: %d hash: %s", height, hash.String())
		return ErrBlockNotExist
	}
	height = block.Height()
	oldStableBlock := bc.stableBlock.Load().(*types.Block)
	if block.Height() <= oldStableBlock.Height() {
		return nil
	}
	// set stable
	if err := bc.db.SetStableBlock(hash); err != nil {
		log.Errorf("SetStableBlock error. height:%d hash:%s, err:%s", height, common.ToHex(hash[:]), err.Error())
		return ErrSetStableBlockToDB
	}
	bc.stableBlock.Store(block)

	defer func() {
		log.Infof("Consensus. height:%d hash:%s", block.Height(), block.Hash().Prefix())
	}()
	if len(bc.chainForksHead) > 1 {
		res := strings.Builder{}
		for _, v := range bc.chainForksHead {
			res.WriteString(fmt.Sprintf("height: %d. hash: %s. parent: %s\r\n", v.Height(), v.Hash().Prefix(), v.ParentHash().Prefix()))
		}
		log.Debugf("total forks: %s", res.String())
	}

	if err := bc.checkCurrentBlock(hash, height); err != nil {
		return err
	}

	// notify
	subscribe.Send(subscribe.NewStableBlock, block)
	log.Infof("stable height reach to: %d", height)
	return nil
}

// checkCurrentBlock
func (bc *BlockChain) checkCurrentBlock(stableHash common.Hash, height uint32) error {
	curBlock := bc.currentBlock.Load().(*types.Block)
	bc.chainForksLock.Lock()
	defer bc.chainForksLock.Unlock()

	tmp := make(map[common.Hash][]*types.Block) // record all fork's parent, reach to stable block
	maxLength := uint32(0)
	// prune forks
	for fHash, fBlock := range bc.chainForksHead {
		if fBlock.Height() < height {
			delete(bc.chainForksHead, fHash)
			continue
		}
		if fBlock.Height() == height {
			if fBlock.Hash() != stableHash {
				delete(bc.chainForksHead, fHash)
			} else {
				tmp[stableHash] = []*types.Block{}
			}
			continue
		}

		// fBlock.Height()> height
		length := fBlock.Height() - height - 1
		if length > maxLength {
			maxLength = length
		}
		// get the same height block on current fork
		pars := make([]*types.Block, length+1)
		parBlock := fBlock
		for i := fBlock.Height(); i > height; i-- {
			pars[i-height-1] = parBlock
			parBlock = bc.GetBlockByHash(parBlock.ParentHash())
		}
		// current chain and stable chain is same
		if parBlock.ParentHash() != stableHash {
			// prune
			delete(bc.chainForksHead, fHash)
		} else {
			// parBlock is son of fBlock
			tmp[fHash] = pars
		}
	}
	var newCurBlock *types.Block
	// choose current block
	if maxLength == uint32(0) {
		// they are same height sons of the stable block
		// TODO choose randomly
		for k, _ := range tmp {
			newCurBlock = bc.GetBlockByHash(k)
		}
	} else {
		for i := uint32(0); i <= maxLength; i++ {
			var b *types.Block
			for k, blocks := range tmp {
				if int(i) >= len(blocks) {
					continue
				}
				if b == nil {
					b = blocks[i]
					newCurBlock = b
				} else {
					bHash := b.Hash()
					vHash := blocks[i].Hash()
					if (b.Time() > blocks[i].Time()) || (b.Time() == blocks[i].Time() && bytes.Compare(bHash[:], vHash[:]) > 0) {
						b = blocks[i]
						newCurBlock = b
					} else {
						delete(tmp, k)
					}
				}

			}
		}
	}
	// check if the new or old current block's height is smaller than stable block's
	if newCurBlock != nil && newCurBlock.Height() < height || newCurBlock == nil && curBlock.Height() < height {
		log.Debug("current block's height < stable block's height")
		newCurBlock = bc.stableBlock.Load().(*types.Block)
	}
	if newCurBlock != nil {
		oldCurHash := curBlock.Hash()
		newCurHash := newCurBlock.Hash()
		// switch fork
		if bytes.Compare(newCurHash[:], oldCurHash[:]) != 0 {
			bc.currentBlock.Store(newCurBlock)
			bc.newBlockNotify(newCurBlock)
			log.Infof("chain forked-2! oldCurHash{ h: %d, hash: %s}, newCurBlock{h:%d, hash: %s}", curBlock.Height(), oldCurHash.Prefix(), newCurBlock.Height(), newCurHash.Prefix())
		}
	} else {
		log.Debug("not have new current block")
	}
	return nil
}

// isIgnorableBlock check the block is exist or not
func (bc *BlockChain) isIgnorableBlock(block *types.Block) bool {
	if has, _ := bc.db.IsExistByHash(block.Hash()); has {
		return true
	}
	sb := bc.stableBlock.Load().(*types.Block)
	if sb.Height() >= block.Height() {
		// the block may not correct, it is dangerous
		log.Debug("ignore the block whose height is smaller than stable block")
		return true
	}
	return false
}

// VerifyAndSeal verify block then create a new block
func (bc *BlockChain) VerifyAndSeal(block *types.Block) (*types.Block, error) {
	// ignore exist block as soon as possible
	if ok := bc.isIgnorableBlock(block); ok {
		return nil, ErrExistBlock
	}

	// verify every things that can be verified before tx processing
	if err := bc.engine.VerifyBeforeTxProcess(block); err != nil {
		return nil, err
	}
	// freeze deputy nodes before transaction process
	var deputySnapshot deputynode.DeputyNodes
	if deputynode.IsSnapshotBlock(block.Height()) {
		deputySnapshot = bc.SnapshotDeputyNodes()
	}

	// execute tx
	gasUsed, err := bc.processor.Process(block.Header, block.Txs)
	if err != nil {
		if err == ErrInvalidTxInBlock {
			return nil, err
		}
		log.Errorf("processor internal error: %v", err)
		panic("processor internal error")
	}
	// Finalize accounts
	if err = bc.engine.Finalize(block.Header.Height, bc.am, bc.dm); err != nil {
		log.Errorf("Finalize accounts error: %v", err)
		return nil, err
	}
	newBlock, err := bc.engine.Seal(block.Header, bc.am.GetTxsProduct(block.Txs, gasUsed), block.Confirms, deputySnapshot)
	if err != nil {
		log.Errorf("Seal block error: %v", err)
		return nil, err
	}

	// verify the things computed by tx
	if err := bc.engine.VerifyAfterTxProcess(block, newBlock); err != nil {
		return nil, err
	}
	return newBlock, nil
}

// confirmBlock confirm a block and send confirm event
func (bc *BlockChain) confirmBlock(hash common.Hash, height uint32) {
	// sign
	privateKey := deputynode.GetSelfNodeKey()
	sig, err := crypto.Sign(hash[:], privateKey)
	if err != nil {
		log.Error("sign for confirm data error")
		return
	}
	var signData types.SignData
	copy(signData[:], sig)

	// save
	if bc.HasBlock(hash) {
		if err := bc.db.SetConfirm(hash, signData); err != nil {
			log.Errorf("SetConfirm failed: %v", err)
		}
	}

	// notify
	subscribe.Send(subscribe.NewConfirm, &network.BlockConfirmData{
		Hash:     hash,
		Height:   height,
		SignInfo: signData,
	})
	log.Debugf("subscribe.NewConfirm.Send(%d)", height)
}

// ReceiveConfirm
func (bc *BlockChain) ReceiveConfirm(info *network.BlockConfirmData) (err error) {
	err = bc.engine.VerifyConfirmPacket(info.Height, info.Hash, []types.SignData{info.SignInfo})
	if err != nil {
		return err
	}

	// has block consensus
	stableBlock := bc.stableBlock.Load().(*types.Block)
	if stableBlock.Height() >= info.Height { // stable block's confirm info
		if ok, err := bc.hasEnoughConfirmInfo(info.Hash, info.Height); err == nil && !ok {
			if err = bc.db.SetConfirm(info.Hash, info.SignInfo); err != nil {
				log.Errorf("SetConfirm failed: %v", err)
			}
		}
		return nil
	}

	// cache confirm info
	if err = bc.db.SetConfirm(info.Hash, info.SignInfo); err != nil {
		log.Errorf("can't SetConfirmInfo. height: %d, hash:%s, err: %v", info.Height, info.Hash.Hex()[:16], err)
		return nil
	}

	if ok, _ := bc.hasEnoughConfirmInfo(info.Hash, info.Height); ok {
		bc.mux.Lock()
		defer bc.mux.Unlock()
		if err = bc.SetStableBlock(info.Hash, info.Height); err != nil {
			log.Errorf("ReceiveConfirm: setStableBlock failed. height: %d, hash:%s, err: %v", info.Height, info.Hash.Hex()[:16], err)
		}
	}
	return nil
}

func (bc *BlockChain) hasEnoughConfirmInfo(hash common.Hash, height uint32) (bool, error) {
	confirmCount, err := bc.getConfirmCount(hash)
	if err != nil {
		return false, err
	}
	nodeCount := bc.dm.GetDeputiesCount(height)
	minCount := int(math.Ceil(float64(nodeCount) * 2.0 / 3.0))
	if confirmCount >= minCount {
		return true, nil
	}
	return false, nil
}

// getConfirmCount get confirm count by hash
func (bc *BlockChain) getConfirmCount(hash common.Hash) (int, error) {
	pack, err := bc.db.GetConfirms(hash)
	if err != nil {
		log.Errorf("Can't GetConfirmInfo. hash:%s. error: %v", hash.Hex(), err)
		return -1, err
	}
	return len(pack), nil
}

// GetConfirms get all confirm info of special block
func (bc *BlockChain) GetConfirms(query *network.GetConfirmInfo) []types.SignData {
	res, err := bc.db.GetConfirms(query.Hash)
	if err != nil {
		log.Warnf("Can't GetConfirms. hash:%s height:%d. error: %v", query.Hash.Hex(), query.Height, err)
		return nil
	}
	return res
}

// ReceiveConfirms receive confirm package from net connection
func (bc *BlockChain) ReceiveConfirms(pack network.BlockConfirms) {
	if pack.Hash == (common.Hash{}) || pack.Pack == nil || len(pack.Pack) == 0 {
		return
	}
	if err := bc.engine.VerifyConfirmPacket(pack.Height, pack.Hash, pack.Pack); err != nil {
		log.Debugf("ReceiveConfirms: %v", err)
		return
	}
	if err := bc.db.SetConfirms(pack.Hash, pack.Pack); err != nil {
		log.Debugf("ReceiveConfirms: %v", err)
	}
}

// Stop stop block chain
func (bc *BlockChain) Stop() {
	if !atomic.CompareAndSwapInt32(&bc.running, 0, 1) {
		return
	}
	close(bc.quitCh)
	log.Info("BlockChain stop")
}

func (bc *BlockChain) Db() db.ChainDB {
	return bc.db
}

// SnapshotDeputyNodes get next epoch deputy nodes for snapshot block
func (bc *BlockChain) SnapshotDeputyNodes() deputynode.DeputyNodes {
	result := make(deputynode.DeputyNodes, 0, bc.dm.DeputyCount)
	list := bc.db.GetCandidatesTop(bc.CurrentBlock().Hash())
	if len(list) > bc.dm.DeputyCount {
		list = list[:bc.dm.DeputyCount]
	}

	for i, n := range list {
		acc := bc.am.GetAccount(n.GetAddress())
		candidate := acc.GetCandidate()
		strID := candidate[types.CandidateKeyNodeID]
		dn, err := deputynode.NewDeputyNode(n.GetTotal(), uint32(i), n.GetAddress(), strID)
		if err != nil {
			continue
		}
		result = append(result, dn)
	}
	return result
}
