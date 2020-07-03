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
	// InsertBlock insert a block to local chain. Return error for distribution project
	InsertBlock(block *types.Block) error
	// InsertConfirms received a block's confirm info
	InsertConfirms(height uint32, blockHash common.Hash, sigList []types.SignData)
	// IsInBlackList
	IsInBlackList(b *types.Block) bool
}

type TxPool interface {
	/* 本节点出块时，从交易池中取出交易进行打包，但并不从交易池中删除 */
	GetTxs(time uint32, size int) types.Transactions
	/* 收到一笔新的交易 */
	AddTx(tx *types.Transaction) error
}
