package chain

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/store"
)

type EngineTestForChain struct{}

func (engine *EngineTestForChain) VerifyHeader(block *types.Block) error             { return nil }
func (engine *EngineTestForChain) Finalize(height uint32, am *account.Manager) error { return nil }
func (engine *EngineTestForChain) Seal(header *types.Header, txProduct *account.TxsProduct, confirms []types.SignData, dNodes deputynode.DeputyNodes) (*types.Block, error) {
	return types.NewBlock(header, txProduct.Txs, txProduct.ChangeLogs), nil
}

func broadcastStableBlock(block *types.Block) {}

func broadcastConfirmInfo(hash common.Hash, height uint32) {}

func NewBlockChainForTest() (*BlockChain, chan *types.Block, error) {
	chainID := uint16(99)
	db := store.NewChainDataBase(store.GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)

	genesis := DefaultGenesisBlock()
	SetupGenesisBlock(db, genesis)

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
