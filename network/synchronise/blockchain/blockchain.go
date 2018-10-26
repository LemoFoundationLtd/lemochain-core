package blockchain

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
)

// BlockChain 同步用 需要被外界实现
type BlockChain interface {
	// HasBlock 链上是否有此块
	HasBlock(hash common.Hash) bool
	// GetBlockByHeight 通过区块高度获取区块
	GetBlockByHeight(height uint32) *types.Block
	// GetBlockByHash 通过区块HASH获取区块
	GetBlockByHash(hash common.Hash) *types.Block
	// CurrentBlock 获取当前最新区块
	CurrentBlock() *types.Block
	// StableBlock 获取当前最新被共识的区块
	StableBlock() *types.Block
	// InsertChain 插入一个区块到链上
	InsertChain(block *types.Block, isSyncing bool) error
	// SetStableBlock 设置最新的稳定区块
	SetStableBlock(hash common.Hash, height uint32, isSyncing bool) error
	// Verify 验证区块是否合法
	Verify(block *types.Block) error
}
