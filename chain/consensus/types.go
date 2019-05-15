package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

// Config holds consensus options.
type Config struct {
	// Show every forks change
	LogForks bool
	// RewardManager is the owner of reward setting precompiled contract
	RewardManager common.Address
	ChainID       uint16
	MineTimeout   uint64
}

// BlockMaterial is used for mine a new block
type BlockMaterial struct {
	Extra         []byte
	MineTimeLimit int64
	Txs           types.Transactions
	Deputies      types.DeputyNodes
}

// BlockLoader is the interface of ChainDB
type BlockLoader interface {
	IterateUnConfirms(fn func(*types.Block))
	GetBlockByHash(hash common.Hash) (*types.Block, error)
	// GetBlockByHeight returns stable blocks
	GetBlockByHeight(height uint32) (*types.Block, error)
}

// StableBlockStore is the interface of ChainDB
type StableBlockStore interface {
	LoadLatestBlock() (*types.Block, error)
	SetStableBlock(hash common.Hash) ([]*types.Block, error)
}

type TxPool interface {
	Get(time uint32, size int) []*types.Transaction
	DelInvalidTxs(txs []*types.Transaction)
	VerifyTxInBlock(block *types.Block) bool
	RecvBlock(block *types.Block)
	PruneBlock(block *types.Block)
}

type CandidateLoader interface {
	LoadTopCandidates(blockHash common.Hash) types.DeputyNodes
}
