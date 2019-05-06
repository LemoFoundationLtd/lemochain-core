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
	TxQueue *TxQueue

	/* 最近1个小时的所有交易 */
	TxRecently *TxRecently

	/* 从当前高度向后的3600个块 */
	BlockCache *BlocksTrie
}

func NewTxPool() *TxPool {
	return &TxPool{
		TxQueue:    NewTxQueue(),
		TxRecently: NewTxRecently(),
		BlockCache: NewBlocksTrie(),
	}
}

/* 本节点出块时，从交易池中取出交易进行打包，但并不从交易池中删除 */
func (pool *TxPool) Get(time uint32, size int) []*types.Transaction {
	return pool.TxQueue.Pop(time, size)
}

/* 本节点出块时，执行交易后，发现错误的交易通过该接口进行删除 */
func (pool *TxPool) DelErrTxs(txs []*types.Transaction) {
	if len(txs) <= 0 {
		return
	}

	hashes := make([]common.Hash, 0, len(txs))
	for _, tx := range txs {
		hashes = append(hashes, tx.Hash())
	}

	pool.TxQueue.DelBatch(hashes)
}

func (pool *TxPool) isInBlocks(hashes map[common.Hash]bool, blocks []*TrieNode) bool {
	if len(hashes) <= 0 || len(blocks) <= 0 {
		return false
	}

	for _, v := range blocks {
		_, ok := hashes[v.Header.Hash()]
		if !ok {
			continue
		} else {
			return false
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

	blocks := pool.TxRecently.GetPath(block.Txs)
	minHeight, maxHeight, hashes := pool.distance(blocks)
	nodes := pool.BlockCache.Path(block.Hash(), block.Height(), uint32(minHeight), uint32(maxHeight))
	return !pool.isInBlocks(hashes, nodes)
}

func (pool *TxPool) distance(hashes map[common.Hash]*TxInBlocks) (int64, int64, map[common.Hash]bool) {
	minHeight := int64(^uint64(0) >> 1)
	maxHeight := int64(-1)
	blocks := make(map[common.Hash]bool)
	for _, v := range hashes {
		minHeightTmp, maxHeightTmp, blocksTmp := v.distance()
		if len(blocks) <= 0 {
			continue
		} else {
			if minHeight < minHeightTmp {
				minHeight = minHeightTmp
			}

			if maxHeight > maxHeightTmp {
				maxHeight = maxHeightTmp
			}

			for k, _ := range blocksTmp {
				blocks[k] = true
			}
		}
	}

	return minHeight, maxHeight, blocks
}

/* 新收到一个通过验证的新块（包括本节点出的块），需要从交易池中删除该块中已打包的交易 */
func (pool *TxPool) RecvBlock(block *types.Block) {
	if block == nil {
		return
	}

	if len(block.Txs) > 0 {
		hashes := make([]common.Hash, 0, len(block.Txs))
		for _, v := range block.Txs {
			hashes = append(hashes, v.Hash())
		}

		pool.TxQueue.DelBatch(hashes)
		pool.TxRecently.RecvBlock(block.Hash(), int64(block.Height()), block.Txs)
	}

	pool.BlockCache.PushBlock(block)
}

/* 收到一笔新的交易 */
func (pool *TxPool) RecvTx(tx *types.Transaction) {
	if tx == nil {
		return
	}

	pool.TxRecently.RecvTx(tx)
	pool.TxQueue.Push(tx)
}

/* 对链进行剪枝，剪下的块中的交易需要回归交易池 */
func (pool *TxPool) PruneBlock(block *types.Block) {
	if block == nil {
		return
	}

	if len(block.Txs) > 0 {
		pool.TxQueue.PushBatch(block.Txs)
		pool.TxRecently.PruneBlock(block.Hash(), int64(block.Height()), block.Txs)
	}

	pool.BlockCache.DelBlock(block)
}
