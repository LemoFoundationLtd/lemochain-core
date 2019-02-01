package types

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

type testAccount struct {
	AccountData
}

func (f *testAccount) GetTxCount() uint32 { return f.GetTxCount() }

func (f *testAccount) SetTxCount(count uint32) {
	f.SetTxCount(count)
}

func (f *testAccount) GetVoteFor() common.Address { return f.GetVoteFor() }

func (f *testAccount) SetVoteFor(addr common.Address) {
	f.SetVoteFor(addr)
}

func (f *testAccount) GetVotes() *big.Int {
	return f.GetVotes()
}

func (f *testAccount) SetVotes(votes *big.Int) {
	f.SetVotes(votes)
}

func (f *testAccount) GetCandidateProfile() CandidateProfile {
	return f.GetCandidateProfile()
}

func (f *testAccount) SetCandidateProfile(profile CandidateProfile) {
	f.SetCandidateProfile(profile)
}

func (f *testAccount) GetAddress() common.Address  { return f.AccountData.Address }
func (f *testAccount) GetBalance() *big.Int        { return f.AccountData.Balance }
func (f *testAccount) SetBalance(balance *big.Int) { f.AccountData.Balance = balance }

// GetBaseVersion returns the version of specific change log from the base block. It is not changed by tx processing until the finalised
func (f *testAccount) GetBaseVersion(logType ChangeLogType) uint32 {
	return f.AccountData.NewestRecords[logType].Version
}
func (f *testAccount) GetSuicide() bool                                    { return false }
func (f *testAccount) SetSuicide(suicided bool)                            {}
func (f *testAccount) GetCodeHash() common.Hash                            { return f.AccountData.CodeHash }
func (f *testAccount) SetCodeHash(codeHash common.Hash)                    { f.AccountData.CodeHash = codeHash }
func (f *testAccount) GetCode() (Code, error)                              { return nil, nil }
func (f *testAccount) SetCode(code Code)                                   {}
func (f *testAccount) GetStorageRoot() common.Hash                         { return f.AccountData.StorageRoot }
func (f *testAccount) SetStorageRoot(root common.Hash)                     { f.AccountData.StorageRoot = root }
func (f *testAccount) GetStorageState(key common.Hash) ([]byte, error)     { return nil, nil }
func (f *testAccount) SetStorageState(key common.Hash, value []byte) error { return nil }
func (f *testAccount) IsEmpty() bool {
	for _, record := range f.AccountData.NewestRecords {
		if record.Version != 0 {
			return false
		}
	}
	return true
}

func (f *testAccount) GetVersion(logType ChangeLogType) uint32 {
	return f.AccountData.NewestRecords[logType].Version
}

type testProcessor struct {
	Accounts map[common.Address]*testAccount
}

func (p *testProcessor) GetAccount(address common.Address) AccountAccessor {
	account, ok := p.Accounts[address]
	if !ok {
		account = &testAccount{
			AccountData: AccountData{
				Address: address,
			},
		}
		p.Accounts[address] = account
	}
	return account
}

func (p *testProcessor) createAccount(logType ChangeLogType, version uint32) *testAccount {
	if p.Accounts == nil {
		p.Accounts = make(map[common.Address]*testAccount)
	}
	index := len(p.Accounts) + 1
	address := common.BigToAddress(big.NewInt(int64(index)))
	account := &testAccount{
		AccountData: AccountData{
			Address:       address,
			Balance:       big.NewInt(100),
			NewestRecords: map[ChangeLogType]VersionRecord{logType: {Version: version, Height: 10}},
		},
	}
	p.Accounts[address] = account
	return account
}

func (p *testProcessor) createChangeLog(logType ChangeLogType) *ChangeLog {
	account := p.createAccount(logType, 0)
	return &ChangeLog{LogType: logType, Address: account.GetAddress(), Version: 1}
}

func (p *testProcessor) GetNextVersion(logType ChangeLogType, addr common.Address) uint32 {
	return 123
}

func (p *testProcessor) PushEvent(event *Event) {}
func (p *testProcessor) PopEvent() error        { return nil }

type testCustomTypeConfig struct {
	input     *ChangeLog
	str       string
	json      string
	hash      string
	rlp       string
	decoded   string
	decodeErr error
}

type testStruct struct {
	A uint
	B string
}

func registerCustomType(logTypeInt int) {
	// NewVal: empty interface{}, Extra: testStruct
	decodeEmptyInterface := func(s *rlp.Stream) (interface{}, error) {
		var result interface{}
		err := s.Decode(&result)
		return nil, err
	}
	decodeStruct := func(s *rlp.Stream) (interface{}, error) {
		var result testStruct
		err := s.Decode(&result)
		return result, err
	}
	doFunc := func(*ChangeLog, ChangeLogProcessor) error { return nil }
	RegisterChangeLog(ChangeLogType(logTypeInt), fmt.Sprintf("ChangeLogType(%d)", logTypeInt), decodeEmptyInterface, decodeStruct, doFunc, doFunc)
}

func getCustomTypeData() []testCustomTypeConfig {
	tests := make([]testCustomTypeConfig, 0)

	// 0 custom type
	tests = append(tests, testCustomTypeConfig{
		input:     &ChangeLog{LogType: ChangeLogType(0), Address: common.Address{}, Version: 1888, OldVal: "str", NewVal: []byte{128, 0xff}},
		str:       "ChangeLogType(0){Account: Lemo888888888888888888888888888888888888, Version: 1888, OldVal: str, NewVal: [128 255]}",
		json:      `{"type":"0","address":"Lemo888888888888888888888888888888888888","version":"1888","newValue":"0x8280ff","extra":""}`,
		hash:      "0xafee1464750a367208437ec1061ddbf793b2120588445389610d8143ad5d1035",
		rlp:       "0xdd809400000000000000000000000000000000000000008207608280ffc0",
		decodeErr: ErrUnknownChangeLogType,
	})

	// 1 output: empty interface{}, extra: testStruct
	registerCustomType(10001)
	structData := testStruct{11, "abc"}
	tests = append(tests, testCustomTypeConfig{
		input:   &ChangeLog{LogType: ChangeLogType(10001), Extra: structData},
		str:     "ChangeLogType(10001){Account: Lemo888888888888888888888888888888888888, Version: 0, Extra: {11 abc}}",
		json:    `{"type":"10001","address":"Lemo888888888888888888888888888888888888","version":"0","newValue":"","extra":"0xc50b83616263"}`,
		hash:    "0xc2f5e2f55f2d6be2ef0e6b2f826bd2c1d9fcb4c2cd88a5b39677eb7564ff5629",
		rlp:     "0xe082271194000000000000000000000000000000000000000080c0c50b83616263",
		decoded: "ChangeLogType(10001){Account: Lemo888888888888888888888888888888888888, Version: 0, Extra: {11 abc}}",
	})

	// 2 empty ChangeLog
	tests = append(tests, testCustomTypeConfig{
		input:     &ChangeLog{},
		str:       "ChangeLogType(0){Account: Lemo888888888888888888888888888888888888, Version: 0}",
		json:      `{"type":"0","address":"Lemo888888888888888888888888888888888888","version":"0","newValue":"","extra":""}`,
		hash:      "0xae191db75787cf40e7a29c1287c1e65ab4b24e8a9bc7c7037e49575241943f65",
		rlp:       "0xd98094000000000000000000000000000000000000000080c0c0",
		decodeErr: ErrUnknownChangeLogType,
	})
	return tests
}

func TestChangeLog_String(t *testing.T) {
	tests := getCustomTypeData()
	for i, test := range tests {
		assert.Equal(t, test.str, test.input.String(), "index=%d %s", i, test.input)
	}
}

func TestChangeLog_Hash(t *testing.T) {
	tests := getCustomTypeData()
	for i, test := range tests {
		assert.Equal(t, test.hash, test.input.Hash().Hex(), "index=%d %s", i, test.input)
	}
}

func TestChangeLog_EncodeRLP_DecodeRLP(t *testing.T) {
	tests := getCustomTypeData()
	for i, test := range tests {
		enc, err := rlp.EncodeToBytes(test.input)
		assert.NoError(t, err, "index=%d %s", i, test.input)
		assert.Equal(t, test.rlp, hexutil.Encode(enc), "index=%d %s", i, test.input)

		decodeResult := new(ChangeLog)
		err = rlp.DecodeBytes(enc, decodeResult)
		assert.Equal(t, test.decodeErr, err, "index=%d %s", i, test.input)
		if test.decodeErr == nil {
			assert.NoError(t, err, "index=%d %s", i, test.input)
			assert.Equal(t, test.decoded, decodeResult.String(), "index=%d %s", i, test.input)
		}
	}
	// TODO build some rlp codes to test the decode error
}

func TestChangeLog_MarshalJSON_UnmarshalJSON(t *testing.T) {
	tests := getCustomTypeData()
	for i, test := range tests {
		json, err := test.input.MarshalJSON()
		assert.NoError(t, err, "index=%d %s", i, test.input)
		assert.Equal(t, test.json, string(json), "index=%d %s", i, test.input)

		decodeResult := new(ChangeLog)
		err = decodeResult.UnmarshalJSON(json)
		assert.Equal(t, test.decodeErr, err, "index=%d %s", i, test.input)
		if test.decodeErr == nil {
			test.input.OldVal = nil
			assert.Equal(t, test.input, decodeResult, "index=%d %s", i, test.input)
		}
	}
}

func TestChangeLog_Undo(t *testing.T) {
	processor := &testProcessor{}

	// unknown type
	log := processor.createChangeLog(ChangeLogType(0))
	err := log.Undo(processor)
	assert.Equal(t, ErrUnknownChangeLogType, err)

	// custom type
	registerCustomType(10002)
	log = processor.createChangeLog(ChangeLogType(10002))
	err = log.Undo(processor)
	assert.NoError(t, err)
}

func TestChangeLog_Redo(t *testing.T) {
	processor := &testProcessor{}

	// unknown type
	log := &ChangeLog{LogType: ChangeLogType(0), Address: common.Address{}, Version: 1}
	err := log.Redo(processor)
	assert.Equal(t, ErrUnknownChangeLogType, err)

	// custom type
	registerCustomType(10003)
	log = processor.createChangeLog(ChangeLogType(10003))
	err = log.Redo(processor)
	assert.NoError(t, err)
}

// TODO
func TestChangeLogSlice(t *testing.T) {

}
