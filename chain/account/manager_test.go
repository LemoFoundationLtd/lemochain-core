package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestNewManager_withoutDB(t *testing.T) {
	defer func() {
		err := recover()
		assert.Equal(t, "account.NewManager is called without a database", err)
	}()
	NewManager(common.Hash{}, nil)
}

func TestNewManager(t *testing.T) {
	NewManager(common.Hash{}, newDB())
}

func TestNewManager_GetAccount(t *testing.T) {
	db := newDB()

	// exist in db
	manager := NewManager(defaultBlock.hash, db)
	account, err := manager.GetAccount(defaultAccounts[0].Address)
	assert.NoError(t, err)
	assert.Equal(t, uint32(100), account.GetVersion())
	assert.Equal(t, false, account.IsEmpty())
	// not exist in db
	account, err = manager.GetAccount(common.HexToAddress("0xaaa"))
	assert.NoError(t, err)
	assert.Equal(t, common.HexToAddress("0xaaa"), account.GetAddress())
	assert.Equal(t, true, account.IsEmpty())

	// load from genesis' parent block
	manager = NewManager(common.Hash{}, db)
	account, err = manager.GetAccount(common.Address{})
	assert.NoError(t, err)
	assert.Equal(t, common.Address{}, account.GetAddress())
	assert.Equal(t, account, manager.accountCache[common.Address{}])
}

func TestNewManager_GetCanonicalAccount(t *testing.T) {
	db := newDB()

	// exist in db
	manager := NewManager(defaultBlock.hash, db)
	account, err := manager.GetCanonicalAccount(defaultAccounts[0].Address)
	assert.NoError(t, err)
	assert.Equal(t, uint32(100), account.GetVersion())
	// not exist in db
	account, err = manager.GetCanonicalAccount(common.HexToAddress("0xaaa"))
	assert.NoError(t, err)
	assert.Equal(t, common.HexToAddress("0xaaa"), account.GetAddress())

	// load from genesis' parent block
	manager = NewManager(common.Hash{}, db)
	account, err = manager.GetCanonicalAccount(common.Address{})
	assert.NoError(t, err)
	assert.Equal(t, common.Address{}, account.GetAddress())
	assert.Equal(t, account, manager.accountCache[common.Address{}])
}

func TestChangeLogProcessor_GetAccount(t *testing.T) {
	manager := NewManager(defaultBlock.hash, newDB())

	// not exist in db
	address := common.HexToAddress("0xaaa")
	rawAccount, err := manager.processor.GetAccount(address)
	assert.NoError(t, err)
	assert.Equal(t, address, rawAccount.GetAddress())
	assert.NotEmpty(t, manager.accountCache[address])

	// exist in db
	rawAccount, err = manager.processor.GetAccount(defaultAccounts[0].Address)
	assert.NoError(t, err)
	assert.Equal(t, uint32(100), rawAccount.GetVersion())

	// change rawAccount, account should change
	account, err := manager.GetAccount(defaultAccounts[0].Address)
	rawAccount.SetBalance(big.NewInt(2))
	assert.Equal(t, rawAccount.GetBalance(), account.GetBalance())

	// change account, rawAccount should change
	account.SetBalance(big.NewInt(3))
	assert.Equal(t, account.GetBalance(), rawAccount.GetBalance())
}

func TestChangeLogProcessor_PushEvent_PopEvent(t *testing.T) {
	manager := NewManager(defaultBlock.hash, newDB())

	// push
	manager.processor.PushEvent(&types.Event{Address: common.HexToAddress("0x1"), TxHash: th(1), BlockHeight: 11})
	manager.processor.PushEvent(&types.Event{Address: common.HexToAddress("0x1"), BlockHeight: 22})
	events := manager.GetEvents()
	assert.Equal(t, 2, len(events))
	assert.Equal(t, uint32(22), events[1].BlockHeight)

	// pop
	err := manager.processor.PopEvent()
	assert.NoError(t, err)
	events = manager.GetEvents()
	assert.Equal(t, 1, len(events))
	assert.Equal(t, uint32(11), events[0].BlockHeight)
	err = manager.processor.PopEvent()
	assert.NoError(t, err)
	err = manager.processor.PopEvent()
	assert.Equal(t, ErrNoEvents, err)
}

func TestChangeLogProcessor_PushChangeLog_GetChangeLogs(t *testing.T) {
	manager := NewManager(defaultBlock.hash, newDB())

	manager.processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(101),
	})
	account, _ := manager.GetAccount(defaultAccounts[0].Address)

	// create a new change log
	account.SetBalance(big.NewInt(999))
	logs := manager.GetChangeLogs()
	assert.Equal(t, 2, len(logs))
	assert.Equal(t, types.ChangeLogType(101), logs[0].LogType)
	assert.Equal(t, *big.NewInt(999), logs[1].NewVal.(big.Int))
}

func TestNewManager_Snapshot_RevertToSnapshot(t *testing.T) {
	manager := NewManager(defaultBlock.hash, newDB())

	// snapshot when empty
	assert.Equal(t, 0, len(manager.processor.validRevisions))
	assert.Equal(t, 0, manager.processor.nextRevisionId)
	newId := manager.Snapshot()
	assert.Equal(t, 0, newId)
	assert.Equal(t, 1, len(manager.processor.validRevisions))
	assert.Equal(t, 1, manager.processor.nextRevisionId)
	// revert to current snapshot
	manager.RevertToSnapshot(newId)
	assert.Equal(t, 0, len(manager.processor.validRevisions))

	// revert not exist version
	assert.PanicsWithValue(t, ErrRevisionNotExist, func() {
		manager.RevertToSnapshot(100)
	})

	// create a new snapshot than revert
	account, _ := manager.GetAccount(defaultAccounts[0].Address)
	newId = manager.Snapshot()
	assert.Equal(t, 1, newId)
	account.SetBalance(big.NewInt(999))
	manager.RevertToSnapshot(newId)
	assert.Equal(t, defaultAccounts[0].Balance, account.GetBalance())

	// create two new snapshot than revert one
	account, _ = manager.GetAccount(common.HexToAddress("0x1"))
	newId = manager.Snapshot()
	account.SetBalance(big.NewInt(999))
	newId = manager.Snapshot()
	account.SetBalance(big.NewInt(1))
	assert.Equal(t, 2, len(manager.processor.validRevisions))
	manager.RevertToSnapshot(newId)
	assert.Equal(t, 1, len(manager.processor.validRevisions))
	assert.Equal(t, big.NewInt(999), account.GetBalance())
}

func TestNewManager_AddEvent(t *testing.T) {
	manager := NewManager(defaultBlock.hash, newDB())

	event1 := &types.Event{Address: common.HexToAddress("0x1"), TxHash: th(1), BlockHeight: 11}
	event2 := &types.Event{Address: common.HexToAddress("0x1"), TxHash: th(1), BlockHeight: 22}
	err := manager.AddEvent(event1)
	assert.NoError(t, err)
	err = manager.AddEvent(event2)
	assert.NoError(t, err)
	events := manager.GetEvents()
	assert.Equal(t, 2, len(events))
	assert.Equal(t, uint32(11), events[0].BlockHeight)
	logs := manager.GetChangeLogs()
	assert.Equal(t, 2, len(logs))
	assert.Equal(t, AddEventLog, logs[0].LogType)
	assert.Equal(t, event1, logs[0].NewVal.(*types.Event))
}

func TestNewManager_GetVersionRoot(t *testing.T) {
	db := newDB()

	// empty version trie
	manager := NewManager(common.Hash{}, db)
	root := manager.GetVersionRoot()
	assert.Equal(t, emptyTrieRoot, root)

	// empty version trie
	manager = NewManager(defaultBlock.hash, db)
	root = manager.GetVersionRoot()
	assert.Equal(t, defaultBlock.versionRoot, root)
}

func TestNewManager_Reset(t *testing.T) {
	manager := NewManager(common.Hash{}, newDB())

	account, _ := manager.GetAccount(common.HexToAddress("0x1"))
	account.SetBalance(big.NewInt(2))
	manager.GetVersionRoot()
	assert.NotEmpty(t, manager.accountCache)
	assert.NotEmpty(t, manager.processor.changeLogs)
	assert.NotEmpty(t, manager.versionTrie)
	manager.Reset(defaultBlock.hash)
	assert.Equal(t, defaultBlock.hash, manager.baseBlockHash)
	assert.Empty(t, manager.accountCache)
	assert.Empty(t, manager.processor.changeLogs)
	assert.Empty(t, manager.processor.events)
	assert.Empty(t, manager.versionTrie)
}

func TestNewManager_Finalise_Save(t *testing.T) {
	db := newDB()
	manager := NewManager(defaultBlock.hash, db)

	// nothing to finalise
	account, err := manager.GetAccount(common.HexToAddress("0x1"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(manager.accountCache))
	assert.Equal(t, false, account.(*SafeAccount).IsDirty())
	err = manager.Finalise()
	assert.NoError(t, err)
	// save
	err = manager.Save(b(1))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(manager.accountCache))

	// finalise dirty
	account, err = manager.GetAccount(defaultAccounts[0].Address)
	assert.NoError(t, err)
	account.SetBalance(big.NewInt(2))
	err = account.SetStorageState(k(1), []byte{100})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(manager.accountCache))
	assert.Equal(t, true, account.(*SafeAccount).IsDirty())
	assert.Equal(t, 2, len(manager.processor.changeLogs))
	root := manager.GetVersionRoot()
	assert.Equal(t, defaultBlock.versionRoot, root)
	err = manager.Finalise()
	assert.NoError(t, err)
	root = manager.GetVersionRoot()
	assert.NotEqual(t, defaultBlock.versionRoot, root)
	// save
	block := &types.Block{}
	block.SetHeader(&types.Header{VersionRoot: root})
	err = db.SetBlock(b(2), block)
	assert.NoError(t, err)
	err = manager.Save(b(2))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(manager.accountCache))
	assert.Equal(t, 0, len(manager.processor.changeLogs))
	manager = NewManager(b(2), db)
	assert.Equal(t, root, manager.GetVersionRoot())
}

func TestNewManager_Finalise_Save2(t *testing.T) {
	db := newDB()
	// load from genesis' parent block
	manager := NewManager(common.Hash{}, db)

	// nothing to finalise
	account, err := manager.GetAccount(common.HexToAddress("0x1"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(manager.accountCache))
	assert.Equal(t, false, account.(*SafeAccount).IsDirty())
	err = manager.Finalise()
	assert.NoError(t, err)
	// save
	err = manager.Save(b(11))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(manager.accountCache))

	// finalise dirty
	account, err = manager.GetAccount(common.HexToAddress("0x2"))
	assert.NoError(t, err)
	account.SetBalance(big.NewInt(2))
	err = account.SetStorageState(k(1), []byte{100})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(manager.accountCache))
	assert.Equal(t, true, account.(*SafeAccount).IsDirty())
	assert.Equal(t, 2, len(manager.processor.changeLogs))
	root := manager.GetVersionRoot()
	assert.Equal(t, emptyTrieRoot, root)
	err = manager.Finalise()
	assert.NoError(t, err)
	root = manager.GetVersionRoot()
	assert.NotEqual(t, emptyTrieRoot, root)
	// save
	block := &types.Block{}
	block.SetHeader(&types.Header{VersionRoot: root})
	err = db.SetBlock(b(12), block)
	assert.NoError(t, err)
	err = manager.Save(b(12))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(manager.accountCache))
	assert.Equal(t, 0, len(manager.processor.changeLogs))
	manager = NewManager(b(12), db)
	assert.Equal(t, root, manager.GetVersionRoot())
}
