package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

type testAccount struct {
	Account
}

type testProcessor struct {
	Accounts map[common.Address]*testAccount
	Events   []*types.Event
}

func (p *testProcessor) GetAccount(address common.Address) types.AccountAccessor {
	account, ok := p.Accounts[address]
	if !ok {
		account = &testAccount{
			Account: Account{
				data: &types.AccountData{
					Address: address,
				},
			},
		}
		p.Accounts[address] = account
	}
	return account
}

func (p *testProcessor) PushEvent(event *types.Event) {
	if p.Events == nil {
		p.Events = make([]*types.Event, 0)
	}
	p.Events = append(p.Events, event)
}

func (p *testProcessor) PopEvent() error {
	p.Events = p.Events[:len(p.Events)-1]
	return nil
}

func (p *testProcessor) createAccount(logType types.ChangeLogType, version uint32) *testAccount {
	if p.Accounts == nil {
		p.Accounts = make(map[common.Address]*testAccount)
	}
	index := len(p.Accounts) + 1
	address := common.BigToAddress(big.NewInt(int64(index)))
	account := &testAccount{
		Account: *NewAccount(nil, address, &types.AccountData{
			Address:       address,
			Balance:       big.NewInt(100),
			NewestRecords: map[types.ChangeLogType]types.VersionRecord{logType: {Version: version, Height: 10}},
		}, 10),
	}
	account.cachedStorage = map[common.Hash][]byte{
		common.HexToHash("0xaaa"): {45, 67},
	}
	p.Accounts[address] = account
	return account
}

type testCustomTypeConfig struct {
	input     *types.ChangeLog
	str       string
	hash      string
	rlp       string
	decoded   string
	decodeErr error
}

func NewStorageLogWithoutError(t *testing.T, account types.AccountAccessor, key common.Hash, newVal []byte) *types.ChangeLog {
	log, err := NewStorageLog(account, key, newVal)
	assert.NoError(t, err)
	return log
}

func getCustomTypeData(t *testing.T) []testCustomTypeConfig {
	processor := &testProcessor{}
	tests := make([]testCustomTypeConfig, 0)

	// 0 BalanceLog
	tests = append(tests, testCustomTypeConfig{
		input:   NewBalanceLog(processor.createAccount(BalanceLog, 0), big.NewInt(0)),
		str:     "BalanceLog{Account: 0x0000000000000000000000000000000000000001, Version: 1, OldVal: 100, NewVal: 0}",
		hash:    "0x9532d32f3b2253bb6fb438cb8ac394882b15a1a2883e6619398d50f059ea2692",
		rlp:     "0xd9019400000000000000000000000000000000000000010180c0",
		decoded: "BalanceLog{Account: 0x0000000000000000000000000000000000000001, Version: 1, NewVal: 0}",
	})

	// 1 StorageLog
	tests = append(tests, testCustomTypeConfig{
		input:   NewStorageLogWithoutError(t, processor.createAccount(StorageLog, 0), common.HexToHash("0xaaa"), []byte{67, 89}),
		str:     "StorageLog{Account: 0x0000000000000000000000000000000000000002, Version: 1, OldVal: [45 67], NewVal: [67 89], Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 10 170]}",
		hash:    "0x47c7805a8377ab0140d96e092a045a03e8e41c6d0b3bb00307d6123d62c59dba",
		rlp:     "0xf83b0294000000000000000000000000000000000000000201824359a00000000000000000000000000000000000000000000000000000000000000aaa",
		decoded: "StorageLog{Account: 0x0000000000000000000000000000000000000002, Version: 1, NewVal: [67 89], Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 10 170]}",
	})

	// 2 CodeLog
	tests = append(tests, testCustomTypeConfig{
		input: NewCodeLog(processor.createAccount(CodeLog, 0), []byte{0x12, 0x34}),
		str:   "CodeLog{Account: 0x0000000000000000000000000000000000000003, Version: 1, NewVal: [18 52]}",
		hash:  "0xcec04ee7ea02f669bfee54633269673d706b59ab821127b5c5491d4dc1c4076a",
		rlp:   "0xdb0394000000000000000000000000000000000000000301821234c0",
	})

	// 3 AddEventLog
	newEvent := &types.Event{
		Address: common.HexToAddress("0xaaa"),
		Topics:  []common.Hash{common.HexToHash("bbb"), common.HexToHash("ccc")},
		Data:    []byte{0x80, 0x0},
	}
	tests = append(tests, testCustomTypeConfig{
		input: NewAddEventLog(processor.createAccount(AddEventLog, 0), newEvent),
		str:   "AddEventLog{Account: 0x0000000000000000000000000000000000000004, Version: 1, NewVal: event: 0000000000000000000000000000000000000aaa [0000000000000000000000000000000000000000000000000000000000000bbb 0000000000000000000000000000000000000000000000000000000000000ccc] 8000 0000000000000000000000000000000000000000000000000000000000000000 0 0000000000000000000000000000000000000000000000000000000000000000 0}",
		hash:  "0x89761d5f9ee931d7e514de58289cce64c8f89491305668bdabd3cf3be815282b",
		rlp:   "0xf8760494000000000000000000000000000000000000000401f85c940000000000000000000000000000000000000aaaf842a00000000000000000000000000000000000000000000000000000000000000bbba00000000000000000000000000000000000000000000000000000000000000ccc828000c0",
	})

	// 4 SuicideLog
	tests = append(tests, testCustomTypeConfig{
		input:   NewSuicideLog(processor.createAccount(SuicideLog, 0)),
		str:     "SuicideLog{Account: 0x0000000000000000000000000000000000000005, Version: 1, OldVal: {Address: Lemo222222222222222222222, Balance: 100}}",
		hash:    "0x6204ac1a4be1e52c77942259e094c499b263ad7176ae9060d5bae2b856c9743a",
		rlp:     "0xd90594000000000000000000000000000000000000000501c0c0",
		decoded: "SuicideLog{Account: 0x0000000000000000000000000000000000000005, Version: 1}",
	})

	return tests
}

func TestChangeLog_String(t *testing.T) {
	tests := getCustomTypeData(t)
	for i, test := range tests {
		assert.Equal(t, test.str, test.input.String(), "index=%d %s", i, test.input)
	}
}

func TestChangeLog_Hash(t *testing.T) {
	tests := getCustomTypeData(t)
	for i, test := range tests {
		assert.Equal(t, test.hash, test.input.Hash().Hex(), "index=%d %s", i, test.input)
	}
}

func TestChangeLog_EncodeRLP_DecodeRLP(t *testing.T) {
	tests := getCustomTypeData(t)
	for i, test := range tests {
		enc, err := rlp.EncodeToBytes(test.input)
		assert.NoError(t, err, "index=%d %s", i, test.input)
		assert.Equal(t, test.rlp, hexutil.Encode(enc), "index=%d %s", i, test.input)

		decodeResult := new(types.ChangeLog)
		err = rlp.DecodeBytes(enc, decodeResult)
		assert.Equal(t, test.decodeErr, err, "index=%d %s", i, test.input)
		if test.decodeErr == nil {
			if test.decoded != "" {
				assert.Equal(t, test.decoded, decodeResult.String(), "index=%d %s", i, test.input)
			} else {
				assert.Equal(t, test.str, decodeResult.String(), "index=%d %s", i, test.input)
			}
		}
	}
}

func TestIsValuable(t *testing.T) {
	tests := getCustomTypeData(t)
	for i, test := range tests {
		assert.Equal(t, true, IsValuable(test.input), "index=%d %s", i, test.input)
	}
}

func findEvent(processor *testProcessor, txHash common.Hash) []*types.Event {
	result := make([]*types.Event, 0)
	for _, event := range processor.Events {
		if event.TxHash == txHash {
			result = append(result, event)
		}
	}
	return result
}

func TestChangeLog_Undo(t *testing.T) {
	processor := &testProcessor{}
	event1 := &types.Event{TxHash: common.HexToHash("0x666")}
	processor.PushEvent(&types.Event{})
	processor.PushEvent(event1)

	tests := []struct {
		input      *types.ChangeLog
		undoErr    error
		afterCheck func(types.AccountAccessor)
	}{
		// 0 NewBalanceLog
		{
			input: NewBalanceLog(processor.createAccount(BalanceLog, 1), big.NewInt(120)),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, big.NewInt(100), accessor.GetBalance())
			},
		},
		// 1 NewBalanceLog no OldVal
		{
			input:   &types.ChangeLog{LogType: BalanceLog, Address: processor.createAccount(BalanceLog, 1).GetAddress(), Version: 1},
			undoErr: types.ErrWrongChangeLogData,
		},
		// 2 NewStorageLog
		{
			input: NewStorageLogWithoutError(t, processor.createAccount(StorageLog, 1), common.HexToHash("0xaaa"), []byte{12}),
			afterCheck: func(accessor types.AccountAccessor) {
				currentVal, err := accessor.GetStorageState(common.HexToHash("0xaaa"))
				assert.Equal(t, []byte{45, 67}, currentVal)
				assert.NoError(t, err)
			},
		},
		// 3 NewStorageLog no OldVal
		{
			input:   &types.ChangeLog{LogType: StorageLog, Address: processor.createAccount(StorageLog, 1).GetAddress(), Version: 1},
			undoErr: types.ErrWrongChangeLogData,
		},
		// 4 NewStorageLog no Extra
		{
			input:   &types.ChangeLog{LogType: StorageLog, Address: processor.createAccount(StorageLog, 1).GetAddress(), Version: 1, OldVal: []byte{45, 67}},
			undoErr: types.ErrWrongChangeLogData,
		},
		// 5 NewCodeLog
		{
			input: NewCodeLog(processor.createAccount(CodeLog, 1), []byte{12}),
			afterCheck: func(accessor types.AccountAccessor) {
				code, err := accessor.GetCode()
				assert.Empty(t, code)
				assert.NoError(t, err)
			},
		},
		// 6 NewAddEventLog
		{
			input: NewAddEventLog(processor.createAccount(AddEventLog, 1), event1),
			afterCheck: func(accessor types.AccountAccessor) {
				events := findEvent(processor, event1.TxHash)
				assert.Empty(t, events)
			},
		},
		// 7 NewSuicideLog
		{
			input: NewSuicideLog(processor.createAccount(SuicideLog, 1)),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, big.NewInt(100), accessor.GetBalance())
				assert.Equal(t, false, accessor.GetSuicide())
			},
		},
	}

	for i, test := range tests {
		err := test.input.Undo(processor)
		assert.Equal(t, test.undoErr, err, "index=%d %s", i, test.input)
		if test.undoErr == nil && test.afterCheck != nil {
			a := processor.GetAccount(test.input.Address)
			test.afterCheck(a)
		}
	}
}

func TestChangeLog_Redo(t *testing.T) {
	processor := &testProcessor{}

	// decrease account version to make redo available
	decreaseVersion := func(log *types.ChangeLog) *types.ChangeLog {
		account := processor.GetAccount(log.Address)
		account.SetVersion(log.LogType, account.GetVersion(log.LogType)-1)
		return log
	}

	tests := []struct {
		input      *types.ChangeLog
		redoErr    error
		afterCheck func(types.AccountAccessor)
	}{
		// 0 NewBalanceLog
		{
			input: decreaseVersion(NewBalanceLog(processor.createAccount(BalanceLog, 1), big.NewInt(120))),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, big.NewInt(120), accessor.GetBalance())
			},
		},
		// 1 NewBalanceLog no NewVal
		{
			input:   &types.ChangeLog{LogType: BalanceLog, Address: processor.createAccount(BalanceLog, 0).GetAddress(), Version: 1},
			redoErr: types.ErrWrongChangeLogData,
		},
		// 2 NewStorageLog
		{
			input: decreaseVersion(NewStorageLogWithoutError(t, processor.createAccount(StorageLog, 1), common.HexToHash("0xaaa"), []byte{12})),
			afterCheck: func(accessor types.AccountAccessor) {
				currentVal, err := accessor.GetStorageState(common.HexToHash("0xaaa"))
				assert.Equal(t, []byte{12}, currentVal)
				assert.NoError(t, err)
			},
		},
		// 3 NewStorageLog no NewVal
		{
			input:   &types.ChangeLog{LogType: StorageLog, Address: processor.createAccount(StorageLog, 0).GetAddress(), Version: 1},
			redoErr: types.ErrWrongChangeLogData,
		},
		// 4 NewStorageLog no Extra
		{
			input:   &types.ChangeLog{LogType: StorageLog, Address: processor.createAccount(StorageLog, 0).GetAddress(), Version: 1, OldVal: []byte{45, 67}},
			redoErr: types.ErrWrongChangeLogData,
		},
		// 5 NewCodeLog
		{
			input: decreaseVersion(NewCodeLog(processor.createAccount(CodeLog, 1), []byte{12})),
			afterCheck: func(accessor types.AccountAccessor) {
				code, err := accessor.GetCode()
				assert.Equal(t, types.Code{12}, code)
				assert.NoError(t, err)
			},
		},
		// 6 NewCodeLog no NewVal
		{
			input:   &types.ChangeLog{LogType: CodeLog, Address: processor.createAccount(CodeLog, 0).GetAddress(), Version: 1},
			redoErr: types.ErrWrongChangeLogData,
		},
		// 7 NewAddEventLog
		{
			input: decreaseVersion(NewAddEventLog(processor.createAccount(AddEventLog, 1), &types.Event{
				Address:   common.HexToAddress("0xaaa"),
				Topics:    []common.Hash{common.HexToHash("bbb"), common.HexToHash("ccc")},
				Data:      []byte{0x80, 0x0},
				BlockHash: common.HexToHash("0xddd"),
				TxHash:    common.HexToHash("0x777"),
			})),
			afterCheck: func(accessor types.AccountAccessor) {
				events := findEvent(processor, common.HexToHash("0x777"))
				assert.Equal(t, 1, len(events))
				assert.Equal(t, []byte{0x80, 0x0}, events[0].Data)
			},
		},
		// 8 NewAddEventLog no NewVal
		{
			input:   &types.ChangeLog{LogType: AddEventLog, Address: processor.createAccount(AddEventLog, 0).GetAddress(), Version: 1},
			redoErr: types.ErrWrongChangeLogData,
		},
		// 9 NewBalanceLog
		{
			input: decreaseVersion(NewSuicideLog(processor.createAccount(BalanceLog, 1))),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, true, accessor.GetSuicide())
				assert.Equal(t, big.NewInt(0), accessor.GetBalance())
			},
		},
	}

	for i, test := range tests {
		err := test.input.Redo(processor)
		assert.Equal(t, test.redoErr, err, "index=%d %s", i, test.input)
		if test.redoErr == nil && test.afterCheck != nil {
			a := processor.GetAccount(test.input.Address)
			test.afterCheck(a)
		}
	}
}
