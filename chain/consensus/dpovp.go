package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/transaction"
	"github.com/LemoFoundationLtd/lemochain-core/chain/txpool"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/subscribe"
	"github.com/LemoFoundationLtd/lemochain-core/metrics"
	"github.com/LemoFoundationLtd/lemochain-core/network"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"sync"
	"time"
)

var (
	blockInsertTimer   = metrics.NewTimer(metrics.BlockInsert_timerName)   // 统计区块插入链中的速率和所用时间的分布情况
	mineBlockTimer     = metrics.NewTimer(metrics.MineBlock_timerName)     // 统计出块速率和时间分布
	verifyBlockMeter   = metrics.NewMeter(metrics.VerifyBlock_meterName)   // 统计验证区块失败的频率
	unStableBlockMeter = metrics.NewMeter(metrics.UnStableBlock_meterName) // 未稳定区块过多，可能影响到换届
)

// DPoVP process the fork logic
type DPoVP struct {
	db         protocol.ChainDB
	dm         *deputynode.Manager
	am         *account.Manager
	txPool     *txpool.TxPool
	txGuard    *txpool.TxGuard
	minerExtra []byte // Extra data in mined block header. It is short than 256bytes

	stableManager *StableManager           // used to process stable logic
	forkManager   *ForkManager             // forks manager
	validator     *Validator               // block validator
	processor     *transaction.TxProcessor // transaction processor
	assembler     *BlockAssembler          // block assembler
	confirmer     *Confirmer               // used to sign block confirm package

	// show chain change detail in log
	logForks bool
	// lock if need change chain state
	chainLock sync.Mutex

	// all dpovp events are here
	stableFeed        subscribe.Feed // stable block change event
	currentFeed       subscribe.Feed // head block change event
	confirmFeed       subscribe.Feed // new confirm event
	fetchConfirmsFeed subscribe.Feed // fetch confirms event
}

const delayFetchConfirmsTime = time.Second * 30

func NewDPoVP(config Config, db protocol.ChainDB, dm *deputynode.Manager, am *account.Manager, loader transaction.ParentBlockLoader, txPool *txpool.TxPool, txGuard *txpool.TxGuard) *DPoVP {
	stable, _ := db.LoadLatestBlock()
	dpovp := &DPoVP{
		db:            db,
		dm:            dm,
		am:            am,
		txPool:        txPool,
		txGuard:       txGuard,
		stableManager: NewStableManager(dm, db),
		forkManager:   NewForkManager(dm, db, stable),
		processor:     transaction.NewTxProcessor(config.RewardManager, config.ChainID, loader, am, db, dm),
		confirmer:     NewConfirmer(dm, db, db, db),
		minerExtra:    config.MinerExtra,
		logForks:      config.LogForks,
	}
	dpovp.validator = NewValidator(config.MineTimeout, db, dm, txGuard, dpovp)
	dpovp.assembler = NewBlockAssembler(am, dm, dpovp.processor, dpovp)
	return dpovp
}

func (dp *DPoVP) StableBlock() *types.Block {
	return dp.stableManager.StableBlock()
}

func (dp *DPoVP) CurrentBlock() *types.Block {
	return dp.forkManager.GetHeadBlock()
}
func (dp *DPoVP) TxGuard() *txpool.TxGuard {
	return dp.txGuard
}

func (dp *DPoVP) TxProcessor() *transaction.TxProcessor {
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

// SubscribeFetchConfirm subscribe fetch block confirms
func (dp *DPoVP) SubscribeFetchConfirm(ch chan []network.GetConfirmInfo) subscribe.Subscription {
	return dp.fetchConfirmsFeed.Subscribe(ch)
}

func (dp *DPoVP) MineBlock(txProcessTimeout int64) (*types.Block, error) {
	defer mineBlockTimer.UpdateSince(time.Now())

	dp.chainLock.Lock()
	defer dp.chainLock.Unlock()
	parentHeader := dp.CurrentBlock().Header
	log.Debug("🔨 Start mine block", "height", parentHeader.Height+1)
	// mine and seal
	header, err := dp.assembler.PrepareHeader(parentHeader, dp.minerExtra)
	if err != nil {
		return nil, err
	}
	err = dp.validator.VerifyMiner(header, parentHeader)
	if err != nil {
		log.Warn("Mining is stuck by something or stable block changed. we have to wait to next mine window")
		return nil, err
	}

	txs := dp.txPool.GetTxs(header.Time, params.MaxTxsForMiner)
	log.Debugf("pick %d txs from txPool", len(txs))
	block, invalidTxs, err := dp.assembler.MineBlock(header, txs, txProcessTimeout)
	if err != nil {
		return nil, err
	}
	log.Info("Mined a new block", "block", block.ShortString(), "txsCount", len(block.Txs))
	// remove invalid txs from pool
	dp.txPool.DelTxs(invalidTxs)

	// save
	if err = dp.saveNewBlock(block); err != nil {
		return nil, err
	}
	return block, nil
}

func (dp *DPoVP) InsertBlock(rawBlock *types.Block) (*types.Block, error) {
	defer blockInsertTimer.UpdateSince(time.Now())

	// ignore exist block as soon as possible
	if ok := dp.isIgnorableBlock(rawBlock); ok {
		return nil, ErrIgnoreBlock
	}

	dp.chainLock.Lock()
	defer dp.chainLock.Unlock()
	log.Debug("🎁 Start insert block to chain", "block", rawBlock.ShortString())

	// verify and create a new block witch filled by transaction products
	block, err := dp.VerifyAndSeal(rawBlock)
	if err != nil {
		verifyBlockMeter.Mark(1) // 统计调用频率用
		log.Errorf("block verify failed: %v", err)
		return nil, ErrVerifyBlockFailed
	}

	// sign confirm before save to store. So that we can save and confirm the block in the same time
	sig, ok := dp.confirmer.TryConfirm(block)
	if ok {
		go dp.broadcastConfirm(block, sig)
	}

	// save
	if err = dp.saveNewBlock(block); err != nil {
		return nil, err
	}

	// for security
	go func() {
		if isEvil := dp.validator.JudgeDeputy(block); isEvil {
			dp.dm.PutEvilDeputyNode(block.MinerAddress(), block.Height())
		}
	}()

	return block, nil
}

// saveNewBlock save block then update the current and stable block
func (dp *DPoVP) saveNewBlock(block *types.Block) error {
	// save
	if err := dp.saveToStore(block); err != nil {
		return err
	}
	dp.txGuard.SaveBlock(block)

	// save last sig because we are the miner. If we clear db and restart, this will be useful
	if IsMinedByself(block) {
		dp.confirmer.SetLastSig(block)
	}
	// try update stable block if there are enough confirms
	stableChanged, err := dp.UpdateStable(block)
	if err != nil {
		log.Errorf("update stable block %s fail", block.ShortString())
		return ErrSaveBlock
	}

	// try update current block or switch to another fork
	oldCurrent := dp.CurrentBlock()
	currentChanged := dp.forkManager.UpdateFork(block, dp.StableBlock())
	if currentChanged {
		dp.onCurrentChanged(oldCurrent, dp.CurrentBlock())
	} else {
		// 该块插入到了其他分支上，把该block中的交易push到本分支状态的交易池中
		dp.txPool.AddTxs(block.Txs)
	}

	// 如果是出现了新的稳定块
	if stableChanged {
		dp.onStableChanged(block)
	}

	// To confirm a block from another fork, we need a height distance that more than 2/3 deputies count.
	// But the new current's height is 2/3 deputies count bigger at most, so we don't need to try to confirm the new current block here
	// dp.confirmer.TryConfirm(block)

	// 当前分支未稳定区块个数等于过渡期区块的十分之九则需要告警
	if dp.CurrentBlock().Height()-dp.StableBlock().Height() == params.InterimDuration*9/10 {
		unStableBlockMeter.Mark(1)
	}
	dp.logCurrentChange(oldCurrent)

	return nil
}

// onCurrentChanged
func (dp *DPoVP) onCurrentChanged(oldCurrent, newCurrent *types.Block) {
	if newCurrent.ParentHash() == oldCurrent.Hash() {
		// remove the transactions on new current block
		dp.txPool.DelTxs(newCurrent.Txs)
	} else {
		// fork switched!
		// get the transactions from old fork and new fork to the same parent of them
		oldForkTxs, newForkTxs, err := dp.txGuard.GetTxsByBranch(oldCurrent, newCurrent)
		if err != nil {
			log.Errorf("Get branch txs error. error: %v", err)
		}
		// TODO diff transaction lists
		// put the transactions on old fork to tx pool
		dp.txPool.AddTxs(oldForkTxs)
		// remove the transactions on new fork from tx pool
		dp.txPool.DelTxs(newForkTxs)
	}

	// send current block change event
	go dp.currentFeed.Send(newCurrent)
}

// onStableChanged
func (dp *DPoVP) onStableChanged(newStable *types.Block) {
	dp.txGuard.DelOldBlocks(newStable.Time())
}

// saveToStore save block and account state to db. They are still unstable now
func (dp *DPoVP) saveToStore(block *types.Block) error {
	hash := block.Hash()
	if err := dp.db.SetBlock(hash, block); err != nil {
		log.Error("Insert block to cache fail", "block", block.ShortString())
		return ErrSaveBlock
	}
	log.Info("Save block to store", "block", block.ShortString(), "time", block.Time(), "parent", block.ParentHash())

	if err := dp.am.Save(hash); err != nil {
		log.Error("Save account error!", "block", block.ShortString(), "err", err)
		return ErrSaveAccount
	}
	return nil
}

func (dp *DPoVP) broadcastConfirm(block *types.Block, sig types.SignData) {
	// only broadcast confirm info within 3 minutes
	if time.Now().Unix()-int64(block.Time()) >= 3*60 {
		return
	}

	pack := &network.BlockConfirmData{
		Hash:     block.Hash(),
		Height:   block.Height(),
		SignInfo: sig,
	}
	dp.confirmFeed.Send(pack)
}

// fetchConfirmsFromRemote fetch confirms from remote peer after 30s
func (dp *DPoVP) fetchConfirmsFromRemote(startHeight, endHeight uint32) {
	// time.AfterFunc its own goroutine
	time.AfterFunc(delayFetchConfirmsTime, func() {
		info := dp.confirmer.NeedConfirmList(startHeight, endHeight)
		if info == nil || len(info) == 0 {
			return
		}
		dp.fetchConfirmsFeed.Send(info)
	})
}

// BatchConfirm confirm and broadcast unsigned stable blocks one by one
func (dp *DPoVP) batchConfirmStable(startHeight, endHeight uint32) {
	result := dp.confirmer.BatchConfirmStable(startHeight, endHeight)
	for _, confirmPack := range result {
		dp.confirmFeed.Send(confirmPack)
	}
}

// saveSnapshot find snapshot block than save its deputy nodes
func (dp *DPoVP) saveSnapshot(startHeight, endHeight uint32) {
	for i := startHeight; i <= endHeight; i++ {
		if deputynode.IsSnapshotBlock(i) {
			block, err := dp.db.GetBlockByHeight(i)
			if err != nil {
				log.Error("load block for snapshot fail", "height", i)
			} else {
				dp.dm.SaveSnapshot(i, block.DeputyNodes)
			}
		}
	}
}

// UpdateStable check if the block can be stable. Then send notification and return true if the stable block changed
func (dp *DPoVP) UpdateStable(block *types.Block) (bool, error) {
	oldStable := dp.StableBlock()
	changed, _, err := dp.stableManager.UpdateStable(block)
	if err != nil {
		return false, err
	}

	if changed {
		// Update deputy nodes map
		// This may not be a litter late, but it's fine. Because deputy nodes snapshot will be used after the interim duration, it's about 1000 blocks
		dp.saveSnapshot(oldStable.Height()+1, dp.StableBlock().Height())

		// notify new stable
		go dp.stableFeed.Send(block)
		// confirm from oldStable to newStable
		go dp.batchConfirmStable(oldStable.Height()+1, dp.StableBlock().Height())
		// after 30s fetch confirms from peer
		go dp.fetchConfirmsFromRemote(oldStable.Height()+1, dp.StableBlock().Height())
	}

	return changed, nil
}

func (dp *DPoVP) logCurrentChange(oldCurrent *types.Block) {
	newCurrent := dp.CurrentBlock()
	if newCurrent.Hash() == oldCurrent.Hash() {
		log.Debugf("Current block is still %s", oldCurrent.ShortString())
	} else if newCurrent.ParentHash() == oldCurrent.Hash() {
		log.Debugf("Current fork length +1. Current block changes from %s to %s", oldCurrent.ShortString(), newCurrent.ShortString())
	} else {
		log.Debugf("Switch fork! Current block changes from %s to %s", oldCurrent.ShortString(), newCurrent.ShortString())
	}
	if dp.logForks {
		log.Debug(dp.db.SerializeForks(newCurrent.Hash()))
	}
}

// isIgnorableBlock check the block is exist or not
func (dp *DPoVP) isIgnorableBlock(block *types.Block) bool {
	if has, _ := dp.db.IsExistByHash(block.Hash()); has {
		log.Debug("Ignore the existed block")
		return true
	}
	if dp.StableBlock().Height() >= block.Height() {
		// the block may not correct, it is not verified
		log.Debug("Ignore the block whose height is not greater than the stable block")
		return true
	}
	return false
}

// VerifyAndSeal verify block then create a new block
func (dp *DPoVP) VerifyAndSeal(block *types.Block) (*types.Block, error) {
	// verify every things that can be verified before tx processing
	if err := dp.validator.VerifyBeforeTxProcess(block, dp.processor.ChainID); err != nil {
		return nil, ErrInvalidBlock
	}
	// filter the valid confirms
	confirms := block.Confirms
	block.Confirms = nil
	block.Confirms, _ = dp.validator.VerifyNewConfirms(block, confirms, dp.dm)
	log.Debug("Verify confirms done", "validCount", len(block.Confirms))

	// parse block, change local state and seal a new block
	newBlock, err := dp.assembler.RunBlock(block)
	if err != nil {
		if err == transaction.ErrInvalidTxInBlock {
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
	dp.chainLock.Lock()
	defer dp.chainLock.Unlock()
	oldCurrent := dp.CurrentBlock()
	log.Debug("👍 Start insert confirm", "height", info.Height, "hash", info.Hash[:3])

	newBlock, err := dp.insertConfirms(info.Height, info.Hash, []types.SignData{info.SignInfo})
	if err != nil {
		log.Warnf("InsertConfirm failed: %v", err)
		return err
	}

	stableChanged, err := dp.UpdateStable(newBlock)
	if err != nil {
		log.Errorf("ReceiveConfirm: setStableBlock failed. height: %d, hash:%s, err: %v", info.Height, info.Hash.Hex()[:16], err)
		return err
	}

	// update the current block
	currentChanged := dp.forkManager.UpdateForkForConfirm(dp.StableBlock())
	if currentChanged {
		dp.onCurrentChanged(oldCurrent, dp.CurrentBlock())
		dp.logCurrentChange(oldCurrent)
	}

	if stableChanged {
		dp.onStableChanged(newBlock)
	}

	return nil
}

// InsertStableConfirms receive confirm package from net connection. The block of these confirms has been confirmed by its son block already
func (dp *DPoVP) InsertStableConfirms(pack network.BlockConfirms) {
	_, err := dp.insertConfirms(pack.Height, pack.Hash, pack.Pack)
	if err != nil {
		log.Warnf("InsertStableConfirms fail: %v", err)
	}
}

// insertConfirms save signature list to store, then return a new block
func (dp *DPoVP) insertConfirms(height uint32, blockHash common.Hash, sigList []types.SignData) (*types.Block, error) {
	if len(sigList) == 0 {
		return nil, ErrIgnoreConfirm
	}
	block, err := dp.db.GetBlockByHash(blockHash)
	if err != nil {
		return nil, ErrBlockNotExist
	}
	if IsConfirmEnough(block, dp.dm) {
		return nil, ErrIgnoreConfirm
	}
	validConfirms, err := dp.validator.VerifyConfirmPacket(height, blockHash, sigList)
	if len(validConfirms) == 0 {
		return nil, err
	}

	return dp.confirmer.SaveConfirm(block, validConfirms)
}

// SnapshotDeputyNodes get next epoch deputy nodes for snapshot block
func (dp *DPoVP) LoadTopCandidates(blockHash common.Hash) types.DeputyNodes {
	result := make(types.DeputyNodes, 0, dp.dm.DeputyCount)
	list := dp.db.GetCandidatesTop(blockHash)
	if len(list) > dp.dm.DeputyCount {
		list = list[:dp.dm.DeputyCount]
	}

	for i, n := range list {
		acc := dp.am.GetAccount(n.GetAddress())
		candidate := acc.GetCandidate()
		strID := candidate[types.CandidateKeyNodeID]
		dn := types.NewDeputyNode(acc.GetVotes(), uint32(i), n.GetAddress(), strID)
		result = append(result, dn)
	}
	return result
}

// LoadRefundCandidates get the address list of candidates who need to refund
func (dp *DPoVP) LoadRefundCandidates(height uint32) ([]common.Address, error) {
	result := make([]common.Address, 0)
	addrList, err := dp.db.GetAllCandidates()
	if err != nil {
		log.Errorf("Load all candidates fail: %v", err)
		return nil, err
	}
	for _, addr := range addrList {
		// 判断addr的candidate信息
		candidateAcc := dp.am.GetAccount(addr)
		depositString := candidateAcc.GetCandidateState(types.CandidateKeyDepositAmount)
		nodeId := candidateAcc.GetCandidateState(types.CandidateKeyNodeID)
		if candidateAcc.GetCandidateState(types.CandidateKeyIsCandidate) == types.NotCandidateNode && depositString != "" { // 满足退还押金的条件
			// 判断该地址是否为本届的共识节点
			if !dp.dm.IsNodeDeputy(height, common.FromHex(nodeId)) {
				result = append(result, addr)
			}
		}
	}
	return result, nil
}
