package account

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
)

var (
	ErrSaveReadOnly = errors.New("can not save a read only account")
)

// ReadOnlyAccount is used to block any save action on Account
type ReadOnlyAccount struct {
	Account
}

func NewReadOnlyAccount(db protocol.ChainDB, address common.Address, data *types.AccountData) *ReadOnlyAccount {
	return &ReadOnlyAccount{Account: *NewAccount(db, address, data)}
}

func (a *ReadOnlyAccount) Finalise() error {
	return ErrSaveReadOnly
}

func (a *ReadOnlyAccount) Save() error {
	return ErrSaveReadOnly
}

// ReadOnlyManager is used to access the newest readonly account data
type ReadOnlyManager struct {
	stableOnly   bool // 是否只读稳定account
	db           protocol.ChainDB
	acctDb       *store.AccountTrieDB
	accountCache map[common.Address]*ReadOnlyAccount
}

// NewManager creates a new Manager. It is used to maintain account changes based on the block environment which specified by blockHash
func NewReadOnlyManager(db protocol.ChainDB, stableOnly bool) *ReadOnlyManager {
	if db == nil {
		panic("account.NewManager is called without a database")
	}
	manager := &ReadOnlyManager{
		stableOnly:   stableOnly,
		db:           db,
		accountCache: make(map[common.Address]*ReadOnlyAccount),
	}

	return manager
}

// Reset clears out all data and switch state to the new block environment. It is not necessary to reset if only use stable accounts data
func (am *ReadOnlyManager) Reset(blockHash common.Hash) {
	exist, err := am.db.IsExistByHash(blockHash)
	if err != nil || !exist {
		log.Errorf("Reset ReadOnlyManager to block[%#x] fail: %s", blockHash, err)
		return
	}

	am.acctDb, _ = am.db.GetActDatabase(blockHash)
	am.accountCache = make(map[common.Address]*ReadOnlyAccount)
}

// getAccount return stable account from db
func (am *ReadOnlyManager) getAccount(address common.Address, stable bool) types.AccountAccessor {
	var data *types.AccountData
	var err error
	if stable {
		data, err = am.db.GetAccount(address)
	} else {
		data, err = am.acctDb.Get(address)
	}
	if err != nil && err != store.ErrNotExist {
		panic(err)
	}
	account := NewReadOnlyAccount(am.db, address, data)
	// cache it
	am.accountCache[address] = account
	return account
}

// GetAccount
func (am *ReadOnlyManager) GetAccount(address common.Address) types.AccountAccessor {

	if am.stableOnly { // 只读稳定的account
		return am.getAccount(address, true)
	} else {
		// 只读最新的account
		cached := am.accountCache[address]
		if cached == nil {
			// If acctDB is exist, then we use the newest unstable account, otherwise the newest stable account
			if am.acctDb != nil {
				return am.getAccount(address, false)
			} else {
				return am.getAccount(address, true)
			}
		} else {
			return cached
		}
	}
}

func (am *ReadOnlyManager) RevertToSnapshot(int) {
}

func (am *ReadOnlyManager) Snapshot() int {
	return 0
}

func (am *ReadOnlyManager) AddEvent(*types.Event) {
}
