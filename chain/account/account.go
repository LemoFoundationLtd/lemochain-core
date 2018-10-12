package account

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"github.com/LemoFoundationLtd/lemochain-go/store/trie"
	"math/big"
	"sort"
)

var (
	sha3Nil            = crypto.Keccak256Hash(nil)
	ErrNegativeBalance = errors.New("balance can't be negative")
	ErrLoadCodeFail    = errors.New("can't load contract code")
	ErrTrieFail        = errors.New("can't load contract storage trie")
	ErrTrieChanged     = errors.New("the trie has changed after Finalise")
)

type Storage map[common.Hash][]byte

func (s Storage) String() (str string) {
	for key, value := range s {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

func (s Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range s {
		cpy[key] = value
	}

	return cpy
}

// Account is used to read and write account data. the code and dirty storage K/V would be cached till they are flushing to db
type Account struct {
	data   *types.AccountData
	db     protocol.ChainDB    // used to access account data in cache or file
	trie   *trie.SecureTrie    // contract storage trie
	trieDb *store.TrieDatabase // used to access tire data in file

	// trie Trie // storage trie
	code types.Code // contract byte code

	cachedStorage Storage // Storage entry cache to avoid duplicate reads
	dirtyStorage  Storage // Storage entries that need to be flushed to disk

	codeIsDirty bool // true if the code was updated
	suicided    bool // will be delete from the trie during the "save" phase
}

// NewAccount wrap an AccountData object, or creates a new one if it's nil.
func NewAccount(db protocol.ChainDB, address common.Address, data *types.AccountData) *Account {
	if data == nil {
		// create new one
		data = &types.AccountData{Address: address}
	} else {
		data = data.Copy()
	}
	if data.Balance == nil {
		data.Balance = new(big.Int)
	}
	if (data.CodeHash == common.Hash{}) {
		data.CodeHash = sha3Nil
	}
	if data.Versions == nil {
		data.Versions = make(map[types.ChangeLogType]uint32)
	}
	return &Account{
		data:          data,
		db:            db,
		cachedStorage: make(Storage),
		dirtyStorage:  make(Storage),
	}
}

// MarshalJSON encodes the lemoClient RPC account format.
func (a *Account) MarshalJSON() ([]byte, error) {
	return a.data.MarshalJSON()
}

// UnmarshalJSON decodes the lemoClient RPC account format.
func (a *Account) UnmarshalJSON(input []byte) error {
	var dec types.AccountData
	if err := dec.UnmarshalJSON(input); err != nil {
		return err
	}
	// TODO a.db is nil
	*a = *NewAccount(a.db, dec.Address, &dec)
	return nil
}

// Implement AccountAccessor. Access Account without changelog
func (a *Account) GetAddress() common.Address                    { return a.data.Address }
func (a *Account) GetBalance() *big.Int                          { return a.data.Balance }
func (a *Account) GetVersion(logType types.ChangeLogType) uint32 { return a.data.Versions[logType] }
func (a *Account) GetSuicide() bool                              { return a.suicided }
func (a *Account) GetCodeHash() common.Hash                      { return a.data.CodeHash }

// StorageRoot wouldn't change until Account.updateTrie() is called
func (a *Account) GetStorageRoot() common.Hash { return a.data.StorageRoot }

func (a *Account) SetBalance(balance *big.Int) {
	if balance.Sign() < 0 {
		log.Errorf("can't set negative balance %v to account %06x", balance, a.data.Address)
		panic(ErrNegativeBalance)
	}
	a.data.Balance = balance
}
func (a *Account) SetVersion(logType types.ChangeLogType, version uint32) {
	a.data.Versions[logType] = version
}
func (a *Account) SetSuicide(suicided bool) {
	if suicided {
		a.SetBalance(new(big.Int))
		a.SetCodeHash(common.Hash{})
		a.SetStorageRoot(common.Hash{})
	}
	a.suicided = suicided
}

func (a *Account) SetCodeHash(codeHash common.Hash) {
	a.data.CodeHash = codeHash
	if (a.data.CodeHash == common.Hash{}) {
		a.data.CodeHash = sha3Nil
	}
	a.code = nil
}
func (a *Account) SetStorageRoot(root common.Hash) {
	a.data.StorageRoot = root
	a.dirtyStorage = make(Storage)
	a.trie = nil
}

// Code returns the contract code associated with this account, if any.
func (a *Account) GetCode() (types.Code, error) {
	if a.code != nil {
		return a.code, nil
	}
	codeHash := a.data.CodeHash
	if codeHash == sha3Nil {
		return nil, nil
	}
	code, err := a.db.GetContractCode(codeHash)
	if err != nil {
		log.Errorf("can't load code hash %x: %v", a.data.CodeHash, err)
		return nil, err
	} else if code == nil {
		log.Errorf("can't load code hash %x", a.data.CodeHash)
		return nil, ErrLoadCodeFail
	} else {
		a.code = *code
	}
	return a.code, nil
}

func (a *Account) SetCode(code types.Code) {
	a.code = code
	oldHash := a.data.CodeHash
	newHash := crypto.Keccak256Hash(code)
	// hash changed and not both hash are empty
	if oldHash != newHash && !(oldHash == common.Hash{} && newHash == sha3Nil) {
		a.data.CodeHash = newHash
		a.codeIsDirty = true
	}
}

// GetState returns a value in account storage.
func (a *Account) GetStorageState(key common.Hash) ([]byte, error) {
	value, exists := a.cachedStorage[key]
	if exists {
		return value, nil
	}
	// Load from DB in case it is missing.
	tr, err := a.getTrie()
	if err != nil {
		log.Errorf("load trie by root 0x%x fail: %v", a.data.StorageRoot, err)
		return nil, ErrTrieFail
	}
	value, err = tr.TryGet(key[:])
	// ignore ErrNotExist, just return empty []byte
	if err != nil && err != store.ErrNotExist {
		return nil, err
	}
	if len(value) != 0 {
		a.cachedStorage[key] = value
	}
	return value, nil
}

// SetState updates a value in account storage.
func (a *Account) SetStorageState(key common.Hash, value []byte) error {
	// TODO the key is Hash already, but secureTrie hash it again?
	a.cachedStorage[key] = value
	a.dirtyStorage[key] = value
	return nil
}

// IsEmpty returns whether the state object is either non-existent or empty (version = 0)
func (a *Account) IsEmpty() bool {
	for _, v := range a.data.Versions {
		if v != 0 {
			return false
		}
	}
	return true
}

func (a *Account) getTrie() (*trie.SecureTrie, error) {
	if a.trie == nil {
		if a.trieDb == nil {
			a.trieDb = a.db.GetTrieDatabase()
		}
		var err error
		a.trie, err = trie.NewSecure(a.data.StorageRoot, a.trieDb, MaxTrieCacheGen)
		if err != nil {
			return nil, err
			// a.trie, _ = trie.NewSecure(common.Hash{}, trieDb, MaxTrieCacheGen)
		}
	}
	return a.trie, nil
}

// updateTrie writes cached storage modifications into storage trie.
func (a *Account) updateTrie() error {
	tr, err := a.getTrie()
	if err != nil {
		log.Errorf("load trie by root 0x%x fail: %v", a.data.StorageRoot, err)
		return ErrTrieFail
	}
	for key, value := range a.dirtyStorage {
		delete(a.dirtyStorage, key)
		if len(value) == 0 {
			err = tr.TryDelete(key[:])
			if err != nil {
				return err
			}
			continue
		}
		v := bytes.TrimLeft(value, "\x00")
		err = tr.TryUpdate(key[:], v)
		if err != nil {
			return err
		}
	}
	a.data.StorageRoot = tr.Hash()
	return nil
}

// Finalise finalises the state, clears the change caches and update tries.
func (a *Account) Finalise(blockHeight uint32) error {
	// update storage trie
	err := a.updateTrie()
	if err != nil {
		return err
	}
	// save the newest record
	if a.data.NewestRecords == nil {
		a.data.NewestRecords = make(map[types.ChangeLogType]types.VersionRecord)
	}
	for logType, version := range a.data.Versions {
		record, ok := a.data.NewestRecords[logType]
		if !ok || record.Version != version {
			a.data.NewestRecords[logType] = types.VersionRecord{Height: blockHeight, Version: version}
		}
	}
	return nil
}

// Save writes dirty data into db.
func (a *Account) Save() error {
	tr, err := a.getTrie()
	if err != nil {
		log.Errorf("load trie by root 0x%x fail: %v", a.data.StorageRoot, err)
		return ErrTrieFail
	}
	// update contract storage trie nodes' hash
	root, err := tr.Commit(nil)
	if err != nil {
		return err
	}
	if root != a.data.StorageRoot {
		return ErrTrieChanged
	}
	// save contract storage trie
	err = a.trieDb.Commit(root, false)
	if err != nil {
		log.Error("save contract storage fail", "address", a.data.Address)
		return err
	}
	// save code
	if a.codeIsDirty {
		if err := a.db.SetContractCode(a.data.CodeHash, &a.code); err != nil {
			return err
		}
		a.codeIsDirty = false
	}
	return nil
}

// LoadNewestChangeLogs loads newest change logs
func (a *Account) LoadNewestChangeLogs() ([]*types.ChangeLog, error) {
	var logs types.ChangeLogSlice
	// TODO the db interface has not been implemented
	// for height, version := range a.data.NewestRecords {
	// 	if version >= fromVersion {
	// 		oneBlockLogs, err := a.db.LoadChangeLog(a.data.Address, height)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		logs = append(logs, oneBlockLogs)
	// 	}
	// }
	sort.Sort(logs)
	return logs, nil
}
