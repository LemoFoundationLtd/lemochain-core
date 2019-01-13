package account

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestLogProcessor_GetAccount(t *testing.T) {
	db := newDB()
	manager := NewManager(newestBlock.Hash(), db)
	processor := manager.processor

	// not exist in db
	address := common.HexToAddress("0xaaa")
	account := processor.GetAccount(address)
	assert.Equal(t, address, account.GetAddress())
	assert.NotEmpty(t, manager.accountCache[address])

	// exist in db
	account = processor.GetAccount(defaultAccounts[0].Address)
	assert.Equal(t, uint32(100), account.GetBaseVersion(BalanceLog))

	// change account, safeAccount should change
	safeAccount := manager.GetAccount(defaultAccounts[0].Address)
	account.SetBalance(big.NewInt(2))
	assert.Equal(t, account.GetBalance(), safeAccount.GetBalance())

	// change safeAccount, account should change
	safeAccount.SetBalance(big.NewInt(3))
	assert.Equal(t, safeAccount.GetBalance(), account.GetBalance())
}

func TestLogProcessor_PushEvent_PopEvent(t *testing.T) {
	db := newDB()
	processor := NewManager(newestBlock.Hash(), db).processor

	// push
	processor.PushEvent(&types.Event{Address: common.HexToAddress("0x1"), TxHash: th(1), BlockHeight: 11})
	processor.PushEvent(&types.Event{Address: common.HexToAddress("0x1"), BlockHeight: 22})
	events := processor.GetEvents()
	assert.Equal(t, 2, len(events))
	assert.Equal(t, uint32(22), events[1].BlockHeight)

	// pop
	err := processor.PopEvent()
	assert.NoError(t, err)
	events = processor.GetEvents()
	assert.Equal(t, 1, len(events))
	assert.Equal(t, uint32(11), events[0].BlockHeight)
	err = processor.PopEvent()
	assert.NoError(t, err)
	err = processor.PopEvent()
	assert.Equal(t, ErrNoEvents, err)
}

func TestLogProcessor_PushChangeLog_GetChangeLogs(t *testing.T) {
	db := newDB()
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
	db := newDB()
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
	db := newDB()
	processor := NewManager(newestBlock.Hash(), db).processor
	// prepare account version record
	account := processor.GetAccount(common.HexToAddress("0x1"))
	account.(*Account).SetVersion(types.ChangeLogType(101), 10, 20)

	// no log
	version := processor.GetNextVersion(types.ChangeLogType(101), common.HexToAddress("0x1"))
	assert.Equal(t, uint32(11), version)

	// 1 log
	processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(101),
		Address: common.HexToAddress("0x1"),
		Version: 11,
	})
	version = processor.GetNextVersion(types.ChangeLogType(101), common.HexToAddress("0x1"))
	assert.Equal(t, uint32(12), version)
	version = processor.GetNextVersion(types.ChangeLogType(102), common.HexToAddress("0x1"))
	assert.Equal(t, uint32(1), version)
	version = processor.GetNextVersion(types.ChangeLogType(101), common.HexToAddress("0x2"))
	assert.Equal(t, uint32(1), version)

	// 2 logs
	processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(101),
		Address: common.HexToAddress("0x1"),
		Version: 12,
	})
	version = processor.GetNextVersion(types.ChangeLogType(101), common.HexToAddress("0x1"))
	assert.Equal(t, uint32(13), version)

	// push log for different account
	processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(101),
		Address: common.HexToAddress("0x2"),
		Version: 1,
	})
	version = processor.GetNextVersion(types.ChangeLogType(101), common.HexToAddress("0x1"))
	assert.Equal(t, uint32(13), version)
	version = processor.GetNextVersion(types.ChangeLogType(101), common.HexToAddress("0x2"))
	assert.Equal(t, uint32(2), version)

	// push log for different type
	processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(102),
		Address: common.HexToAddress("0x1"),
		Version: 1,
	})
	version = processor.GetNextVersion(types.ChangeLogType(101), common.HexToAddress("0x1"))
	assert.Equal(t, uint32(13), version)
	version = processor.GetNextVersion(types.ChangeLogType(102), common.HexToAddress("0x1"))
	assert.Equal(t, uint32(2), version)
}

func TestLogProcessor_Clear(t *testing.T) {
	db := newDB()
	processor := NewManager(newestBlock.Hash(), db).processor

	processor.PushEvent(&types.Event{Address: common.HexToAddress("0x1"), TxHash: th(1), BlockHeight: 11})
	processor.PushChangeLog(&types.ChangeLog{
		LogType: types.ChangeLogType(101),
	})
	processor.Snapshot()

	processor.Clear()
	assert.Equal(t, 0, len(processor.GetEvents()))
	assert.Equal(t, 0, len(processor.GetChangeLogs()))
	assert.Equal(t, 0, len(processor.revisions))
	assert.Equal(t, 0, processor.nextRevisionId)

	processor.Clear()
}

// generate change log by safe account
func TestLogProcessor_Snapshot_RevertToSnapshot(t *testing.T) {
	db := newDB()
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
	db := newDB()
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

func TestLogProcessor_MergeChangeLogs(t *testing.T) {
	db := newDB()
	manager := NewManager(newestBlock.Hash(), db)
	processor := manager.processor

	// merge nothing
	processor.MergeChangeLogs(0)

	safeAccount1 := manager.GetAccount(defaultAccounts[0].Address)
	safeAccount2 := manager.GetAccount(common.HexToAddress("0x1"))
	safeAccount3 := manager.GetAccount(common.HexToAddress("0x2"))

	// balance log, balance log, custom log, balance log, balance log
	safeAccount1.SetBalance(big.NewInt(111))  // 0
	processor.PushChangeLog(&types.ChangeLog{ // 1
		LogType: types.ChangeLogType(101),
	})
	safeAccount2.SetBalance(big.NewInt(222)) // 2
	safeAccount3.SetBalance(big.NewInt(444)) // 3
	safeAccount1.SetBalance(big.NewInt(333)) // 4
	safeAccount3.SetBalance(big.NewInt(0))   // 5
	safeAccount1.SetBalance(big.NewInt(100)) // 6
	logs := processor.GetChangeLogs()
	assert.Equal(t, 7, len(logs))
	assert.Equal(t, *big.NewInt(111), processor.GetChangeLogs()[0].NewVal)

	// merge different account's change log
	processor.MergeChangeLogs(5)
	logs = processor.GetChangeLogs()
	assert.Equal(t, 7, len(logs))

	// successfully merge
	processor.MergeChangeLogs(0)
	logs = processor.GetChangeLogs()
	fmt.Println(logs)
	assert.Equal(t, 1, len(logs))
	assert.Equal(t, uint32(0), safeAccount3.GetBaseVersion(BalanceLog))
	// the first change log has been sorted to the last one
	assert.Equal(t, *big.NewInt(222), processor.GetChangeLogs()[0].NewVal)

	// broke snapshot
	safeAccount2.SetBalance(big.NewInt(444))
	processor.Snapshot()
	assert.PanicsWithValue(t, ErrSnapshotIsBroken, func() {
		processor.MergeChangeLogs(0)
	})
}

// func createChangeLog(processor *testProcessor, accountVersion uint32, logType ChangeLogType, logVersion uint32) *ChangeLog {
// 	account := processor.createAccount(logType, accountVersion)
// 	return &ChangeLog{LogType: logType, Address: account.GetAddress(), Version: logVersion}
// }
//
// func removeAddress(log *ChangeLog) *ChangeLog {
// 	log.Address = common.Address{}
// 	return log
// }
//
// func TestChangeLog_Undo(t *testing.T) {
// 	processor := &testProcessor{}
// 	registerCustomType(10002)
//
// 	tests := []struct {
// 		input      *ChangeLog
// 		undoErr    error
// 		afterCheck func(AccountAccessor)
// 	}{
// 		// 0 custom type
// 		{
// 			input:   createChangeLog(processor, 0, ChangeLogType(0), 1),
// 			undoErr: ErrUnknownChangeLogType,
// 		},
// 		// 1 lower version
// 		{
// 			input:   createChangeLog(processor, 2, ChangeLogType(10002), 1),
// 			undoErr: ErrWrongChangeLogVersion,
// 		},
// 		// 2 same version
// 		{
// 			input: createChangeLog(processor, 1, ChangeLogType(10002), 1),
// 			afterCheck: func(accessor AccountAccessor) {
// 				assert.Equal(t, uint32(0), accessor.(*testAccount).GetVersion(ChangeLogType(10002)))
// 				assert.Equal(t, uint32(0), accessor.(*testAccount).GetVersion(ChangeLogType(10002)))
// 			},
// 		},
// 		// 3 higher version
// 		{
// 			input:   createChangeLog(processor, 2, ChangeLogType(10002), 1),
// 			undoErr: ErrWrongChangeLogVersion,
// 		},
// 		// 4 no account
// 		{
// 			input:   removeAddress(createChangeLog(processor, 2, ChangeLogType(10002), 1)),
// 			undoErr: ErrWrongChangeLogVersion,
// 		},
// 	}
//
// 	for i, test := range tests {
// 		err := test.input.Undo(processor)
// 		assert.Equal(t, test.undoErr, err, "index=%d %s", i, test.input)
// 		if test.undoErr == nil && test.afterCheck != nil {
// 			a := processor.GetAccount(test.input.Address)
// 			test.afterCheck(a)
// 		}
// 	}
// }
//
// func TestChangeLog_Redo(t *testing.T) {
// 	processor := &testProcessor{}
// 	registerCustomType(10003)
//
// 	tests := []struct {
// 		input      *ChangeLog
// 		redoErr    error
// 		afterCheck func(AccountAccessor)
// 	}{
// 		// 0 custom type
// 		{
// 			input:   &ChangeLog{LogType: ChangeLogType(0), Address: common.Address{}, Version: 1},
// 			redoErr: ErrUnknownChangeLogType,
// 		},
// 		// 1 lower version
// 		{
// 			input:   createChangeLog(processor, 2, ChangeLogType(10003), 1),
// 			redoErr: ErrAlreadyRedo,
// 		},
// 		// 2 same version
// 		{
// 			input:   createChangeLog(processor, 1, ChangeLogType(10003), 1),
// 			redoErr: ErrAlreadyRedo,
// 		},
// 		// 3 correct version
// 		{
// 			input: createChangeLog(processor, 0, ChangeLogType(10003), 1),
// 			afterCheck: func(accessor AccountAccessor) {
// 				assert.Equal(t, uint32(1), accessor.(*testAccount).GetVersion(ChangeLogType(10003)))
// 			},
// 		},
// 		// 3 higher version
// 		{
// 			input:   createChangeLog(processor, 0, ChangeLogType(10003), 2),
// 			redoErr: ErrWrongChangeLogVersion,
// 		},
// 		// 4 no account
// 		{
// 			input:   removeAddress(createChangeLog(processor, 1, ChangeLogType(10003), 2)),
// 			redoErr: ErrWrongChangeLogVersion,
// 		},
// 	}
//
// 	for i, test := range tests {
// 		err := test.input.Redo(processor)
// 		assert.Equal(t, test.redoErr, err, "index=%d %s", i, test.input)
// 		if test.redoErr == nil && test.afterCheck != nil {
// 			a := processor.GetAccount(test.input.Address)
// 			test.afterCheck(a)
// 		}
// 	}
// }
