package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-go/store"
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

func (p *testProcessor) GetNextVersion(logType types.ChangeLogType, addr common.Address) uint32 {
	account := p.Accounts[addr]
	return account.GetBaseVersion(logType) + 1
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
	account := p.createEmptyAccount()
	account.data.Balance = big.NewInt(100)
	account.data.NewestRecords = map[types.ChangeLogType]types.VersionRecord{logType: {Version: version, Height: 10}}
	account.data.VoteFor = common.HexToAddress("0x0001")
	account.cachedStorage = map[common.Hash][]byte{
		common.HexToHash("0xaaa"): {45, 67},
	}
	account.data.Candidate.Votes = big.NewInt(200)
	account.data.Candidate.Profile[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	return account
}

func (p *testProcessor) createEmptyAccount() *testAccount {
	if p.Accounts == nil {
		p.Accounts = make(map[common.Address]*testAccount)
	}
	index := len(p.Accounts) + 1
	address := common.BigToAddress(big.NewInt(int64(index)))
	account := &testAccount{
		Account: *NewAccount(nil, address, &types.AccountData{
			Address: address,
			Balance: new(big.Int),
		}),
	}
	p.Accounts[address] = account
	return account
}

type testLogConfig struct {
	input      *types.ChangeLog
	isValuable bool
	str        string
	hash       string
	rlp        string
	decoded    string
	decodeErr  error
}

func NewStorageLogWithoutError(t *testing.T, processor *testProcessor, version uint32, key common.Hash, newVal []byte) *types.ChangeLog {
	account := processor.createAccount(StorageLog, version)
	log, err := NewStorageLog(processor, account, key, newVal)
	assert.NoError(t, err)
	return log
}

func getTestLogs(t *testing.T) []testLogConfig {
	processor := &testProcessor{}
	tests := make([]testLogConfig, 0)

	// 0 BalanceLog
	tests = append(tests, testLogConfig{
		input:      NewBalanceLog(processor, processor.createAccount(BalanceLog, 0), big.NewInt(0)),
		isValuable: true,
		str:        "BalanceLog{Account: Lemo8888888888888888888888888888888888BW, Version: 1, OldVal: 100, NewVal: 0}",
		hash:       "0x9532d32f3b2253bb6fb438cb8ac394882b15a1a2883e6619398d50f059ea2692",
		rlp:        "0xd9019400000000000000000000000000000000000000010180c0",
		decoded:    "BalanceLog{Account: Lemo8888888888888888888888888888888888BW, Version: 1, NewVal: 0}",
	})
	// 1 BalanceLog
	tests = append(tests, testLogConfig{
		input:      NewBalanceLog(processor, processor.createAccount(BalanceLog, 0), big.NewInt(100)),
		isValuable: false,
		str:        "BalanceLog{Account: Lemo8888888888888888888888888888888888QR, Version: 1, OldVal: 100, NewVal: 100}",
		hash:       "0xdca6aeea6698dc07743ef2eec68d49117906684ff3540cf055b02c58b8b05ada",
		rlp:        "0xd9019400000000000000000000000000000000000000020164c0",
		decoded:    "BalanceLog{Account: Lemo8888888888888888888888888888888888QR, Version: 1, NewVal: 100}",
	})

	// 2 StorageLog
	tests = append(tests, testLogConfig{
		input:      NewStorageLogWithoutError(t, processor, 0, common.HexToHash("0xaaa"), []byte{67, 89}),
		isValuable: true,
		str:        "StorageLog{Account: Lemo88888888888888888888888888888888835N, Version: 1, OldVal: [45 67], NewVal: [67 89], Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 10 170]}",
		hash:       "0x797089e263ae76c3b21513415e431fc21ffe4e9cca51e2991f95c82ba3b66d46",
		rlp:        "0xf83b0294000000000000000000000000000000000000000301824359a00000000000000000000000000000000000000000000000000000000000000aaa",
		decoded:    "StorageLog{Account: Lemo88888888888888888888888888888888835N, Version: 1, NewVal: [67 89], Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 10 170]}",
	})
	// 3 StorageLog
	tests = append(tests, testLogConfig{
		input:      NewStorageLogWithoutError(t, processor, 0, common.HexToHash("0xaaa"), []byte{45, 67}),
		isValuable: false,
		str:        "StorageLog{Account: Lemo8888888888888888888888888888888883GH, Version: 1, OldVal: [45 67], NewVal: [45 67], Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 10 170]}",
		hash:       "0xe06a8520e227694035f190f8cf4e7e63d6ad771e78aba3da34ee8ed86c4735be",
		rlp:        "0xf83b0294000000000000000000000000000000000000000401822d43a00000000000000000000000000000000000000000000000000000000000000aaa",
		decoded:    "StorageLog{Account: Lemo8888888888888888888888888888888883GH, Version: 1, NewVal: [45 67], Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 10 170]}",
	})

	// 4 CodeLog
	tests = append(tests, testLogConfig{
		input:      NewCodeLog(processor, processor.createAccount(CodeLog, 0), []byte{0x12, 0x34}),
		isValuable: true,
		str:        "CodeLog{Account: Lemo8888888888888888888888888888888883WD, Version: 1, NewVal: 0x1234}",
		hash:       "0x71414949bfacba2baae60c0d947827bdab6c92e0e388a60f110228245d395a63",
		rlp:        "0xdb0394000000000000000000000000000000000000000501821234c0",
	})
	// 5 CodeLog
	tests = append(tests, testLogConfig{
		input:      NewCodeLog(processor, processor.createAccount(CodeLog, 0), []byte{}),
		isValuable: false,
		str:        "CodeLog{Account: Lemo88888888888888888888888888888888849A, Version: 1, NewVal: }",
		hash:       "0x7f95953470d2e6140ba7f18e8d7471708f6cd828bdcdb94d5e625e2df5765e56",
		rlp:        "0xd9039400000000000000000000000000000000000000060180c0",
	})

	// 6 AddEventLog
	newEvent := &types.Event{
		Address: common.HexToAddress("0xaaa"),
		Topics:  []common.Hash{common.HexToHash("bbb"), common.HexToHash("ccc")},
		Data:    []byte{0x80, 0x0},
	}
	tests = append(tests, testLogConfig{
		input:      NewAddEventLog(processor, processor.createAccount(AddEventLog, 0), newEvent),
		isValuable: true,
		str:        "AddEventLog{Account: Lemo8888888888888888888888888888888884N7, Version: 1, NewVal: event: 0000000000000000000000000000000000000aaa [0000000000000000000000000000000000000000000000000000000000000bbb 0000000000000000000000000000000000000000000000000000000000000ccc] 8000 0000000000000000000000000000000000000000000000000000000000000000 0 0000000000000000000000000000000000000000000000000000000000000000 0}",
		hash:       "0xf2471fc3ebed4f84c29ceded41910de27f1fbdfdff0430b3421e379ea2ed4554",
		rlp:        "0xf8760494000000000000000000000000000000000000000701f85c940000000000000000000000000000000000000aaaf842a00000000000000000000000000000000000000000000000000000000000000bbba00000000000000000000000000000000000000000000000000000000000000ccc828000c0",
	})
	// It is not possible to set NewVal in AddEventLog to nil. We can't test is because we can't rlp encode a (*types.Event)(nil)

	// 7 SuicideLog
	tests = append(tests, testLogConfig{
		input:      NewSuicideLog(processor, processor.createAccount(SuicideLog, 0)),
		isValuable: true,
		str:        "Lemo888888888888888888888888888888888534, Version: 1, OldVal: {Address: Lemo888888888888888888888888888888888888, Balance: 100, VoteFor: Lemo888888888888888888888888888888888888, TxCount: 0}}",
		hash:       "0xe2d92fe499dfbd00be20be9a041ac164f0c985a81c7ae8f5921f22a9f9d98090",
		rlp:        "0xd90594000000000000000000000000000000000000000801c0c0",
		decoded:    "SuicideLog{Account: Lemo888888888888888888888888888888888534, Version: 1}",
	})
	// 8 SuicideLog
	tests = append(tests, testLogConfig{
		input:      NewSuicideLog(processor, processor.createEmptyAccount()),
		isValuable: false,
		str:        "SuicideLog{Account: Lemo888888888888888888888888888888888534, Version: 1, OldVal: {Address: Lemo888888888888888888888888888888888888, Balance: 100, VoteFor: Lemo888888888888888888888888888888888888, TxCount: 0}}",
		hash:       "0x8d05be4a001f2ad367c9642b30a1825621eefa9fd56f7383eb7ef74a67fed1d4",
		rlp:        "0xd90594000000000000000000000000000000000000000901c0c0",
		decoded:    "SuicideLog{Account: Lemo8888888888888888888888888888888885CZ, Version: 1}",
	})

	// 9 VotesLog
	tests = append(tests, testLogConfig{
		input:      NewVotesLog(processor, processor.createAccount(VotesLog, 0), big.NewInt(1000)),
		isValuable: true,
		str:        "VotesLog{Account: Lemo8888888888888888888888888888888885RT, Version: 1, OldVal: 200, NewVal: 1000}",
		hash:       "0x06d9d7c475a3d6b50dd790452e3a753e2f01adf0f7448d3e3ddb8f394d63741a",
		rlp:        "0xdb0794000000000000000000000000000000000000000a018203e8c0",
		decoded:    "VotesLog{Account: Lemo8888888888888888888888888888888885RT, Version: 1, NewVal: 1000}",
	})
	// 10 VotesLog
	tests = append(tests, testLogConfig{
		input:      NewVotesLog(processor, processor.createAccount(VotesLog, 0), big.NewInt(200)),
		isValuable: false,
		str:        "VotesLog{Account: Lemo88888888888888888888888888888888866Q, Version: 1, OldVal: 200, NewVal: 200}",
		hash:       "0x890f53b1b8b23a4086db0feb9f602101b950e5407770f58be537e7dc1c5aea8e",
		rlp:        "0xda0794000000000000000000000000000000000000000b0181c8c0",
		decoded:    "VotesLog{Account: Lemo88888888888888888888888888888888866Q, Version: 1, NewVal: 200}",
	})

	// 11 VoteForLog
	tests = append(tests, testLogConfig{
		input:      NewVoteForLog(processor, processor.createAccount(VoteForLog, 0), common.HexToAddress("0x0002")),
		isValuable: true,
		str:        "VoteForLog{Account: Lemo8888888888888888888888888888888886HK, Version: 1, OldVal: Lemo8888888888888888888888888888888888BW, NewVal: Lemo8888888888888888888888888888888888QR}",
		hash:       "0x738ccb6ac44d3b4ee4feaaaec6be9290bedd24a180ecbf2097536f8b4d0c2136",
		rlp:        "0xed0694000000000000000000000000000000000000000c01940000000000000000000000000000000000000002c0",
		decoded:    "VoteForLog{Account: Lemo8888888888888888888888888888888886HK, Version: 1, NewVal: Lemo8888888888888888888888888888888888QR}",
	})
	// 12 VoteForLog
	tests = append(tests, testLogConfig{
		input:      NewVoteForLog(processor, processor.createAccount(VoteForLog, 0), common.HexToAddress("0x0001")),
		isValuable: false,
		str:        "VoteForLog{Account: Lemo8888888888888888888888888888888886YG, Version: 1, OldVal: Lemo8888888888888888888888888888888888BW, NewVal: Lemo8888888888888888888888888888888888BW}",
		hash:       "0x4bf77f5842a27b81509583e0872f647b7116fb65225af79fb4932f0c7f4bac82",
		rlp:        "0xed0694000000000000000000000000000000000000000d01940000000000000000000000000000000000000001c0",
		decoded:    "VoteForLog{Account: Lemo8888888888888888888888888888888886YG, Version: 1, NewVal: Lemo8888888888888888888888888888888888BW}",
	})

	return tests
}

func TestChangeLog_String(t *testing.T) {
	tests := getTestLogs(t)
	for i, test := range tests {
		assert.Equal(t, test.str, test.input.String(), "index=%d %s", i, test.input)
	}
}

func TestChangeLog_Hash(t *testing.T) {
	tests := getTestLogs(t)
	for i, test := range tests {
		assert.Equal(t, test.hash, test.input.Hash().Hex(), "index=%d %s", i, test.input)
	}
}

func TestChangeLog_EncodeRLP_DecodeRLP(t *testing.T) {
	tests := getTestLogs(t)
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
	tests := getTestLogs(t)
	for i, test := range tests {
		assert.Equal(t, test.isValuable, IsValuable(test.input), "index=%d %s", i, test.input)
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
			input: NewBalanceLog(processor, processor.createAccount(BalanceLog, 1), big.NewInt(120)),
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
			input: NewStorageLogWithoutError(t, processor, 1, common.HexToHash("0xaaa"), []byte{12}),
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
			input: NewCodeLog(processor, processor.createAccount(CodeLog, 1), []byte{12}),
			afterCheck: func(accessor types.AccountAccessor) {
				code, err := accessor.GetCode()
				assert.Empty(t, code)
				assert.NoError(t, err)
			},
		},
		// 6 NewAddEventLog
		{
			input: NewAddEventLog(processor, processor.createAccount(AddEventLog, 1), event1),
			afterCheck: func(accessor types.AccountAccessor) {
				events := findEvent(processor, event1.TxHash)
				assert.Empty(t, events)
			},
		},
		// 7 NewSuicideLog
		{
			input: NewSuicideLog(processor, processor.createAccount(SuicideLog, 1)),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, big.NewInt(100), accessor.GetBalance())
				assert.Equal(t, false, accessor.GetSuicide())
			},
		},
		// 8 VoteFor
		{
			input: NewVoteForLog(processor, processor.createAccount(VoteForLog, 1), common.HexToAddress("0x0002")),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, common.HexToAddress("0x0001"), accessor.GetVoteFor())
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
		// TODO
		// account := processor.GetAccount(log.Address)
		// account.SetVersion(log.LogType, account.GetVersion(log.LogType)-1)
		return log
	}

	tests := []struct {
		input      *types.ChangeLog
		redoErr    error
		afterCheck func(types.AccountAccessor)
	}{
		// 0 NewBalanceLog
		{
			input: decreaseVersion(NewBalanceLog(processor, processor.createAccount(BalanceLog, 1), big.NewInt(120))),
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
			input: decreaseVersion(NewStorageLogWithoutError(t, processor, 1, common.HexToHash("0xaaa"), []byte{12})),
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
			input: decreaseVersion(NewCodeLog(processor, processor.createAccount(CodeLog, 1), []byte{12})),
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
			input: decreaseVersion(NewAddEventLog(processor, processor.createAccount(AddEventLog, 1), &types.Event{
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
			input: decreaseVersion(NewSuicideLog(processor, processor.createAccount(BalanceLog, 1))),
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

// Set twice, and the first changeLog shouldn't change
func TestChangeLog_valueShouldBeStableCandidateProfile(t *testing.T) {
	store.ClearData()
	db := newDB()
	manager := NewManager(common.Hash{}, db)
	account := manager.GetAccount(common.HexToAddress("0x1"))

	// BalanceLog
	manager.clear()
	balance := big.NewInt(123)
	account.SetBalance(balance)
	assert.Equal(t, 1, len(manager.GetChangeLogs()))
	// get and set again
	balance = account.GetBalance()
	balance.Set(big.NewInt(234))
	account.SetBalance(balance)
	assert.Equal(t, 2, len(manager.GetChangeLogs()))
	oldBalance := manager.GetChangeLogs()[0].NewVal.(big.Int)
	assert.Equal(t, *big.NewInt(123), oldBalance)

	// StorageLog
	manager.clear()
	val := make([]byte, 1)
	val[0] = 56
	_ = account.SetStorageState(h(1), val)
	assert.Equal(t, 1, len(manager.GetChangeLogs()))
	// get and set again
	val, _ = account.GetStorageState(h(1))
	val[0] = 48
	_ = account.SetStorageState(h(1), val)
	assert.Equal(t, 2, len(manager.GetChangeLogs()))
	oldVal := manager.GetChangeLogs()[0].NewVal.([]byte)
	assert.Equal(t, byte(56), oldVal[0])

	// VoteForLog
	manager.clear()
	voteFor := common.HexToAddress("0x123")
	account.SetVoteFor(voteFor)
	assert.Equal(t, 1, len(manager.GetChangeLogs()))
	// get and set again
	voteFor = account.GetVoteFor()
	voteFor[0] = 88
	account.SetVoteFor(voteFor)
	assert.Equal(t, 2, len(manager.GetChangeLogs()))
	oldVoteFor := manager.GetChangeLogs()[0].NewVal.(common.Address)
	assert.Equal(t, common.HexToAddress("0x123"), oldVoteFor)

	// Votes
	manager.clear()
	votes := big.NewInt(123)
	account.SetVotes(votes)
	assert.Equal(t, 1, len(manager.GetChangeLogs()))
	// get and set again
	votes = account.GetVotes()
	votes.Set(big.NewInt(234))
	account.SetVotes(votes)
	assert.Equal(t, 2, len(manager.GetChangeLogs()))
	oldVotes := manager.GetChangeLogs()[0].NewVal.(big.Int)
	assert.Equal(t, *big.NewInt(123), oldVotes)

	// CandidateProfileLog
	manager.clear()
	profile := make(types.CandidateProfile, 0)
	profile["aa"] = "bb"
	account.SetCandidateProfile(profile)
	assert.Equal(t, 1, len(manager.GetChangeLogs()))
	// get and set again
	profile = account.GetCandidateProfile()
	profile["aa"] = "cc"
	account.SetCandidateProfile(profile)
	assert.Equal(t, 2, len(manager.GetChangeLogs()))
	oldProfile := manager.GetChangeLogs()[0].NewVal.(*types.CandidateProfile)
	assert.Equal(t, "bb", (*oldProfile)["aa"])

	// The value in TxCountLog is uint32. No need to test
}
