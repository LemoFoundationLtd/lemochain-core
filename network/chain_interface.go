package network

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

// BlockChain
type BlockChain interface {
	Genesis() *types.Block
	// HasBlock if block exist in local chain
	HasBlock(hash common.Hash) bool
	// GetBlockByHeight get block by  height from local chain
	GetBlockByHeight(height uint32) *types.Block
	// GetBlockByHash get block by hash from local chain
	GetBlockByHash(hash common.Hash) *types.Block
	// CurrentBlock local chain's current block
	CurrentBlock() *types.Block
	// StableBlock local chain's latest stable block
	StableBlock() *types.Block
	// InsertBlock insert a block to local chain
	InsertBlock(block *types.Block) error
	// ReceiveConfirm received a confirm message from remote peer
	InsertConfirm(info *BlockConfirmData)
	// GetConfirms get a block's confirms from local chain
	GetConfirms(query *GetConfirmInfo) []types.SignData
	// IsConfirmEnough test if the confirms in block is enough
	IsConfirmEnough(block *types.Block) bool
	// ReceiveStableConfirms received a block's confirm info
	ReceiveStableConfirms(pack BlockConfirms)
	// IsInBlackList
	IsInBlackList(b *types.Block) bool
}

type TxPool interface {
	// AddTxs add transaction
	RecvTxs(tx []*types.Transaction)
	// Remove remove transaction
	RecvBlock(block *types.Block)
}
