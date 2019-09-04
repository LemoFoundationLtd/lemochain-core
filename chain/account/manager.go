package account

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"github.com/LemoFoundationLtd/lemochain-core/store/trie"
	"math/big"
	"sort"
)

// Trie cache generation limit after which to evict trie nodes from memory.
const MaxTrieCacheGen = uint16(120)

var (
	ErrRevisionNotExist = errors.New("revision cannot be reverted")
	ErrNoEvents         = errors.New("the times of pop event is more than push")
	ErrSnapshotIsBroken = errors.New("the snapshot is broken")
)

// TxsProduct is the product of transaction execution
type TxsProduct struct {
	Txs         types.Transactions // The transactions executed indeed. These transactions will be packaged in a block
	GasUsed     uint64             // gas used by all transactions
	ChangeLogs  types.ChangeLogSlice
	VersionRoot common.Hash
}

// Manager is used to maintain the newest and not confirmed account data. It will save all data to the db when finished a block's transactions processing.
type Manager struct {
	db     protocol.ChainDB
	trieDb *store.TrieDatabase // used to access tire data in file
	acctDb *store.AccountTrieDB
	// Manager loads all data from the branch where the baseBlock is
	baseBlock     *types.Block
	baseBlockHash common.Hash

	// This map holds 'live' accounts, which will get modified while processing a state transition.
	accountCache map[common.Address]*SafeAccount

	processor   *LogProcessor
	versionTrie *trie.SecureTrie
}

// NewManager creates a new Manager. It is used to maintain account changes based on the block environment which specified by blockHash
func NewManager(blockHash common.Hash, db protocol.ChainDB) *Manager {
	if db == nil {
		panic("account.NewManager is called without a database")
	}
	manager := &Manager{
		db:            db,
		baseBlockHash: blockHash,
		accountCache:  make(map[common.Address]*SafeAccount),
		trieDb:        db.GetTrieDatabase(),
	}
	if err := manager.loadBaseBlock(); err != nil {
		log.Errorf("load block[%s] fail: %s\n", manager.baseBlockHash.Hex(), err.Error())
		panic(err)
	}

	manager.acctDb, _ = db.GetActDatabase(blockHash)
	manager.processor = NewLogProcessor(manager)
	return manager
}

// GetAccount loads account from cache or db, or creates a new one if it's not exist.
func (am *Manager) GetAccount(address common.Address) types.AccountAccessor {
	cached := am.accountCache[address]
	if cached == nil {
		data, _ := am.acctDb.Get(address)
		account := NewAccount(am.db, address, data)
		cached = NewSafeAccount(am.processor, account)
		// cache it
		am.accountCache[address] = cached
	}
	return cached
}

// GetCanonicalAccount loads an readonly account object from confirmed block in db, or creates a new one if it's not exist. The Modification of the account will not be recorded to store.
func (am *Manager) GetCanonicalAccount(address common.Address) types.AccountAccessor {
	data, err := am.db.GetAccount(address)
	if err != nil && err != store.ErrNotExist {
		panic(err)
	}
	return NewAccount(am.db, address, data)
}

// getRawAccount loads an account same as GetAccount, but editing the account of this method returned is not going to generate change logs.
// This method is used for ChangeLog.Redo/Undo.
func (am *Manager) getRawAccount(address common.Address) *Account {
	safeAccount := am.GetAccount(address)
	// Change this account will change safeAccount. They are same pointers
	return safeAccount.(*SafeAccount).rawAccount
}

// AddEvent records the event during transaction's execution.
func (am *Manager) AddEvent(event *types.Event) {
	if (event.Address == common.Address{}) {
		panic("account.Manager.AddEvent() is called without a Address or TxHash")
	}

	account := am.GetAccount(event.Address)
	account.PushEvent(event)
}

// // GetEvents returns all events since last reset
func (am *Manager) GetEvents() []*types.Event {
	events := make([]*types.Event, 0)
	for _, v := range am.accountCache {
		events = append(events, v.GetEvents()...)
	}
	return events
}

// GetEvents returns all events since last reset
// func (am *Manager) GetEventsByTx(txHash common.Hash) []*types.Event {
// 	result := make([]*types.Event, 0)
// 	for _, event := range am.processor.GetEvents() {
// 		if event.TxHash == txHash {
// 			result = append(result, event)
// 		}
// 	}
// 	return result
// }

// GetChangeLogs returns all change logs since last reset
func (am *Manager) GetChangeLogs() types.ChangeLogSlice {
	return am.processor.GetChangeLogs()
}

// getVersionTrie loads version trie by the version root from baseBlockHash
func (am *Manager) getVersionTrie() *trie.SecureTrie {
	if am.versionTrie == nil {
		var root common.Hash
		// not genesis block
		if (am.baseBlockHash != common.Hash{}) {
			// load last version trie root
			root = am.baseBlock.Header.VersionRoot
		}

		var err error
		am.versionTrie, err = trie.NewSecure(root, am.trieDb, MaxTrieCacheGen)
		if err != nil {
			panic(err)
		}
	}
	return am.versionTrie
}

func (am *Manager) GetVersionRoot() common.Hash {
	return am.getVersionTrie().Hash()
}

// clear clears all data from one block so that Manager can get ready to process another block's transactions
func (am *Manager) clear() {
	am.accountCache = make(map[common.Address]*SafeAccount)
	am.processor.Clear()
	am.versionTrie = nil
}

// Reset clears out all data and switch state to the new block environment.
func (am *Manager) Reset(blockHash common.Hash) {
	am.baseBlockHash = blockHash
	if err := am.loadBaseBlock(); err != nil {
		log.Errorf("Reset to block[%s] fail: %s\n", am.baseBlockHash.Hex(), err.Error())
		panic(err)
	}

	am.acctDb, _ = am.db.GetActDatabase(blockHash)
	am.clear()
}

func (am *Manager) loadBaseBlock() (err error) {
	var block *types.Block
	if (am.baseBlockHash != common.Hash{}) {
		block, err = am.db.GetBlockByHash(am.baseBlockHash)
		if err != nil {
			panic("load base block err: " + err.Error())
		}
	}
	am.baseBlock = block
	return err
}

// Snapshot returns an identifier for the current revision of the state.
func (am *Manager) Snapshot() int {
	return am.processor.Snapshot()
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (am *Manager) RevertToSnapshot(revid int) {
	am.processor.RevertToSnapshot(revid)
}

func (am *Manager) logGrouping() map[common.Address]types.ChangeLogSlice {
	logsByAccount := make(map[common.Address]types.ChangeLogSlice)
	logs := am.processor.changeLogs
	for _, log := range logs {
		logsByAccount[log.Address] = append(logsByAccount[log.Address], log)
	}
	return logsByAccount
}

// updateVersion
func (am *Manager) updateVersion(logs types.ChangeLogSlice, account *SafeAccount) error {
	currentHeight := am.CurrentBlockHeight()
	versionTrie := am.getVersionTrie()
	eventIndex := uint(0)
	for _, changeLog := range logs {
		if changeLog.LogType == AddEventLog {
			newVal := changeLog.NewVal.(*types.Event)
			newVal.Index = eventIndex
			eventIndex = eventIndex + 1
		}

		nextVersion := account.rawAccount.GetVersion(changeLog.LogType) + 1

		// set version record in rawAccount.data.NewestRecords
		changeLog.Version = nextVersion
		account.rawAccount.SetVersion(changeLog.LogType, nextVersion, currentHeight)

		// update version trie
		k := versionTrieKey(account.GetAddress(), changeLog.LogType)
		if err := versionTrie.TryUpdate(k, big.NewInt(int64(changeLog.Version)).Bytes()); err != nil {
			return err
		}
	}
	return nil
}

// Finalise finalises the state, clears the change caches and update tries.
func (am *Manager) Finalise() error {
	logsByAccount := am.logGrouping()

	// 排序
	addressList := make(common.AddressSlice, 0, len(am.accountCache))
	for addr := range am.accountCache {
		addressList = append(addressList, addr)
	}
	sort.Sort(addressList)

	var account *SafeAccount
	for _, addr := range addressList {
		account = am.accountCache[addr]
		logs := logsByAccount[account.GetAddress()]
		if len(logs) <= 0 {
			continue
		}
		// 更新root log
		oldStorageRoot := account.rawAccount.GetStorageRoot()
		oldAssetCodeRoot := account.rawAccount.GetAssetCodeRoot()
		oldAssetIdRoot := account.rawAccount.GetAssetIdRoot()
		oldEquityRoot := account.rawAccount.GetEquityRoot()

		// update account and contract storage
		if err := account.rawAccount.Finalise(); err != nil {
			return err
		}

		newStorageRoot := account.rawAccount.GetStorageRoot()
		newAssetCodeRoot := account.rawAccount.GetAssetCodeRoot()
		newAssetIdRoot := account.rawAccount.GetAssetIdRoot()
		newEquityRoot := account.rawAccount.GetEquityRoot()

		if newStorageRoot != oldStorageRoot {
			log := NewStorageRootLog(account.GetAddress(), am.processor, oldStorageRoot, newStorageRoot)
			am.processor.PushChangeLog(log)
		}

		if newAssetCodeRoot != oldAssetCodeRoot {
			log, _ := NewAssetCodeRootLog(account.GetAddress(), am.processor, oldAssetCodeRoot, newAssetCodeRoot)
			am.processor.PushChangeLog(log)
		}

		if newAssetIdRoot != oldAssetIdRoot {
			log, _ := NewAssetIdRootLog(account.GetAddress(), am.processor, oldAssetIdRoot, newAssetIdRoot)
			am.processor.PushChangeLog(log)
		}

		if newEquityRoot != oldEquityRoot {
			log, _ := NewEquityRootLog(account.GetAddress(), am.processor, oldEquityRoot, newEquityRoot)
			am.processor.PushChangeLog(log)
		}
		// 更新change log的版本号
		if err := am.updateVersion(logs, account); err != nil {
			return err
		}
	}
	return nil
}

func versionTrieKey(address common.Address, logType types.ChangeLogType) []byte {
	return append(address.Bytes(), big.NewInt(int64(logType)).Bytes()...)
}

// Save writes dirty data into db.
func (am *Manager) Save(newBlockHash common.Hash) error {
	logsByAccount := am.logGrouping()
	acctDatabase, _ := am.db.GetActDatabase(newBlockHash)
	for _, account := range am.accountCache {
		if len(logsByAccount[account.GetAddress()]) <= 0 {
			continue
		}
		if err := account.rawAccount.Save(); err != nil {
			return err
		}

		// save accounts to db
		acctDatabase.Put(account.rawAccount.data, am.CurrentBlockHeight())
	}

	am.db.CandidatesRanking(newBlockHash)

	// update version trie nodes' hash
	root, err := am.getVersionTrie().Commit(nil)
	if err != nil {
		return err
	}
	// save version trie
	err = am.trieDb.Commit(root, false)
	if err != nil {
		log.Errorf("save version trie fail: %v", err)
		return err
	}
	log.Debugf("save version trie success: %#x", root)
	am.clear()
	return nil
}

// GetTxProduct get the product of transaction execution
func (am *Manager) GetTxsProduct(txs types.Transactions, gasUsed uint64) *TxsProduct {
	return &TxsProduct{
		Txs:         txs,
		GasUsed:     gasUsed,
		ChangeLogs:  am.GetChangeLogs(),
		VersionRoot: am.GetVersionRoot(),
	}
}

// Rebuild loads and redo all change logs to update account to the newest state.
func (am *Manager) Rebuild(address common.Address, logs types.ChangeLogSlice) error {
	_, err := am.processor.Rebuild(address, logs)
	if err != nil {
		return err
	}
	return nil
	// save account
	// return am.db.SetAccounts(am.baseBlockHash, []*types.AccountData{account.data})
}

// MergeChangeLogs merges the change logs for same account in block. Then update the version of change logs and account.
func (am *Manager) MergeChangeLogs() {
	am.processor.MergeChangeLogs()
}

func (am *Manager) Stop(graceful bool) error {
	return nil
}

func (am *Manager) CurrentBlockHeight() uint32 {
	if am.baseBlock == nil {
		return 0
	}
	return am.baseBlock.Height() + 1
}

func (am *Manager) RebuildAll(b *types.Block) error {
	am.Reset(b.ParentHash())
	if b.ChangeLogs != nil {
		for _, cl := range b.ChangeLogs {
			if cl.LogType == StorageRootLog ||
				cl.LogType == AssetIdRootLog ||
				cl.LogType == AssetCodeRootLog ||
				cl.LogType == EquityRootLog {
				continue
			}

			if err := cl.Redo(am.processor); err != nil {
				return err
			}

			am.processor.changeLogs = append(am.processor.changeLogs, cl)
		}
	}
	return nil
}
