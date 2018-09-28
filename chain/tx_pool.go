package chain

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
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

type TxsCache struct {
	txs []*TransactionWithTime
	cap int
	cnt int
}

func NewTxsCache() *TxsCache {
	cache := &TxsCache{}
	cache.cap = 2
	cache.cnt = 0
	cache.txs = make([]*TransactionWithTime, cache.cap)

	return cache
}

func (cache *TxsCache) push(tx *types.Transaction) {
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

	cache.cnt = cache.cnt + 1
}

func (cache *TxsCache) pop(size int) []*types.Transaction {
	if size <= 0 {
		return make([]*types.Transaction, 0)
	}

	if cache.cnt <= size {
		txs := make([]*types.Transaction, cache.cnt)
		for index := 0; index < cache.cnt; index++ {
			txs[index] = cache.txs[index].Tx
		}

		return txs
	} else {
		txs := make([]*types.Transaction, size)
		for index := 0; index < size; index++ {
			txs[index] = cache.txs[index].Tx
		}

		return txs
	}
}

func (cache *TxsCache) remove(size int) {
	if cache.cnt <= size {
		cache.cap = 512
		cache.txs = make([]*TransactionWithTime, cache.cap)
		cache.cnt = 0
	} else {
		cache.txs = append(cache.txs[:size], cache.txs[size+1:]...)
		cache.cnt = cache.cnt - size
	}
}

type TxsRecent struct {
	lastTime int64
	recent   map[common.Hash]bool
}

func NewRecent() *TxsRecent {
	t := time.Now()
	return &TxsRecent{
		lastTime: t.Unix(),
		recent:   make(map[common.Hash]bool),
	}
}

func (recent *TxsRecent) isExist(hash common.Hash) bool {
	if recent.recent[hash] {
		return true
	} else {
		return false
	}
}

func (recent *TxsRecent) put(hash common.Hash) {
	next := time.Now().Unix()

	if next-recent.lastTime > TransactionTimeOut {
		recent.lastTime = next
		recent.recent = make(map[common.Hash]bool)
	}

	recent.recent[hash] = true
}

type TxPool struct {
	am       *account.Manager
	txsCache *TxsCache
	recent   *TxsRecent
	mux      sync.Mutex
}

func NewTxPool() *TxPool {
	pool := &TxPool{
		txsCache: NewTxsCache(),
		recent:   NewRecent(),
	}

	return pool
}

func (pool *TxPool) AddTx(tx *types.Transaction) error {
	hash := tx.Hash()
	pool.mux.Lock()
	defer pool.mux.Unlock()
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
		return nil
	}
}

func (pool *TxPool) Pending(size int) []*types.Transaction {
	pool.mux.Lock()
	defer pool.mux.Unlock()
	return pool.txsCache.pop(size)
}

func (pool *TxPool) Remove(size int) {
	pool.mux.Lock()
	defer pool.mux.Unlock()
	pool.txsCache.remove(size)
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
