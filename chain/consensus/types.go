package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
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
	Deputies      deputynode.DeputyNodes
}

// BlockLoader supports retrieving headers and consensus parameters from the
// current blockchain to be used during transaction processing.
type BlockLoader interface {
	// GetBlockByHash returns the hash corresponding to their hash.
	GetBlockByHash(hash common.Hash) *types.Block
	// GetBlockByHash returns the hash corresponding to their hash.
	GetParentByHeight(height uint32, sonBlockHash common.Hash) *types.Block
}

type TxPool interface {
	Get(time uint32, size int) []*types.Transaction
	DelInvalidTxs(txs []*types.Transaction)
	VerifyTxInBlock(block *types.Block) bool
	RecvBlock(block *types.Block)
	PruneBlock(block *types.Block)
}

type CandidateLoader interface {
	LoadTopCandidates(blockHash common.Hash) deputynode.DeputyNodes
}
