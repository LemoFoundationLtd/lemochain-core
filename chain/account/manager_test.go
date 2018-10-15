package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestNewManager_Interface(t *testing.T) {
	var _ vm.AccountManager = (*Manager)(nil)
	var _ types.ChangeLogProcessor = (*logProcessor)(nil)
}

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
	manager := NewManager(newestBlock.Hash(), db)
	account := manager.GetAccount(defaultAccounts[0].Address)
	assert.Equal(t, uint32(100), account.GetVersion(BalanceLog))
	assert.Equal(t, false, account.IsEmpty())
	// not exist in db
	account = manager.GetAccount(common.HexToAddress("0xaaa"))
	assert.Equal(t, common.HexToAddress("0xaaa"), account.GetAddress())
	assert.Equal(t, true, account.IsEmpty())

	// load from older block
	manager = NewManager(defaultBlockInfos[0].hash, db)
	account = manager.GetAccount(defaultAccounts[0].Address)
	assert.Equal(t, uint32(100), account.GetVersion(BalanceLog))
	assert.Equal(t, false, account.IsEmpty())

	// load from genesis' parent block
	manager = NewManager(common.Hash{}, db)
	account = manager.GetAccount(common.Address{})
	assert.Equal(t, common.Address{}, account.GetAddress())
	assert.Equal(t, account, manager.accountCache[common.Address{}])

	// get twice
	account2 := manager.GetAccount(common.Address{})
	account.SetBalance(big.NewInt(200))
	assert.Equal(t, big.NewInt(200), account.GetBalance())
	account2.SetBalance(big.NewInt(100))
	assert.Equal(t, big.NewInt(100), account2.GetBalance())
}

func TestNewManager_GetCanonicalAccount(t *testing.T) {
	db := newDB()

	// exist in db
	manager := NewManager(newestBlock.Hash(), db)
	account := manager.GetCanonicalAccount(defaultAccounts[0].Address)
	assert.Equal(t, uint32(100), account.GetVersion(BalanceLog))
	// not exist in db
	account = manager.GetCanonicalAccount(common.HexToAddress("0xaaa"))
	assert.Equal(t, common.HexToAddress("0xaaa"), account.GetAddress())

	// load from genesis' parent block
	manager = NewManager(common.Hash{}, db)
	account = manager.GetCanonicalAccount(common.Address{})
	assert.Equal(t, common.Address{}, account.GetAddress())
	assert.Empty(t, manager.accountCache[common.Address{}])
}

func TestChangeLogProcessor_GetAccount(t *testing.T) {
	manager := NewManager(newestBlock.Hash(), newDB())

	// not exist in db
	address := common.HexToAddress("0xaaa")
	rawAccount := manager.processor.GetAccount(address)
	assert.Equal(t, address, rawAccount.GetAddress())
	assert.NotEmpty(t, manager.accountCache[address])

	// exist in db
	rawAccount = manager.processor.GetAccount(defaultAccounts[0].Address)
	assert.Equal(t, uint32(100), rawAccount.GetVersion(BalanceLog))

	// change rawAccount, account should change
	account := manager.GetAccount(defaultAccounts[0].Address)
	rawAccount.SetBalance(big.NewInt(2))
	assert.Equal(t, rawAccount.GetBalance(), account.GetBalance())

	// change account, rawAccount should change
	account.SetBalance(big.NewInt(3))
	assert.Equal(t, account.GetBalance(), rawAccount.GetBalance())
}

func TestChangeLogProcessor_PushEvent_PopEvent(t *testing.T) {
	manager := NewManager(newestBlock.Hash(), newDB())

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
	manager := NewManager(newestBlock.Hash(), newDB())

	manager.processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(101),
	})
	account := manager.GetAccount(defaultAccounts[0].Address)

	// create a new change log
	account.SetBalance(big.NewInt(999))
	logs := manager.GetChangeLogs()
	assert.Equal(t, 2, len(logs))
	assert.Equal(t, types.ChangeLogType(101), logs[0].LogType)
	assert.Equal(t, *big.NewInt(999), logs[1].NewVal.(big.Int))
}

func TestNewManager_Snapshot_RevertToSnapshot(t *testing.T) {
	manager := NewManager(newestBlock.Hash(), newDB())

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
	account := manager.GetAccount(defaultAccounts[0].Address)
	newId = manager.Snapshot()
	assert.Equal(t, 1, newId)
	account.SetBalance(big.NewInt(999))
	manager.RevertToSnapshot(newId)
	assert.Equal(t, defaultAccounts[0].Balance, account.GetBalance())

	// create two new snapshot than revert one
	account = manager.GetAccount(common.HexToAddress("0x1"))
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
	manager := NewManager(newestBlock.Hash(), newDB())

	event1 := &types.Event{Address: common.HexToAddress("0x1"), TxHash: th(1), BlockHeight: 11}
	event2 := &types.Event{Address: common.HexToAddress("0x1"), TxHash: th(1), BlockHeight: 22}
	manager.AddEvent(event1)
	assert.Equal(t, uint(0), event1.Index)
	manager.AddEvent(event2)
	assert.Equal(t, uint(1), event2.Index)
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

	// not empty version trie
	manager = NewManager(newestBlock.Hash(), db)
	root = manager.GetVersionRoot()
	assert.Equal(t, newestBlock.VersionRoot(), root)
	// read version from trie
	key := append(defaultAccounts[0].Address.Bytes(), big.NewInt(int64(BalanceLog)).Bytes()...)
	value, err := ReadTrie(db, root, key)
	assert.NoError(t, err)
	assert.Equal(t, uint32(100), manager.GetAccount(defaultAccounts[0].Address).GetVersion(BalanceLog))
	assert.Equal(t, big.NewInt(100), new(big.Int).SetBytes(value))
}

func TestNewManager_Reset(t *testing.T) {
	manager := NewManager(common.Hash{}, newDB())

	account := manager.GetAccount(common.HexToAddress("0x1"))
	account.SetBalance(big.NewInt(2))
	manager.GetVersionRoot()
	assert.NotEmpty(t, manager.accountCache)
	assert.NotEmpty(t, manager.processor.changeLogs)
	assert.NotEmpty(t, manager.versionTrie)
	manager.Reset(newestBlock.Hash())
	assert.Equal(t, newestBlock.Hash(), manager.baseBlockHash)
	assert.Empty(t, manager.accountCache)
	assert.Empty(t, manager.processor.changeLogs)
	assert.Empty(t, manager.processor.events)
	assert.Empty(t, manager.versionTrie)
}

// saving from the newest block
func TestNewManager_Finalise_Save(t *testing.T) {
	db := newDB()
	manager := NewManager(newestBlock.Hash(), db)

	// nothing to finalise
	account := manager.GetAccount(common.HexToAddress("0x1"))
	assert.Equal(t, 1, len(manager.accountCache))
	assert.Equal(t, false, account.(*SafeAccount).IsDirty())
	err := manager.Finalise()
	assert.NoError(t, err)
	// save
	err = manager.Save(b(1))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(manager.accountCache))

	// finalise dirty
	account = manager.GetAccount(defaultAccounts[0].Address)
	account.SetBalance(big.NewInt(2))
	err = account.SetStorageState(k(1), []byte{100})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(manager.accountCache))
	assert.Equal(t, true, account.(*SafeAccount).IsDirty())
	assert.Equal(t, 2, len(manager.processor.changeLogs))
	root := manager.GetVersionRoot()
	assert.Equal(t, newestBlock.VersionRoot(), root)
	err = manager.Finalise()
	assert.NoError(t, err)
	root = manager.GetVersionRoot()
	assert.NotEqual(t, newestBlock.VersionRoot(), root)
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

// saving genesis block and first block
func TestNewManager_Finalise_Save2(t *testing.T) {
	db := newDB()
	// load from genesis' parent block
	manager := NewManager(common.Hash{}, db)

	// nothing to finalise
	account := manager.GetAccount(common.HexToAddress("0x1"))
	assert.Equal(t, 1, len(manager.accountCache))
	assert.Equal(t, false, account.(*SafeAccount).IsDirty())
	err := manager.Finalise()
	assert.NoError(t, err)
	// save
	err = manager.Save(b(11))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(manager.accountCache))

	// finalise dirty
	account = manager.GetAccount(common.HexToAddress("0x2"))
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

func TestManager_Save_Reset(t *testing.T) {
	db := newDB()
	// load from genesis' parent block
	manager := NewManager(common.Hash{}, db)

	// save balance to 1 in block1
	account := manager.GetAccount(common.HexToAddress("0x1"))
	account.SetBalance(big.NewInt(1))
	assert.Equal(t, uint32(1), account.GetVersion(BalanceLog))
	err := manager.Finalise()
	assert.NoError(t, err)
	block := &types.Block{}
	block.SetHeader(&types.Header{Height: 0, VersionRoot: manager.GetVersionRoot()})
	err = db.SetBlock(block.Hash(), block)
	assert.NoError(t, err)
	err = manager.Save(block.Hash())
	assert.NoError(t, err)

	// save balance to 2 in block2
	block1Hash := block.Hash()
	manager.Reset(block1Hash)
	account = manager.GetAccount(common.HexToAddress("0x1"))
	account.SetBalance(big.NewInt(2))
	assert.Equal(t, uint32(2), account.GetVersion(BalanceLog))
	err = manager.Finalise()
	assert.NoError(t, err)
	block = &types.Block{}
	block.SetHeader(&types.Header{Height: 1, ParentHash: block1Hash, VersionRoot: manager.GetVersionRoot()})
	err = db.SetBlock(block.Hash(), block)
	assert.NoError(t, err)
	err = manager.Save(block.Hash())
	assert.NoError(t, err)

	// load state from block1
	manager.Reset(block1Hash)
	account = manager.GetAccount(common.HexToAddress("0x1"))
	assert.Equal(t, big.NewInt(1), account.GetBalance())
	assert.Equal(t, uint32(1), account.GetVersion(BalanceLog))
}

func TestManager_MergeChangeLogs(t *testing.T) {
	manager := NewManager(common.Hash{}, newDB())

	// merge nothing
	manager.MergeChangeLogs(0)

	account1 := manager.GetAccount(defaultAccounts[0].Address)
	account2 := manager.GetAccount(common.HexToAddress("0x1"))
	account3 := manager.GetAccount(common.HexToAddress("0x2"))

	// balance log, custom log, balance log, balance log
	account1.SetBalance(big.NewInt(111))
	manager.processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(101),
	})
	account2.SetBalance(big.NewInt(222))
	account3.SetBalance(big.NewInt(444))
	account1.SetBalance(big.NewInt(333))
	account3.SetBalance(big.NewInt(0))
	logs := manager.GetChangeLogs()
	assert.Equal(t, 6, len(logs))
	assert.Equal(t, *big.NewInt(111), manager.GetChangeLogs()[0].NewVal)
	assert.Equal(t, uint32(2), account1.GetVersion(BalanceLog))

	// merge different account's change log
	manager.MergeChangeLogs(4)
	logs = manager.GetChangeLogs()
	assert.Equal(t, 6, len(logs))

	// successfully merge
	manager.MergeChangeLogs(0)
	logs = manager.GetChangeLogs()
	assert.Equal(t, 2, len(logs))
	assert.Equal(t, uint32(1), account1.GetVersion(BalanceLog))
	assert.Equal(t, uint32(0), account3.GetVersion(BalanceLog))
	// the first change log has been sorted to the last one
	assert.Equal(t, *big.NewInt(333), manager.GetChangeLogs()[1].NewVal)

	// broke snapshot
	account1.SetBalance(big.NewInt(444))
	manager.Snapshot()
	assert.PanicsWithValue(t, ErrSnapshotIsBroken, func() {
		manager.MergeChangeLogs(0)
	})
}
