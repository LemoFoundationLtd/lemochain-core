package chain

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/subscribe"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"sync"
	"time"
)

var (
	// ErrInvalidSender is returned if the transaction contains an invalid signature.
	ErrInvalidSender = errors.New("invalid sender")
	ErrTxChainID     = errors.New("Tx chainID unequal to node chainID ")
	ErrNegativeValue = errors.New("negative value")

	// ErrInsufficientFunds is returned if the total cost of executing a transaction
	// is higher than the balance of the user's account.
	ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")
)

var TransactionTimeOut = int64(3600)

type TransactionWithTime struct {
	Tx      *types.Transaction
	RecTime int64
	DelFlg  bool
}

// type AccountLoader interface {
// 	GetCanonicalAccount(address common.Address) types.AccountAccessor
// }

type TxsSort interface {
	push(tx *types.Transaction)

	pop(size int) []*types.Transaction

	removeBatch(keys []common.Hash)

	remove(key common.Hash)

	len() int
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
		DelFlg:  false,
	}
	cache.index[tx.Hash()] = cache.cnt

	cache.cnt = cache.cnt + 1
}

func (cache *TxsSortByTime) pop(size int) []*types.Transaction {
	txs := make([]*types.Transaction, 0)
	if (size <= 0) || (len(cache.txs) <= 0) {
		return txs
	} else {
		cnt := 0
		for index := 0; (index < cache.cnt) && (cnt < size); index++ {
			if cache.txs[index].DelFlg {
				// delete(cache.index, cache.txs[index].Tx.Hash())
				// cache.txs = append(cache.txs[:index], cache.txs[index+1:]...)
				// cache.cnt = cache.cnt - 1
				// index = index - 1
			} else {
				txs = append(txs, cache.txs[index].Tx)
				cnt = cnt + 1
			}
		}

		if len(txs) <= 0 {
			cache.txs = make([]*TransactionWithTime, cache.cap)
			cache.index = make(map[common.Hash]int)
			cache.cnt = 0
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
	pos, ok := cache.index[key]
	if ok && pos >= 0 && !cache.txs[pos].DelFlg {
		cache.txs[pos].DelFlg = true
	} else if !ok {
		log.Debug("Txs are not synchronized to local tx_pool")
	} else {
		log.Error("delete Txs again!")
	}
}

func (cache *TxsSortByTime) len() int {
	return cache.cnt
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
	_, isExist1 := recent.recent[0][hash]
	_, isExist2 := recent.recent[1][hash]

	if isExist1 || isExist2 {
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
	// am      AccountLoader
	chainID uint16

	txsCache TxsSort

	recent *TxsRecent
	mux    sync.Mutex
}

func NewTxPool(chainID uint16) *TxPool {
	pool := &TxPool{
		// am:      am,
		chainID: chainID,
		recent:  NewRecent(),
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
		err := pool.validateTx(tx)
		if err != nil {
			return err
		}
		pool.recent.put(hash)
		pool.txsCache.push(tx)
		subscribe.Send(subscribe.NewTx, tx)
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

func (pool *TxPool) AddKey(hash common.Hash) {
	pool.mux.Lock()
	defer pool.mux.Unlock()
	pool.recent.put(hash)
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
	_, err := tx.From()
	if err != nil {
		return err
	}
	if tx.ChainID() != pool.chainID {
		return ErrTxChainID
	}
	if tx.Amount().Sign() < 0 {
		return ErrNegativeValue
	}
	// Make sure the transaction is signed properly
	if len(tx.GasPayerSig()) == 0 {
		_, err = types.MakeSigner().GetSigner(tx)
	} else {
		// reimbursement transaction
		_, err = types.MakeReimbursementTxSigner().GetSigner(tx)
		_, err = types.MakeGasPayerSigner().GetSigner(tx)
	}

	if err != nil {
		return err
	} else {
		return nil
	}
	// fromAccount := pool.am
	// balance := fromAccount.GetBalance()
	// if balance.Cmp(tx.Cost()) < 0 {
	// 	return ErrInsufficientFunds
	// } else {
	// 	return nil
	// }
}
