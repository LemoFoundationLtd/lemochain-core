package account

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/stretchr/testify/assert"
	"math/big"
	"strconv"
	"testing"
)

func TestLogProcessor_GetAccount(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	manager := NewManager(newestBlock.Hash(), db)
	processor := manager.processor

	// not exist in db
	address := common.HexToAddress("0xaaa")
	account := processor.GetAccount(address)
	assert.Equal(t, address, account.GetAddress())
	assert.NotEmpty(t, manager.accountCache[address])

	// exist in db
	account = processor.GetAccount(defaultAccounts[0].Address)
	assert.Equal(t, uint32(100), account.GetVersion(BalanceLog))

	// change account, safeAccount should change
	safeAccount := manager.GetAccount(defaultAccounts[0].Address)
	account.SetBalance(big.NewInt(2))
	assert.Equal(t, account.GetBalance(), safeAccount.GetBalance())

	// change safeAccount, account should change
	safeAccount.SetBalance(big.NewInt(3))
	assert.Equal(t, safeAccount.GetBalance(), account.GetBalance())
}

func TestLogProcessor_PushEvent_PopEvent(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	manager := NewManager(newestBlock.Hash(), db)
	// push
	manager.AddEvent(&types.Event{Address: common.HexToAddress("0x1"), TxHash: th(1)})
	manager.AddEvent(&types.Event{Address: common.HexToAddress("0x1")})
	events := manager.GetEvents()
	assert.Equal(t, 2, len(events))
}

func TestLogProcessor_PushChangeLog_GetChangeLogs(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	processor := NewManager(newestBlock.Hash(), db).processor
	assert.Equal(t, 0, len(processor.GetChangeLogs()))

	processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(101),
	})
	assert.Equal(t, 1, len(processor.GetChangeLogs()))

	processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(102),
	})
	assert.Equal(t, 2, len(processor.GetChangeLogs()))
}

func TestLogProcessor_GetLogsByAddress(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	processor := NewManager(newestBlock.Hash(), db).processor

	// no log
	logs := processor.GetLogsByAddress(common.HexToAddress("0x1"))
	assert.Empty(t, logs)

	// 3 logs belongs to 2 account
	processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(101),
		Address: common.HexToAddress("0x1"),
	})
	processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(102),
		Address: common.HexToAddress("0x2"),
	})
	processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(103),
		Address: common.HexToAddress("0x1"),
	})
	logs = processor.GetLogsByAddress(common.HexToAddress("0x1"))
	assert.Equal(t, 2, len(logs))
	assert.Equal(t, types.ChangeLogType(103), logs[1].LogType)
	logs = processor.GetLogsByAddress(common.HexToAddress("0x2"))
	assert.Equal(t, 1, len(logs))
	logs = processor.GetLogsByAddress(common.HexToAddress("0x3"))
	assert.Empty(t, logs)
}

func TestLogProcessor_GetNextVersion(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	processor := NewManager(newestBlock.Hash(), db).processor
	// prepare account version record
	account := processor.GetAccount(common.HexToAddress("0x1"))
	account.(*Account).SetVersion(types.ChangeLogType(101), 10, 20)

	// no log
	version := processor.GetAccount(common.HexToAddress("0x1")).GetNextVersion(types.ChangeLogType(101))
	assert.Equal(t, uint32(11), version)

	// 1 log
	processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(101),
		Address: common.HexToAddress("0x1"),
		Version: 11,
	})
	version = processor.GetAccount(common.HexToAddress("0x1")).GetNextVersion(types.ChangeLogType(101))
	assert.Equal(t, uint32(12), version)
	version = processor.GetAccount(common.HexToAddress("0x2")).GetNextVersion(types.ChangeLogType(102))
	assert.Equal(t, uint32(1), version)
	version = processor.GetAccount(common.HexToAddress("0x2")).GetNextVersion(types.ChangeLogType(101))
	assert.Equal(t, uint32(1), version)

	// 2 logs
	processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(101),
		Address: common.HexToAddress("0x1"),
		Version: 12,
	})
	version = processor.GetAccount(common.HexToAddress("0x1")).GetNextVersion(types.ChangeLogType(101))
	assert.Equal(t, uint32(13), version)

	// push log for different account
	processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(101),
		Address: common.HexToAddress("0x2"),
		Version: 1,
	})
	version = processor.GetAccount(common.HexToAddress("0x1")).GetNextVersion(types.ChangeLogType(101))
	assert.Equal(t, uint32(14), version)
	version = processor.GetAccount(common.HexToAddress("0x2")).GetNextVersion(types.ChangeLogType(101))
	assert.Equal(t, uint32(2), version)

	// push log for different type
	processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(102),
		Address: common.HexToAddress("0x1"),
		Version: 1,
	})
	version = processor.GetAccount(common.HexToAddress("0x1")).GetNextVersion(types.ChangeLogType(101))
	assert.Equal(t, uint32(15), version)
	version = processor.GetAccount(common.HexToAddress("0x1")).GetNextVersion(types.ChangeLogType(102))
	assert.Equal(t, uint32(1), version)
}

func TestLogProcessor_Clear(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	processor := NewManager(newestBlock.Hash(), db).processor

	processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(101),
	})
	processor.Snapshot()

	processor.Clear()
	assert.Equal(t, 0, len(processor.GetChangeLogs()))
	assert.Equal(t, 0, len(processor.revisions))
	assert.Equal(t, 0, processor.nextRevisionId)

	processor.Clear()
}

// generate change log by safe account
func TestLogProcessor_Snapshot_RevertToSnapshot(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	manager := NewManager(newestBlock.Hash(), db)
	processor := manager.processor

	// snapshot when empty
	assert.Equal(t, 0, len(processor.revisions))
	assert.Equal(t, 0, processor.nextRevisionId)
	newId := processor.Snapshot()
	assert.Equal(t, 0, newId)
	assert.Equal(t, 1, len(processor.revisions))
	assert.Equal(t, 0, processor.revisions[0].id)
	assert.Equal(t, 0, processor.revisions[0].journalIndex)
	assert.Equal(t, 1, processor.nextRevisionId)
	// revert to current snapshot
	processor.RevertToSnapshot(newId)
	assert.Equal(t, 0, len(processor.revisions))
	assert.Equal(t, 1, processor.nextRevisionId)

	// revert not exist version
	assert.PanicsWithValue(t, ErrRevisionNotExist, func() {
		processor.RevertToSnapshot(100)
	})

	// create a new snapshot then revert
	safeAccount := manager.GetAccount(defaultAccounts[0].Address)
	newId = processor.Snapshot()
	assert.Equal(t, 1, newId)
	safeAccount.SetBalance(big.NewInt(999))
	processor.RevertToSnapshot(newId)
	assert.Equal(t, defaultAccounts[0].Balance, safeAccount.GetBalance())

	// create two new snapshot then revert one
	safeAccount = manager.GetAccount(common.HexToAddress("0x1"))
	newId = processor.Snapshot()
	safeAccount.SetBalance(big.NewInt(999))
	newId = processor.Snapshot()
	safeAccount.SetBalance(big.NewInt(1))
	assert.Equal(t, 2, len(processor.revisions))
	processor.RevertToSnapshot(newId)
	assert.Equal(t, 1, len(processor.revisions))
	assert.Equal(t, big.NewInt(999), safeAccount.GetBalance())

	// snapshot twice
	safeAccount = manager.GetAccount(common.HexToAddress("0x2"))
	safeAccount.SetBalance(big.NewInt(999))
	newId = processor.Snapshot()
	newId2 := processor.Snapshot()
	safeAccount.SetBalance(big.NewInt(1))
	processor.RevertToSnapshot(newId2)
	processor.RevertToSnapshot(newId)
	assert.Equal(t, 1, len(processor.revisions))
	assert.Equal(t, big.NewInt(999), safeAccount.GetBalance())
}

// test invalid change logs
func TestLogProcessor_Snapshot_RevertToSnapshot2(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	manager := NewManager(newestBlock.Hash(), db)
	processor := manager.processor
	// prepare account version record
	safeAccount := manager.GetAccount(common.HexToAddress("0x1")).(*SafeAccount)
	safeAccount.rawAccount.SetVersion(BalanceLog, 10, 20)

	// version is little than in account
	newId := processor.Snapshot()
	safeAccount.SetBalance(big.NewInt(999))
	assert.Equal(t, 1, len(processor.GetChangeLogs()))
	processor.changeLogs[0].Version = 9
	assert.PanicsWithValue(t, types.ErrWrongChangeLogVersion, func() {
		processor.RevertToSnapshot(newId)
	})

	// version is continuous
	processor.Clear()
	newId = processor.Snapshot()
	log.Errorf("safe account version: " + strconv.Itoa(int(safeAccount.GetVersion(BalanceLog))))
	log.Errorf("safe account next version: " + strconv.Itoa(int(safeAccount.GetNextVersion(BalanceLog))))
	safeAccount.SetBalance(big.NewInt(200))
	safeAccount.SetBalance(big.NewInt(201))
	assert.Equal(t, 2, len(processor.GetChangeLogs()))
	processor.RevertToSnapshot(newId)

	// version is not continuous
	newId = processor.Snapshot()
	safeAccount.SetBalance(big.NewInt(300))
	safeAccount.SetBalance(big.NewInt(301))
	processor.changeLogs[1].Version++
	assert.PanicsWithValue(t, types.ErrWrongChangeLogVersion, func() {
		processor.RevertToSnapshot(newId)
	})
}

func TestLogProcessor_MergeChangeLogs1(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	manager := NewManager(newestBlock.Hash(), db)
	processor := manager.processor

	// merge nothing
	processor.MergeChangeLogs(0)

	safeAccount1 := manager.GetAccount(defaultAccounts[0].Address)
	safeAccount2 := manager.GetAccount(common.HexToAddress("0x1"))

	// balance log, balance log, custom log, balance log, balance log
	safeAccount1.SetBalance(big.NewInt(111)) // 0
	safeAccount2.SetBalance(big.NewInt(222)) // 1
	logs := processor.GetChangeLogs()
	assert.Equal(t, 2, len(logs))
	assert.Equal(t, *big.NewInt(111), logs[0].NewVal)

	// merge different account's change log
	processor.MergeChangeLogs(0)
	logs = processor.GetChangeLogs()
	assert.Equal(t, 2, len(logs))

	// merge unchanged
	safeAccount2.SetBalance(big.NewInt(333)) // 2
	safeAccount2.SetBalance(big.NewInt(444)) // 3
	safeAccount2.SetBalance(big.NewInt(333)) // 4
	processor.MergeChangeLogs(2)
	logs = processor.GetChangeLogs()
	assert.Equal(t, 3, len(logs))
}

func TestLogProcessor_MergeChangeLogs2(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	manager := NewManager(newestBlock.Hash(), db)
	processor := manager.processor

	safeAccount1 := manager.GetAccount(defaultAccounts[0].Address)
	safeAccount2 := manager.GetAccount(common.HexToAddress("0x1"))
	safeAccount3 := manager.GetAccount(common.HexToAddress("0x2"))

	// balance log, balance log, custom log, balance log, balance log
	safeAccount1.SetBalance(big.NewInt(111))           // 0
	safeAccount1.SetVoteFor(safeAccount2.GetAddress()) // 1
	safeAccount2.SetBalance(big.NewInt(222))           // 2
	safeAccount3.SetBalance(big.NewInt(444))           // 3
	safeAccount1.SetBalance(big.NewInt(333))           // 4
	safeAccount3.SetBalance(big.NewInt(0))             // 5
	safeAccount1.SetBalance(big.NewInt(111))           // 6
	logs := processor.GetChangeLogs()
	assert.Equal(t, 7, len(logs))
	assert.Equal(t, *big.NewInt(111), logs[0].NewVal)

	// successfully merge
	// the 6th overwrite 4th and 0th
	// the 5th overwrite 3th. but the 5th is not valuable, so we remove 5th too
	// then sort logs by address. the result sequence is: 2, 0, 1
	processor.MergeChangeLogs(0)
	logs = processor.GetChangeLogs()
	assert.Equal(t, 3, len(logs))
	assert.Equal(t, uint32(0), safeAccount3.GetVersion(BalanceLog))
	// 2
	assert.Equal(t, BalanceLog, logs[0].LogType)
	assert.Equal(t, safeAccount2.GetAddress(), logs[0].Address)
	assert.Equal(t, *big.NewInt(0), logs[0].OldVal)
	assert.Equal(t, *big.NewInt(222), logs[0].NewVal)
	assert.Equal(t, safeAccount2.GetVersion(BalanceLog)+1, logs[0].Version)
	// 0
	assert.Equal(t, BalanceLog, logs[1].LogType)
	assert.Equal(t, safeAccount1.GetAddress(), logs[1].Address)
	assert.Equal(t, *defaultAccounts[0].Balance, logs[1].OldVal)
	assert.Equal(t, *big.NewInt(111), logs[1].NewVal)
	assert.Equal(t, safeAccount1.GetVersion(BalanceLog)+1, logs[1].Version)
	// 1
	assert.Equal(t, VoteForLog, logs[2].LogType)
	assert.Equal(t, safeAccount1.GetAddress(), logs[2].Address)
	assert.Equal(t, common.Address{}, logs[2].OldVal)
	assert.Equal(t, safeAccount2.GetAddress(), logs[2].NewVal)
	assert.Equal(t, uint32(1), logs[2].Version)

	// broke snapshot
	safeAccount2.SetBalance(big.NewInt(444))
	processor.Snapshot()
	assert.PanicsWithValue(t, ErrSnapshotIsBroken, func() {
		processor.MergeChangeLogs(0)
	})
}

func TestLogProcessor_MergeChangeLogs3(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	manager := NewManager(newestBlock.Hash(), db)
	processor := manager.processor

	safeAccount1 := manager.GetAccount(defaultAccounts[0].Address)
	safeAccount2 := manager.GetAccount(common.HexToAddress("0x1"))
	safeAccount3 := manager.GetAccount(common.HexToAddress("0x2"))

	//processor.PushChangeLog(NewStor)

	// balance log, balance log, custom log, balance log, balance log
	safeAccount1.SetBalance(big.NewInt(111))           // 0
	safeAccount1.SetVoteFor(safeAccount2.GetAddress()) // 1
	safeAccount2.SetBalance(big.NewInt(222))           // 2
	safeAccount3.SetBalance(big.NewInt(444))           // 3
	safeAccount1.SetBalance(big.NewInt(333))           // 4
	safeAccount3.SetBalance(big.NewInt(0))             // 5
	safeAccount1.SetBalance(big.NewInt(111))           // 6
	logs := processor.GetChangeLogs()
	assert.Equal(t, 7, len(logs))
	assert.Equal(t, *big.NewInt(111), logs[0].NewVal)

	// successfully merge
	// the 6th overwrite 4th and 0th
	// the 5th overwrite 3th. but the 5th is not valuable, so we remove 5th too
	// then sort logs by address. the result sequence is: 2, 0, 1
	processor.MergeChangeLogs(0)
	logs = processor.GetChangeLogs()
	assert.Equal(t, 3, len(logs))
	assert.Equal(t, uint32(0), safeAccount3.GetVersion(BalanceLog))
	// 2
	assert.Equal(t, BalanceLog, logs[0].LogType)
	assert.Equal(t, safeAccount2.GetAddress(), logs[0].Address)
	assert.Equal(t, *big.NewInt(0), logs[0].OldVal)
	assert.Equal(t, *big.NewInt(222), logs[0].NewVal)
	assert.Equal(t, safeAccount2.GetVersion(BalanceLog)+1, logs[0].Version)
	// 0
	assert.Equal(t, BalanceLog, logs[1].LogType)
	assert.Equal(t, safeAccount1.GetAddress(), logs[1].Address)
	assert.Equal(t, *defaultAccounts[0].Balance, logs[1].OldVal)
	assert.Equal(t, *big.NewInt(111), logs[1].NewVal)
	assert.Equal(t, safeAccount1.GetVersion(BalanceLog)+1, logs[1].Version)
	// 1
	assert.Equal(t, VoteForLog, logs[2].LogType)
	assert.Equal(t, safeAccount1.GetAddress(), logs[2].Address)
	assert.Equal(t, common.Address{}, logs[2].OldVal)
	assert.Equal(t, safeAccount2.GetAddress(), logs[2].NewVal)
	assert.Equal(t, uint32(1), logs[2].Version)

	// broke snapshot
	safeAccount2.SetBalance(big.NewInt(444))
	processor.Snapshot()
	assert.PanicsWithValue(t, ErrSnapshotIsBroken, func() {
		processor.MergeChangeLogs(0)
	})
}
