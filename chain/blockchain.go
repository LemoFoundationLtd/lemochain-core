package chain

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/flag"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/subscribe"
	"github.com/LemoFoundationLtd/lemochain-core/network"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	db "github.com/LemoFoundationLtd/lemochain-core/store/protocol"
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
	genesisBlock *types.Block // genesis block
	lastSig      blockSig

	mux       sync.Mutex
	engine    Engine       // consensus engine
	processor *TxProcessor // state processor
	running   int32

	NewCurrentBlockFeed subscribe.Feed

	setStableMux sync.Mutex

	quitCh chan struct{}
}

type blockSig struct {
	Height uint32
	Hash   common.Hash
}

func NewBlockChain(chainID uint16, engine Engine, dm *deputynode.Manager, db db.ChainDB, flags flag.CmdFlags) (bc *BlockChain, err error) {
	bc = &BlockChain{
		chainID: chainID,
		db:      db,
		dm:      dm,
		// newBlockCh:     newBlockCh,
		flags:  flags,
		engine: engine,
		quitCh: make(chan struct{}),
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
	bc.am = account.NewManager(block.Hash(), bc.db)
	bc.lastSig.Height = block.Height()
	bc.lastSig.Hash = block.Hash()
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
	if height > currentBlock.Height() {
		return nil
	}

	var block *types.Block
	var err error
	if height <= bc.StableBlock().Height() {
		// stable block
		block, err = bc.db.GetBlockByHeight(height)
	} else {
		// unstable block
		block, err = bc.db.GetUnConfirmByHeight(height, currentBlock.Hash())
	}
	if err != nil {
		log.Error("load stable parent block fail", "height", height, "err", err)
		return nil
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
	block, err := bc.db.LoadLatestBlock()
	if err != nil {
		log.Warn("load stable block fail")
		// We would make sure genesis is available at least. So err is not tolerable
		panic(err)
	}
	return block
}

// SetMinedBlock 挖到新块
func (bc *BlockChain) SetMinedBlock(block *types.Block) error {
	sb := bc.StableBlock()
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

	// update current block
	oldCurrent := bc.currentBlock.Load().(*types.Block)
	bc.currentBlock.Store(block)
	log.Debugf("current block changed: %d-%s -> %d-%s", oldCurrent.Height(), oldCurrent.Hash().Prefix(), block.Height(), block.Hash().Prefix())

	// update stable block if there are less then 3 deputy nodes
	if err = bc.UpdateStable(block); err != nil {
		log.Errorf("can't update stable block. height:%d hash:%s", block.Height(), block.Hash().Prefix())
		return ErrSaveBlock
	}

	// notify
	go func() {
		bc.NewCurrentBlockFeed.Send(block)
		subscribe.Send(subscribe.NewMinedBlock, block)
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

// InsertChain insert block of non-self to chain
func (bc *BlockChain) InsertChain(rawBlock *types.Block, isSynchronising bool) (err error) {
	bc.mux.Lock()
	defer bc.mux.Unlock()

	hash := rawBlock.Hash()
	oldCurrent := bc.currentBlock.Load().(*types.Block)
	oldCurrentHash := oldCurrent.Hash()

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

	// update current block
	if block.ParentHash() == oldCurrentHash {
		bc.currentBlock.Store(block)
		log.Debugf("current block changed: %d-%s -> %d-%s", oldCurrent.Height(), oldCurrentHash.Prefix(), block.Height(), block.Hash().Prefix())
	}
	// try update stable block if it has enough confirms
	if err = bc.UpdateStable(block); err != nil {
		log.Errorf("can't check stable block. height:%d hash:%s", block.Height(), hash.Prefix())
		return ErrSaveBlock
	}
	newCurrent := bc.CurrentBlock()
	newCurrentHash := newCurrent.Hash()

	// some logs
	if newCurrent.ParentHash() == oldCurrentHash {
		log.Debug("current chain length +1")
	} else if newCurrentHash == oldCurrentHash {
		log.Debug("insert to fork chain")
	} else {
		log.Debug("switch fork")
	}
	log.Debugf("current block: %d, %s, parent: %s", newCurrent.Height(), newCurrentHash.Prefix(), newCurrent.ParentHash().Prefix())

	// for security
	go bc.JudgeDeputy(block)

	return nil
}

// JudgeDeputy check if the deputy node is evil by his new block
func (bc *BlockChain) JudgeDeputy(newBlock *types.Block) {
	// check if the deputy mine two blocks at same height
	bc.db.IterateUnConfirms(func(node *types.Block) {
		if node.Height() == newBlock.Height() && node.Hash() != newBlock.Hash() {
			nodeID, err := newBlock.SignerNodeID()
			if err != nil {
				log.Error("no NodeID, can't judge the deputy", "err", err)
				return
			}
			log.Warnf("The deputy %x is evil !!! It mined block %s and %s at same height %d", nodeID, newBlock.Hash().Prefix(), node.Hash().Prefix(), newBlock.Height())
			// TODO add the deputy to blacklist
		}
	})
}

// UpdateStable check if the block can be stable. Then send notification if the stable block changed
func (bc *BlockChain) UpdateStable(block *types.Block) error {
	bc.setStableMux.Lock()
	defer bc.setStableMux.Unlock()

	var (
		hash           = block.Hash()
		oldStable      = bc.StableBlock()
		oldCurrent     = bc.currentBlock.Load().(*types.Block)
		oldCurrentHash = oldCurrent.Hash()
	)
	if block.Height() <= oldStable.Height() {
		return nil
	}

	// update stable block
	if bc.engine.CanBeStable(block.Height(), len(block.Confirms)) {
		if err := bc.db.SetStableBlock(hash); err != nil {
			log.Errorf("SetStableBlock error. height:%d hash:%s, err:%s", block.Height(), common.ToHex(hash[:]), err.Error())
			return ErrSetStableBlockToDB
		}
		// TODO confirm from oldStable to newStable in coroutine
		bc.updateDeputyNodes(block)
	}
	newStable := bc.StableBlock()

	// update fork
	stableChanged := newStable.Hash() != oldStable.Hash()
	if stableChanged {
		bc.CheckFork()
	}

	// notify
	go func() {
		if stableChanged {
			log.Debugf("stable block changed: %d-%s -> %d-%s", oldCurrent.Height(), oldCurrentHash.Prefix(), newStable.Height(), newStable.Hash().Prefix())
			subscribe.Send(subscribe.NewStableBlock, block)
		}
	}()
	return nil
}

// CheckFork check and update the current fork
func (bc *BlockChain) CheckFork() *types.Block {
	var (
		oldCurrent = bc.currentBlock.Load().(*types.Block)
		newCurrent *types.Block
	)
	// TODO 这里需要把最新收到的块传进来，假如不切分支的正常情况，需要让current高度+1。可能UpdateCurrent要从UpdateCurrentAndStable里拆出去

	// Test if currentBlock is still there. It may be pruned by stable block updating
	if _, err := bc.db.GetUnConfirmByHeight(oldCurrent.Height(), oldCurrent.Hash()); err == store.ErrNotExist {
		// choose the longest fork to be new current block
		newCurrent = bc.engine.ChooseNewFork()
		if newCurrent == nil {
			newCurrent = bc.StableBlock()
		}
	} else {
		// try to switch fork
		newCurrent = bc.engine.TrySwitchFork(bc.StableBlock(), oldCurrent)
	}
	bc.currentBlock.Store(newCurrent)

	// notify
	go func() {
		currentChanged := newCurrent.Hash() != oldCurrent.Hash()
		if currentChanged {
			log.Debugf("switch fork! current block changed: %d-%s -> %d-%s", oldCurrent.Height(), oldCurrent.Hash().Prefix(), newCurrent.Height(), newCurrent.Hash().Prefix())
			bc.NewCurrentBlockFeed.Send(newCurrent)
		}

		if bc.dm.IsSelfDeputyNode(newCurrent.Height()) && bc.needConfirmCurrent() {
			bc.confirmCurrent()
		}
	}()
	return newCurrent
}

// isIgnorableBlock check the block is exist or not
func (bc *BlockChain) isIgnorableBlock(block *types.Block) bool {
	if has, _ := bc.db.IsExistByHash(block.Hash()); has {
		return true
	}
	if bc.StableBlock().Height() >= block.Height() {
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
	if err = bc.engine.Finalize(block.Header.Height, bc.am); err != nil {
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

// signBlock sign a block and return signData
func (bc *BlockChain) signBlock(hash common.Hash) (types.SignData, error) {
	// sign
	privateKey := deputynode.GetSelfNodeKey()
	sig, err := crypto.Sign(hash[:], privateKey)
	if err != nil {
		return types.SignData{}, err
	}
	var signData types.SignData
	copy(signData[:], sig)
	return signData, nil
}

func (bc *BlockChain) needConfirmCurrent() bool {
	block := bc.CurrentBlock()
	// the block is at same fork with last signed block
	if block.ParentHash() == bc.lastSig.Hash {
		return true
	}
	// the block is deputyCount*2/3 far from signed block
	nodeCount := bc.dm.GetDeputiesCount(block.Height())
	signDistance := twoThird(nodeCount)
	if block.Height() > bc.lastSig.Height+signDistance {
		return true
	}

	return false
}

// confirmCurrent confirm current block and send confirm event
func (bc *BlockChain) confirmCurrent() {
	block := bc.CurrentBlock()
	hash := block.Hash()
	sig, err := bc.signBlock(hash)
	if err != nil {
		log.Error("sign for confirm data error", "err", err)
	}
	// save
	if err := bc.db.SetConfirm(hash, sig); err != nil {
		log.Errorf("SetConfirm failed: %v", err)
	}

	bc.lastSig.Height = block.Height()
	bc.lastSig.Hash = hash

	// only broadcast confirm info within 3 minutes
	if time.Now().Unix()-int64(block.Time()) < 3*60 {
		// notify
		subscribe.Send(subscribe.NewConfirm, &network.BlockConfirmData{
			Hash:     bc.lastSig.Hash,
			Height:   bc.lastSig.Height,
			SignInfo: sig,
		})
		log.Debug("confirm notify", "height", block.Height())
	}
}

func (bc *BlockChain) hasSigned(hash common.Hash) (bool, error) {
	if hash == bc.lastSig.Hash {
		return true, nil
	}

	confirms, err := bc.db.GetConfirms(hash)
	if err != nil {
		log.Error("load confirms fail", "err", err)
		return false, err
	}
	sig, err := bc.signBlock(hash)
	if err != nil {
		log.Error("sign for confirm data error", "err", err)
		return false, err
	}
	for _, confirm := range confirms {
		if confirm == sig {
			return true, nil
		}
	}

	return false, nil
}

// ReceiveConfirm
func (bc *BlockChain) ReceiveConfirm(info *network.BlockConfirmData) (err error) {
	err = bc.engine.VerifyConfirmPacket(info.Height, info.Hash, []types.SignData{info.SignInfo})
	if err != nil {
		return err
	}
	block, err := bc.db.GetBlockByHash(info.Hash)
	if err != nil {
		log.Errorf("load block for confirm fail. hash:%s. error: %v", info.Hash.Hex(), err)
		return err
	}

	// confirm for a stable block
	if bc.StableBlock().Height() >= info.Height { // stable block's confirm info
		if !bc.engine.CanBeStable(info.Height, len(block.Confirms)) {
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

	if bc.engine.CanBeStable(info.Height, len(block.Confirms)+1) {
		bc.mux.Lock()
		defer bc.mux.Unlock()

		if err = bc.UpdateStable(block); err != nil {
			log.Errorf("ReceiveConfirm: setStableBlock failed. height: %d, hash:%s, err: %v", info.Height, info.Hash.Hex()[:16], err)
			return err
		}
	}
	return nil
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

// ReceiveConfirms receive confirm package from net connection. The block of these confirms has been confirmed by its son block already
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
