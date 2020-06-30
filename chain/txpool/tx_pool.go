package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/metrics"
	"sync"
)

var (
	txPoolTotalNumberCounter = metrics.NewCounter(metrics.TxpoolNumber_counterName) // 交易池中剩下的总交易数量
	blockTradeAmount         = common.Lemo2Mo("500000")                             // 如果交易的amount 大于此值则进行事件通知
	defaultPoolCap           = 128
)

// TxPool saves the transactions which could be packaged in current fork
type TxPool struct {
	txs types.Transactions
	cap int

	// it is used to find the index of txs by tx hash. sub tx hash will be point to its box tx index
	hashIndexMap map[common.Hash]int

	RW sync.RWMutex
}

func NewTxPool() *TxPool {
	pool := &TxPool{}
	pool.cap = defaultPoolCap
	pool.txs = make(types.Transactions, 0, pool.cap)
	pool.hashIndexMap = make(map[common.Hash]int)
	return pool
}

func (pool *TxPool) IsEmpty() bool {
	pool.RW.Lock()
	defer pool.RW.Unlock()

	return len(pool.hashIndexMap) <= 0
}

func (pool *TxPool) addTx(tx *types.Transaction) error {
	if tx == nil {
		return ErrInvalidTx
	}
	if pool.isTxExist(tx) {
		return ErrTxIsExist
	}

	// extend storage
	if pool.cap-len(pool.txs) < 1 {
		pool.cap *= 2
		tmp := make(types.Transactions, 0, pool.cap)
		copy(tmp, pool.txs)
		pool.txs = tmp
	}

	// save transaction
	index := len(pool.txs)
	pool.txs = append(pool.txs, tx)
	pool.hashIndexMap[tx.Hash()] = index
	// save sub transactions in box transaction
	if tx.Type() == params.BoxTx {
		for _, subTx := range getSubTxs(tx) {
			pool.hashIndexMap[subTx.Hash()] = index
		}
	}

	// report tx count increase
	txPoolTotalNumberCounter.Inc(1)
	if tx.Amount().Cmp(blockTradeAmount) >= 0 {
		toString := "[nil]"
		if tx.To() != nil {
			toString = tx.To().String()
		}
		log.Eventf(log.TxEvent, "Block trade appear. %s send %s to %s", tx.From().String(), tx.Amount().String(), toString)
	}

	return nil
}

func (pool *TxPool) AddTx(tx *types.Transaction) error {
	pool.RW.Lock()
	defer pool.RW.Unlock()

	return pool.addTx(tx)
}

// AddTxs push txs into pool and return the number of new txs
func (pool *TxPool) AddTxs(txs types.Transactions) int {
	if len(txs) == 0 {
		return 0
	}
	pool.RW.Lock()
	defer pool.RW.Unlock()

	log.Debugf("Put %d transactions into pool", len(txs))
	count := 0
	for _, tx := range txs {
		if err := pool.addTx(tx); err == nil {
			count++
		}
	}
	return count
}

/* 本节点出块时，从交易池中取出交易进行打包，但并不从交易池中删除 */
func (pool *TxPool) GetTxs(time uint32, size int) types.Transactions {
	result := make([]*types.Transaction, 0, size)
	if size <= 0 {
		return result
	}

	pool.RW.Lock()
	defer pool.RW.Unlock()

	for _, tx := range pool.txs {
		if tx == nil {
			continue
		}
		if isTxTimeOut(tx, time) {
			pool.delTx(tx)
			continue
		}
		result = append(result, tx)
		if len(result) >= size {
			break
		}
	}
	return result
}

func (pool *TxPool) delTx(tx *types.Transaction) {
	if tx == nil {
		return
	}

	hash := tx.Hash()
	if index, ok := pool.hashIndexMap[hash]; ok {
		// If tx is a sub tx (from other miner's block). This will delete the box tx. So the box tx would not be packaged, it will be deleted when expired
		// There is a small problem is that the other sub txs in the box could not be add into pool, because they are already in hashIndexMap, but they are not in txs
		pool.txs[index] = nil
		delete(pool.hashIndexMap, hash)

		// report tx count decrease
		txPoolTotalNumberCounter.Dec(1)
	}

	// delete indexes of sub transactions in box transaction
	if tx.Type() == params.BoxTx {
		for _, subTx := range getSubTxs(tx) {
			delete(pool.hashIndexMap, subTx.Hash())
		}
	}
}

/* 新收到一个通过验证的新块（包括本节点出的块），需要从交易池中删除该块中已打包的交易 */
func (pool *TxPool) DelTxs(txs types.Transactions) {
	if len(txs) <= 0 {
		return
	}

	pool.RW.Lock()
	defer pool.RW.Unlock()

	log.Debugf("Delete %d transactions from pool", len(txs))

	for _, tx := range txs {
		pool.delTx(tx)
	}

	pool.gc()
}

func (pool *TxPool) isTxExist(tx *types.Transaction) bool {
	hash := tx.Hash()
	if _, ok := pool.hashIndexMap[hash]; ok {
		return true
	}

	// check sub transactions in box transaction
	if tx.Type() == params.BoxTx {
		for _, subTx := range getSubTxs(tx) {
			if _, ok := pool.hashIndexMap[subTx.Hash()]; ok {
				return true
			}
		}
	}

	return false
}

func (pool *TxPool) gc() {
	log.Debug("TxPool gc", "indexesLength", len(pool.hashIndexMap))
	if len(pool.hashIndexMap) <= 0 {
		if pool.cap > defaultPoolCap {
			pool.cap--
		}
		pool.txs = make(types.Transactions, 0, pool.cap)
		pool.hashIndexMap = make(map[common.Hash]int)
	}
}

func isTxTimeOut(tx *types.Transaction, time uint32) bool {
	if tx.Expiration() < uint64(time) {
		return true
	}

	// check sub transactions in box transaction
	if tx.Type() == params.BoxTx {
		for _, subTx := range getSubTxs(tx) {
			if subTx.Expiration() < uint64(time) {
				return true
			}
		}
	}

	return false
}

func getSubTxs(tx *types.Transaction) types.Transactions {
	box, err := types.GetBox(tx.Data())
	if err != nil {
		log.Debug("get box from tx err: " + err.Error())
		return make(types.Transactions, 0)
	} else {
		return box.SubTxList
	}
}
