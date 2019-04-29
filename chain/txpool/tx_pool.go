package txpool

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

var TransactionExpiration = 30 * 60

var (
	TxPoolErrExist = errors.New("transaction is exist")
)

type TxPool struct {
	/* 还未被打包进块的交易 */
	TxCache TxSliceByTime

	/* 最近1个小时的所有交易 */
	TxRecently TxRecently

	/* 从当前高度向后的3600个块 */
	BlockCache BlocksTrie
}

/* 本节点出块时，从交易池中取出交易进行打包，但并不从交易池中删除 */
func (pool *TxPool) Get(size int) []*types.Transaction {
	if size <= 0 {
		return make([]*types.Transaction, 0)
	}

	return pool.TxCache.GetBatch(size)
}

/* 本节点出块时，执行交易后，发现错误的交易通过该接口进行删除 */
func (pool *TxPool) DelErrTxs(txs []*types.Transaction) {
	if len(txs) <= 0 {
		return
	}
	pool.TxCache.DelBatchByTx(txs)
}

func (pool *TxPool) isInBlocks(hashs []common.Hash, blocks []*TrieNode) bool {
	if len(hashs) <= 0 || len(blocks) <= 0 {
		return false
	}

	for index := 0; index < len(hashs); index++ {
		hash := hashs[index]
		for _, v := range blocks {
			if v.hashIsExist(hash) {
				return true
			} else {
				continue
			}
		}
	}

	return true
}

/* 新收一个块时，验证块中的交易是否被同一条分叉上的其他块打包了 */
func (pool *TxPool) BlockIsValid(block *types.Block) bool {
	if block == nil {
		return false
	}

	if len(block.Txs) <= 0 {
		return true
	}

	minHeight, maxHeight, hashs := pool.TxRecently.GetPath(block.Txs)
	nodes := pool.BlockCache.Path(block.Hash(), block.Height(), uint32(minHeight), uint32(maxHeight))
	return !pool.isInBlocks(hashs, nodes)
}

/* 新收到一个通过验证的新块（包括本节点出的块），需要从交易池中删除该块中已打包的交易 */
func (pool *TxPool) RecvBlock(block *types.Block) {
	if block == nil {
		return
	}

	if len(block.Txs) > 0 {
		pool.TxCache.DelBatchByTx(block.Txs)
		pool.TxRecently.RecvBlock(int64(block.Height()), block.Txs)
	}

	timeOutTxs := pool.BlockCache.PushBlock(block)
	pool.TxRecently.DelBatch(timeOutTxs)
}

/* 收到一笔新的交易 */
func (pool *TxPool) RecvTx(tx *types.Transaction) {
	if tx == nil {
		return
	}

	pool.TxRecently.RecvTx(tx)
	timeOutTxs := pool.TxCache.add(tx)
	pool.TxRecently.DelBatch(timeOutTxs)
}

/* 对链进行剪枝，剪下的块中的交易需要回归交易池 */
func (pool *TxPool) PruneBlock(block *types.Block) {
	if block == nil {
		return
	}

	if len(block.Txs) > 0 {
		timeOutTxs := pool.TxCache.AddBatch(block.Txs)
		pool.TxRecently.DelBatch(timeOutTxs)
	}

	pool.BlockCache.DelBlock(block)
}
