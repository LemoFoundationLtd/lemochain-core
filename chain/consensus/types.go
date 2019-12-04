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
	MinerExtra    []byte // Extra data in mined block header. It is short than 256bytes
}

// BlockMaterial is used for mine a new block
type BlockMaterial struct {
	ParentHeader *types.Header
	Time         uint32 // new block time in header
	Extra        []byte
	Txs          types.Transactions
}

// BlockLoader is the interface of ChainDB
type BlockLoader interface {
	IterateUnConfirms(fn func(*types.Block))
	GetUnConfirmByHeight(height uint32, leafBlockHash common.Hash) (*types.Block, error)
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
	ExistPendingTx(time uint32) bool
	DelInvalidTxs(txs []*types.Transaction)
	PushTx(tx *types.Transaction) bool
	RecvBlock(block *types.Block)
	SetTxsFlag(txs []*types.Transaction, isPending bool) bool
}

type CandidateLoader interface {
	LoadTopCandidates(blockHash common.Hash) types.DeputyNodes
	LoadRefundCandidates(height uint32) ([]common.Address, error)
}
