package miner

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/stretchr/testify/assert"
	"testing"
)

type EngineTestForMiner struct{}

func (engine *EngineTestForMiner) VerifyHeader(block *types.Block) error { return nil }

func (engine *EngineTestForMiner) Seal(header *types.Header, txs []*types.Transaction, changeLog []*types.ChangeLog, events []*types.Event) (*types.Block, error) {
	return nil, nil
}

func (engine *EngineTestForMiner) Finalize(header *types.Header, am *account.Manager) {}

func broadcastStableBlock(block *types.Block) {}

func broadcastConfirmInfo(hash common.Hash, height uint32) {}

func newBlockChain() (*chain.BlockChain, chan *types.Block, error) {
	chainId := uint16(99)
	db, err := store.NewCacheChain(store.GetStorePath())
	if err != nil {
		return nil, nil, err
	}

	genesis := chain.DefaultGenesisBlock()
	_, err = chain.SetupGenesisBlock(db, genesis)
	if err != nil {
		return nil, nil, err
	}

	var engine EngineTestForMiner
	ch := make(chan *types.Block)
	blockChain, err := chain.NewBlockChain(chainId, &engine, db, ch, nil)
	if err != nil {
		return nil, nil, err
	}

	blockChain.BroadcastStableBlock = broadcastStableBlock
	blockChain.BroadcastConfirmInfo = broadcastConfirmInfo

	deputynode.Instance().Add(0, chain.DefaultDeputyNodes)
	return blockChain, ch, nil
}

func newMiner(key string) (*Miner, chan *types.Block, chan *types.Block, error) {
	store.ClearData()

	tmp, err := crypto.HexToECDSA(key)
	if err != nil {
		return nil, nil, nil, err
	} else {
		deputynode.SetSelfNodeKey(tmp)
	}

	blockChain, _, err := newBlockChain()
	if err != nil {
		return nil, nil, nil, err
	}

	cnf := &MineConfig{SleepTime: 3000, Timeout: 10000}
	txPool := chain.NewTxPool(blockChain.AccountManager(), nil)
	mineNewBlockCh := make(chan *types.Block)
	recvBlockCh := make(chan *types.Block)

	return New(cnf,
		blockChain,
		txPool,
		mineNewBlockCh, recvBlockCh,
		new(EngineTestForMiner)), mineNewBlockCh, recvBlockCh, nil
}

func TestMiner_modifyTimer(t *testing.T) {
	store.ClearData()

	me := "c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"
	miner, _, _, err := newMiner(me)
	assert.NoError(t, err)

	reset := miner.getSleepTime()
	assert.Equal(t, 0, reset)
}
