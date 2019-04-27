package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/subscribe"
	"github.com/LemoFoundationLtd/lemochain-core/network"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"time"
)

// DPoVP process the fork logic
type DPoVP struct {
	db     protocol.ChainDB
	dm     *deputynode.Manager
	am     *account.Manager
	txPool TxPool

	stableManager *StableManager  // used to process stable logic
	forkManager   *ForkManager    // forks manager
	validator     *Validator      // block validator
	processor     *TxProcessor    // transaction processor
	assembler     *BlockAssembler // block assembler
	confirmer     *Confirmer      // used to sign block confirm package

	// all dpovp events are here
	stableFeed  subscribe.Feed // stable block change event
	currentFeed subscribe.Feed // head block change event
	confirmFeed subscribe.Feed // new confirm event
}

func NewDPoVP(config Config, db protocol.ChainDB, dm *deputynode.Manager, am *account.Manager, loader BlockLoader, txPool TxPool, stable *types.Block) *DPoVP {
	dpovp := &DPoVP{
		db:            db,
		dm:            dm,
		am:            am,
		txPool:        txPool,
		stableManager: NewStableManager(dm, db),
		forkManager:   NewForkManager(dm, db, stable),
		processor:     NewTxProcessor(config, loader, am, db),
		confirmer:     NewConfirmer(dm, db, stable),
	}
	dpovp.validator = NewValidator(config.MineTimeout, db, dm, dpovp)
	dpovp.assembler = NewBlockAssembler(db, am, dm, dpovp.processor, dpovp)
	return dpovp
}

func (dp *DPoVP) StableBlock() *types.Block {
	return dp.stableManager.StableBlock()
}

func (dp *DPoVP) CurrentBlock() *types.Block {
	return dp.forkManager.GetHeadBlock()
}

func (dp *DPoVP) TxProcessor() *TxProcessor {
	return dp.processor
}

// SubscribeStable subscribe the stable block update notification
func (dp *DPoVP) SubscribeStable(ch chan *types.Block) subscribe.Subscription {
	return dp.stableFeed.Subscribe(ch)
}

// SubscribeCurrent subscribe the current block update notification. The blocks may be not continuous
func (dp *DPoVP) SubscribeCurrent(ch chan *types.Block) subscribe.Subscription {
	return dp.currentFeed.Subscribe(ch)
}

// SubscribeConfirm subscribe the new confirm notification
func (dp *DPoVP) SubscribeConfirm(ch chan *network.BlockConfirmData) subscribe.Subscription {
	return dp.confirmFeed.Subscribe(ch)
}

func (dp *DPoVP) MineBlock(material *BlockMaterial) (*types.Block, error) {
	// mine and seal
	block, err := dp.assembler.MineBlock(dp.CurrentBlock(), material.MinerAddr, material.Extra, dp.txPool, material.MineTimeLimit)
	if err != nil {
		return nil, err
	}
	log.Info("mined a new block", "height", block.Height(), "hash", block.Hash(), "txs count", len(block.Txs))

	// save
	if err := dp.saveToStore(block); err != nil {
		return nil, err
	}

	// update current block
	oldCurrent := dp.CurrentBlock()
	dp.setCurrent(block)
	log.Debugf("Current block changed: %d-%s -> %d-%s", oldCurrent.Height(), oldCurrent.Hash().Prefix(), block.Height(), block.Hash().Prefix())
	dp.txPool.RemoveTxs(block.Txs)

	// update stable block if there are less then 3 deputy nodes
	if _, err = dp.UpdateStable(block); err != nil {
		log.Errorf("can't update stable block. height:%d hash:%s", block.Height(), block.Hash().Prefix())
		return nil, ErrSaveBlock
	}

	// Mined block is always on current fork. So there is no need to switch fork

	return block, nil
}

func (dp *DPoVP) InsertBlock(rawBlock *types.Block) (*types.Block, error) {
	hash := rawBlock.Hash()
	oldCurrent := dp.CurrentBlock()
	oldCurrentHash := oldCurrent.Hash()

	// verify and create a new block witch filled by transaction products
	block, err := dp.VerifyAndSeal(rawBlock)
	if err != nil {
		log.Errorf("block verify failed: %v", err)
		return nil, ErrVerifyBlockFailed
	}

	// sign confirm before save to store. So that we can save the block and confirm in same time
	sig, ok := dp.confirmer.TryConfirm(block)
	if ok {
		go dp.broadcastConfirm(block, sig)
	}

	// save
	if err := dp.saveToStore(block); err != nil {
		return nil, err
	}

	// update current block
	if block.ParentHash() == oldCurrentHash {
		dp.setCurrent(block)
		log.Debugf("Current block changed: %d-%s -> %d-%s", oldCurrent.Height(), oldCurrentHash.Prefix(), block.Height(), block.Hash().Prefix())
	}
	dp.txPool.RemoveTxs(block.Txs)
	// for security
	go func() {
		isEvil := dp.validator.JudgeDeputy(block)
		if isEvil {
			// TODO mark the evil deputy node
		}
	}()

	// try update stable block if there are enough confirms
	stableChanged, err := dp.UpdateStable(block)
	if err != nil {
		log.Errorf("can't check stable block. height:%d hash:%s", block.Height(), hash.Prefix())
		return nil, ErrSaveBlock
	}

	// Maybe a block on other fork is stable now. So we need check if the current fork is still there
	if stableChanged && dp.CheckFork() {
		// If the current is cut, we will choose a best fork. So no need to try switch fork now
		return block, nil
	}
	// The new block is inserted to other fork. So maybe we need to update fork
	if block.ParentHash() != oldCurrentHash {
		dp.TrySwitchFork()
	}

	return block, nil
}

// saveToStore save block and account state to db. They are still unstable now
func (dp *DPoVP) saveToStore(block *types.Block) error {
	hash := block.Hash()
	if err := dp.db.SetBlock(hash, block); err != nil {
		log.Error("Insert block to cache fail", "height", block.Height(), "hash", hash.Hex())
		return ErrSaveBlock
	}
	log.Info("Save block to store", "height", block.Height(), "hash", hash.Prefix(), "time", block.Time(), "parent", block.ParentHash().Prefix())

	if err := dp.am.Save(hash); err != nil {
		log.Error("Save account error!", "height", block.Height(), "hash", hash.Prefix(), "err", err)
		return ErrSaveAccount
	}
	return nil
}

func (dp *DPoVP) broadcastConfirm(block *types.Block, sig types.SignData) {
	// only broadcast confirm info within 3 minutes
	if time.Now().Unix()-int64(block.Time()) >= 3*60 {
		return
	}

	func() {
		pack := &network.BlockConfirmData{
			Hash:     block.Hash(),
			Height:   block.Height(),
			SignInfo: sig,
		}
		dp.confirmFeed.Send(pack)
	}()
}

// BatchConfirm confirm and broadcast unsigned stable blocks one by one
func (dp *DPoVP) batchConfirmStable(startHeight, endHeight uint32) {
	result := dp.confirmer.BatchConfirmStable(startHeight, endHeight)
	for _, confirmPack := range result {
		dp.confirmFeed.Send(confirmPack)
	}
}

// UpdateStable check if the block can be stable. Then send notification and return true if the stable block changed
func (dp *DPoVP) UpdateStable(block *types.Block) (bool, error) {
	oldStable := dp.StableBlock()
	changed, err := dp.stableManager.UpdateStable(block)

	// notify
	if err == nil && changed {
		go dp.stableFeed.Send(block)
		go dp.batchConfirmStable(oldStable.Height()+1, dp.StableBlock().Height()-1)
	}

	return changed, err
}

// TrySwitchFork try to switch to a better fork
func (dp *DPoVP) TrySwitchFork() {
	oldCurrent := dp.CurrentBlock()

	// try to switch fork
	newCurrent, switched := dp.forkManager.TrySwitchFork(dp.StableBlock(), oldCurrent)
	if !switched {
		return
	}

	dp.setCurrent(newCurrent)
}

// CheckFork check the current fork and update it if it is cut. Return true if the current fork change
func (dp *DPoVP) CheckFork() bool {
	oldCurrent := dp.CurrentBlock()

	// Test if currentBlock is still there. It may be pruned by stable block updating
	_, err := dp.db.GetUnConfirmByHeight(oldCurrent.Height(), oldCurrent.Hash())
	if err == nil || err != store.ErrNotExist {
		return false
	}

	// The current block is cut. Choose the longest fork to be new current block
	dp.setCurrent(dp.forkManager.ChooseNewFork())
	return true
}

// setCurrent update current block and send a notification
func (dp *DPoVP) setCurrent(block *types.Block) {
	if block == nil {
		block = dp.StableBlock()
	}

	oldCurrent := dp.CurrentBlock()
	if oldCurrent.Hash() == block.Hash() {
		return
	}

	dp.forkManager.SetHeadBlock(block)

	dp.logCurrentChange(oldCurrent)

	// To confirm a block from another fork, we need a height distance that more than 2/3 deputies count.
	// But the new current's height is 2/3 deputies count bigger at most, so we don't need to try to confirm the new current block here
	// dp.confirmer.TryConfirm(block)

	// notify
	go func() {
		dp.currentFeed.Send(block)
	}()
}

func (dp *DPoVP) logCurrentChange(oldCurrent *types.Block) {
	newCurrent := dp.CurrentBlock()
	if newCurrent.Hash() == oldCurrent.Hash() {
		log.Debugf("Insert to another fork chain. current block is still [%d]%s", oldCurrent.Height(), oldCurrent.Hash().Prefix())
	} else if newCurrent.ParentHash() == oldCurrent.Hash() {
		log.Debugf("Current fork length +1. current block change from [%d]%s to [%d]%s", oldCurrent.Height(), oldCurrent.Hash().Prefix(), newCurrent.Height(), newCurrent.Hash().Prefix())
	} else {
		log.Debugf("Switch fork! current block change from [%d]%s to [%d]%s", oldCurrent.Height(), oldCurrent.Hash().Prefix(), newCurrent.Height(), newCurrent.Hash().Prefix())
	}
}

// isIgnorableBlock check the block is exist or not
func (dp *DPoVP) isIgnorableBlock(block *types.Block) bool {
	if has, _ := dp.db.IsExistByHash(block.Hash()); has {
		return true
	}
	if dp.StableBlock().Height() >= block.Height() {
		// the block may not correct, it is dangerous
		log.Debug("ignore the block whose height is smaller than stable block")
		return true
	}
	return false
}

// VerifyAndSeal verify block then create a new block
func (dp *DPoVP) VerifyAndSeal(block *types.Block) (*types.Block, error) {
	// ignore exist block as soon as possible
	if ok := dp.isIgnorableBlock(block); ok {
		return nil, ErrExistBlock
	}

	// verify every things that can be verified before tx processing
	if err := dp.validator.VerifyBeforeTxProcess(block); err != nil {
		return nil, ErrInvalidBlock
	}
	// filter the valid confirms
	confirms := block.Confirms
	block.Confirms = nil
	block.Confirms, _ = dp.validator.VerifyNewConfirms(block, confirms, dp.dm)

	// parse block, change local state and seal a new block
	newBlock, err := dp.assembler.RunBlock(block)
	if err != nil {
		if err == ErrInvalidTxInBlock {
			return nil, ErrInvalidBlock
		}
		log.Errorf("RunBlock internal error: %v", err)
		// panic("processor internal error")
		return nil, err
	}

	// verify the things computed by tx
	if err := dp.validator.VerifyAfterTxProcess(block, newBlock); err != nil {
		return nil, ErrInvalidBlock
	}
	return newBlock, nil
}

func (dp *DPoVP) InsertConfirm(info *network.BlockConfirmData) error {
	validConfirms, err := dp.validator.VerifyConfirmPacket(info.Height, info.Hash, []types.SignData{info.SignInfo})
	if len(validConfirms) == 0 {
		return err
	}
	block, _ := dp.db.GetBlockByHash(info.Hash)

	// save
	if err = dp.confirmer.SaveConfirm(block, validConfirms); err != nil {
		log.Errorf("InsertConfirm failed: %v", err)
		return nil
	}

	changed, err := dp.UpdateStable(block)
	if err != nil {
		log.Errorf("ReceiveConfirm: setStableBlock failed. height: %d, hash:%s, err: %v", info.Height, info.Hash.Hex()[:16], err)
		return err
	}

	// maybe the current block is cut
	if changed {
		dp.CheckFork()
	}
	return nil
}

// ReceiveStableConfirms receive confirm package from net connection. The block of these confirms has been confirmed by its son block already
func (dp *DPoVP) InsertStableConfirms(pack network.BlockConfirms) {
	if pack.Hash == (common.Hash{}) || pack.Pack == nil || len(pack.Pack) == 0 {
		return
	}
	validConfirms, err := dp.validator.VerifyConfirmPacket(pack.Height, pack.Hash, pack.Pack)
	if len(validConfirms) == 0 {
		log.Debugf("InsertStableConfirms: %v", err)
		return
	}

	block, _ := dp.db.GetBlockByHash(pack.Hash)
	if err := dp.confirmer.SaveConfirm(block, validConfirms); err != nil {
		log.Debugf("InsertStableConfirms: %v", err)
	}
}

// SnapshotDeputyNodes get next epoch deputy nodes for snapshot block
func (dp *DPoVP) LoadTopCandidates(blockHash common.Hash) deputynode.DeputyNodes {
	result := make(deputynode.DeputyNodes, 0, dp.dm.DeputyCount)
	list := dp.db.GetCandidatesTop(blockHash)
	if len(list) > dp.dm.DeputyCount {
		list = list[:dp.dm.DeputyCount]
	}

	for i, n := range list {
		acc := dp.am.GetAccount(n.GetAddress())
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
