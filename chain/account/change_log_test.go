package account

import (
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
			VoteFor:       common.HexToAddress("0x0001"),
		}),
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

func NewStorageLogWithoutError(t *testing.T, processor *testProcessor, version uint32, key common.Hash, newVal []byte) *types.ChangeLog {
	account := processor.createAccount(StorageLog, version)
	log, err := NewStorageLog(processor, account, key, newVal)
	assert.NoError(t, err)
	return log
}

func getCustomTypeData(t *testing.T) []testCustomTypeConfig {
	processor := &testProcessor{}
	tests := make([]testCustomTypeConfig, 0)

	// 0 BalanceLog
	tests = append(tests, testCustomTypeConfig{
		input:   NewBalanceLog(processor, processor.createAccount(BalanceLog, 0), big.NewInt(0)),
		str:     "BalanceLog{Account: Lemo8888888888888888888888888888888888BW, Version: 1, OldVal: 100, NewVal: 0}",
		hash:    "0x9532d32f3b2253bb6fb438cb8ac394882b15a1a2883e6619398d50f059ea2692",
		rlp:     "0xd9019400000000000000000000000000000000000000010180c0",
		decoded: "BalanceLog{Account: Lemo8888888888888888888888888888888888BW, Version: 1, NewVal: 0}",
	})

	// 1 StorageLog
	tests = append(tests, testCustomTypeConfig{
		input:   NewStorageLogWithoutError(t, processor, 0, common.HexToHash("0xaaa"), []byte{67, 89}),
		str:     "StorageLog{Account: Lemo8888888888888888888888888888888888QR, Version: 1, OldVal: [45 67], NewVal: [67 89], Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 10 170]}",
		hash:    "0x47c7805a8377ab0140d96e092a045a03e8e41c6d0b3bb00307d6123d62c59dba",
		rlp:     "0xf83b0294000000000000000000000000000000000000000201824359a00000000000000000000000000000000000000000000000000000000000000aaa",
		decoded: "StorageLog{Account: Lemo8888888888888888888888888888888888QR, Version: 1, NewVal: [67 89], Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 10 170]}",
	})

	// 2 CodeLog
	tests = append(tests, testCustomTypeConfig{
		input: NewCodeLog(processor, processor.createAccount(CodeLog, 0), []byte{0x12, 0x34}),
		str:   "CodeLog{Account: Lemo88888888888888888888888888888888835N, Version: 1, NewVal: 0x1234}",
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
		input: NewAddEventLog(processor, processor.createAccount(AddEventLog, 0), newEvent),
		str:   "AddEventLog{Account: Lemo8888888888888888888888888888888883GH, Version: 1, NewVal: event: 0000000000000000000000000000000000000aaa [0000000000000000000000000000000000000000000000000000000000000bbb 0000000000000000000000000000000000000000000000000000000000000ccc] 8000 0000000000000000000000000000000000000000000000000000000000000000 0 0000000000000000000000000000000000000000000000000000000000000000 0}",
		hash:  "0x89761d5f9ee931d7e514de58289cce64c8f89491305668bdabd3cf3be815282b",
		rlp:   "0xf8760494000000000000000000000000000000000000000401f85c940000000000000000000000000000000000000aaaf842a00000000000000000000000000000000000000000000000000000000000000bbba00000000000000000000000000000000000000000000000000000000000000ccc828000c0",
	})

	// 4 SuicideLog
	tests = append(tests, testCustomTypeConfig{
		input:   NewSuicideLog(processor, processor.createAccount(SuicideLog, 0)),
		str:     "SuicideLog{Account: Lemo8888888888888888888888888888888883WD, Version: 1, OldVal: {Address: Lemo888888888888888888888888888888888888, Balance: 100, VoteFor: Lemo888888888888888888888888888888888888, TxCount: 0}}",
		hash:    "0x6204ac1a4be1e52c77942259e094c499b263ad7176ae9060d5bae2b856c9743a",
		rlp:     "0xd90594000000000000000000000000000000000000000501c0c0",
		decoded: "SuicideLog{Account: Lemo8888888888888888888888888888888883WD, Version: 1}",
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

		// VoteFor
		{
			input: NewVoteForLog(processor, processor.createAccount(VoteForLog, 1), common.HexToAddress("0x0002")),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, common.HexToAddress("0x0001"), accessor.GetVoteFor())
				// assert.Equal(t, false, accessor.GetSuicide())
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
