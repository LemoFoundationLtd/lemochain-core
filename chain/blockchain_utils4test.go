package chain

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/store"
)

type EngineTestForChain struct{}

func (engine *EngineTestForChain) VerifyHeader(block *types.Block) error { return nil }

func (engine *EngineTestForChain) Seal(header *types.Header, txs []*types.Transaction, changeLog []*types.ChangeLog, events []*types.Event) (*types.Block, error) {
	return nil, nil
}

func (engine *EngineTestForChain) Finalize(header *types.Header, am *account.Manager) {}

func broadcastStableBlock(block *types.Block) {}

func broadcastConfirmInfo(hash common.Hash, height uint32) {}

func NewBlockChainForTest() (*BlockChain, chan *types.Block, error) {
	chainID := uint16(99)
	db, err := store.NewCacheChain(store.GetStorePath())
	if err != nil {
		return nil, nil, err
	}

	genesis := DefaultGenesisBlock()
	_, err = SetupGenesisBlock(db, genesis)
	if err != nil {
		return nil, nil, err
	}

	var engine EngineTestForChain
	ch := make(chan *types.Block)
	blockChain, err := NewBlockChain(chainID, &engine, db, nil)
	if err != nil {
		return nil, nil, err
	}

	// blockChain.BroadcastStableBlock = broadcastStableBlock
	// blockChain.BroadcastConfirmInfo = broadcastConfirmInfo

	deputynode.Instance().Add(0, DefaultDeputyNodes)
	return blockChain, ch, nil
}
