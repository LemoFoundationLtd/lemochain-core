package chain

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"sync"
	"time"
)

var (
	// ErrInvalidSender is returned if the transaction contains an invalid signature.
	ErrInvalidSender = errors.New("invalid sender")

	// ErrInsufficientFunds is returned if the total cost of executing a transaction
	// is higher than the balance of the user's account.
	ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")
)

var TransactionTimeOut = int64(10)

type TransactionWithTime struct {
	Tx      *types.Transaction
	RecTime int64
}

type TxsSort interface {
	push(tx *types.Transaction)

	pop(size int) []*types.Transaction

	removeBatch(keys []common.Hash)

	remove(key common.Hash)
}

type TxsSortByTime struct {
	txs   []*TransactionWithTime
	index map[common.Hash]int
	cap   int
	cnt   int
}

func NewTxsSortByTime() TxsSort {
	cache := &TxsSortByTime{}
	cache.cap = 2
	cache.cnt = 0
	cache.txs = make([]*TransactionWithTime, cache.cap)
	cache.index = make(map[common.Hash]int)
	return cache
}

func (cache *TxsSortByTime) push(tx *types.Transaction) {

	if cache.cap-cache.cnt < 1 {
		cache.cap = 2 * cache.cap
		tmp := make([]*TransactionWithTime, cache.cap)
		copy(tmp, cache.txs)
		cache.txs = tmp
	}

	t := time.Now()
	cache.txs[cache.cnt] = &TransactionWithTime{
		Tx:      tx,
		RecTime: t.Unix(),
	}
	cache.index[tx.Hash()] = cache.cnt

	cache.cnt = cache.cnt + 1
}

func (cache *TxsSortByTime) pop(size int) []*types.Transaction {
	if size <= 0 {
		return make([]*types.Transaction, 0)
	} else {
		minCnt := size
		if cache.cnt < minCnt {
			minCnt = cache.cnt
		}

		txs := make([]*types.Transaction, minCnt)
		for index := 0; index < minCnt; index++ {
			txs[index] = cache.txs[index].Tx
		}

		return txs
	}
}

func (cache *TxsSortByTime) removeBatch(keys []common.Hash) {
	if len(keys) <= 0 {
		return
	} else {
		for index := 0; index < len(keys); index++ {
			cache.remove(keys[index])
		}
	}
}

func (cache *TxsSortByTime) remove(key common.Hash) {
	pos := cache.index[key]
	if pos >= 0 {
		delete(cache.index, key)
		cache.txs = append(cache.txs[:pos], cache.txs[pos+1:]...)
		cache.cnt = cache.cnt - 1
	}
}

type TxsRecent struct {
	lastTime int64
	index    store.Index
	recent   []map[common.Hash]bool
}

func NewRecent() *TxsRecent {
	t := time.Now()
	recent := &TxsRecent{lastTime: t.Unix()}

	recent.index.Init()
	recent.recent = make([]map[common.Hash]bool, 2)
	recent.recent[0] = make(map[common.Hash]bool)
	recent.recent[1] = make(map[common.Hash]bool)

	return recent
}

func (recent *TxsRecent) isExist(hash common.Hash) bool {
	if recent.recent[0][hash] || recent.recent[1][hash] {
		return true
	} else {
		return false
	}
}

func (recent *TxsRecent) put(hash common.Hash) {
	next := time.Now().Unix()

	if next-recent.lastTime > TransactionTimeOut {
		recent.lastTime = next

		recent.recent[recent.index.Bak()] = make(map[common.Hash]bool)
		recent.index.Swap()
	}

	recent.recent[recent.index.Cur()][hash] = true
}

type TxPool struct {
	am    *account.Manager
	txsCh chan types.Transactions

	txsCache TxsSort

	recent *TxsRecent
	mux    sync.Mutex
}

func NewTxPool(am *account.Manager, txsCh chan types.Transactions) *TxPool {
	pool := &TxPool{
		am:     am,
		txsCh:  txsCh,
		recent: NewRecent(),
	}
	pool.txsCache = NewTxsSortByTime()

	return pool
}

func (pool *TxPool) AddTx(tx *types.Transaction) error {

	pool.mux.Lock()
	defer pool.mux.Unlock()

	hash := tx.Hash()
	isExist := pool.recent.isExist(hash)
	if isExist {
		return nil
	} else {
		// err := pool.validateTx(tx)
		// if err != nil {
		// 	return err
		// }
		pool.recent.put(hash)
		pool.txsCache.push(tx)
		pool.txsCh <- types.Transactions{tx}
		return nil
	}
}

func (pool *TxPool) AddTxs(txs []*types.Transaction) error {
	if len(txs) <= 0 {
		return nil
	} else {
		for index := 0; index < len(txs); index++ {
			err := pool.AddTx(txs[index])
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func (pool *TxPool) Pending(size int) []*types.Transaction {
	pool.mux.Lock()
	defer pool.mux.Unlock()

	return pool.txsCache.pop(size)
}

func (pool *TxPool) Remove(keys []common.Hash) {
	pool.mux.Lock()
	defer pool.mux.Unlock()

	pool.txsCache.removeBatch(keys)
}

func (pool *TxPool) validateTx(tx *types.Transaction) error {
	from, err := tx.From()
	if err != nil {
		return ErrInvalidSender
	}

	fromAccount := pool.am.GetAccount(from)
	balance := fromAccount.GetBalance()
	if balance.Cmp(tx.Cost()) < 0 {
		return ErrInsufficientFunds
	} else {
		return nil
	}
}
