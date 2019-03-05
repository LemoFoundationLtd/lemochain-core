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
	account.storage.cached = map[common.Hash][]byte{
		common.HexToHash("0xaaa"): {45, 67},
	}

	val, _ := rlp.EncodeToBytes(&types.Asset{
		Category:        1,
		IsDivisible:     true,
		AssetCode:       common.HexToHash("0x11"),
		Decimals:        0,
		TotalSupply:     new(big.Int).SetInt64(1),
		IsReplenishable: false,
		Issuer:          common.HexToAddress("0x22"),
		Profile: map[string]string{
			"lemokey": "lemoval",
		},
	})
	account.assetCode.cached = map[common.Hash][]byte{
		common.HexToHash("0x33"): val,
	}

	account.assetId.cached = map[common.Hash][]byte{
		common.HexToHash("0x33"): []byte("old"),
	}

	val, _ = rlp.EncodeToBytes(&types.AssetEquity{
		AssetCode: common.HexToHash("0x22"),
		AssetId:   common.HexToHash("0x33"),
		Equity:    new(big.Int).SetInt64(100),
	})
	account.equity.cached = map[common.Hash][]byte{
		common.HexToHash("0x33"): val,
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

	// storage root
	tests = append(tests, testLogConfig{
		input:      NewStorageRootLog(processor, processor.createAccount(BalanceLog, 0), common.HexToHash("0x01"), common.HexToHash("0x02")),
		isValuable: true,
		str:        "StorageRootLog{Account: Lemo8888888888888888888888888888888883WD, Version: 1, OldVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1], NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
		hash:       "0x9b40bebbdc2758db2b756f64eed632f0484e48faa3794adbc503cc55ac5ac334",
		rlp:        "0xf8390394000000000000000000000000000000000000000501a00000000000000000000000000000000000000000000000000000000000000002c0",
		decoded:    "StorageRootLog{Account: Lemo8888888888888888888888888888888883WD, Version: 1, NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
	})

	// 4 CodeLog
	tests = append(tests, testLogConfig{
		input:      NewCodeLog(processor, processor.createAccount(CodeLog, 0), []byte{0x12, 0x34}),
		isValuable: true,
		str:        "CodeLog{Account: Lemo88888888888888888888888888888888849A, Version: 1, NewVal: 0x1234}",
		hash:       "0x68d9b4a1e48a3f52774beb565d8123281421a4bb3519b30ce7e7cbb24d0dd308",
		rlp:        "0xdb0e94000000000000000000000000000000000000000601821234c0",
	})

	// 5 CodeLog
	tests = append(tests, testLogConfig{
		input:      NewCodeLog(processor, processor.createAccount(CodeLog, 0), []byte{}),
		isValuable: false,
		str:        "CodeLog{Account: Lemo8888888888888888888888888888888884N7, Version: 1, NewVal: }",
		hash:       "0x73cefd2f2312068337e16f784f8cb06ea132b701b8d614c30aff7f5867d2d6f3",
		rlp:        "0xd90e9400000000000000000000000000000000000000070180c0",
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
		str:        "AddEventLog{Account: Lemo888888888888888888888888888888888534, Version: 1, NewVal: event: 0000000000000000000000000000000000000aaa [0000000000000000000000000000000000000000000000000000000000000bbb 0000000000000000000000000000000000000000000000000000000000000ccc] 8000 0000000000000000000000000000000000000000000000000000000000000000 0 0000000000000000000000000000000000000000000000000000000000000000 0}",
		hash:       "0x9819dd922773475d634fbb6d775e464bd07e3a4217982b78c3f1230dac488de4",
		rlp:        "0xf8760f94000000000000000000000000000000000000000801f85c940000000000000000000000000000000000000aaaf842a00000000000000000000000000000000000000000000000000000000000000bbba00000000000000000000000000000000000000000000000000000000000000ccc828000c0",
	})
	// It is not possible to set NewVal in AddEventLog to nil. We can't test is because we can't rlp encode a (*types.Event)(nil)

	// 7 SuicideLog
	tests = append(tests, testLogConfig{
		input:      NewSuicideLog(processor, processor.createAccount(SuicideLog, 0)),
		isValuable: true,
		str:        "SuicideLog{Account: Lemo8888888888888888888888888888888885CZ, Version: 1, OldVal: {Address: Lemo888888888888888888888888888888888888, Balance: 100, VoteFor: Lemo888888888888888888888888888888888888}}",
		hash:       "0x7ee159c6b3a060c673a26e97e15438a5a16d7b8b21ebcff99f59f2455a8b6147",
		rlp:        "0xd91094000000000000000000000000000000000000000901c0c0",
		decoded:    "SuicideLog{Account: Lemo8888888888888888888888888888888885CZ, Version: 1}",
	})
	// 8 SuicideLog
	tests = append(tests, testLogConfig{
		input:      NewSuicideLog(processor, processor.createEmptyAccount()),
		isValuable: false,
		str:        "SuicideLog{Account: Lemo8888888888888888888888888888888885RT, Version: 1, OldVal: {Address: Lemo888888888888888888888888888888888888, Balance: 0, VoteFor: Lemo888888888888888888888888888888888888}}",
		hash:       "0x6d7a3da06cd453572ac14e54994b53dcbc907299ed32760b3eeb1f456099cfbd",
		rlp:        "0xd91094000000000000000000000000000000000000000a01c0c0",
		decoded:    "SuicideLog{Account: Lemo8888888888888888888888888888888885RT, Version: 1}",
	})

	// 9 VotesLog
	tests = append(tests, testLogConfig{
		input:      NewVotesLog(processor, processor.createAccount(VotesLog, 0), big.NewInt(1000)),
		isValuable: true,
		str:        "VotesLog{Account: Lemo88888888888888888888888888888888866Q, Version: 1, OldVal: 200, NewVal: 1000}",
		hash:       "0xfc049109e9882cbd4ca27b628958e17242eef720de85cc807a2a5f63313b492f",
		rlp:        "0xdb1294000000000000000000000000000000000000000b018203e8c0",
		decoded:    "VotesLog{Account: Lemo88888888888888888888888888888888866Q, Version: 1, NewVal: 1000}",
	})
	// 10 VotesLog
	tests = append(tests, testLogConfig{
		input:      NewVotesLog(processor, processor.createAccount(VotesLog, 0), big.NewInt(200)),
		isValuable: false,
		str:        "VotesLog{Account: Lemo8888888888888888888888888888888886HK, Version: 1, OldVal: 200, NewVal: 200}",
		hash:       "0xe555e6c1771827e5edc0e25ac953e5a2b7c4e7fba2c2976a5b608b5133daf646",
		rlp:        "0xda1294000000000000000000000000000000000000000c0181c8c0",
		decoded:    "VotesLog{Account: Lemo8888888888888888888888888888888886HK, Version: 1, NewVal: 200}",
	})

	// 11 VoteForLog
	tests = append(tests, testLogConfig{
		input:      NewVoteForLog(processor, processor.createAccount(VoteForLog, 0), common.HexToAddress("0x0002")),
		isValuable: true,
		str:        "VoteForLog{Account: Lemo8888888888888888888888888888888886YG, Version: 1, OldVal: Lemo8888888888888888888888888888888888BW, NewVal: Lemo8888888888888888888888888888888888QR}",
		hash:       "0xe377afa192cccb5db82fa76173b993909c11d8eb0e5dd006ddc538e8338ff480",
		rlp:        "0xed1194000000000000000000000000000000000000000d01940000000000000000000000000000000000000002c0",
		decoded:    "VoteForLog{Account: Lemo8888888888888888888888888888888886YG, Version: 1, NewVal: Lemo8888888888888888888888888888888888QR}",
	})
	// 12 VoteForLog
	tests = append(tests, testLogConfig{
		input:      NewVoteForLog(processor, processor.createAccount(VoteForLog, 0), common.HexToAddress("0x0001")),
		isValuable: false,
		str:        "VoteForLog{Account: Lemo8888888888888888888888888888888887AC, Version: 1, OldVal: Lemo8888888888888888888888888888888888BW, NewVal: Lemo8888888888888888888888888888888888BW}",
		hash:       "0x78cfaf7515a698c7399c8699f214294ab23392afbdeac14715ca967f7171a553",
		rlp:        "0xed1194000000000000000000000000000000000000000e01940000000000000000000000000000000000000001c0",
		decoded:    "VoteForLog{Account: Lemo8888888888888888888888888888888887AC, Version: 1, NewVal: Lemo8888888888888888888888888888888888BW}",
	})

	// 13 AssetCode
	log, _ := NewAssetCodeLog(processor, processor.createAccount(AssetCodeLog, 0), common.HexToHash("0x33"), new(types.Asset))
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "AssetCodeLog{Account: Lemo8888888888888888888888888888888887P9, Version: 1, OldVal: {Category: 1, IsDivisible: true, AssetCode: 0x0000000000000000000000000000000000000000000000000000000000000011, Decimals: 0, Issuer: Lemo888888888888888888888888888888888FY4, IsReplenishable: false, TotalSupply: 1, Profiles: {lemokey => lemoval}}, NewVal: {Category: 0, IsDivisible: false, AssetCode: 0x0000000000000000000000000000000000000000000000000000000000000000, Decimals: 0, Issuer: Lemo888888888888888888888888888888888888, IsReplenishable: false, TotalSupply: 0, Profile: []}, Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 51]}",
		hash:       "0x18cff028352d5e0dcf7f0e54b6700de3792959f4dece767acf64e5a2617d44cd",
		rlp:        "0xf8760494000000000000000000000000000000000000000f01f83c8080a00000000000000000000000000000000000000000000000000000000000000000808080940000000000000000000000000000000000000000c0a00000000000000000000000000000000000000000000000000000000000000033",
		decoded:    "AssetCodeLog{Account: Lemo8888888888888888888888888888888887P9, Version: 1, NewVal: {Category: 0, IsDivisible: false, AssetCode: 0x0000000000000000000000000000000000000000000000000000000000000000, Decimals: 0, Issuer: Lemo888888888888888888888888888888888888, IsReplenishable: false, TotalSupply: 0, Profile: []}, Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 51]}",
	})

	log, _ = NewAssetCodeStateLog(processor, processor.createAccount(AssetCodeStateLog, 0), common.HexToHash("0x33"), "key", "new")
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "AssetCodeStateLog{Account: Lemo888888888888888888888888888888888246, Version: 1, OldVal: , NewVal: new, Extra: {UUID: 0x0000000000000000000000000000000000000000000000000000000000000033, Key: key}}",
		hash:       "0xca473e00e10bced9d57451b9b7bf2a6d2af434f62a570479442e8352c463ba90",
		rlp:        "0xf8410594000000000000000000000000000000000000001001836e6577e5a00000000000000000000000000000000000000000000000000000000000000033836b6579",
		decoded:    "AssetCodeStateLog{Account: Lemo888888888888888888888888888888888246, Version: 1, NewVal: new, Extra: {UUID: 0x0000000000000000000000000000000000000000000000000000000000000033, Key: key}}",
	})

	log, _ = NewAssetCodeRootLog(processor, processor.createAccount(AssetCodeRootLog, 0), common.HexToHash("0x01"), common.HexToHash("0x02"))
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "AssetCodeRootLog{Account: Lemo8888888888888888888888888888888882F3, Version: 1, OldVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1], NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
		hash:       "0x82174d68a2a7b0c1d8a351d58842cbf66bcc5860482155992dc6fadbd87ad584",
		rlp:        "0xf8390694000000000000000000000000000000000000001101a00000000000000000000000000000000000000000000000000000000000000002c0",
		decoded:    "AssetCodeRootLog{Account: Lemo8888888888888888888888888888888882F3, Version: 1, NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
	})

	log, _ = NewAssetIdLog(processor, processor.createAccount(AssetIdLog, 0), common.HexToHash("0x33"), "new")
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "AssetIdLog{Account: Lemo8888888888888888888888888888888882SY, Version: 1, OldVal: old, NewVal: new, Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 51]}",
		hash:       "0xd293ba6abe604251f8a9b33feeca4b3c730ac526ac90e2e2a80c56bbf3ae83dc",
		rlp:        "0xf83c0894000000000000000000000000000000000000001201836e6577a00000000000000000000000000000000000000000000000000000000000000033",
		decoded:    "AssetIdLog{Account: Lemo8888888888888888888888888888888882SY, Version: 1, NewVal: new, Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 51]}",
	})

	log, _ = NewAssetIdRootLog(processor, processor.createAccount(AssetIdRootLog, 0), common.HexToHash("0x01"), common.HexToHash("0x02"))
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "AssetIdRootLog{Account: Lemo88888888888888888888888888888888897S, Version: 1, OldVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1], NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
		hash:       "0x3a536384f429704678fc645e3efd73d16a0b902f35b211c142b37811c1f258c0",
		rlp:        "0xf8390994000000000000000000000000000000000000001301a00000000000000000000000000000000000000000000000000000000000000002c0",
		decoded:    "AssetIdRootLog{Account: Lemo88888888888888888888888888888888897S, Version: 1, NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
	})

	log, _ = NewEquityLog(processor, processor.createAccount(EquityLog, 0), common.HexToHash("0x33"), new(types.AssetEquity))
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "EquityLog{Account: Lemo8888888888888888888888888888888889JP, Version: 1, OldVal: {AssetCode: 0x0000000000000000000000000000000000000000000000000000000000000022, AssetId: 0x0000000000000000000000000000000000000000000000000000000000000033, Equity: 100}, NewVal: {AssetCode: 0x0000000000000000000000000000000000000000000000000000000000000000, AssetId: 0x0000000000000000000000000000000000000000000000000000000000000000, Equity: 0}, Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 51]}",
		hash:       "0xab78c9401d21445580b71f94ce2976b2bcdf85ac9d7f556d02345018a8ca151f",
		rlp:        "0xf87d0a94000000000000000000000000000000000000001401f843a00000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000000080a00000000000000000000000000000000000000000000000000000000000000033",
		decoded:    "EquityLog{Account: Lemo8888888888888888888888888888888889JP, Version: 1, NewVal: {AssetCode: 0x0000000000000000000000000000000000000000000000000000000000000000, AssetId: 0x0000000000000000000000000000000000000000000000000000000000000000, Equity: 0}, Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 51]}",
	})

	log, _ = NewEquityRootLog(processor, processor.createAccount(EquityRootLog, 0), common.HexToHash("0x01"), common.HexToHash("0x02"))
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "EquityRootLog{Account: Lemo8888888888888888888888888888888889ZJ, Version: 1, OldVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1], NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
		hash:       "0xf23e2a21d65ab7b4c5211f49d26d2665ff8d0cfe2d6897fb3ab874ae285205c1",
		rlp:        "0xf8390b94000000000000000000000000000000000000001501a00000000000000000000000000000000000000000000000000000000000000002c0",
		decoded:    "EquityRootLog{Account: Lemo8888888888888888888888888888888889ZJ, Version: 1, NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
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
		// 10 Votes
		{
			input: NewVotesLog(processor, processor.createAccount(VotesLog, 1), new(big.Int).SetInt64(500)),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, new(big.Int).SetInt64(200), accessor.GetVotes())
			},
		},
		// 11 Profile
		{
			input: NewCandidateLog(processor, processor.createAccount(VotesLog, 1), map[string]string{types.CandidateKeyIsCandidate: "false", types.CandidateKeyHost: "host"}),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, "true", accessor.GetCandidate()[types.CandidateKeyIsCandidate])
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
		// 10 VoteFor
		{
			input: decreaseVersion(NewVoteForLog(processor, processor.createAccount(VoteForLog, 1), common.HexToAddress("0x0002"))),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, common.HexToAddress("0x0002"), accessor.GetVoteFor())
			},
		},
		// 12 Votes
		{
			input: decreaseVersion(NewVotesLog(processor, processor.createAccount(VotesLog, 1), new(big.Int).SetInt64(500))),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, new(big.Int).SetInt64(500), accessor.GetVotes())
			},
		},
		// 13 Profile
		{
			input: decreaseVersion(NewCandidateLog(processor, processor.createAccount(VotesLog, 1), map[string]string{types.CandidateKeyIsCandidate: "false", types.CandidateKeyHost: "host"})),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, "false", accessor.GetCandidate()[types.CandidateKeyIsCandidate])
				assert.Equal(t, "host", accessor.GetCandidate()[types.CandidateKeyHost])
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
	defer db.Close()

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
	profile := make(types.Profile, 0)
	profile["aa"] = "bb"
	account.SetCandidate(profile)
	assert.Equal(t, 1, len(manager.GetChangeLogs()))
	// get and set again
	profile = account.GetCandidate()
	profile["aa"] = "cc"
	account.SetCandidate(profile)
	assert.Equal(t, 2, len(manager.GetChangeLogs()))
	oldProfile := manager.GetChangeLogs()[0].NewVal.(*types.Profile)
	assert.Equal(t, "bb", (*oldProfile)["aa"])

	// The value in TxCountLog is uint32. No need to test
}
