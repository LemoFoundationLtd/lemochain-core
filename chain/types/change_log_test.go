package types

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

type testAccount struct {
	AccountData
}

func (f *testAccount) SetSingers(signers Signers) error {
	panic("implement me")
}

func (f *testAccount) GetSigners() Signers {
	panic("implement me")
}

func (f *testAccount) GetCandidate() Profile {
	panic("implement me")
}

func (f *testAccount) SetCandidate(profile Profile) {
	panic("implement me")
}

func (f *testAccount) GetCandidateState(key string) string {
	panic("implement me")
}

func (f *testAccount) SetCandidateState(key string, val string) {
	panic("implement me")
}

func (f *testAccount) GetAssetCodeRoot() common.Hash {
	panic("implement me")
}

func (f *testAccount) SetAssetCodeRoot(root common.Hash) {
	panic("implement me")
}

func (f *testAccount) GetAssetIdRoot() common.Hash {
	panic("implement me")
}

func (f *testAccount) SetAssetIdRoot(root common.Hash) {
	panic("implement me")
}

func (f *testAccount) GetEquityRoot() common.Hash {
	panic("implement me")
}

func (f *testAccount) SetEquityRoot(root common.Hash) {
	panic("implement me")
}

func (f *testAccount) GetAssetCode(code common.Hash) (*Asset, error) {
	panic("implement me")
}

func (f *testAccount) SetAssetCode(code common.Hash, asset *Asset) error {
	panic("implement me")
}

func (f *testAccount) GetAssetCodeTotalSupply(code common.Hash) (*big.Int, error) {
	panic("implement me")
}

func (f *testAccount) SetAssetCodeTotalSupply(code common.Hash, val *big.Int) error {
	panic("implement me")
}

func (f *testAccount) GetAssetCodeState(code common.Hash, key string) (string, error) {
	panic("implement me")
}

func (f *testAccount) SetAssetCodeState(code common.Hash, key string, val string) error {
	panic("implement me")
}

func (f *testAccount) GetAssetIdState(id common.Hash) (string, error) {
	panic("implement me")
}

func (f *testAccount) SetAssetIdState(id common.Hash, data string) error {
	panic("implement me")
}

func (f *testAccount) GetEquityState(id common.Hash) (*AssetEquity, error) {
	panic("implement me")
}

func (f *testAccount) SetEquityState(id common.Hash, equity *AssetEquity) error {
	panic("implement me")
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

func (f *testAccount) GetCandidateProfile() Profile {
	return f.GetCandidateProfile()
}

func (f *testAccount) SetCandidateProfile(profile Profile) {
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
func (f *testAccount) MarshalISON() ([]byte, error) {
	return f.AccountData.MarshalJSON()
}
func (f *testAccount) GetVersion(logType ChangeLogType) uint32 {
	return f.AccountData.NewestRecords[logType].Version
}
func (f *testAccount) GetNextVersion(logType ChangeLogType) uint32 {
	panic("implement me")
}
func (f *testAccount) PushEvent(event *Event) {}
func (f *testAccount) PopEvent() error        { return nil }
func (f *testAccount) GetEvents() []*Event {
	panic("implement me")
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
		json:      `{"type":"0","address":"Lemo888888888888888888888888888888888888","version":"1888","newValue":"gP8=","extra":null}`,
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
		json:    `{"type":"10001","address":"Lemo888888888888888888888888888888888888","version":"0","newValue":null,"extra":{"A":11,"B":"abc"}}`,
		hash:    "0xc2f5e2f55f2d6be2ef0e6b2f826bd2c1d9fcb4c2cd88a5b39677eb7564ff5629",
		rlp:     "0xe082271194000000000000000000000000000000000000000080c0c50b83616263",
		decoded: "ChangeLogType(10001){Account: Lemo888888888888888888888888888888888888, Version: 0, Extra: {11 abc}}",
	})

	// 2 empty ChangeLog
	tests = append(tests, testCustomTypeConfig{
		input:     &ChangeLog{},
		str:       "ChangeLogType(0){Account: Lemo888888888888888888888888888888888888, Version: 0}",
		json:      `{"type":"0","address":"Lemo888888888888888888888888888888888888","version":"0","newValue":null,"extra":null}`,
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
	}
}

type testMarshalChangeLog struct {
	changeLog *ChangeLog
	json      string
}

type testProfileChangeLogExtra struct {
	UUID common.Hash
	Key  string
}

// testMarshal 生成changeLog marshal的数据
func testMarshal() []*testMarshalChangeLog {
	tests := make([]*testMarshalChangeLog, 0, 19)
	// balanceLog
	test01 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(1), Address: common.HexToAddress("0x111"), Version: 0, NewVal: *big.NewInt(999)},
		json:      `{"type":"1","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":"999","extra":null}`,
	}
	// StorageLog
	test02 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(2), Address: common.HexToAddress("0x111"), Version: 0, NewVal: []byte{1, 2, 3, 4}, Extra: common.HexToHash("0x583ecc2aa7344eb26d2a78c66989f9fa2ef6b207d3a6d3a91bd5f71747a5cbad")},
		json:      `{"type":"2","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":"AQIDBA==","extra":"0x583ecc2aa7344eb26d2a78c66989f9fa2ef6b207d3a6d3a91bd5f71747a5cbad"}`,
	}
	// StorageRootLog
	test03 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(3), Address: common.HexToAddress("0x111"), Version: 0, NewVal: common.HexToHash("0x583ecc2aa7344eb26d2a78c66989f9fa2ef6b207d3a6d3a91bd5f71747a5cbad")},
		json:      `{"type":"3","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":"0x583ecc2aa7344eb26d2a78c66989f9fa2ef6b207d3a6d3a91bd5f71747a5cbad","extra":null}`,
	}

	// AssetCodeLog
	profile := make(Profile)
	profile["description"] = "test asset"
	profile["freeze"] = "false"
	newValue := &Asset{
		Category:        1,
		IsDivisible:     false,
		AssetCode:       common.HexToHash("0x0f061d7e4a210df4231e2b6f5f87ac1c35cdc9cb50f31269f3dc0cd741101db6"),
		Decimal:         18,
		TotalSupply:     big.NewInt(1000),
		IsReplenishable: false,
		Issuer:          common.HexToAddress("0x222"),
		Profile:         profile,
	}
	test04 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(4), Address: common.HexToAddress("0x111"), Version: 0, NewVal: newValue, Extra: common.HexToHash("0x583ecc2aa7344eb26d2a78c66989f9fa2ef6b207d3a6d3a91bd5f71747a5cbad")},
		json:      `{"type":"4","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":{"category":"1","isDivisible":false,"assetCode":"0x0f061d7e4a210df4231e2b6f5f87ac1c35cdc9cb50f31269f3dc0cd741101db6","decimal":"18","totalSupply":"1000","isReplenishable":false,"issuer":"Lemo888888888888888888888888888888889YS2","profile":{"description":"test asset","freeze":"false"}},"extra":"0x583ecc2aa7344eb26d2a78c66989f9fa2ef6b207d3a6d3a91bd5f71747a5cbad"}`,
	}
	// AssetCodeStateLog
	extra := &testProfileChangeLogExtra{
		UUID: common.HexToHash("0x9999"),
		Key:  "aaabbb",
	}
	test05 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(5), Address: common.HexToAddress("0x111"), Version: 0, NewVal: "aaabbb", Extra: extra},
		json:      `{"type":"5","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":"aaabbb","extra":{"UUID":"0x0000000000000000000000000000000000000000000000000000000000009999","Key":"aaabbb"}}`,
	}
	// AssetCodeRootLog
	test06 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(6), Address: common.HexToAddress("0x111"), Version: 0, NewVal: common.HexToHash("0x888")},
		json:      `{"type":"6","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":"0x0000000000000000000000000000000000000000000000000000000000000888","extra":null}`,
	}
	// AssetCodeTotalSupplyLog
	test07 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(7), Address: common.HexToAddress("0x111"), Version: 0, NewVal: *(big.NewInt(800000)), Extra: common.HexToHash("0x888")},
		json:      `{"type":"7","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":"800000","extra":"0x0000000000000000000000000000000000000000000000000000000000000888"}`,
	}
	// AssetIdLog
	test08 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(8), Address: common.HexToAddress("0x111"), Version: 0, NewVal: "asset code log", Extra: common.HexToHash("0x777")},
		json:      `{"type":"8","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":"asset code log","extra":"0x0000000000000000000000000000000000000000000000000000000000000777"}`,
	}
	// AssetIdRootLog
	test09 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(9), Address: common.HexToAddress("0x111"), Version: 0, NewVal: common.HexToHash("0x777")},
		json:      `{"type":"9","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":"0x0000000000000000000000000000000000000000000000000000000000000777","extra":null}`,
	}
	// EquityLog
	equity := &AssetEquity{
		AssetCode: common.HexToHash("0x11"),
		AssetId:   common.HexToHash("0x11"),
		Equity:    big.NewInt(5000),
	}
	test10 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(10), Address: common.HexToAddress("0x111"), Version: 0, NewVal: equity, Extra: common.HexToHash("0x777")},
		json:      `{"type":"10","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":{"assetCode":"0x0000000000000000000000000000000000000000000000000000000000000011","assetId":"0x0000000000000000000000000000000000000000000000000000000000000011","equity":"5000"},"extra":"0x0000000000000000000000000000000000000000000000000000000000000777"}`,
	}
	// EquityRootLog
	test11 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(11), Address: common.HexToAddress("0x111"), Version: 0, NewVal: common.HexToHash("0x111")},
		json:      `{"type":"11","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":"0x0000000000000000000000000000000000000000000000000000000000000111","extra":null}`,
	}
	// CandidateLog
	candidateProfile := make(Profile)
	candidateProfile[CandidateKeyPort] = "8001"
	candidateProfile[CandidateKeyHost] = "www.lemochain.com"
	test12 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(12), Address: common.HexToAddress("0x111"), Version: 0, NewVal: &candidateProfile},
		json:      `{"type":"12","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":{"host":"www.lemochain.com","port":"8001"},"extra":null}`,
	}
	// CandidateStateLog
	test13 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(13), Address: common.HexToAddress("0x111"), Version: 0, NewVal: "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG", Extra: "incommeAddress"},
		json:      `{"type":"13","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":"Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG","extra":"incommeAddress"}`,
	}
	// CodeLog
	test14 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(14), Address: common.HexToAddress("0x111"), Version: 0, NewVal: Code{1, 2, 3, 4, 5}},
		json:      `{"type":"14","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":"AQIDBAU=","extra":null}`,
	}
	// AddEventLog
	test15 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(15), Address: common.HexToAddress("0x111"), Version: 0, NewVal: &Event{
			Address: common.HexToAddress("0x333"),
			Topics:  []common.Hash{common.HexToHash("0x11"), common.HexToHash("0x22")},
			Data:    []byte{1, 2, 3, 4, 5, 6, 7},
			TxHash:  common.HexToHash("0x444"),
			TxIndex: 1,
			Index:   2,
			Removed: false,
		}},
		json: `{"type":"15","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":{"address":"Lemo88888888888888888888888888888888DY7T","topics":["0x0000000000000000000000000000000000000000000000000000000000000011","0x0000000000000000000000000000000000000000000000000000000000000022"],"data":"0x01020304050607","transactionHash":"0x0000000000000000000000000000000000000000000000000000000000000444","transactionIndex":"1","eventIndex":"2","removed":false},"extra":null}`,
	}
	// SuicideLog
	test16 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(16), Address: common.HexToAddress("0x111"), Version: 0},
		json:      `{"type":"16","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":null,"extra":null}`,
	}
	// VoteForLog
	test17 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(17), Address: common.HexToAddress("0x111"), Version: 0, NewVal: common.HexToAddress("0x122")},
		json:      `{"type":"17","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":"Lemo8888888888888888888888888888888867TQ","extra":null}`,
	}
	// VotesLog
	test18 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(18), Address: common.HexToAddress("0x111"), Version: 0, NewVal: *(big.NewInt(9999))},
		json:      `{"type":"18","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":"9999","extra":null}`,
	}
	// SignerLog
	signers := make(Signers, 0)
	signAccount01 := SignAccount{Address: common.HexToAddress("0x456"), Weight: 20}
	signAccount02 := SignAccount{Address: common.HexToAddress("0x666"), Weight: 60}
	signers = append(signers, signAccount01, signAccount02)
	test19 := &testMarshalChangeLog{
		changeLog: &ChangeLog{LogType: ChangeLogType(19), Address: common.HexToAddress("0x111"), Version: 0, NewVal: signers},
		json:      `{"type":"19","address":"Lemo888888888888888888888888888888885ZCK","version":"0","newValue":[{"address":"Lemo88888888888888888888888888888888K6FC","weight":"20"},{"address":"Lemo88888888888888888888888888888888WTDP","weight":"60"}],"extra":null}`,
	}
	return append(tests, test01, test02, test03, test04, test05, test06, test07, test08, test09, test10,
		test11, test12, test13, test14, test15, test16, test17, test18, test19)
}

func TestChangeLog_MarshalJSON(t *testing.T) {
	tests := testMarshal()
	for _, test := range tests {
		js, err := test.changeLog.MarshalJSON()
		assert.NoError(t, err)
		assert.Equal(t, test.json, string(js))
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
