package account

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"github.com/LemoFoundationLtd/lemochain-core/store/trie"
	"math/big"
)

var (
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

type StorageCache struct {
	db     protocol.ChainDB    // used to access account data in cache or file
	trie   *trie.SecureTrie    // contract storage trie
	trieDb *store.TrieDatabase // used to access tire data in file
	cached Storage             // Storage entry cache to avoid duplicate reads
	dirty  Storage             // Storage entries that need to be flushed to disk
}

func NewStorageCache(db protocol.ChainDB) *StorageCache {
	return &StorageCache{
		db:     db,
		cached: make(Storage),
		dirty:  make(Storage),
	}
}

func (cache *StorageCache) Reset() {
	cache.trie = nil
	cache.cached = make(Storage)
	cache.dirty = make(Storage)
}

func (cache *StorageCache) GetTrie(root common.Hash) (*trie.SecureTrie, error) {
	if cache.trie == nil {
		if cache.trieDb == nil {
			cache.trieDb = cache.db.GetTrieDatabase()
		}

		var err error
		cache.trie, err = trie.NewSecure(root, cache.trieDb, MaxTrieCacheGen)
		if err != nil {
			return nil, err
		}
	}

	return cache.trie, nil
}

func (cache *StorageCache) Save(root common.Hash) error {
	if len(cache.dirty) > 0 {
		return ErrTrieChanged
	}

	if root != (common.Hash{}) {
		tr, err := cache.GetTrie(root)
		if err != nil {
			log.Errorf("load trie by root 0x%x fail: %v", root, err)
			return ErrTrieFail
		}
		// update contract storage trie nodes' hash
		result, err := tr.Commit(nil)
		if err != nil {
			return err
		}
		if root != result {
			return ErrTrieChanged
		}
		// save contract storage trie
		err = cache.trieDb.Commit(result, false)
		if err != nil {
			log.Error("save contract storage fail", "address")
			return err
		}
	}
	return nil
}

// updateTrie writes cached storage modifications into storage trie.
func (cache *StorageCache) Update(root common.Hash) (common.Hash, error) {
	if root == (common.Hash{}) && len(cache.dirty) == 0 {
		return common.Hash{}, nil
	}

	tr, err := cache.GetTrie(root)
	if err != nil {
		log.Errorf("load trie by root 0x%x fail: %v", root, err)
		return common.Hash{}, ErrTrieFail
	}

	for key, value := range cache.dirty {
		delete(cache.dirty, key)
		if len(value) == 0 {
			err = tr.TryDelete(key[:])
			if err != nil {
				return common.Hash{}, err
			} else {
				continue
			}
		}
		v := bytes.TrimLeft(value, "\x00")
		err = tr.TryUpdate(key[:], v)
		if err != nil {
			return common.Hash{}, err
		}
	}

	return tr.Hash(), nil
}

func (cache *StorageCache) SetState(key common.Hash, value []byte) error {
	cache.cached[key] = value
	cache.dirty[key] = value
	return nil
}

func (cache *StorageCache) DelState(key common.Hash) error {
	delete(cache.cached, key)
	delete(cache.dirty, key)
	return nil
}

func (cache *StorageCache) GetState(root common.Hash, key common.Hash) ([]byte, error) {
	value, exists := cache.cached[key]
	if exists {
		return value, nil
	}
	// Load from DB in case it is missing.
	tr, err := cache.GetTrie(root)
	if err != nil {
		log.Errorf("load trie by root 0x%x fail: %v", root, err)
		return nil, ErrTrieFail
	}
	value, err = tr.TryGet(key[:])
	// ignore ErrNotExist, just return empty []byte
	if err != nil && err != store.ErrNotExist {
		return nil, err
	}
	if len(value) != 0 {
		cache.cached[key] = value
	}
	return value, nil
}

// Account is used to read and write account data. the code and dirty storage K/V would be cached till they are flushing to db
type Account struct {
	data *types.AccountData
	db   protocol.ChainDB // used to access account data in cache or file

	storage   *StorageCache
	assetCode *StorageCache
	assetId   *StorageCache
	equity    *StorageCache

	code        types.Code // contract byte code
	codeIsDirty bool       // true if the code was updated

	events        []*types.Event
	newestRecords map[types.ChangeLogType]uint32
	suicided      bool // will be delete from the trie during the "save" phase
}

func (a *Account) SetSingers(signers types.Signers) error {
	if len(signers) <= 0 {
		a.data.Signers = make(types.Signers, 0)
	} else {
		a.data.Signers = make(types.Signers, 0, len(signers))
		a.data.Signers = append(a.data.Signers, signers...)
	}

	return nil
}

func (a *Account) GetSigners() types.Signers {
	if len(a.data.Signers) <= 0 {
		return make(types.Signers, 0)
	} else {
		result := make(types.Signers, 0, len(a.data.Signers))
		return append(result, a.data.Signers...)
	}
}

func (a *Account) PushEvent(event *types.Event) {
	a.events = append(a.events, event)
}

func (a *Account) PopEvent() error {
	size := len(a.events)
	if size == 0 {
		return ErrNoEvents
	}
	a.events = a.events[:size-1]
	return nil
}

func (a *Account) GetEvents() []*types.Event {
	return a.events
}

// NewAccount wrap an AccountData object, or creates a new one if it's nil.
func NewAccount(db protocol.ChainDB, address common.Address, data *types.AccountData) *Account {
	if data == nil {
		data = &types.AccountData{Address: address}
	} else {
		data = data.Copy()
	}

	if data.Balance == nil {
		data.Balance = new(big.Int)
	}

	if data.NewestRecords == nil {
		data.NewestRecords = make(map[types.ChangeLogType]types.VersionRecord)
	}

	if data.Signers == nil {
		data.Signers = make(types.Signers, 0)
	}

	if data.Candidate.Profile == nil {
		data.Candidate.Profile = make(types.Profile)
	}

	if data.Candidate.Votes == nil {
		data.Candidate.Votes = new(big.Int)
	}

	account := &Account{
		data:          data,
		db:            db,
		events:        make([]*types.Event, 0),
		newestRecords: make(map[types.ChangeLogType]uint32),
		storage:       NewStorageCache(db),
		assetCode:     NewStorageCache(db),
		assetId:       NewStorageCache(db),
		equity:        NewStorageCache(db),
	}

	for k, v := range data.NewestRecords {
		account.newestRecords[k] = v.Version
	}
	return account
}

// MarshalJSON encodes the client RPC account format.
func (a *Account) MarshalJSON() ([]byte, error) {
	return a.data.MarshalJSON()
}

// UnmarshalJSON decodes the client RPC account format.
func (a *Account) UnmarshalJSON(input []byte) error {
	dec := new(types.AccountData)
	if err := dec.UnmarshalJSON(input); err != nil {
		return err
	}
	// TODO a.db is always nil
	*a = *NewAccount(a.db, dec.Address, dec)
	return nil
}

func (a *Account) String() string {
	return a.data.String()
}

func (a *Account) GetVoteFor() common.Address { return a.data.VoteFor }

func (a *Account) SetVoteFor(addr common.Address) {
	a.data.VoteFor = addr
}

func (a *Account) GetVotes() *big.Int {
	return a.data.Candidate.Votes
}

func (a *Account) SetVotes(votes *big.Int) {
	a.data.Candidate.Votes = new(big.Int).Set(votes)
}

func (a *Account) GetCandidate() types.Profile {
	if a.data.Candidate.Profile == nil {
		return make(types.Profile)
	} else {
		result := make(types.Profile)
		for k, v := range a.data.Candidate.Profile {
			result[k] = v
		}
		return result
	}
}

func (a *Account) SetCandidate(profile types.Profile) {
	a.data.Candidate.Profile = make(types.Profile)
	for k, v := range profile {
		a.data.Candidate.Profile[k] = v
	}
}

func (a *Account) GetCandidateState(key string) string {
	if a.data.Candidate.Profile == nil {
		return ""
	} else {
		return a.data.Candidate.Profile[key]
	}
}

func (a *Account) SetCandidateState(key string, val string) {
	if a.data.Candidate.Profile == nil {
		a.data.Candidate.Profile = make(types.Profile)
	}

	a.data.Candidate.Profile[key] = val
}

// Implement AccountAccessor. Access Account without changelog
func (a *Account) GetAddress() common.Address { return a.data.Address }
func (a *Account) GetBalance() *big.Int       { return new(big.Int).Set(a.data.Balance) }

// GetBaseVersion returns the version of specific change log from the base block. It is not changed by tx processing until the finalised
func (a *Account) GetVersion(logType types.ChangeLogType) uint32 {
	return a.data.NewestRecords[logType].Version
}

// GetBaseVersion returns the version of specific change log from the base block. It is not changed by tx processing until the finalised
func (a *Account) GetNextVersion(logType types.ChangeLogType) uint32 {
	a.newestRecords[logType] = a.newestRecords[logType] + 1
	return a.newestRecords[logType]
}

func (a *Account) SetVersion(logType types.ChangeLogType, version, blockHeight uint32) {
	a.newestRecords[logType] = version
	a.data.NewestRecords[logType] = types.VersionRecord{Version: version, Height: blockHeight}
}
func (a *Account) GetSuicide() bool         { return a.suicided }
func (a *Account) GetCodeHash() common.Hash { return a.data.CodeHash }

// StorageRoot wouldn't change until Account.updateTrie() is called
func (a *Account) GetStorageRoot() common.Hash { return a.data.StorageRoot }

func (a *Account) SetStorageRoot(root common.Hash) {
	a.data.StorageRoot = root
	a.storage.Reset()
}

func (a *Account) GetAssetCodeRoot() common.Hash { return a.data.AssetCodeRoot }

func (a *Account) SetAssetCodeRoot(root common.Hash) {
	a.data.AssetCodeRoot = root
	a.assetCode.Reset()
}

func (a *Account) GetAssetIdRoot() common.Hash { return a.data.AssetIdRoot }

func (a *Account) SetAssetIdRoot(root common.Hash) {
	a.data.AssetIdRoot = root
	a.assetId.Reset()
}

func (a *Account) GetEquityRoot() common.Hash { return a.data.EquityRoot }

func (a *Account) SetEquityRoot(root common.Hash) {
	a.data.EquityRoot = root
	a.equity.Reset()
}

func (a *Account) SetBalance(balance *big.Int) {
	if balance.Sign() < 0 {
		log.Errorf("can't set negative balance %v to account %s", balance, a.data.Address.String())
		panic(ErrNegativeBalance)
	}
	a.data.Balance.Set(balance)
}

func (a *Account) SetSuicide(suicided bool) {
	if suicided {
		a.SetBalance(new(big.Int))
		a.SetCodeHash(common.Hash{})
		a.SetStorageRoot(common.Hash{})
		a.SetAssetCodeRoot(common.Hash{})
		a.SetAssetIdRoot(common.Hash{})
	}
	a.suicided = suicided
}

func (a *Account) SetCodeHash(codeHash common.Hash) {
	a.data.CodeHash = codeHash
	a.code = nil
}

// Code returns the contract code associated with this account, if any.
func (a *Account) GetCode() (types.Code, error) {
	if a.code != nil {
		return a.code, nil
	}
	codeHash := a.data.CodeHash
	if codeHash == common.Sha3Nil || codeHash == (common.Hash{}) {
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
		a.code = code
	}
	return a.code, nil
}

func (a *Account) SetCode(code types.Code) {
	a.code = code
	oldHash := a.data.CodeHash
	newHash := crypto.Keccak256Hash(code)
	// hash changed and not both hash are empty
	if oldHash != newHash && !(oldHash == common.Hash{} && newHash == common.Sha3Nil) {
		a.data.CodeHash = newHash
		a.codeIsDirty = true
	}
}

// GetState returns a value in account storage.
func (a *Account) GetStorageState(key common.Hash) ([]byte, error) {
	return a.storage.GetState(a.data.StorageRoot, key)
}

// SetState updates a value in account storage.
func (a *Account) SetStorageState(key common.Hash, value []byte) error {
	return a.storage.SetState(key, value)
}

func (a *Account) GetAssetCode(code common.Hash) (*types.Asset, error) {
	val, err := a.assetCode.GetState(a.data.AssetCodeRoot, code)
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, store.ErrNotExist
	}

	var asset types.Asset
	asset.TotalSupply = new(big.Int)
	asset.Profile = make(types.Profile)
	err = rlp.DecodeBytes(val, &asset)
	if err != nil {
		return nil, err
	} else {
		return &asset, nil
	}
}

func (a *Account) SetAssetCode(code common.Hash, asset *types.Asset) error {
	if asset == nil {
		return a.assetCode.DelState(code)
	}

	val, err := rlp.EncodeToBytes(asset)
	if err != nil {
		return err
	} else {
		return a.assetCode.SetState(code, val)
	}
}

func (a *Account) GetAssetCodeTotalSupply(code common.Hash) (*big.Int, error) {
	asset, err := a.GetAssetCode(code)
	if err != nil {
		return nil, err
	}

	if asset == nil {
		return new(big.Int).SetInt64(0), nil
	}

	return asset.TotalSupply, nil
}

func (a *Account) SetAssetCodeTotalSupply(code common.Hash, val *big.Int) error {
	asset, err := a.GetAssetCode(code)
	if err != nil {
		return err
	} else {
		asset.TotalSupply = val
		return a.SetAssetCode(code, asset)
	}
}

func (a *Account) GetAssetCodeState(code common.Hash, key string) (string, error) {
	asset, err := a.GetAssetCode(code)
	if err != nil {
		return "", err
	}

	return asset.Profile[key], nil
}

func (a *Account) SetAssetCodeState(code common.Hash, key string, val string) error {
	asset, err := a.GetAssetCode(code)
	if err != nil {
		return err
	} else {
		asset.Profile[key] = val
		return a.SetAssetCode(code, asset)
	}
}

func (a *Account) GetAssetIdState(id common.Hash) (string, error) {
	root := a.GetAssetIdRoot()
	val, err := a.assetId.GetState(root, id)
	if err != nil {
		return "", err
	} else {
		return string(val), nil
	}
}

func (a *Account) SetAssetIdState(id common.Hash, data string) error {
	return a.assetId.SetState(id, []byte(data))
}

func (a *Account) GetEquityState(id common.Hash) (*types.AssetEquity, error) {
	val, err := a.equity.GetState(a.data.EquityRoot, id)
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, store.ErrNotExist
	}

	var equity types.AssetEquity
	err = rlp.DecodeBytes(val, &equity)
	if err != nil {
		return nil, err
	} else {
		return &equity, nil
	}
}

func (a *Account) SetEquityState(id common.Hash, equity *types.AssetEquity) error {
	if equity == nil {
		return a.equity.SetState(id, nil)
	} else {
		val, err := rlp.EncodeToBytes(equity)
		if err != nil {
			return err
		} else {
			return a.equity.SetState(id, val)
		}
	}
}

// IsEmpty returns whether the state object is either non-existent or empty (version = 0)
func (a *Account) IsEmpty() bool {
	for _, record := range a.data.NewestRecords {
		if record.Version != 0 {
			return false
		}
	}
	return true
}

// updateTrie writes cached storage modifications into storage trie.
func (a *Account) updateTrie() error {
	hash, err := a.storage.Update(a.data.StorageRoot)
	if err != nil {
		return err
	} else {
		a.data.StorageRoot = hash
	}

	hash, err = a.assetCode.Update(a.data.AssetCodeRoot)
	if err != nil {
		return err
	} else {
		a.data.AssetCodeRoot = hash
	}

	hash, err = a.assetId.Update(a.data.AssetIdRoot)
	if err != nil {
		return err
	} else {
		a.data.AssetIdRoot = hash
	}

	hash, err = a.equity.Update(a.data.EquityRoot)
	if err != nil {
		return err
	} else {
		a.data.EquityRoot = hash
	}

	return nil
}

// Finalise finalises the state, clears the change caches and update tries.
func (a *Account) Finalise() error {
	// update storage trie
	return a.updateTrie()
}

// Save writes dirty data into db.
func (a *Account) Save() error {
	if a.data.StorageRoot != (common.Hash{}) {
		err := a.storage.Save(a.data.StorageRoot)
		if err != nil {
			return err
		}
	}

	if a.data.AssetCodeRoot != (common.Hash{}) {
		err := a.assetCode.Save(a.data.AssetCodeRoot)
		if err != nil {
			return err
		}
	}

	if a.data.AssetIdRoot != (common.Hash{}) {
		err := a.assetId.Save(a.data.AssetIdRoot)
		if err != nil {
			return err
		}
	}

	if a.data.EquityRoot != (common.Hash{}) {
		err := a.equity.Save(a.data.EquityRoot)
		if err != nil {
			return err
		}
	}

	// save code
	if a.codeIsDirty {
		if err := a.db.SetContractCode(a.data.CodeHash, a.code); err != nil {
			return err
		}
		a.codeIsDirty = false
	}
	return nil
}
