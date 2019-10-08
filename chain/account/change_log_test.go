package account

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
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
	return account.GetNextVersion(logType)
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
		Decimal:         0,
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
		Equity:    new(big.Int).SetInt64(200),
	})
	account.equity.cached = map[common.Hash][]byte{
		common.HexToHash("0x33"): val,
	}

	account.data.Candidate.Votes = big.NewInt(200)
	account.data.Candidate.Profile[types.CandidateKeyIsCandidate] = types.IsCandidateNode
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
	log, err := NewStorageLog(account.GetAddress(), processor, key, newVal)
	assert.NoError(t, err)
	return log
}

func getTestLogs(t *testing.T) []testLogConfig {
	processor := &testProcessor{}
	tests := make([]testLogConfig, 0)

	// 0 BalanceLog
	account := processor.createAccount(BalanceLog, 0)
	tests = append(tests, testLogConfig{
		input:      NewBalanceLog(account.GetAddress(), processor, big.NewInt(0)),
		isValuable: true,
		str:        "BalanceLog{Account: Lemo8888888888888888888888888888888888BW, Version: 1, OldVal: 100, NewVal: 0}",
		hash:       "0x9532d32f3b2253bb6fb438cb8ac394882b15a1a2883e6619398d50f059ea2692",
		rlp:        "0xd9019400000000000000000000000000000000000000010180c0",
		decoded:    "BalanceLog{Account: Lemo8888888888888888888888888888888888BW, Version: 1, NewVal: 0}",
	})
	// 1 BalanceLog
	account = processor.createAccount(BalanceLog, 0)
	tests = append(tests, testLogConfig{
		input:      NewBalanceLog(account.GetAddress(), processor, big.NewInt(100)),
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

	// 4 storage root
	account = processor.createAccount(BalanceLog, 0)
	tests = append(tests, testLogConfig{
		input:      NewStorageRootLog(account.GetAddress(), processor, common.HexToHash("0x01"), common.HexToHash("0x02")),
		isValuable: true,
		str:        "StorageRootLog{Account: Lemo8888888888888888888888888888888883WD, Version: 1, OldVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1], NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
		hash:       "0x9b40bebbdc2758db2b756f64eed632f0484e48faa3794adbc503cc55ac5ac334",
		rlp:        "0xf8390394000000000000000000000000000000000000000501a00000000000000000000000000000000000000000000000000000000000000002c0",
		decoded:    "StorageRootLog{Account: Lemo8888888888888888888888888888888883WD, Version: 1, NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
	})

	// 5 CodeLog
	account = processor.createAccount(CodeLog, 0)
	tests = append(tests, testLogConfig{
		input:      NewCodeLog(account.GetAddress(), processor, []byte{0x12, 0x34}),
		isValuable: true,
		str:        "CodeLog{Account: Lemo88888888888888888888888888888888849A, Version: 1, NewVal: 0x1234}",
		hash:       "0x68d9b4a1e48a3f52774beb565d8123281421a4bb3519b30ce7e7cbb24d0dd308",
		rlp:        "0xdb0e94000000000000000000000000000000000000000601821234c0",
	})

	// 6 CodeLog
	account = processor.createAccount(CodeLog, 0)
	tests = append(tests, testLogConfig{
		input:      NewCodeLog(account.GetAddress(), processor, []byte{}),
		isValuable: false,
		str:        "CodeLog{Account: Lemo8888888888888888888888888888888884N7, Version: 1, NewVal: }",
		hash:       "0x73cefd2f2312068337e16f784f8cb06ea132b701b8d614c30aff7f5867d2d6f3",
		rlp:        "0xd90e9400000000000000000000000000000000000000070180c0",
	})

	// 7 AddEventLog
	newEvent := &types.Event{
		Address: common.HexToAddress("0xaaa"),
		Topics:  []common.Hash{common.HexToHash("bbb"), common.HexToHash("ccc")},
		Data:    []byte{0x80, 0x0},
	}

	account = processor.createAccount(AddEventLog, 0)
	tests = append(tests, testLogConfig{
		input:      NewAddEventLog(account.GetAddress(), processor, newEvent),
		isValuable: true,
		str:        "AddEventLog{Account: Lemo888888888888888888888888888888888534, Version: 1, NewVal: event: 0000000000000000000000000000000000000aaa [0000000000000000000000000000000000000000000000000000000000000bbb 0000000000000000000000000000000000000000000000000000000000000ccc] 8000 0000000000000000000000000000000000000000000000000000000000000000 0 0}",
		hash:       "0x9819dd922773475d634fbb6d775e464bd07e3a4217982b78c3f1230dac488de4",
		rlp:        "0xf8760f94000000000000000000000000000000000000000801f85c940000000000000000000000000000000000000aaaf842a00000000000000000000000000000000000000000000000000000000000000bbba00000000000000000000000000000000000000000000000000000000000000ccc828000c0",
	})
	// It is not possible to set NewVal in AddEventLog to nil. We can't test is because we can't rlp encode a (*types.Event)(nil)

	// 8 SuicideLog
	account = processor.createAccount(SuicideLog, 0)
	tests = append(tests, testLogConfig{
		input:      NewSuicideLog(account.GetAddress(), processor),
		isValuable: true,
		str:        "SuicideLog{Account: Lemo8888888888888888888888888888888885CZ, Version: 1, OldVal: {Address: Lemo888888888888888888888888888888888888, Balance: 100, VoteFor: Lemo888888888888888888888888888888888888}}",
		hash:       "0x7ee159c6b3a060c673a26e97e15438a5a16d7b8b21ebcff99f59f2455a8b6147",
		rlp:        "0xd91094000000000000000000000000000000000000000901c0c0",
		decoded:    "SuicideLog{Account: Lemo8888888888888888888888888888888885CZ, Version: 1}",
	})
	// 9 SuicideLog
	account = processor.createEmptyAccount()
	tests = append(tests, testLogConfig{
		input:      NewSuicideLog(account.GetAddress(), processor),
		isValuable: false,
		str:        "SuicideLog{Account: Lemo8888888888888888888888888888888885RT, Version: 1, OldVal: {Address: Lemo888888888888888888888888888888888888, Balance: 0, VoteFor: Lemo888888888888888888888888888888888888}}",
		hash:       "0x6d7a3da06cd453572ac14e54994b53dcbc907299ed32760b3eeb1f456099cfbd",
		rlp:        "0xd91094000000000000000000000000000000000000000a01c0c0",
		decoded:    "SuicideLog{Account: Lemo8888888888888888888888888888888885RT, Version: 1}",
	})

	// 10 VotesLog
	account = processor.createAccount(VotesLog, 0)
	tests = append(tests, testLogConfig{
		input:      NewVotesLog(account.GetAddress(), processor, big.NewInt(1000)),
		isValuable: true,
		str:        "VotesLog{Account: Lemo88888888888888888888888888888888866Q, Version: 1, OldVal: 200, NewVal: 1000}",
		hash:       "0xfc049109e9882cbd4ca27b628958e17242eef720de85cc807a2a5f63313b492f",
		rlp:        "0xdb1294000000000000000000000000000000000000000b018203e8c0",
		decoded:    "VotesLog{Account: Lemo88888888888888888888888888888888866Q, Version: 1, NewVal: 1000}",
	})
	// 11 VotesLog
	account = processor.createAccount(VotesLog, 0)
	tests = append(tests, testLogConfig{
		input:      NewVotesLog(account.GetAddress(), processor, big.NewInt(200)),
		isValuable: false,
		str:        "VotesLog{Account: Lemo8888888888888888888888888888888886HK, Version: 1, OldVal: 200, NewVal: 200}",
		hash:       "0xe555e6c1771827e5edc0e25ac953e5a2b7c4e7fba2c2976a5b608b5133daf646",
		rlp:        "0xda1294000000000000000000000000000000000000000c0181c8c0",
		decoded:    "VotesLog{Account: Lemo8888888888888888888888888888888886HK, Version: 1, NewVal: 200}",
	})

	// 12 VoteForLog
	account = processor.createAccount(VoteForLog, 0)
	tests = append(tests, testLogConfig{
		input:      NewVoteForLog(account.GetAddress(), processor, common.HexToAddress("0x0002")),
		isValuable: true,
		str:        "VoteForLog{Account: Lemo8888888888888888888888888888888886YG, Version: 1, OldVal: Lemo8888888888888888888888888888888888BW, NewVal: Lemo8888888888888888888888888888888888QR}",
		hash:       "0xe377afa192cccb5db82fa76173b993909c11d8eb0e5dd006ddc538e8338ff480",
		rlp:        "0xed1194000000000000000000000000000000000000000d01940000000000000000000000000000000000000002c0",
		decoded:    "VoteForLog{Account: Lemo8888888888888888888888888888888886YG, Version: 1, NewVal: Lemo8888888888888888888888888888888888QR}",
	})
	// 13 VoteForLog
	account = processor.createAccount(VoteForLog, 0)
	tests = append(tests, testLogConfig{
		input:      NewVoteForLog(account.GetAddress(), processor, common.HexToAddress("0x0001")),
		isValuable: false,
		str:        "VoteForLog{Account: Lemo8888888888888888888888888888888887AC, Version: 1, OldVal: Lemo8888888888888888888888888888888888BW, NewVal: Lemo8888888888888888888888888888888888BW}",
		hash:       "0x78cfaf7515a698c7399c8699f214294ab23392afbdeac14715ca967f7171a553",
		rlp:        "0xed1194000000000000000000000000000000000000000e01940000000000000000000000000000000000000001c0",
		decoded:    "VoteForLog{Account: Lemo8888888888888888888888888888888887AC, Version: 1, NewVal: Lemo8888888888888888888888888888888888BW}",
	})

	// 14 AssetCode
	account = processor.createAccount(AssetCodeLog, 0)
	log, _ := NewAssetCodeLog(account.GetAddress(), processor, common.HexToHash("0x33"), new(types.Asset))
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "AssetCodeLog{Account: Lemo8888888888888888888888888888888887P9, Version: 1, OldVal: {Category: 1, IsDivisible: true, AssetCode: 0x0000000000000000000000000000000000000000000000000000000000000011, Decimal: 0, Issuer: Lemo888888888888888888888888888888888FY4, IsReplenishable: false, TotalSupply: 1, Profiles: {lemokey => lemoval}}, NewVal: {Category: 0, IsDivisible: false, AssetCode: 0x0000000000000000000000000000000000000000000000000000000000000000, Decimal: 0, Issuer: Lemo888888888888888888888888888888888888, IsReplenishable: false, TotalSupply: 0, Profile: []}, Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 51]}",
		hash:       "0x18cff028352d5e0dcf7f0e54b6700de3792959f4dece767acf64e5a2617d44cd",
		rlp:        "0xf8760494000000000000000000000000000000000000000f01f83c8080a00000000000000000000000000000000000000000000000000000000000000000808080940000000000000000000000000000000000000000c0a00000000000000000000000000000000000000000000000000000000000000033",
		decoded:    "AssetCodeLog{Account: Lemo8888888888888888888888888888888887P9, Version: 1, NewVal: {Category: 0, IsDivisible: false, AssetCode: 0x0000000000000000000000000000000000000000000000000000000000000000, Decimal: 0, Issuer: Lemo888888888888888888888888888888888888, IsReplenishable: false, TotalSupply: 0, Profile: []}, Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 51]}",
	})

	// 15
	account = processor.createAccount(AssetCodeStateLog, 0)
	log, _ = NewAssetCodeStateLog(account.GetAddress(), processor, common.HexToHash("0x33"), "key", "new")
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "AssetCodeStateLog{Account: Lemo888888888888888888888888888888888246, Version: 1, OldVal: , NewVal: new, Extra: {UUID: 0x0000000000000000000000000000000000000000000000000000000000000033, Key: key}}",
		hash:       "0xca473e00e10bced9d57451b9b7bf2a6d2af434f62a570479442e8352c463ba90",
		rlp:        "0xf8410594000000000000000000000000000000000000001001836e6577e5a00000000000000000000000000000000000000000000000000000000000000033836b6579",
		decoded:    "AssetCodeStateLog{Account: Lemo888888888888888888888888888888888246, Version: 1, NewVal: new, Extra: {UUID: 0x0000000000000000000000000000000000000000000000000000000000000033, Key: key}}",
	})

	// 16
	account = processor.createAccount(AssetCodeRootLog, 0)
	log, _ = NewAssetCodeRootLog(account.GetAddress(), processor, common.HexToHash("0x01"), common.HexToHash("0x02"))
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "AssetCodeRootLog{Account: Lemo8888888888888888888888888888888882F3, Version: 1, OldVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1], NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
		hash:       "0x82174d68a2a7b0c1d8a351d58842cbf66bcc5860482155992dc6fadbd87ad584",
		rlp:        "0xf8390694000000000000000000000000000000000000001101a00000000000000000000000000000000000000000000000000000000000000002c0",
		decoded:    "AssetCodeRootLog{Account: Lemo8888888888888888888888888888888882F3, Version: 1, NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
	})

	account = processor.createAccount(AssetIdLog, 0)
	log, _ = NewAssetIdLog(account.GetAddress(), processor, common.HexToHash("0x33"), "new")
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "AssetIdLog{Account: Lemo8888888888888888888888888888888882SY, Version: 1, OldVal: old, NewVal: new, Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 51]}",
		hash:       "0xd293ba6abe604251f8a9b33feeca4b3c730ac526ac90e2e2a80c56bbf3ae83dc",
		rlp:        "0xf83c0894000000000000000000000000000000000000001201836e6577a00000000000000000000000000000000000000000000000000000000000000033",
		decoded:    "AssetIdLog{Account: Lemo8888888888888888888888888888888882SY, Version: 1, NewVal: new, Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 51]}",
	})

	account = processor.createAccount(AssetIdRootLog, 0)
	log, _ = NewAssetIdRootLog(account.GetAddress(), processor, common.HexToHash("0x01"), common.HexToHash("0x02"))
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "AssetIdRootLog{Account: Lemo88888888888888888888888888888888897S, Version: 1, OldVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1], NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
		hash:       "0x3a536384f429704678fc645e3efd73d16a0b902f35b211c142b37811c1f258c0",
		rlp:        "0xf8390994000000000000000000000000000000000000001301a00000000000000000000000000000000000000000000000000000000000000002c0",
		decoded:    "AssetIdRootLog{Account: Lemo88888888888888888888888888888888897S, Version: 1, NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
	})

	account = processor.createAccount(EquityLog, 0)
	log, _ = NewEquityLog(account.GetAddress(), processor, common.HexToHash("0x33"), new(types.AssetEquity))
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "EquityLog{Account: Lemo8888888888888888888888888888888889JP, Version: 1, OldVal: {AssetCode: 0x0000000000000000000000000000000000000000000000000000000000000022, AssetId: 0x0000000000000000000000000000000000000000000000000000000000000033, Equity: 200}, NewVal: {AssetCode: 0x0000000000000000000000000000000000000000000000000000000000000000, AssetId: 0x0000000000000000000000000000000000000000000000000000000000000000, Equity: 0}, Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 51]}",
		hash:       "0xab78c9401d21445580b71f94ce2976b2bcdf85ac9d7f556d02345018a8ca151f",
		rlp:        "0xf87d0a94000000000000000000000000000000000000001401f843a00000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000000080a00000000000000000000000000000000000000000000000000000000000000033",
		decoded:    "EquityLog{Account: Lemo8888888888888888888888888888888889JP, Version: 1, NewVal: {AssetCode: 0x0000000000000000000000000000000000000000000000000000000000000000, AssetId: 0x0000000000000000000000000000000000000000000000000000000000000000, Equity: 0}, Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 51]}",
	})

	account = processor.createAccount(EquityRootLog, 0)
	log, _ = NewEquityRootLog(account.GetAddress(), processor, common.HexToHash("0x01"), common.HexToHash("0x02"))
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "EquityRootLog{Account: Lemo8888888888888888888888888888888889ZJ, Version: 1, OldVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1], NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
		hash:       "0xf23e2a21d65ab7b4c5211f49d26d2665ff8d0cfe2d6897fb3ab874ae285205c1",
		rlp:        "0xf8390b94000000000000000000000000000000000000001501a00000000000000000000000000000000000000000000000000000000000000002c0",
		decoded:    "EquityRootLog{Account: Lemo8888888888888888888888888888888889ZJ, Version: 1, NewVal: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2]}",
	})

	account = processor.createAccount(AssetCodeTotalSupplyLog, 0)
	log, _ = NewAssetCodeTotalSupplyLog(account.GetAddress(), processor, common.HexToHash("0x33"), new(big.Int).SetInt64(10))
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "AssetCodeTotalSupplyLog{Account: Lemo888888888888888888888888888888888ABF, Version: 1, OldVal: 1, NewVal: 10, Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 51]}",
		hash:       "0x6ff638cf6b1615f41285390e903a9c4796fcb47a25dfe5e107303f952951d63e",
		rlp:        "0xf83907940000000000000000000000000000000000000016010aa00000000000000000000000000000000000000000000000000000000000000033",
		decoded:    "AssetCodeTotalSupplyLog{Account: Lemo888888888888888888888888888888888ABF, Version: 1, NewVal: 10, Extra: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 51]}",
	})

	account = processor.createAccount(SignerLog, 0)

	signers := make(types.Signers, 0)
	signers = append(signers, types.SignAccount{
		Address: common.HexToAddress("0x01"),
		Weight:  99,
	})
	log, _ = NewSignerLog(account.GetAddress(), processor, nil, signers)
	tests = append(tests, testLogConfig{
		input:      log,
		isValuable: true,
		str:        "SignerLog{Account: Lemo888888888888888888888888888888888AQB, Version: 1, NewVal: [{Addr: 0x0000000000000000000000000000000000000001, Weight: 99}]}",
		hash:       "0xc9773be0b6d8eda739a5593ba9280d23e6c48763ee79ec7226cdeb3d4bd1ce08",
		rlp:        "0xf01394000000000000000000000000000000000000001701d7d694000000000000000000000000000000000000000163c0",
		decoded:    "SignerLog{Account: Lemo888888888888888888888888888888888AQB, Version: 1, NewVal: [{Addr: 0x0000000000000000000000000000000000000001, Weight: 99}]}",
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
	// assert.NotEqual(t, 2, 2)
}

func findEvent(processor *testProcessor, txHash common.Hash) []*types.Event {
	accounts := processor.Accounts
	result := make([]*types.Event, 0)
	for _, v := range accounts {
		for _, event := range v.GetEvents() {
			if event.TxHash == txHash {
				result = append(result, event)
			}
		}
	}

	return result
}

func TestChangeLog_Undo(t *testing.T) {
	processor := &testProcessor{}
	event1 := &types.Event{TxHash: common.HexToHash("0x666")}
	processor.PushEvent(&types.Event{})
	processor.PushEvent(event1)

	account := processor.createAccount(AssetCodeLog, 1)
	assetCodeLog, _ := NewAssetCodeLog(account.GetAddress(), processor, common.HexToHash("0x33"), new(types.Asset))
	account = processor.createAccount(AssetCodeRootLog, 1)
	assetCodeRootLog, _ := NewAssetCodeRootLog(account.GetAddress(), processor, common.HexToHash("0x01"), common.HexToHash("0x02"))
	account = processor.createAccount(AssetCodeStateLog, 1)
	assetCodeStateLog, _ := NewAssetCodeStateLog(account.GetAddress(), processor, common.HexToHash("0x33"), "lemokey", "newVal")
	account = processor.createAccount(AssetIdLog, 1)
	assetIdLog, _ := NewAssetIdLog(account.GetAddress(), processor, common.HexToHash("0x033"), "newVal")
	account = processor.createAccount(AssetIdRootLog, 1)
	assetIdRootLog, _ := NewAssetIdRootLog(account.GetAddress(), processor, common.HexToHash("0x11"), common.HexToHash("0x22"))
	account = processor.createAccount(EquityLog, 1)
	equityStateLog, _ := NewEquityLog(account.GetAddress(), processor, common.HexToHash("0x33"), &types.AssetEquity{
		AssetCode: common.HexToHash("0x22"),
		AssetId:   common.HexToHash("0x33"),
		Equity:    new(big.Int).SetInt64(100),
	})
	account = processor.createAccount(EquityRootLog, 1)
	equityRootLog, _ := NewEquityRootLog(account.GetAddress(), processor, common.HexToHash("0x11"), common.HexToHash("0x22"))
	account = processor.createAccount(AssetCodeTotalSupplyLog, 1)
	assetCodeTotalSupplyLog, _ := NewAssetCodeTotalSupplyLog(account.GetAddress(), processor, common.HexToHash("0x33"), new(big.Int).SetInt64(500))

	account0 := processor.createAccount(BalanceLog, 1)
	account3 := processor.createAccount(StorageRootLog, 1)
	account6 := processor.createAccount(CodeLog, 1)
	account7 := processor.createAccount(AddEventLog, 1)
	account8 := processor.createAccount(SuicideLog, 1)
	account9 := processor.createAccount(VoteForLog, 1)
	account10 := processor.createAccount(VotesLog, 1)
	account11 := processor.createAccount(VotesLog, 1)

	signers := make(types.Signers, 0)
	signers = append(signers, types.SignAccount{
		Address: common.HexToAddress("0x01"),
		Weight:  99,
	})
	signerLog, _ := NewSignerLog(account.GetAddress(), processor, make(types.Signers, 0), signers)

	tests := []struct {
		input      *types.ChangeLog
		undoErr    error
		afterCheck func(types.AccountAccessor)
	}{
		// 0 NewBalanceLog
		{
			input: NewBalanceLog(account0.GetAddress(), processor, big.NewInt(120)),
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
		// 3 NewStorageRootLog
		{
			input: NewStorageRootLog(account3.GetAddress(), processor, common.HexToHash("0x01"), common.HexToHash("0x02")),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, common.HexToHash("0x01"), accessor.GetStorageRoot())
			},
		},
		// 4 NewStorageLog no OldVal
		{
			input:   &types.ChangeLog{LogType: StorageLog, Address: processor.createAccount(StorageLog, 1).GetAddress(), Version: 1},
			undoErr: types.ErrWrongChangeLogData,
		},
		// 5 NewStorageLog no Extra
		{
			input:   &types.ChangeLog{LogType: StorageLog, Address: processor.createAccount(StorageRootLog, 1).GetAddress(), Version: 1, OldVal: []byte{45, 67}},
			undoErr: types.ErrWrongChangeLogData,
		},
		// 6 NewCodeLog
		{
			input: NewCodeLog(account6.GetAddress(), processor, []byte{12}),
			afterCheck: func(accessor types.AccountAccessor) {
				code, err := accessor.GetCode()
				assert.Empty(t, code)
				assert.NoError(t, err)
			},
		},
		// 7 NewAddEventLog
		{
			input: NewAddEventLog(account7.GetAddress(), processor, event1),
			afterCheck: func(accessor types.AccountAccessor) {
				events := findEvent(processor, event1.TxHash)
				assert.Empty(t, events)
			},
		},
		// 8 NewSuicideLog
		{
			input: NewSuicideLog(account8.GetAddress(), processor),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, big.NewInt(100), accessor.GetBalance())
				assert.Equal(t, false, accessor.GetSuicide())
			},
		},
		// 9 VoteFor
		{
			input: NewVoteForLog(account9.GetAddress(), processor, common.HexToAddress("0x0002")),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, common.HexToAddress("0x0001"), accessor.GetVoteFor())
			},
		},
		// 10 Votes
		{
			input: NewVotesLog(account10.GetAddress(), processor, new(big.Int).SetInt64(500)),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, new(big.Int).SetInt64(200), accessor.GetVotes())
			},
		},
		// 11 Profile
		{
			input: NewCandidateLog(account11.GetAddress(), processor, map[string]string{types.CandidateKeyIsCandidate: "false", types.CandidateKeyHost: "host"}),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, "true", accessor.GetCandidate()[types.CandidateKeyIsCandidate])
			},
		},
		// 12 NewAssetCodeLog
		{
			input: assetCodeLog,
			afterCheck: func(accessor types.AccountAccessor) {
				GetAssetCode := func(accessor types.AccountAccessor, code common.Hash) *types.Asset {
					result, _ := accessor.GetAssetCode(code)
					return result
				}

				assert.Equal(t, common.HexToHash("0x11"), GetAssetCode(accessor, common.HexToHash("0x33")).AssetCode)
				assert.Equal(t, common.HexToAddress("0x22"), GetAssetCode(accessor, common.HexToHash("0x33")).Issuer)
				assert.Equal(t, 1, len(GetAssetCode(accessor, common.HexToHash("0x33")).Profile))
			},
		},
		// 13 NewAssetCodeRootLog
		{
			input: assetCodeRootLog,
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, common.HexToHash("0x01"), accessor.GetAssetCodeRoot())
			},
		},
		// 14 NewAssetCodeStateLog
		{
			input: assetCodeStateLog,
			afterCheck: func(accessor types.AccountAccessor) {
				GetAssetCodeState := func(accessor types.AccountAccessor, code common.Hash, key string) string {
					result, _ := accessor.GetAssetCodeState(code, key)
					return result
				}
				assert.Equal(t, "lemoval", GetAssetCodeState(accessor, common.HexToHash("0x33"), "lemokey"))
			},
		},
		// 15 NewAssetIdLog
		{
			input: assetIdLog,
			afterCheck: func(accessor types.AccountAccessor) {
				GetAssetIdState := func(id common.Hash) string {
					result, _ := accessor.GetAssetIdState(id)
					return result
				}
				assert.Equal(t, "old", GetAssetIdState(common.HexToHash("0x33")))
			},
		},
		// 16NewAssetIdRootLog
		{
			input: assetIdRootLog,
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, common.HexToHash("0x11"), accessor.GetAssetIdRoot())
			},
		},
		// 17NewEquityLog
		{
			input: equityStateLog,
			afterCheck: func(accessor types.AccountAccessor) {
				GetEquity := func(id common.Hash) *types.AssetEquity {
					result, _ := accessor.GetEquityState(id)
					return result
				}
				assert.Equal(t, new(big.Int).SetInt64(200), GetEquity(common.HexToHash("0x33")).Equity)
			},
		},
		// 18NewEquityRootLog
		{
			input: equityRootLog,
			afterCheck: func(accessor types.AccountAccessor) {
				root := accessor.GetEquityRoot()
				assert.Equal(t, common.HexToHash("0x11"), root)
			},
		},
		// 19 AssetCodeTotalSupplyLog
		{
			input: assetCodeTotalSupplyLog,
			afterCheck: func(accessor types.AccountAccessor) {
				result, _ := accessor.GetAssetCodeTotalSupply(common.HexToHash("0x33"))
				assert.Equal(t, new(big.Int).SetInt64(1), result)
			},
		},
		// 19 AssetCodeTotalSupplyLog
		{
			input: assetCodeTotalSupplyLog,
			afterCheck: func(accessor types.AccountAccessor) {
				result, _ := accessor.GetAssetCodeTotalSupply(common.HexToHash("0x33"))
				assert.Equal(t, new(big.Int).SetInt64(1), result)
			},
		},
		// 20 signerLog
		{
			input: signerLog,
			afterCheck: func(accessor types.AccountAccessor) {
				result := accessor.GetSigners()
				assert.Equal(t, 0, len(result))
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

	account := processor.createAccount(AssetCodeLog, 1)
	assetCodeLog, _ := NewAssetCodeLog(account.GetAddress(), processor, common.HexToHash("0x33"), new(types.Asset))
	account = processor.createAccount(AssetCodeRootLog, 1)
	assetCodeRootLog, _ := NewAssetCodeRootLog(account.GetAddress(), processor, common.HexToHash("0x01"), common.HexToHash("0x02"))
	account = processor.createAccount(AssetCodeStateLog, 1)
	assetCodeStateLog, _ := NewAssetCodeStateLog(account.GetAddress(), processor, common.HexToHash("0x33"), "lemokey", "newVal")
	account = processor.createAccount(AssetIdLog, 1)
	assetIdLog, _ := NewAssetIdLog(account.GetAddress(), processor, common.HexToHash("0x033"), "newVal")
	account = processor.createAccount(AssetIdRootLog, 1)
	assetIdRootLog, _ := NewAssetIdRootLog(account.GetAddress(), processor, common.HexToHash("0x11"), common.HexToHash("0x22"))
	account = processor.createAccount(EquityLog, 1)
	equityStateLog, _ := NewEquityLog(account.GetAddress(), processor, common.HexToHash("0x33"), &types.AssetEquity{
		AssetCode: common.HexToHash("0x22"),
		AssetId:   common.HexToHash("0x33"),
		Equity:    new(big.Int).SetInt64(100),
	})
	account = processor.createAccount(EquityRootLog, 1)
	equityRootLog, _ := NewEquityRootLog(account.GetAddress(), processor, common.HexToHash("0x11"), common.HexToHash("0x22"))
	account = processor.createAccount(AssetCodeTotalSupplyLog, 1)
	assetCodeTotalSupplyLog, _ := NewAssetCodeTotalSupplyLog(account.GetAddress(), processor, common.HexToHash("0x33"), new(big.Int).SetInt64(500))

	account0 := processor.createAccount(BalanceLog, 1)
	account5 := processor.createAccount(CodeLog, 1)
	account7 := processor.createAccount(AddEventLog, 1)
	account9 := processor.createAccount(BalanceLog, 1)
	account10 := processor.createAccount(VoteForLog, 1)
	account12 := processor.createAccount(VotesLog, 1)
	account13 := processor.createAccount(VotesLog, 1)

	signers := make(types.Signers, 0)
	signers = append(signers, types.SignAccount{
		Address: common.HexToAddress("0x01"),
		Weight:  99,
	})
	signerLog, _ := NewSignerLog(account.GetAddress(), processor, make(types.Signers, 0), signers)

	tests := []struct {
		input      *types.ChangeLog
		redoErr    error
		afterCheck func(types.AccountAccessor)
	}{
		// 0 NewBalanceLog
		{
			input: decreaseVersion(NewBalanceLog(account0.GetAddress(), processor, big.NewInt(120))),
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
			input: decreaseVersion(NewCodeLog(account5.GetAddress(), processor, []byte{12})),
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
			input: decreaseVersion(NewAddEventLog(account7.GetAddress(), processor, &types.Event{
				Address: common.HexToAddress("0xaaa"),
				Topics:  []common.Hash{common.HexToHash("bbb"), common.HexToHash("ccc")},
				Data:    []byte{0x80, 0x0},
				TxHash:  common.HexToHash("0x777"),
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
			input: decreaseVersion(NewSuicideLog(account9.GetAddress(), processor)),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, true, accessor.GetSuicide())
				assert.Equal(t, big.NewInt(0), accessor.GetBalance())
			},
		},
		// 10 VoteFor
		{
			input: decreaseVersion(NewVoteForLog(account10.GetAddress(), processor, common.HexToAddress("0x0002"))),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, common.HexToAddress("0x0002"), accessor.GetVoteFor())
			},
		},
		// 12 Votes
		{
			input: decreaseVersion(NewVotesLog(account12.GetAddress(), processor, new(big.Int).SetInt64(500))),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, new(big.Int).SetInt64(500), accessor.GetVotes())
			},
		},
		// 13 Profile
		{
			input: decreaseVersion(NewCandidateLog(account13.GetAddress(), processor, map[string]string{types.CandidateKeyIsCandidate: "false", types.CandidateKeyHost: "host"})),
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, "false", accessor.GetCandidate()[types.CandidateKeyIsCandidate])
				assert.Equal(t, "host", accessor.GetCandidate()[types.CandidateKeyHost])
			},
		},

		// 12 NewAssetCodeLog
		{
			input: assetCodeLog,
			afterCheck: func(accessor types.AccountAccessor) {
				GetAssetCode := func(accessor types.AccountAccessor, code common.Hash) *types.Asset {
					result, _ := accessor.GetAssetCode(code)
					return result
				}

				assert.Equal(t, common.Hash{}, GetAssetCode(accessor, common.HexToHash("0x33")).AssetCode)
				assert.Equal(t, common.Address{}, GetAssetCode(accessor, common.HexToHash("0x33")).Issuer)
				assert.Equal(t, 0, len(GetAssetCode(accessor, common.HexToHash("0x33")).Profile))
			},
		},
		// 13 NewAssetCodeRootLog
		{
			input: assetCodeRootLog,
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, common.HexToHash("0x02"), accessor.GetAssetCodeRoot())
			},
		},
		// 14 NewAssetCodeStateLog
		{
			input: assetCodeStateLog,
			afterCheck: func(accessor types.AccountAccessor) {
				GetAssetCodeState := func(accessor types.AccountAccessor, code common.Hash, key string) string {
					result, _ := accessor.GetAssetCodeState(code, key)
					return result
				}
				assert.Equal(t, "newVal", GetAssetCodeState(accessor, common.HexToHash("0x33"), "lemokey"))
			},
		},
		// 15 NewAssetIdLog
		{
			input: assetIdLog,
			afterCheck: func(accessor types.AccountAccessor) {
				GetAssetIdState := func(id common.Hash) string {
					result, _ := accessor.GetAssetIdState(id)
					return result
				}
				assert.Equal(t, "newVal", GetAssetIdState(common.HexToHash("0x33")))
			},
		},
		// 16NewAssetIdRootLog
		{
			input: assetIdRootLog,
			afterCheck: func(accessor types.AccountAccessor) {
				assert.Equal(t, common.HexToHash("0x22"), accessor.GetAssetIdRoot())
			},
		},
		// 17NewEquityLog
		{
			input: equityStateLog,
			afterCheck: func(accessor types.AccountAccessor) {
				GetEquity := func(id common.Hash) *types.AssetEquity {
					result, _ := accessor.GetEquityState(id)
					return result
				}
				assert.Equal(t, new(big.Int).SetInt64(100), GetEquity(common.HexToHash("0x33")).Equity)
			},
		},
		// 18NewEquityRootLog
		{
			input: equityRootLog,
			afterCheck: func(accessor types.AccountAccessor) {
				root := accessor.GetEquityRoot()
				assert.Equal(t, common.HexToHash("0x22"), root)
			},
		},
		// 19 AssetCodeTotalSupplyLog
		{
			input: assetCodeTotalSupplyLog,
			afterCheck: func(accessor types.AccountAccessor) {
				result, _ := accessor.GetAssetCodeTotalSupply(common.HexToHash("0x33"))
				assert.Equal(t, new(big.Int).SetInt64(500), result)
			},
		},
		// 20 signer log
		{
			input: signerLog,
			afterCheck: func(accessor types.AccountAccessor) {
				result := accessor.GetSigners()
				assert.Equal(t, 1, len(result))
				assert.Equal(t, signers[0], result[0])
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
	ClearData()
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
