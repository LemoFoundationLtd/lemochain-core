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
	// InsertStableConfirms received a block's confirm info
	InsertStableConfirms(pack BlockConfirms)
	// IsInBlackList
	IsInBlackList(b *types.Block) bool
}

type TxPool interface {
	/* 本节点出块时，从交易池中取出交易进行打包，但并不从交易池中删除 */
	Get(time uint32, size int) []*types.Transaction

	/* 本节点出块时，执行交易后，发现错误的交易通过该接口进行删除 */
	DelInvalidTxs(txs []*types.Transaction)

	/* 新收一个块时，验证块中的交易是否被同一条分叉上的其他块打包了 */
	VerifyTxInBlock(block *types.Block) bool

	/* 新收到一个通过验证的新块（包括本节点出的块），需要从交易池中删除该块中已打包的交易 */
	RecvBlock(block *types.Block)

	/* 收到一笔新的交易 */
	RecvTx(tx *types.Transaction) bool

	/* 对链进行剪枝，剪下的块中的交易需要回归交易池 */
	PruneBlock(block *types.Block)
}
