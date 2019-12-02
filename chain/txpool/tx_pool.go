package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/metrics"
	"sync"
)

var (
	txPoolTotalNumberCounter = metrics.NewCounter(metrics.TxpoolNumber_counterName) // 交易池中剩下的总交易数量
	blockTradeAmount         = common.Lemo2Mo("500000")                             // 如果交易的amount 大于此值则进行事件通知
)

type TxPool struct {
	/* 还未被打包进块的交易 */
	PendingTxs *TxQueue

	/* 最近1个小时的所有交易 */
	RecentTxs *RecentTx

	RW sync.RWMutex
}

func NewTxPool() *TxPool {
	return &TxPool{
		PendingTxs: NewTxQueue(),
		RecentTxs:  NewTxRecently(),
	}
}

/* 本节点出块时，从交易池中取出交易进行打包，但并不从交易池中删除 */
func (pool *TxPool) Get(time uint32, size int) []*types.Transaction {
	pool.RW.Lock()
	defer pool.RW.Unlock()
	return pool.PendingTxs.Pop(time, size)
}

// ExistCanPackageTx 存在可以打包的交易
func (pool *TxPool) ExistPendingTx(time uint32) bool {
	pool.RW.Lock()
	defer pool.RW.Unlock()
	return pool.PendingTxs.ExistPendingTx(time)
}

/* 本节点出块时，执行交易后，发现错误的交易通过该接口进行删除 */
func (pool *TxPool) DelInvalidTxs(txs []*types.Transaction) {
	pool.RW.Lock()
	defer pool.RW.Unlock()

	if len(txs) <= 0 {
		return
	}

	hashes := make([]common.Hash, 0, len(txs))
	for _, tx := range txs {
		hashes = append(hashes, tx.Hash())
	}
	pool.PendingTxs.DelBatch(hashes)
}

/* 新收到一个通过验证的新块（包括本节点出的块），需要从交易池中删除该块中已打包的交易 */
func (pool *TxPool) RecvBlock(block *types.Block) {
	pool.RW.Lock()
	defer pool.RW.Unlock()

	if block == nil {
		return
	}

	txs := block.Txs
	if len(txs) <= 0 {
		return
	}

	hashes := make([]common.Hash, 0, len(txs))
	for _, v := range txs {
		hashes = append(hashes, v.Hash())
	}

	pool.PendingTxs.DelBatch(hashes)
	pool.RecentTxs.RecvBlock(block.Hash(), int64(block.Height()), txs)
}

/* 收到一笔新的交易 */
func (pool *TxPool) PushTx(tx *types.Transaction) bool {
	if tx == nil {
		return false
	}

	pool.RW.Lock()
	defer pool.RW.Unlock()

	isExist := pool.RecentTxs.IsExist(tx)
	if isExist {
		log.Debug("tx is already exist. hash: " + tx.Hash().Hex())
		return false
	} else {
		pool.RecentTxs.RecvTx(tx)
		pool.PendingTxs.Push(tx)
		txPoolTotalNumberCounter.Inc(1) // 记录收到一笔交易
		if tx.Amount().Cmp(blockTradeAmount) >= 0 {
			toString := "[nil]"
			if tx.To() != nil {
				toString = tx.To().String()
			}
			log.Eventf(log.TxEvent, "Block trade appear. %s send %s to %s", tx.From().String(), tx.Amount().String(), toString)
		}
		return true
	}
}

func (pool *TxPool) SetTxsFlag(txs []*types.Transaction, isPending bool) bool {
	pool.RW.Lock()
	defer pool.RW.Unlock()

	if len(txs) <= 0 {
		return false
	}

	for _, v := range txs {
		pool.RecentTxs.RecvTx(v)
		pool.PendingTxs.Push(v)
	}

	if isPending {
		return true
	}

	// 这样组合操作的目的是，不想改PendingTxs的接口导致引入新bug
	for _, v := range txs {
		pool.PendingTxs.Del(v.Hash())
	}

	return true
}
