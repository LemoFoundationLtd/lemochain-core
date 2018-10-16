package account

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"github.com/LemoFoundationLtd/lemochain-go/store/trie"
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

// Manager is used to maintain the newest and not confirmed account data. It will save all data to the db when finished a block's transactions processing.
type Manager struct {
	db     protocol.ChainDB
	trieDb *store.TrieDatabase // used to access tire data in file
	// Manager loads all data from the branch where the baseBlock is
	baseBlock     *types.Block
	baseBlockHash common.Hash

	// This map holds 'live' accounts, which will get modified while processing a state transition.
	accountCache map[common.Address]*SafeAccount

	processor   *logProcessor
	versionTrie *trie.SecureTrie
}

// NewManager creates a new Manager. it is used to maintain account changes based on the block environment which specified by blockHash
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
	manager.processor = &logProcessor{
		manager:    manager,
		changeLogs: make([]*types.ChangeLog, 0),
		events:     make([]*types.Event, 0),
	}
	return manager
}

// GetAccount loads account from cache or db, or creates a new one if it's not exist.
func (am *Manager) GetAccount(address common.Address) types.AccountAccessor {
	cached := am.accountCache[address]
	if cached == nil {
		data, err := am.db.GetAccount(am.baseBlockHash, address)
		if err != nil && err != store.ErrNotExist {
			panic(err)
		}
		account := NewAccount(am.db, address, data, am.baseBlockHeight())
		cached = NewSafeAccount(am.processor, account)
		// cache it
		am.accountCache[address] = cached
	}
	return cached
}

// GetCanonicalAccount loads an readonly account object from confirmed block in db, or creates a new one if it's not exist. The Modification of the account will not be recorded to store.
func (am *Manager) GetCanonicalAccount(address common.Address) types.AccountAccessor {
	data, err := am.db.GetCanonicalAccount(address)
	if err != nil && err != store.ErrNotExist {
		panic(err)
	}
	return NewAccount(am.db, address, data, am.baseBlockHeight())
}

// getRawAccount loads an account same as GetAccount, but editing the account of this method returned is not going to generate change logs.
// This method is used for ChangeLog.Redo/Undo.
func (am *Manager) getRawAccount(address common.Address) types.AccountAccessor {
	safeAccount := am.GetAccount(address)
	// Change this account will change safeAccount. They are same pointers
	return safeAccount.(*SafeAccount).rawAccount
}

// IsExist reports whether the given account address exists in the db.
// Notably this also returns true for suicided accounts.
func (am *Manager) IsExist(address common.Address) bool {
	_, err := am.db.GetAccount(am.baseBlockHash, address)
	return err == nil || err != store.ErrNotExist
}

// AddEvent records the event during transaction's execution.
func (am *Manager) AddEvent(event *types.Event) {
	if (event.Address == common.Address{}) {
		panic("account.Manager.AddEvent() is called without a Address or TxHash")
	}
	account := am.getRawAccount(event.Address)
	event.Index = uint(len(am.processor.events))
	am.processor.PushChangeLog(NewAddEventLog(account, event))
	am.processor.PushEvent(event)
}

// GetEvents returns all events since last reset
func (am *Manager) GetEvents() []*types.Event {
	return am.processor.events[:]
}

// GetEvents returns all events since last reset
func (am *Manager) GetEventsByTx(txHash common.Hash) []*types.Event {
	result := make([]*types.Event, 0)
	for _, event := range am.processor.events {
		if event.TxHash == txHash {
			result = append(result, event)
		}
	}
	return result
}

// GetChangeLogs returns all change logs since last reset
func (am *Manager) GetChangeLogs() []*types.ChangeLog {
	return am.processor.changeLogs
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
	am.processor.clear()
	am.versionTrie = nil
}

// Reset clears out all data and switch state to the new block environment.
func (am *Manager) Reset(blockHash common.Hash) {
	am.baseBlockHash = blockHash
	if err := am.loadBaseBlock(); err != nil {
		log.Errorf("Reset to block[%s] fail: %s\n", am.baseBlockHash.Hex(), err.Error())
		panic(err)
	}
	am.clear()
}

func (am *Manager) loadBaseBlock() (err error) {
	var block *types.Block
	if (am.baseBlockHash != common.Hash{}) {
		block, err = am.db.GetBlockByHash(am.baseBlockHash)
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

// Finalise finalises the state, clears the change caches and update tries.
func (am *Manager) Finalise() error {
	versionTrie := am.getVersionTrie()
	for _, account := range am.accountCache {
		if !account.IsDirty() {
			continue
		}
		// update account and contract storage
		if err := account.rawAccount.Finalise(); err != nil {
			return err
		}
		// update version trie
		for logType, record := range account.rawAccount.data.NewestRecords {
			k := versionTrieKey(account.GetAddress(), logType)
			version := big.NewInt(int64(record.Version)).Bytes()
			if err := versionTrie.TryUpdate(k, version); err != nil {
				return err
			}
		}
	}
	return nil
}

func versionTrieKey(address common.Address, logType types.ChangeLogType) []byte {
	return append(address.Bytes(), big.NewInt(int64(logType)).Bytes()...)
}

// Save writes dirty data into db.
func (am *Manager) Save(newBlockHash common.Hash) error {
	dirtyAccounts := make([]*types.AccountData, 0, len(am.accountCache))
	for _, account := range am.accountCache {
		if !account.IsDirty() {
			continue
		}
		if err := account.rawAccount.Save(); err != nil {
			return err
		}
		dirtyAccounts = append(dirtyAccounts, account.rawAccount.data)
	}
	// save accounts to db
	if len(dirtyAccounts) != 0 {
		if err := am.db.SetAccounts(newBlockHash, dirtyAccounts); err != nil {
			log.Errorf("save accounts to db fail: %v", err)
			return err
		}
	}
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
	am.clear()
	return nil
}

type revision struct {
	id           int
	journalIndex int
}

// logProcessor records change logs and contract events during block's transaction execution. It can access raw account data for undo/redo
type logProcessor struct {
	manager *Manager

	// The change logs generated by transactions from the same block
	changeLogs     []*types.ChangeLog
	validRevisions []revision
	nextRevisionId int

	// This map holds contract events generated by every transactions
	events []*types.Event
}

func (h *logProcessor) GetAccount(addr common.Address) types.AccountAccessor {
	return h.manager.getRawAccount(addr)
}

func (h *logProcessor) PushEvent(event *types.Event) {
	h.events = append(h.events, event)
}

func (h *logProcessor) PopEvent() error {
	size := len(h.events)
	if size == 0 {
		return ErrNoEvents
	}
	h.events = h.events[:size-1]
	return nil
}

func (h *logProcessor) PushChangeLog(log *types.ChangeLog) {
	h.changeLogs = append(h.changeLogs, log)
}

func (h *logProcessor) clear() {
	h.changeLogs = make([]*types.ChangeLog, 0)
	h.events = make([]*types.Event, 0)
}

// Snapshot returns an identifier for the current revision of the change log.
func (h *logProcessor) Snapshot() int {
	id := h.nextRevisionId
	h.nextRevisionId++
	h.validRevisions = append(h.validRevisions, revision{id, len(h.changeLogs)})
	return id
}

// checkRevisionAvailable check if the newest revision is accessible
func (h *logProcessor) checkRevisionAvailable() bool {
	if len(h.validRevisions) == 0 {
		return true
	}
	last := h.validRevisions[len(h.validRevisions)-1]
	return last.journalIndex <= len(h.changeLogs)
}

// RevertToSnapshot reverts all changes made since the given revision.
func (h *logProcessor) RevertToSnapshot(revid int) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(h.validRevisions), func(i int) bool {
		return h.validRevisions[i].id >= revid
	})
	if idx == len(h.validRevisions) || h.validRevisions[idx].id != revid {
		log.Errorf("revision id %v cannot be reverted", revid)
		panic(ErrRevisionNotExist)
	}
	snapshot := h.validRevisions[idx].journalIndex

	// Replay the change log to undo changes.
	for i := len(h.changeLogs) - 1; i >= snapshot; i-- {
		h.changeLogs[i].Undo(h)
	}
	h.changeLogs = h.changeLogs[:snapshot]

	// Remove invalidated snapshots from the stack.
	h.validRevisions = h.validRevisions[:idx]
}

// Rebuild loads and redo all change logs to update account to the newest state.
//
// TODO Changelog maybe retrieved from other node, so the account in local store is not contain the newest NewestRecords.
// We'd better change this function to "Rebuild(address common.Address, logs []types.ChangeLog)" cause the Rebuild function is called by change log synchronization module
func (am *Manager) Rebuild(address common.Address) error {
	accountAccessor := am.getRawAccount(address)
	account := accountAccessor.(*Account)
	logs, err := account.LoadNewestChangeLogs()
	for _, log := range logs {
		err = log.Redo(&logProcessor{manager: am})
		if err != nil && err != types.ErrAlreadyRedo {
			return err
		}
	}
	// save account
	return am.db.SetAccounts(am.baseBlockHash, []*types.AccountData{account.data})
}

// MergeChangeLogs merges the change logs for same account in block. Then update the version of change logs and account.
func (am *Manager) MergeChangeLogs(fromIndex int) {
	needMerge := am.processor.changeLogs[fromIndex:]
	mergedLogs, versionLogs := MergeChangeLogs(needMerge)
	am.processor.changeLogs = append(am.processor.changeLogs[:fromIndex], mergedLogs...)
	for _, changeLog := range mergedLogs {
		am.getRawAccount(changeLog.Address).SetVersion(changeLog.LogType, changeLog.Version)
	}
	for _, changeLog := range versionLogs {
		am.getRawAccount(changeLog.Address).SetVersion(changeLog.LogType, changeLog.Version)
	}
	// make sure the snapshot still work
	if !am.processor.checkRevisionAvailable() {
		panic(ErrSnapshotIsBroken)
	}
}

func (am *Manager) Stop(graceful bool) error {
	return nil
}

func (am *Manager) baseBlockHeight() uint32 {
	if am.baseBlock == nil {
		return 0
	}
	return am.baseBlock.Height()
}
