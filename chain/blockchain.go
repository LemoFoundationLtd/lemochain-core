package chain

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/consensus"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/transaction"
	"github.com/LemoFoundationLtd/lemochain-core/chain/txpool"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/flag"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/subscribe"
	"github.com/LemoFoundationLtd/lemochain-core/network"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	db "github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"sync/atomic"
)

var ErrNoGenesis = errors.New("can't get genesis block")
var ErrLoadBlock = errors.New("load block fail")

type BlockChain struct {
	chainID      uint16
	flags        flag.CmdFlags
	genesisBlock *types.Block // genesis block

	db     db.ChainDB
	am     *account.Manager
	dm     *deputynode.Manager
	engine *consensus.DPoVP

	stopped int32
	quitCh  chan struct{}
}

// Config holds chain options.
type Config struct {
	ChainID     uint16
	MineTimeout uint64 // milliseconds
}

func NewBlockChain(config Config, dm *deputynode.Manager, db db.ChainDB, flags flag.CmdFlags, txPool *txpool.TxPool) (bc *BlockChain, err error) {
	bc = &BlockChain{
		chainID: config.ChainID,
		db:      db,
		dm:      dm,
		flags:   flags,
		quitCh:  make(chan struct{}),
	}
	bc.genesisBlock, err = bc.db.GetBlockByHeight(0)
	if err != nil {
		return nil, ErrNoGenesis
	}

	// stable latestStableBlock
	latestStableBlock, err := bc.db.LoadLatestBlock()
	if err != nil {
		log.Errorf("Can't load last state: %v", err)
		return nil, err
	}

	bc.am = account.NewManager(latestStableBlock.Hash(), bc.db)
	dpovpCfg := consensus.Config{
		LogForks:      bc.flags.Int(common.LogLevel)-1 >= 3,
		RewardManager: bc.Founder(),
		ChainID:       bc.chainID,
		MineTimeout:   config.MineTimeout,
		MinerExtra:    nil,
	}
	txGuard := txpool.NewTxGuard(latestStableBlock.Time())
	bc.engine = consensus.NewDPoVP(dpovpCfg, bc.db, bc.dm, bc.am, bc, txPool, txGuard)

	bc.initTxPool(latestStableBlock, txPool, txGuard)
	go bc.runFeedTranspondLoop()

	log.Info("BlockChain is ready", "stableHeight", bc.StableBlock().Height(), "stableHash", bc.StableBlock().Hash(), "currentHeight", bc.CurrentBlock().Height(), "currentHash", bc.CurrentBlock().Hash())
	return bc, nil
}

func (bc *BlockChain) initTxPool(block *types.Block, txPool *txpool.TxPool, txGuard *txpool.TxGuard) {
	if block == nil {
		log.Debug("init tx pool. block is nil.")
		return
	}

	stableTime := block.Time()
	height := block.Height()
	iter := block
	// 需要初始化的block交易条件为，区块时间戳距离最新的稳定区块的时间戳不大于30分钟
	for stableTime-iter.Time() <= uint32(params.MaxTxLifeTime) {
		txGuard.SaveBlock(iter)
		if height <= 0 {
			break
		}
		// be careful height is a uint32
		height--
		iter = bc.GetBlockByHeight(height)
		if iter == nil {
			log.Errorf("get block by height error when init tx pool. height: %d.", height)
			panic(ErrLoadBlock)
		}
	}
	log.Debugf("Finish init tx pool, start block height: %d timestamp: %d to end block height: %d timestamp: %d. ", iter.Height(), iter.Time(), block.Height(), block.Time())
}

func (bc *BlockChain) TxGuard() *txpool.TxGuard {
	return bc.engine.TxGuard()
}

func (bc *BlockChain) AccountManager() *account.Manager {
	return bc.am
}

func (bc *BlockChain) DeputyManager() *deputynode.Manager {
	return bc.dm
}

func (bc *BlockChain) IsInBlackList(b *types.Block) bool {
	return bc.dm.IsEvilDeputyNode(b.MinerAddress(), b.Height())
}

// runFeedTranspondLoop transpond dpovp feed to global event bus
func (bc *BlockChain) runFeedTranspondLoop() {
	currentCh := make(chan *types.Block)
	currentSub := bc.engine.SubscribeCurrent(currentCh)
	stableCh := make(chan *types.Block)
	stableSub := bc.engine.SubscribeStable(stableCh)
	confirmCh := make(chan *network.BlockConfirmData)
	confirmSub := bc.engine.SubscribeConfirm(confirmCh)
	fetchConfirmCh := make(chan []network.GetConfirmInfo)
	fetchConfirmSub := bc.engine.SubscribeFetchConfirm(fetchConfirmCh)
	for {
		select {
		case block := <-currentCh:
			go subscribe.Send(subscribe.NewCurrentBlock, block)
		case block := <-stableCh:
			go subscribe.Send(subscribe.NewStableBlock, block)
		case confirm := <-confirmCh:
			go subscribe.Send(subscribe.NewConfirm, confirm)
		case confirmsInfo := <-fetchConfirmCh:
			go subscribe.Send(subscribe.FetchConfirms, confirmsInfo)
		case <-bc.quitCh:
			currentSub.Unsubscribe()
			stableSub.Unsubscribe()
			confirmSub.Unsubscribe()
			fetchConfirmSub.Unsubscribe()
			return
		}
	}
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

func (bc *BlockChain) TxProcessor() *transaction.TxProcessor {
	return bc.engine.TxProcessor()
}

func (bc *BlockChain) Flags() flag.CmdFlags {
	return bc.flags
}

// HasBlock has special block in local
func (bc *BlockChain) HasBlock(hash common.Hash) bool {
	ok, _ := bc.db.IsExistByHash(hash)
	return ok
}

func (bc *BlockChain) GetBlockByHeight(height uint32) *types.Block {
	return bc.GetParentByHeight(height, bc.CurrentBlock().Hash())
}

func (bc *BlockChain) GetParentByHeight(height uint32, sonBlockHash common.Hash) *types.Block {
	var block *types.Block
	var err error
	if height <= bc.StableBlock().Height() {
		// stable block
		block, err = bc.db.GetBlockByHeight(height)
	} else {
		// unstable block
		block, err = bc.db.GetUnConfirmByHeight(height, sonBlockHash)
	}
	if err != nil {
		log.Error("load block by height fail", "height", height, "err", err)
		return nil
	}
	return block
}

func (bc *BlockChain) GetBlockByHash(hash common.Hash) *types.Block {
	block, err := bc.db.GetBlockByHash(hash)
	if err != nil {
		log.Warn("load block fail", "hash", hash.Hex(), "err", err)
		return nil
	}
	return block
}

// CurrentBlock get latest current block
func (bc *BlockChain) CurrentBlock() *types.Block {
	return bc.engine.CurrentBlock()
}

// StableBlock get latest stable block
func (bc *BlockChain) StableBlock() *types.Block {
	return bc.engine.StableBlock()
}

func (bc *BlockChain) MineBlock(txProcessTimeout int64) {
	if atomic.LoadInt32(&bc.stopped) != 0 {
		return
	}
	block, err := bc.engine.MineBlock(txProcessTimeout)
	// broadcast
	if err == nil {
		go subscribe.Send(subscribe.NewMinedBlock, block)
	}
}

// InsertBlock insert block of non-self to chain
func (bc *BlockChain) InsertBlock(block *types.Block) error {
	if atomic.LoadInt32(&bc.stopped) != 0 {
		return nil
	}
	// verify and create a new block witch filled by transaction products
	_, err := bc.engine.InsertBlock(block)
	return err
}

// InsertConfirms receive confirm package from net connection
func (bc *BlockChain) InsertConfirms(height uint32, blockHash common.Hash, sigList []types.SignData) {
	if atomic.LoadInt32(&bc.stopped) != 0 {
		return
	}
	_ = bc.engine.InsertConfirms(height, blockHash, sigList)
}

func (bc *BlockChain) GetCandidatesTop(hash common.Hash) []*store.Candidate {
	return bc.db.GetCandidatesTop(hash)
}

func (bc *BlockChain) FetchConfirm(height uint32) error {
	block := bc.GetBlockByHeight(height)
	if block == nil {
		return ErrLoadBlock
	}

	fetchList := []network.GetConfirmInfo{
		{Height: height, Hash: block.Hash()},
	}
	go subscribe.Send(subscribe.FetchConfirms, fetchList)
	return nil
}

// LogForks print the forks graph
func (bc *BlockChain) LogForks() {
	fmt.Println(bc.db.SerializeForks(bc.CurrentBlock().Hash()))
}

// Stop stop block chain
func (bc *BlockChain) Stop() {
	if !atomic.CompareAndSwapInt32(&bc.stopped, 0, 1) {
		return
	}
	close(bc.quitCh)
	log.Info("BlockChain stop")
}
