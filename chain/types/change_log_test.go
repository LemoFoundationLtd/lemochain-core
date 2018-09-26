package types

import (
	"errors"
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

func (f *testAccount) GetAddress() common.Address                          { return f.AccountData.Address }
func (f *testAccount) GetBalance() *big.Int                                { return f.AccountData.Balance }
func (f *testAccount) SetBalance(balance *big.Int)                         { f.AccountData.Balance = balance }
func (f *testAccount) GetVersion() uint32                                  { return f.AccountData.Version }
func (f *testAccount) SetVersion(version uint32)                           { f.AccountData.Version = version }
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
func (f *testAccount) IsEmpty() bool                                       { return f.AccountData.Version == 0 }

type testProcessor struct {
	Accounts map[common.Address]*testAccount
}

var ErrNoAccount = errors.New("no account from address")

func (p *testProcessor) GetAccount(addr common.Address) (AccountAccessor, error) {
	account, ok := p.Accounts[addr]
	if !ok {
		return nil, ErrNoAccount
	}
	return account, nil
}

func (p *testProcessor) createAccount(version uint32) *testAccount {
	if p.Accounts == nil {
		p.Accounts = make(map[common.Address]*testAccount)
	}
	index := len(p.Accounts) + 1
	address := common.BigToAddress(big.NewInt(int64(index)))
	account := &testAccount{
		AccountData: AccountData{
			Address: address,
			Balance: big.NewInt(100),
			Version: version,
		},
	}
	p.Accounts[address] = account
	return account
}

func (p *testProcessor) PushEvent(event *Event) {}
func (p *testProcessor) PopEvent() error        { return nil }

type testCustomTypeConfig struct {
	input     *ChangeLog
	str       string
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
		str:       "ChangeLogType(0): 0x0000000000000000000000000000000000000000 1888 str [128 255] <nil>",
		hash:      "0xafee1464750a367208437ec1061ddbf793b2120588445389610d8143ad5d1035",
		rlp:       "0xdd809400000000000000000000000000000000000000008207608280ffc0",
		decodeErr: ErrUnknownChangeLogType,
	})

	// 1 output: empty interface{}, extra: testStruct
	registerCustomType(10001)
	structData := testStruct{11, "abc"}
	tests = append(tests, testCustomTypeConfig{
		input:   &ChangeLog{LogType: ChangeLogType(10001), Extra: structData},
		str:     "ChangeLogType(10001): 0x0000000000000000000000000000000000000000 0 <nil> <nil> {11 abc}",
		hash:    "0xc2f5e2f55f2d6be2ef0e6b2f826bd2c1d9fcb4c2cd88a5b39677eb7564ff5629",
		rlp:     "0xe082271194000000000000000000000000000000000000000080c0c50b83616263",
		decoded: "ChangeLogType(10001): 0x0000000000000000000000000000000000000000 0 <nil> <nil> {11 abc}",
	})

	// 2 empty ChangeLog
	tests = append(tests, testCustomTypeConfig{
		input:     &ChangeLog{},
		str:       "ChangeLogType(0): 0x0000000000000000000000000000000000000000 0 <nil> <nil> <nil>",
		hash:      "0xae191db75787cf40e7a29c1287c1e65ab4b24e8a9bc7c7037e49575241943f65",
		rlp:       "0xd98094000000000000000000000000000000000000000080c0c0",
		decodeErr: ErrUnknownChangeLogType,
	})
	return tests
}

func TestChangeLog_String(t *testing.T) {
	tests := getCustomTypeData()
	for i, test := range tests {
		if test.input.String() != test.str {
			t.Errorf("test %d. want str: %s, got: %s", i, test.str, test.input.String())
		}
	}
}

func TestChangeLog_Hash(t *testing.T) {
	tests := getCustomTypeData()
	for i, test := range tests {
		if test.input.Hash().Hex() != test.hash {
			t.Errorf("test %d. want hash: %s, got: %s", i, test.hash, test.input.Hash().Hex())
		}
	}
}

func TestChangeLog_EncodeRLP_DecodeRLP(t *testing.T) {
	tests := getCustomTypeData()
	for i, test := range tests {
		enc, err := rlp.EncodeToBytes(test.input)
		if err != nil {
			t.Errorf("test %d. rlp encode error: %s", i, err)
		} else if hexutil.Encode(enc) != test.rlp {
			t.Errorf("test %d. want rlp: %s, got: %s", i, test.rlp, hexutil.Encode(enc))
		} else {
			decodeResult := new(ChangeLog)
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
	// TODO build some rlp codes to test the decode error
}

func createChangeLog(processor *testProcessor, accountVersion uint32, logType ChangeLogType, logVersion uint32) *ChangeLog {
	account := processor.createAccount(accountVersion)
	return &ChangeLog{LogType: logType, Address: account.GetAddress(), Version: logVersion}
}

func removeAddress(log *ChangeLog) *ChangeLog {
	log.Address = common.Address{}
	return log
}

func TestChangeLog_Undo(t *testing.T) {
	processor := &testProcessor{}
	registerCustomType(10002)

	tests := []struct {
		input      *ChangeLog
		undoErr    error
		afterCheck func(AccountAccessor)
	}{
		// 0 custom type
		{
			input:   createChangeLog(processor, 0, ChangeLogType(0), 1),
			undoErr: ErrUnknownChangeLogType,
		},
		// 1 lower version
		{
			input:   createChangeLog(processor, 2, ChangeLogType(10002), 1),
			undoErr: ErrWrongChangeLogVersion,
		},
		// 2 same version
		{
			input: createChangeLog(processor, 1, ChangeLogType(10002), 1),
			afterCheck: func(accessor AccountAccessor) {
				assert.Equal(t, uint32(0), accessor.GetVersion())
			},
		},
		// 3 higher version
		{
			input:   createChangeLog(processor, 2, ChangeLogType(10002), 1),
			undoErr: ErrWrongChangeLogVersion,
		},
		// 4 no account
		{
			input:   removeAddress(createChangeLog(processor, 2, ChangeLogType(10002), 1)),
			undoErr: ErrNoAccount,
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
	registerCustomType(10003)

	tests := []struct {
		input      *ChangeLog
		redoErr    error
		afterCheck func(AccountAccessor)
	}{
		// 0 custom type
		{
			input:   &ChangeLog{LogType: ChangeLogType(0), Address: common.Address{}, Version: 1},
			redoErr: ErrUnknownChangeLogType,
		},
		// 1 lower version
		{
			input:   createChangeLog(processor, 2, ChangeLogType(10003), 1),
			redoErr: ErrAlreadyRedo,
		},
		// 2 same version
		{
			input:   createChangeLog(processor, 1, ChangeLogType(10003), 1),
			redoErr: ErrAlreadyRedo,
		},
		// 3 correct version
		{
			input: createChangeLog(processor, 0, ChangeLogType(10003), 1),
			afterCheck: func(accessor AccountAccessor) {
				assert.Equal(t, uint32(1), accessor.GetVersion())
			},
		},
		// 3 higher version
		{
			input:   createChangeLog(processor, 0, ChangeLogType(10003), 2),
			redoErr: ErrWrongChangeLogVersion,
		},
		// 4 no account
		{
			input:   removeAddress(createChangeLog(processor, 0, ChangeLogType(10003), 1)),
			redoErr: ErrNoAccount,
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

// TODO
func TestChangeLogSlice(t *testing.T) {

}
