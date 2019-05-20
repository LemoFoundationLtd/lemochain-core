package vm

import (
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"github.com/stretchr/testify/assert"
	"math/big"
	"os"
	"testing"
	"time"
)

func GetStorePath() string {
	return "../testdata/blockchain"
}

func ClearData() {
	err := os.RemoveAll(GetStorePath())
	failCnt := 1
	for err != nil {
		log.Errorf("CLEAR DATA BASE FAIL.%s, SLEEP(%ds) AND CONTINUE", err.Error(), failCnt)
		time.Sleep(time.Duration(failCnt) * time.Second)
		err = os.RemoveAll(GetStorePath())
		failCnt = failCnt + 1
	}
}

// newDB db for test
func newDB() protocol.ChainDB {
	return store.NewChainDataBase(GetStorePath(), "", "")
}

// newTransferAssetData
func newTransferAssetData(assetId common.Hash, amount *big.Int) []byte {
	trad := &types.TradingAsset{
		AssetId: assetId,
		Value:   amount,
		Input:   nil,
	}
	data, _ := json.Marshal(trad)
	return data
}

type TestDb struct {
	issuer common.Address
}

func (t TestDb) GetAssetCode(code common.Hash) (common.Address, error) {
	return t.issuer, nil
}

func NewTestDb(issuer common.Address) *TestDb {
	return &TestDb{
		issuer: issuer,
	}
}

// TestEVM_TransferAssetTx
func TestEVM_TransferAssetTx(t *testing.T) {
	/*
		1. 测试被冻结的资产无法交易
		2. 测试可分割资产余额不足的情况(不可分割资产不存在余额不足的情况)
		3. 测试可分割资产和不可分割资产的交易
	*/
	ClearData()
	db := newDB()
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	assetCode := common.HexToHash("0x14444")
	issuerAddr := common.HexToAddress("0x123456")
	senderAddr := common.HexToAddress("0x223343")
	receiverAddr := common.HexToAddress("0x556544")
	senderAcc := am.GetAccount(senderAddr)
	receiverAcc := am.GetAccount(receiverAddr)
	profile := make(types.Profile)
	equity := big.NewInt(5000)
	// 给sender初始化资产余额
	senderAcc.SetEquityState(assetCode, &types.AssetEquity{
		AssetCode: assetCode,
		AssetId:   assetCode,
		Equity:    equity,
	})
	// 创建evm
	cfg := Config{
		Debug:         false,
		RewardManager: common.HexToAddress("0x12212"),
	}
	newContext := Context{}
	evm := NewEVM(newContext, am, cfg)
	// 查询资产的db
	assetDB := NewTestDb(issuerAddr)

	// var snapshot = am.Snapshot()
	// 1. 测试被冻结的资产无法交易的情况
	// 创建被冻结的资产
	profile[types.AssetFreeze] = "true"
	freezeAsset := &types.Asset{
		Category:        1,
		IsDivisible:     true,
		AssetCode:       assetCode,
		Decimal:         18,
		TotalSupply:     nil,
		IsReplenishable: false,
		Issuer:          issuerAddr,
		Profile:         profile,
	}
	issuerAcc := am.GetAccount(issuerAddr)
	err := issuerAcc.SetAssetCode(assetCode, freezeAsset)
	assert.NoError(t, err)

	// 交易此资产
	data := newTransferAssetData(assetCode, equity)
	_, _, err, _ = evm.TransferAssetTx(senderAcc, receiverAddr, 0, data, assetDB)
	// 返回的是被冻结资产的err
	assert.Equal(t, ErrTransferFrozenAsset, err)

	// am.RevertToSnapshot(snapshot)
	// 2. 测试资产余额不足的情况
	// 创建一个可分割的资产
	profile[types.AssetFreeze] = "false"
	ass := &types.Asset{
		Category:        1,
		IsDivisible:     true,
		AssetCode:       assetCode,
		Decimal:         12,
		TotalSupply:     nil,
		IsReplenishable: false,
		Issuer:          issuerAddr,
		Profile:         profile,
	}
	issuerAcc.SetAssetCode(assetCode, ass)

	// 交易资产
	data = newTransferAssetData(assetCode, new(big.Int).Mul(big.NewInt(2), equity))
	_, _, err, _ = evm.TransferAssetTx(senderAcc, receiverAddr, 0, data, assetDB)
	// 返回余额不足的err
	assert.Equal(t, ErrInsufficientBalance, err)

	// am.RevertToSnapshot(snapshot)
	// 3.1 测试可拆分资产的正常情况
	// 创建一个可拆分资产
	profile[types.AssetFreeze] = "false"
	asset := &types.Asset{
		Category:        1,
		IsDivisible:     true,
		AssetCode:       assetCode,
		Decimal:         10,
		TotalSupply:     nil,
		IsReplenishable: false,
		Issuer:          issuerAddr,
		Profile:         profile,
	}
	err = issuerAcc.SetAssetCode(assetCode, asset)
	assert.NoError(t, err)
	// 交易资产
	data = newTransferAssetData(assetCode, new(big.Int).Div(equity, big.NewInt(2)))
	_, _, err, _ = evm.TransferAssetTx(senderAcc, receiverAddr, 0, data, assetDB)
	assert.NoError(t, err)
	// 查看交易结果
	equity01, err := senderAcc.GetEquityState(assetCode)
	assert.NoError(t, err)
	equity02, err := receiverAcc.GetEquityState(assetCode)
	assert.NoError(t, err)
	assert.Equal(t, new(big.Int).Div(equity, big.NewInt(2)), equity01.Equity)
	assert.Equal(t, new(big.Int).Div(equity, big.NewInt(2)), equity02.Equity)

	// am.RevertToSnapshot(snapshot)
	// 3.2 测试不可拆分资产的正常情况
	profile[types.AssetFreeze] = "false"
	asset = &types.Asset{
		Category:        1,
		IsDivisible:     false,
		AssetCode:       assetCode,
		Decimal:         0,
		TotalSupply:     nil,
		IsReplenishable: false,
		Issuer:          issuerAddr,
		Profile:         profile,
	}
	err = issuerAcc.SetAssetCode(assetCode, asset)
	assert.NoError(t, err)
	// 交易资产
	data = newTransferAssetData(assetCode, new(big.Int).Div(equity, big.NewInt(2)))
	_, _, err, _ = evm.TransferAssetTx(senderAcc, receiverAddr, 0, data, assetDB)
	assert.NoError(t, err)
	// 查看结果
	eq, err := receiverAcc.GetEquityState(assetCode)
	assert.NoError(t, err)
	assert.Equal(t, equity, eq.Equity) // 不可分割资产交易转账余额为资产的所有余额
}
