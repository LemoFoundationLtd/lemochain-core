package chain

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"sync"
)

var (
	// ErrInvalidSender is returned if the transaction contains an invalid signature.
	ErrInvalidSender = errors.New("invalid sender")

	// ErrInsufficientFunds is returned if the total cost of executing a transaction
	// is higher than the balance of the user's account.
	ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")
)

type TxPool struct {
	am       *account.Manager
	all      map[common.Hash]*types.Transaction // 所有等待打包的交易
	tmp      map[common.Hash]types.Transactions // 已被打包的交易，但块尚未确认
	mux      sync.Mutex
	newTxsCh chan types.Transactions
}

func NewTxPool(am *account.Manager, txsCh chan types.Transactions) *TxPool {
	pool := &TxPool{
		am:       am,
		all:      make(map[common.Hash]*types.Transaction),
		newTxsCh: txsCh,
	}

	return pool
}

// Pending 出块时用 获取等待打包的交易
func (pool *TxPool) Pending() types.Transactions {
	pool.mux.Lock()
	defer pool.mux.Unlock()
	pending := make(types.Transactions, 0, len(pool.all))
	for _, v := range pool.all {
		tx := *v
		pending = append(pending, &tx)
	}
	return pending
}

// validateTx 初级验证交易是否合法
func (pool *TxPool) validateTx(tx *types.Transaction) error {
	from, err := tx.From()
	if err != nil {
		return ErrInvalidSender
	}
	fromAccount, err := pool.am.GetAccount(from)
	if err != nil {
		return err
	}
	balance := fromAccount.GetBalance()
	if balance.Cmp(tx.Cost()) < 0 {
		return ErrInsufficientFunds
	}
	// 其他 todo
	return nil
}

func (pool *TxPool) addTx(tx *types.Transaction) error {
	if err := pool.validateTx(tx); err != nil {
		return err
	}
	pool.all[tx.Hash()] = tx
	return nil
}

// AddTx 增加交易到交易池
func (pool *TxPool) AddTx(tx *types.Transaction) error {
	pool.mux.Lock()
	defer pool.mux.Unlock()
	err := pool.addTx(tx)
	if err != nil {
		pool.newTxsCh <- types.Transactions{tx}
	}
	return err
}

// AddTxs 批量增加交易到交易池
func (pool *TxPool) AddTxs(txs types.Transactions) []error {
	pool.mux.Lock()
	defer pool.mux.Unlock()
	errs := make([]error, len(txs))
	for i, tx := range txs {
		errs[i] = pool.addTx(tx)
	}
	pool.newTxsCh <- txs
	return errs
}

// RemoveTxs 从交易池移除交易
func (pool *TxPool) RemoveTxs(txs []common.Hash) {
	pool.mux.Lock()
	defer pool.mux.Unlock()
	for _, hash := range txs {
		if _, ok := pool.all[hash]; ok {
			delete(pool.all, hash)
		}
	}
}

// Stop
func (pool *TxPool) Stop() {

}
