package chain

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/consensus"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
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

type BlockChain struct {
	chainID      uint16
	flags        flag.CmdFlags
	genesisBlock *types.Block // genesis block

	db     db.ChainDB
	am     *account.Manager
	dm     *deputynode.Manager
	engine *consensus.DPoVP

	// receive call event from outside
	receiveBlockCh   chan *types.Block
	mineBlockCh      chan *consensus.BlockMaterial
	receiveConfirmCh chan *network.BlockConfirmData

	running int32
	quitCh  chan struct{}
}

// Config holds chain options.
type Config struct {
	ChainID     uint16
	MineTimeout uint64
}

func NewBlockChain(config Config, dm *deputynode.Manager, db db.ChainDB, flags flag.CmdFlags, txPool *TxPool) (bc *BlockChain, err error) {
	bc = &BlockChain{
		chainID:          config.ChainID,
		db:               db,
		dm:               dm,
		flags:            flags,
		receiveBlockCh:   make(chan *types.Block),
		mineBlockCh:      make(chan *consensus.BlockMaterial),
		receiveConfirmCh: make(chan *network.BlockConfirmData),
		quitCh:           make(chan struct{}),
	}
	bc.genesisBlock, err = bc.db.GetBlockByHeight(0)
	if err != nil {
		return nil, ErrNoGenesis
	}

	// stable block
	block, err := bc.db.LoadLatestBlock()
	if err != nil {
		log.Errorf("Can't load last state: %v", err)
		return nil, err
	}

	bc.am = account.NewManager(block.Hash(), bc.db)
	dpovpCfg := consensus.Config{
		Debug:         bc.Flags().Bool(common.Debug),
		RewardManager: bc.Founder(),
		ChainID:       bc.chainID,
		MineTimeout:   config.MineTimeout,
	}
	bc.engine = consensus.NewDPoVP(dpovpCfg, bc.db, bc.dm, bc.am, bc, txPool, block)

	go bc.runFeedResendLoop()
	go bc.runMainLoop()

	return bc, nil
}

func (bc *BlockChain) AccountManager() *account.Manager {
	return bc.am
}

func (bc *BlockChain) DeputyManager() *deputynode.Manager {
	return bc.dm
}

// runFeedResendLoop resend dpovp feed to global event bus
func (bc *BlockChain) runFeedResendLoop() {
	stableCh := make(chan *types.Block)
	stableSub := bc.engine.SubscribeStable(stableCh)
	confirmCh := make(chan *network.BlockConfirmData)
	confirmSub := bc.engine.SubscribeConfirm(confirmCh)
	for {
		select {
		case block := <-stableCh:
			go subscribe.Send(subscribe.NewStableBlock, block)
		case confirm := <-confirmCh:
			go subscribe.Send(subscribe.NewConfirm, confirm)
		case <-bc.quitCh:
			stableSub.Unsubscribe()
			confirmSub.Unsubscribe()
			return
		}
	}
}

// runMainLoop handle the call events from outside
func (bc *BlockChain) runMainLoop() {
	for {
		// These cases should be executed mutually. So we must not use coroutine
		select {
		case block := <-bc.receiveBlockCh:
			// verify and create a new block witch filled by transaction products
			_, _ = bc.engine.InsertBlock(block)

		case blockMaterial := <-bc.mineBlockCh:
			block, err := bc.engine.MineBlock(blockMaterial)
			if err == nil {
				go subscribe.Send(subscribe.NewMinedBlock, block)
			}

		case confirm := <-bc.receiveConfirmCh:
			_ = bc.engine.InsertConfirm(confirm)

		case <-bc.quitCh:
			return
		}
	}
}

func (bc *BlockChain) ReceiveConfirmCh() <-chan *network.BlockConfirmData {
	return bc.receiveConfirmCh
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

func (bc *BlockChain) TxProcessor() *consensus.TxProcessor {
	return bc.engine.TxProcessor()
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

// SubscribeNewBlock subscribe the current block update notification
func (bc *BlockChain) SubscribeNewBlock(ch chan *types.Block) subscribe.Subscription {
	return bc.engine.SubscribeCurrent(ch)
}

func (bc *BlockChain) MineBlock(material *consensus.BlockMaterial) {
	bc.mineBlockCh <- material
}

// InsertBlock insert block of non-self to chain
func (bc *BlockChain) InsertBlock(block *types.Block) {
	bc.receiveBlockCh <- block
}

// IsConfirmEnough test if the confirms in block is enough
func (bc *BlockChain) IsConfirmEnough(block *types.Block) bool {
	return consensus.IsConfirmEnough(block, bc.dm)
}

// GetConfirms get all confirm info of special block
func (bc *BlockChain) GetConfirms(query *network.GetConfirmInfo) []types.SignData {
	block := bc.GetBlockByHash(query.Hash)
	if block == nil {
		return nil
	}

	return block.Confirms
}

// ReceiveConfirm
func (bc *BlockChain) InsertConfirm(info *network.BlockConfirmData) {
	bc.receiveConfirmCh <- info
}

// ReceiveStableConfirms receive confirm package from net connection. The block of these confirms has been confirmed by its son block already
func (bc *BlockChain) ReceiveStableConfirms(pack network.BlockConfirms) {
	bc.engine.InsertStableConfirms(pack)
}

func (bc *BlockChain) GetCandidatesTop(hash common.Hash) []*store.Candidate {
	return bc.db.GetCandidatesTop(hash)
}

// Stop stop block chain
func (bc *BlockChain) Stop() {
	if !atomic.CompareAndSwapInt32(&bc.running, 0, 1) {
		return
	}
	close(bc.quitCh)
	log.Info("BlockChain stop")
}
