package chain

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/flag"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/subscribe"
	"github.com/LemoFoundationLtd/lemochain-go/network"
	db "github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"math"
	"net"
	"strconv"
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

func NewBlockChain(chainID uint16, engine Engine, db db.ChainDB, flags flag.CmdFlags) (bc *BlockChain, err error) {
	bc = &BlockChain{
		chainID: chainID,
		db:      db,
		// newBlockCh:     newBlockCh,
		flags:          flags,
		engine:         engine,
		chainForksHead: make(map[common.Hash]*types.Block, 16),
		quitCh:         make(chan struct{}),
	}
	bc.genesisBlock = bc.GetBlockByHeight(0)
	if bc.genesisBlock == nil {
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

func (bc *BlockChain) getGenesisFromDb() *types.Block {
	block, err := bc.db.GetBlockByHeight(0)
	if err != nil {
		panic("can't get genesis block")
	}
	return block
}

func (bc *BlockChain) GetBlockByHeight(height uint32) *types.Block {
	// genesis block
	if height == 0 {
		return bc.getGenesisFromDb()
	}

	// not genesis block
	block := bc.currentBlock.Load().(*types.Block)
	currentBlockHeight := block.Height()
	stableBlockHeight := bc.stableBlock.Load().(*types.Block).Height()
	var err error
	if stableBlockHeight >= height {
		block, err = bc.db.GetBlockByHeight(height)
		if err != nil {
			panic(fmt.Sprintf("can't get block. height:%d, err: %v", height, err))
		}
	} else if height <= currentBlockHeight {
		for i := currentBlockHeight - height; i > 0; i-- {
			block, err = bc.db.GetBlockByHash(block.ParentHash())
			if err != nil {
				panic(fmt.Sprintf("can't get block. height:%d, err: %v", height, err))
			}
		}
	} else {
		return nil
	}
	return block
}

func (bc *BlockChain) GetBlockByHash(hash common.Hash) *types.Block {
	block, err := bc.db.GetBlockByHash(hash)
	if err != nil {
		log.Debugf("can't get block. hash:%s", hash.Hex())
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
		log.Debug("mine a block, but height not large than stable block")
		return nil
	}
	if ok, _ := bc.db.IsExistByHash(block.ParentHash()); !ok {
		return ErrParentNotExist
	}
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
	nodeCount := deputynode.Instance().GetDeputiesCount(block.Height())
	bc.currentBlock.Store(block)
	delete(bc.chainForksHead, block.ParentHash())
	bc.chainForksHead[block.Hash()] = block
	if nodeCount == 1 {
		_ = bc.SetStableBlock(block.Hash(), block.Height())
	}
	go func() {
		// notify
		subscribe.Send(subscribe.NewMinedBlock, block)
		msg := bc.createSignInfo(block.Hash(), block.Height())
		subscribe.Send(subscribe.NewConfirm, msg)
	}()
	return nil
}

func (bc *BlockChain) newBlockNotify(block *types.Block) {
	bc.RecvBlockFeed.Send(block)
}

// InsertChain insert block of non-self to chain
func (bc *BlockChain) InsertChain(block *types.Block, isSynchronising bool) (err error) {
	bc.mux.Lock()
	defer bc.mux.Unlock()

	if err := bc.Verify(block); err != nil {
		log.Errorf("block verify failed: %v", err)
		return ErrVerifyBlockFailed
	}

	hash := block.Hash()
	parentHash := block.ParentHash()
	oldCurrentBlock := bc.currentBlock.Load().(*types.Block)
	currentHash := oldCurrentBlock.Hash()
	if has, _ := bc.db.IsExistByHash(hash); has {
		return nil
	}
	if ok, _ := bc.db.IsExistByHash(parentHash); !ok {
		return ErrParentNotExist
	}
	// save
	block.SetEvents(bc.AccountManager().GetEvents())
	block.SetChangeLogs(bc.AccountManager().GetChangeLogs())

	sb := bc.stableBlock.Load().(*types.Block)
	if sb.Height() >= block.Height() {
		log.Debug("mine a block, but height not large than stable block")
		return nil
	}

	if err = bc.db.SetBlock(hash, block); err != nil {
		log.Errorf("can't insert block to cache. height:%d hash:%s", block.Height(), hash.Prefix())
		return ErrSaveBlock
	}
	log.Infof("Insert block to chain. height: %d. hash: %s. time: %d. parent: %s", block.Height(), block.Hash().Prefix(), block.Time(), block.ParentHash().Prefix())
	if err := bc.AccountManager().Save(hash); err != nil {
		log.Error("save account error!", "height", block.Height(), "hash", hash.Prefix(), "err", err)
		return err
	}
	// is synchronise from net or deputy nodes less than 3
	nodeCount := deputynode.Instance().GetDeputiesCount(block.Height())
	if nodeCount < 3 {
		defer func() { _ = bc.SetStableBlock(hash, block.Height()) }()
	} else {
		minCount := int(math.Ceil(float64(nodeCount) * 2.0 / 3.0))
		if len(block.Confirms) >= minCount {
			defer func() { _ = bc.SetStableBlock(hash, block.Height()) }()
		}
	}

	broadcastConfirm := false
	bc.chainForksLock.Lock()
	defer func() {
		bc.chainForksLock.Unlock()
		if broadcastConfirm && deputynode.Instance().IsSelfDeputyNode(block.Height()) {
			// only broadcast confirm info within 3 minutes
			currentTime := time.Now().Unix()
			if currentTime-int64(block.Time()) < 3*60 {
				time.AfterFunc(500*time.Millisecond, func() {
					msg := bc.createSignInfo(block.Hash(), block.Height())
					subscribe.Send(subscribe.NewConfirm, msg)
					log.Debugf("subscribe.NewConfirm.Send(%d)", msg.Height)
				})
			}
		}
		// for debug
		b := bc.currentBlock.Load().(*types.Block)
		log.Debugf("current block: %d, %s, parent: %s", b.Height(), b.Hash().String()[:16], b.ParentHash().String()[:16])
	}()

	// normal, in same chain
	if parentHash == currentHash {
		bc.currentBlock.Store(block)
		delete(bc.chainForksHead, currentHash) // remove old record from fork container
		bc.chainForksHead[hash] = block        // record new fork
		bc.newBlockNotify(block)
		broadcastConfirm = true
		return nil
	}

	fork, err := bc.needFork(block)
	if err != nil {
		log.Errorf("InsertChain: needFork failed: %v", err)
		return nil
	}
	if fork {
		if block.Height() > oldCurrentBlock.Height() {
			broadcastConfirm = true
		}
		bc.currentBlock.Store(block)
		log.Warnf("chain forked-1! current block: height(%d), hash(%s)", block.Height(), block.Hash().Prefix())
	} else {
		log.Debugf("not update current block. block: %d - %s, parent: %s, current: %d - %s",
			block.Height(), block.Hash().Prefix(), block.ParentHash().Prefix(), oldCurrentBlock.Height(), oldCurrentBlock.Hash().Prefix())
	}
	if _, ok := bc.chainForksHead[parentHash]; ok {
		delete(bc.chainForksHead, parentHash)
	}
	bc.chainForksHead[hash] = block
	bc.newBlockNotify(block)
	return nil
}

// needFork
func (bc *BlockChain) needFork(b *types.Block) (bool, error) {
	cB := bc.currentBlock.Load().(*types.Block)
	sB := bc.stableBlock.Load().(*types.Block)
	bFB := b  // 新块所在链 父块
	cFB := cB // 当前链 父块
	var err error
	if b.Height() > cB.Height() {
		// 查找与cB同高度的区块
		for bFB.Height() > cB.Height() {
			hash := bFB.ParentHash()
			height := bFB.Height() + 1
			if bFB, err = bc.db.GetBlockByHash(hash); err != nil {
				log.Debugf("needFork: getBlock failed. height: %d, hash: %s", height, hash.Prefix())
				return false, err
			}
		}
	} else if b.Height() < cB.Height() {
		// 查找与cB同高度的区块
		for cFB.Height() > b.Height() {
			hash := cFB.ParentHash()
			height := cFB.Height() + 1
			if cFB, err = bc.db.GetBlockByHash(hash); err != nil {
				log.Debugf("needFork: getBlock failed. height: %d, hash: %s", height, hash.Prefix())
				return false, err
			}
		}
	}

	// find same ancestor
	for bFB.ParentHash() != cFB.ParentHash() {
		hashB := bFB.ParentHash()
		heightB := bFB.Height() + 1
		hashC := cFB.ParentHash()
		heightC := cFB.Height() + 1
		if bFB, err = bc.db.GetBlockByHash(hashB); err != nil {
			log.Debugf("needFork: getBlock failed. height: %d, hash: %s", heightB, hashB.Prefix())
			return false, err
		}
		if cFB, err = bc.db.GetBlockByHash(hashC); err != nil {
			log.Debugf("needFork: getBlock failed. height: %d, hash: %s", heightC, hashC.Prefix())
			return false, err
		}
	}
	// ancestor's height can't less than stable block's height
	if bFB.Height() <= sB.Height() {
		log.Debugf("bFB.Height(%d) <= sB.Height(%d)", bFB.Height(), sB.Height())
		return false, nil
	}
	bHash := bFB.Hash()
	cHash := cFB.Hash()
	if cFB.Time() > bFB.Time() || (cFB.Time() == bFB.Time() && bytes.Compare(cHash[:], bHash[:]) > 0) {
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

	curBlock := bc.currentBlock.Load().(*types.Block)
	// fork
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
		if fBlock.Height() == height && fBlock.Hash() != hash {
			delete(bc.chainForksHead, fHash)
			continue
		}
		// same height and same hash
		if fBlock.Hash() == hash {
			tmp[hash] = []*types.Block{}
			continue
		}

		length := fBlock.Height() - height - 1
		if length > maxLength {
			maxLength = length
		}
		if length>>31 == 1 {
			panic("internal error")
		}
		pars := make([]*types.Block, length+1)
		pars[length] = fBlock
		length--
		parBlock := fBlock
		// get the same height block on current fork
		for parBlock.Height() > height+1 {
			parBlock = bc.GetBlockByHash(parBlock.ParentHash())
			pars[length] = parBlock
			length--
		}
		// current chain and stable chain is same
		if parBlock.ParentHash() != hash {
			delete(bc.chainForksHead, fHash)
		} else {
			tmp[fHash] = pars
		}
	}
	var newCurBlock *types.Block
	// chose current block
	if maxLength == uint32(0) {
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
	if newCurBlock != nil {
		// fork
		oldCurHash := curBlock.Hash()
		newCurHash := newCurBlock.Hash()
		if bytes.Compare(newCurHash[:], oldCurHash[:]) != 0 {
			bc.currentBlock.Store(newCurBlock)
			bc.newBlockNotify(newCurBlock)
			log.Infof("chain forked-2! oldCurHash{ h: %d, hash: %s}, newCurBlock{h:%d, hash: %s}", curBlock.Height(), oldCurHash.Prefix(), newCurBlock.Height(), newCurHash.Prefix())
		}
	} else {
		log.Debug("not have new current block")
	}
	defer func() {
		sb := bc.stableBlock.Load().(*types.Block)
		cb := bc.currentBlock.Load().(*types.Block)
		if cb.Height() < sb.Height() {
			log.Debug("current block's height < stable block's height")
			bc.currentBlock.Store(sb)
		}
	}()
	// notify
	subscribe.Send(subscribe.NewStableBlock, block)
	log.Infof("stable height reach to: %d", height)
	return nil
}

// Verify verify block
func (bc *BlockChain) Verify(block *types.Block) error {
	// verify header
	if err := bc.engine.VerifyHeader(block); err != nil {
		return err
	}
	// verify body
	if err := bc.verifyBody(block); err != nil {
		return err
	}

	hash := block.Hash()
	// execute tx
	newHeader, err := bc.processor.Process(block)
	if err == ErrInvalidTxInBlock {
		return err
	} else if err == nil {
	} else {
		log.Errorf("processor internal error: %v", err)
		panic("processor internal error")
	}

	// verify block hash
	if newHeader.Hash() != hash {
		log.Errorf("verify block error! oldHeader: %v, newHeader:%v", block.Header, newHeader)
		return ErrVerifyBlockFailed
	}
	return nil
}

// verifyBody verify block body
func (bc *BlockChain) verifyBody(block *types.Block) error {
	header := block.Header
	// verify txRoot
	if hash := types.DeriveTxsSha(block.Txs); hash != header.TxRoot {
		log.Errorf("verify block failed. hash:%s height:%d", block.Hash(), block.Height())
		return ErrVerifyBlockFailed
	}
	// verify deputyRoot
	if block.Height()%params.SnapshotBlock == 0 {
		bRoot := types.DeriveDeputyRootSha(block.DeputyNodes)
		nodes := bc.GetNewDeputyNodes()
		selfRoot := types.DeriveDeputyRootSha(nodes)
		if bytes.Compare(bRoot[:], selfRoot[:]) != 0 {
			log.Errorf("verify block failed. deputyNodes not match.block's nodes: %v, self nodes: %v", block.DeputyNodes, nodes)
			return ErrVerifyBlockFailed
		}
	}
	return nil
}

// createSignInfo create sign info for a block
func (bc *BlockChain) createSignInfo(hash common.Hash, height uint32) *network.BlockConfirmData {
	data := &network.BlockConfirmData{
		Hash:   hash,
		Height: height,
	}
	privateKey := deputynode.GetSelfNodeKey()
	signInfo, err := crypto.Sign(hash[:], privateKey)
	if err != nil {
		log.Error("sign for confirm data error")
		return nil
	}
	copy(data.SignInfo[:], signInfo)
	if bc.HasBlock(hash) {
		if err := bc.db.SetConfirm(hash, data.SignInfo); err != nil {
			log.Errorf("SetConfirm failed: %v", err)
		}
	}
	return data
}

// ReceiveConfirm
func (bc *BlockChain) ReceiveConfirm(info *network.BlockConfirmData) (err error) {
	block, err := bc.db.GetBlockByHash(info.Hash)
	if err != nil {
		return ErrBlockNotExist
	}
	height := block.Height()

	// recover public key
	pubKey, err := crypto.Ecrecover(info.Hash[:], info.SignInfo[:])
	if err != nil {
		log.Warnf("Unavailable confirm info. Can't recover signer. hash:%s SignInfo:%s", info.Hash.Hex(), common.ToHex(info.SignInfo[:]))
		return ErrInvalidSignedConfirmInfo
	}
	// get index of signer
	index := bc.getSignerIndex(pubKey[1:], height)
	if index < 0 {
		log.Warnf("Unavailable confirm info. from: %s", common.ToHex(pubKey[1:]))
		return ErrInvalidConfirmInfo
	}
	// has block consensus
	stableBlock := bc.stableBlock.Load().(*types.Block)
	if stableBlock.Height() >= height { // stable block's confirm info
		if ok, err := bc.hasEnoughConfirmInfo(info.Hash, height); err == nil && !ok {
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

	if ok, _ := bc.hasEnoughConfirmInfo(info.Hash, height); ok {
		bc.mux.Lock()
		defer bc.mux.Unlock()
		if err = bc.SetStableBlock(info.Hash, height); err != nil {
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
	nodeCount := deputynode.Instance().GetDeputiesCount(height)
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

// get index of signer in deputy nodes list
func (bc *BlockChain) getSignerIndex(pubKey []byte, height uint32) int {
	node := deputynode.Instance().GetDeputyByNodeID(height, pubKey)
	if node != nil {
		return int(node.Rank)
	}
	return -1
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
	if pack.Hash != (common.Hash{}) && pack.Pack != nil && len(pack.Pack) > 0 {
		if err := bc.db.SetConfirms(pack.Hash, pack.Pack); err != nil {
			log.Debugf("ReceiveConfirms: %v", err)
		}
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

// GetNewDeputyNodes get next epoch deputy nodes for snapshot block
func (bc *BlockChain) GetNewDeputyNodes() deputynode.DeputyNodes {
	result := make(deputynode.DeputyNodes, 0, 17)
	list := bc.db.GetCandidatesTop(bc.CurrentBlock().Hash())
	for i, n := range list {
		dn := new(deputynode.DeputyNode)
		dn.Votes = n.GetTotal()
		acc := bc.am.GetAccount(n.GetAddress())
		profile := acc.GetCandidateProfile()
		strAddr := profile[types.CandidateKeyMinerAddress]
		addr, err := common.StringToAddress(strAddr)
		if err != nil {
			log.Errorf("GetNewDeputyNodes: profile error, addr: %s", strAddr)
			continue
		}
		dn.MinerAddress = addr
		dn.IP = net.ParseIP(profile[types.CandidateKeyHost])
		port, err := strconv.Atoi(profile[types.CandidateKeyPort])
		if err != nil || (port < 100 || port > 65535) {
			log.Errorf("GetNewDeputyNodes: profile error, port: %s", profile[types.CandidateKeyPort])
			continue
		}
		dn.Port = uint32(port)
		dn.Rank = uint32(i)
		strID := profile[types.CandidateKeyNodeID]
		nID, err := hex.DecodeString(strID)
		if err != nil {
			log.Errorf("GetNewDeputyNodes: profile error, NodeID: %s", strID)
			continue
		}
		dn.NodeID = nID
		result = append(result, dn)
	}
	return result
}
