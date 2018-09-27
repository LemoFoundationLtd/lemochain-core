package account

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

type testAccount struct {
	types.AccountData
	Code     types.Code
	Storage  map[common.Hash][]byte
	suicided bool
}

func (f *testAccount) GetAddress() common.Address  { return f.AccountData.Address }
func (f *testAccount) GetBalance() *big.Int        { return f.AccountData.Balance }
func (f *testAccount) SetBalance(balance *big.Int) { f.AccountData.Balance = balance }
func (f *testAccount) GetVersion() uint32          { return f.AccountData.Version }
func (f *testAccount) SetVersion(version uint32)   { f.AccountData.Version = version }
func (f *testAccount) GetSuicide() bool            { return f.suicided }
func (f *testAccount) SetSuicide(suicided bool) {
	if suicided {
		f.SetBalance(new(big.Int))
		f.SetCodeHash(common.Hash{})
		f.SetStorageRoot(common.Hash{})
	}
	f.suicided = suicided
}
func (f *testAccount) GetCodeHash() common.Hash                        { return f.AccountData.CodeHash }
func (f *testAccount) SetCodeHash(codeHash common.Hash)                { f.AccountData.CodeHash = codeHash }
func (f *testAccount) GetCode() (types.Code, error)                    { return f.Code, nil }
func (f *testAccount) SetCode(code types.Code)                         { f.Code = code }
func (f *testAccount) IsEmpty() bool                                   { return f.AccountData.Version == 0 }
func (f *testAccount) GetStorageRoot() common.Hash                     { return f.AccountData.StorageRoot }
func (f *testAccount) SetStorageRoot(root common.Hash)                 { f.AccountData.StorageRoot = root }
func (f *testAccount) GetStorageState(key common.Hash) ([]byte, error) { return f.Storage[key], nil }
func (f *testAccount) SetStorageState(key common.Hash, value []byte) error {
	f.Storage[key] = value
	return nil
}

type testProcessor struct {
	Accounts map[common.Address]*testAccount
	Events   []*types.Event
}

var ErrNoAccount = errors.New("no account from address")

func (p *testProcessor) GetAccount(addr common.Address) (types.AccountAccessor, error) {
	account, ok := p.Accounts[addr]
	if !ok {
		return nil, ErrNoAccount
	}
	return account, nil
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

func (p *testProcessor) createAccount(version uint32) *testAccount {
	if p.Accounts == nil {
		p.Accounts = make(map[common.Address]*testAccount)
	}
	index := len(p.Accounts) + 1
	address := common.BigToAddress(big.NewInt(int64(index)))
	account := &testAccount{
		AccountData: types.AccountData{
			Address: address,
			Balance: big.NewInt(100),
			Version: version,
		},
		Storage: map[common.Hash][]byte{
			common.HexToHash("0xaaa"): {45, 67},
		},
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
		input:   NewBalanceLog(processor.createAccount(0), big.NewInt(0)),
		str:     "BalanceLog: 0x0000000000000000000000000000000000000001 1 100 0 <nil>",
		hash:    "0x9532d32f3b2253bb6fb438cb8ac394882b15a1a2883e6619398d50f059ea2692",
		rlp:     "0xd9019400000000000000000000000000000000000000010180c0",
		decoded: "BalanceLog: 0x0000000000000000000000000000000000000001 1 <nil> 0 <nil>",
	})

	// 1 StorageLog
	tests = append(tests, testCustomTypeConfig{
		input:   NewStorageLogWithoutError(t, processor.createAccount(0), common.HexToHash("0xaaa"), []byte{67, 89}),
		str:     "StorageLog: 0x0000000000000000000000000000000000000002 1 [45 67] [67 89] [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 10 170]",
		hash:    "0x47c7805a8377ab0140d96e092a045a03e8e41c6d0b3bb00307d6123d62c59dba",
		rlp:     "0xf83b0294000000000000000000000000000000000000000201824359a00000000000000000000000000000000000000000000000000000000000000aaa",
		decoded: "StorageLog: 0x0000000000000000000000000000000000000002 1 <nil> [67 89] [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 10 170]",
	})

	// 2 CodeLog
	tests = append(tests, testCustomTypeConfig{
		input:   NewCodeLog(processor.createAccount(0), []byte{0x12, 0x34}),
		str:     "CodeLog: 0x0000000000000000000000000000000000000003 1 <nil> [18 52] <nil>",
		hash:    "0xcec04ee7ea02f669bfee54633269673d706b59ab821127b5c5491d4dc1c4076a",
		rlp:     "0xdb0394000000000000000000000000000000000000000301821234c0",
		decoded: "CodeLog: 0x0000000000000000000000000000000000000003 1 <nil> [18 52] <nil>",
	})

	// 3 AddEventLog
	newEvent := &types.Event{
		Address: common.HexToAddress("0xaaa"),
		Topics:  []common.Hash{common.HexToHash("bbb"), common.HexToHash("ccc")},
		Data:    []byte{0x80, 0x0},
	}
	tests = append(tests, testCustomTypeConfig{
		input:   NewAddEventLog(processor.createAccount(0), newEvent),
		str:     "AddEventLog: 0x0000000000000000000000000000000000000004 1 <nil> event: 0000000000000000000000000000000000000aaa [0000000000000000000000000000000000000000000000000000000000000bbb 0000000000000000000000000000000000000000000000000000000000000ccc] 8000 0000000000000000000000000000000000000000000000000000000000000000 0 0000000000000000000000000000000000000000000000000000000000000000 0 <nil>",
		hash:    "0x89761d5f9ee931d7e514de58289cce64c8f89491305668bdabd3cf3be815282b",
		rlp:     "0xf8760494000000000000000000000000000000000000000401f85c940000000000000000000000000000000000000aaaf842a00000000000000000000000000000000000000000000000000000000000000bbba00000000000000000000000000000000000000000000000000000000000000ccc828000c0",
		decoded: "AddEventLog: 0x0000000000000000000000000000000000000004 1 <nil> event: 0000000000000000000000000000000000000aaa [0000000000000000000000000000000000000000000000000000000000000bbb 0000000000000000000000000000000000000000000000000000000000000ccc] 8000 0000000000000000000000000000000000000000000000000000000000000000 0 0000000000000000000000000000000000000000000000000000000000000000 0 <nil>",
	})

	// 4 SuicideLog
	tests = append(tests, testCustomTypeConfig{
		input:   NewSuicideLog(processor.createAccount(0)),
		str:     "SuicideLog: 0x0000000000000000000000000000000000000005 1 &{[0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0] 100 0 [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0] [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0] []} <nil> <nil>",
		hash:    "0x6204ac1a4be1e52c77942259e094c499b263ad7176ae9060d5bae2b856c9743a",
		rlp:     "0xd90594000000000000000000000000000000000000000501c0c0",
		decoded: "SuicideLog: 0x0000000000000000000000000000000000000005 1 <nil> <nil> <nil>",
	})

	return tests
}

func TestChangeLog_String(t *testing.T) {
	tests := getCustomTypeData(t)
	for i, test := range tests {
		if test.input.String() != test.str {
			t.Errorf("test %d. want str: %s, got: %s", i, test.str, test.input.String())
		}
	}
}

func TestChangeLog_Hash(t *testing.T) {
	tests := getCustomTypeData(t)
	for i, test := range tests {
		if test.input.Hash().Hex() != test.hash {
			t.Errorf("test %d. want hash: %s, got: %s", i, test.hash, test.input.Hash().Hex())
		}
	}
}

func TestChangeLog_EncodeRLP_DecodeRLP(t *testing.T) {
	tests := getCustomTypeData(t)
	for i, test := range tests {
		enc, err := rlp.EncodeToBytes(test.input)
		if err != nil {
			t.Errorf("test %d. rlp encode error: %s", i, err)
		} else if hexutil.Encode(enc) != test.rlp {
			t.Errorf("test %d. want rlp: %s, got: %s", i, test.rlp, hexutil.Encode(enc))
		} else {
			decodeResult := new(types.ChangeLog)
			err = rlp.DecodeBytes(enc, decodeResult)
			if err != nil {
				if test.decodeErr != err {
					t.Errorf("test %d. want decodeErr: %s, got: %s", i, test.decodeErr, err.Error())
				}
			} else if test.decodeErr != nil {
				t.Errorf("test %d. want decodeErr: %s, got: <nil>", i, test.decodeErr)
			} else if decodeResult.String() != test.decoded {
				t.Errorf("test %d. want decoded: %s, got: %s", i, test.decoded, decodeResult.String())
			}
		}
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
			input: NewBalanceLog(processor.createAccount(1), big.NewInt(120)),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, big.NewInt(100), accessor.GetBalance())
			},
		},
		// 1 NewBalanceLog no OldVal
		{
			input:   &types.ChangeLog{LogType: BalanceLog, Address: processor.createAccount(1).GetAddress(), Version: 1},
			undoErr: types.ErrWrongChangeLogData,
		},
		// 2 NewStorageLog
		{
			input: NewStorageLogWithoutError(t, processor.createAccount(1), common.HexToHash("0xaaa"), []byte{12}),
			afterCheck: func(accessor types.AccountAccessor) {
				currentVal, err := accessor.GetStorageState(common.HexToHash("0xaaa"))
				assert.Equal(t, []byte{45, 67}, currentVal)
				assert.NoError(t, err)
			},
		},
		// 3 NewStorageLog no OldVal
		{
			input:   &types.ChangeLog{LogType: StorageLog, Address: processor.createAccount(1).GetAddress(), Version: 1},
			undoErr: types.ErrWrongChangeLogData,
		},
		// 4 NewStorageLog no Extra
		{
			input:   &types.ChangeLog{LogType: StorageLog, Address: processor.createAccount(1).GetAddress(), Version: 1, OldVal: []byte{45, 67}},
			undoErr: types.ErrWrongChangeLogData,
		},
		// 5 NewCodeLog
		{
			input: NewCodeLog(processor.createAccount(1), []byte{12}),
			afterCheck: func(accessor types.AccountAccessor) {
				code, err := accessor.GetCode()
				assert.Empty(t, code)
				assert.NoError(t, err)
			},
		},
		// 6 NewAddEventLog
		{
			input: NewAddEventLog(processor.createAccount(1), event1),
			afterCheck: func(accessor types.AccountAccessor) {
				events := findEvent(processor, event1.TxHash)
				assert.Empty(t, events)
			},
		},
		// 7 NewSuicideLog
		{
			input: NewSuicideLog(processor.createAccount(1)),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, big.NewInt(100), accessor.GetBalance())
				assert.Equal(t, false, accessor.GetSuicide())
			},
		},
	}

	for i, test := range tests {
		err := test.input.Undo(processor)
		if err != nil {
			if test.undoErr != err {
				t.Errorf("test %d. undo error: %s", i, err)
			}
		} else if test.undoErr != nil {
			t.Errorf("test %d. want undoErr: %s, got: <nil>", i, test.undoErr)
		} else if test.afterCheck != nil {
			a, _ := processor.GetAccount(test.input.Address)
			test.afterCheck(a)
		}
	}
}

func TestChangeLog_Redo(t *testing.T) {
	processor := &testProcessor{}

	// decrease account version to make redo available
	decreaseVersion := func(log *types.ChangeLog) *types.ChangeLog {
		account, _ := processor.GetAccount(log.Address)
		account.SetVersion(account.GetVersion() - 1)
		return log
	}

	tests := []struct {
		input      *types.ChangeLog
		redoErr    error
		afterCheck func(types.AccountAccessor)
	}{
		// 0 NewBalanceLog
		{
			input: decreaseVersion(NewBalanceLog(processor.createAccount(1), big.NewInt(120))),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, big.NewInt(120), accessor.GetBalance())
			},
		},
		// 1 NewBalanceLog no NewVal
		{
			input:   &types.ChangeLog{LogType: BalanceLog, Address: processor.createAccount(0).GetAddress(), Version: 1},
			redoErr: types.ErrWrongChangeLogData,
		},
		// 2 NewStorageLog
		{
			input: decreaseVersion(NewStorageLogWithoutError(t, processor.createAccount(1), common.HexToHash("0xaaa"), []byte{12})),
			afterCheck: func(accessor types.AccountAccessor) {
				currentVal, err := accessor.GetStorageState(common.HexToHash("0xaaa"))
				assert.Equal(t, []byte{12}, currentVal)
				assert.NoError(t, err)
			},
		},
		// 3 NewStorageLog no NewVal
		{
			input:   &types.ChangeLog{LogType: StorageLog, Address: processor.createAccount(0).GetAddress(), Version: 1},
			redoErr: types.ErrWrongChangeLogData,
		},
		// 4 NewStorageLog no Extra
		{
			input:   &types.ChangeLog{LogType: StorageLog, Address: processor.createAccount(0).GetAddress(), Version: 1, OldVal: []byte{45, 67}},
			redoErr: types.ErrWrongChangeLogData,
		},
		// 5 NewCodeLog
		{
			input: decreaseVersion(NewCodeLog(processor.createAccount(1), []byte{12})),
			afterCheck: func(accessor types.AccountAccessor) {
				code, err := accessor.GetCode()
				assert.Equal(t, types.Code{12}, code)
				assert.NoError(t, err)
			},
		},
		// 6 NewCodeLog no NewVal
		{
			input:   &types.ChangeLog{LogType: CodeLog, Address: processor.createAccount(0).GetAddress(), Version: 1},
			redoErr: types.ErrWrongChangeLogData,
		},
		// 7 NewAddEventLog
		{
			input: decreaseVersion(NewAddEventLog(processor.createAccount(1), &types.Event{
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
			input:   &types.ChangeLog{LogType: AddEventLog, Address: processor.createAccount(0).GetAddress(), Version: 1},
			redoErr: types.ErrWrongChangeLogData,
		},
		// 9 NewBalanceLog
		{
			input: decreaseVersion(NewSuicideLog(processor.createAccount(1))),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, true, accessor.GetSuicide())
				assert.Equal(t, big.NewInt(0), accessor.GetBalance())
			},
		},
	}

	for i, test := range tests {
		err := test.input.Redo(processor)
		if err != nil {
			if test.redoErr != err {
				t.Errorf("test %d. redo error: %s", i, err)
			}
		} else if test.redoErr != nil {
			t.Errorf("test %d. want redoErr: %s, got: <nil>", i, test.redoErr)
		} else if test.afterCheck != nil {
			a, _ := processor.GetAccount(test.input.Address)
			test.afterCheck(a)
		}
	}
}
