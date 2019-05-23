package transaction

import (
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

// newCreateAssetTxData
func newCreateAssetTxData(assetName, assetAymbol, assetDescription, freeze string, category uint32, isDivisible, isReplenishable bool) []byte {
	asset := buildAsset(common.HexToAddress("0x11111111111"), common.Hash{}, big.NewInt(10000), assetName, assetAymbol, assetDescription, freeze, "60000", category, isDivisible, isReplenishable)
	data, _ := json.Marshal(asset)
	return data
}

// newIssuerAssetTxData
func newIssuerAssetTxData(assetCode common.Hash, amount *big.Int, metaData string) []byte {
	issue := &types.IssueAsset{
		AssetCode: assetCode,
		MetaData:  metaData,
		Amount:    amount,
	}
	data, _ := json.Marshal(issue)
	return data
}

// newReplenishAssetData
func newReplenishAssetData(assetCode, assetId common.Hash, amount *big.Int) []byte {
	repl := &types.ReplenishAsset{
		AssetCode: assetCode,
		AssetId:   assetId,
		Amount:    amount,
	}
	data, err := json.Marshal(repl)
	if err != nil {
		panic(err)
	}
	return data
}

// newModifyAssetData
func newModifyAssetData(code common.Hash, info types.Profile) []byte {
	modify := &types.ModifyAssetInfo{
		AssetCode: code,
		Info:      info,
	}
	data, err := json.Marshal(modify)
	if err != nil {
		panic(err)
	}
	return data
}

// buildAsset 返回一个自定义的asset对象
func buildAsset(issuer common.Address, assetCode common.Hash, totalSupply *big.Int, assetName, assetAymbol, assetDescription, freeze, suggestGasLimit string, category uint32, isDivisible, isReplenishable bool) *types.Asset {
	newProfile := make(types.Profile)
	newProfile[types.AssetName] = assetName
	newProfile[types.AssetSymbol] = assetAymbol
	newProfile[types.AssetDescription] = assetDescription
	newProfile[types.AssetFreeze] = freeze
	newProfile[types.AssetSuggestedGasLimit] = suggestGasLimit
	newAsset := &types.Asset{
		Category:        category,
		IsDivisible:     isDivisible,
		AssetCode:       assetCode,
		Decimal:         10,
		TotalSupply:     totalSupply,
		IsReplenishable: isReplenishable,
		Issuer:          issuer,
		Profile:         newProfile,
	}
	return newAsset
}

// TestRunAssetEnv_CreateAssetTx 测试创建资产交易执行函数
func TestRunAssetEnv_CreateAssetTx(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	r := NewRunAssetEnv(am)

	var snapshot = r.am.Snapshot()
	// 交易发送者
	sender := common.HexToAddress("0x111111")
	// 交易hash == 资产的assetCode
	txHash := common.HexToHash("0x222222")
	// 生成资产信息所占用字节大于规定的大小的情况
	info := "www.lemochain.comlemolemolemolemolemolemolemolemolemolemolemolemolemolemolemowww.lemochain.comlemolemolemolemolemolemolemolemolemolemolemolemolemolemolemowww.lemochain.comlemolemolemolemolemolemolemolemolemolemolemolemolemolemolemo"
	maxData := newCreateAssetTxData(info, info, info, info, 3, true, true)
	log.Infof("maxData length: ", len(maxData))
	err := r.CreateAssetTx(sender, maxData, txHash)
	assert.Equal(t, ErrMarshalAssetLength, err)

	// 正常情况
	r.am.RevertToSnapshot(snapshot)
	info = "lemo"
	normalData := newCreateAssetTxData(info, info, info, info, 3, true, true)
	err = r.CreateAssetTx(sender, normalData, txHash)
	assert.NoError(t, err)
	// 验证数据
	senderAcc := r.am.GetAccount(sender)
	asset, err := senderAcc.GetAssetCode(txHash)
	assert.NoError(t, err)
	assert.Equal(t, uint32(3), asset.Category)
	assert.Equal(t, big.NewInt(0), asset.TotalSupply)
	assert.True(t, asset.IsDivisible)
	assert.True(t, asset.IsDivisible)
	assert.Equal(t, sender, asset.Issuer)
	assert.Equal(t, info, asset.Profile[types.AssetName])
}

// TestRunAssetEnv_IssueAssetTx 对发行资产中三种资产的特殊逻辑测试
func TestRunAssetEnv_IssueAssetTx(t *testing.T) {
	/*
		1. 发行资产对资产总量的变化
		2. 发行可分割资产和不可分割资产
	*/
	ClearData()
	db := newDB()
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	r := NewRunAssetEnv(am)
	issuer := common.HexToAddress("0x111111")
	assetCode := common.HexToHash("0x222222")
	assetId := common.HexToHash("0x333333")
	receiver := common.HexToAddress("0x444444")
	issuerAcc := r.am.GetAccount(issuer)
	receiverAcc := r.am.GetAccount(receiver)
	amount := big.NewInt(9999)
	meta := "a normal erc20 asset"
	data := newIssuerAssetTxData(assetCode, amount, meta)

	// var snapshot = r.am.Snapshot()

	// 1.1 创建一个可分割资产
	erc20_asset := buildAsset(issuer, assetCode, big.NewInt(0), "lemotest", "LM", "test lemo", "false", "100000", 1, true, true)
	err := issuerAcc.SetAssetCode(assetCode, erc20_asset)
	assert.NoError(t, err)
	// 发行一个erc20资产, 注：erc20资产的assetId == assetCode
	err = r.IssueAssetTx(issuer, receiver, assetId, data)
	assert.NoError(t, err)
	// receiver中获取发行的资产
	equity, err := receiverAcc.GetEquityState(assetCode)
	assert.NoError(t, err)
	assert.Equal(t, assetCode, equity.AssetCode)
	// assetCode == assetId 验证
	assert.Equal(t, equity.AssetId, equity.AssetCode)
	assert.Equal(t, amount, equity.Equity)
	// metaData 验证
	metaData, err := receiverAcc.GetAssetIdState(assetCode)
	assert.NoError(t, err)
	assert.Equal(t, meta, metaData)
	// asset中totalSupply验证
	num, err := issuerAcc.GetAssetCodeTotalSupply(assetCode)
	assert.NoError(t, err)
	assert.Equal(t, amount, num)

	// 回滚到初始状态
	// r.am.RevertToSnapshot(snapshot)

	// 1.2 创建一个不可分割资产来验证TotalSupply只增加1
	notDivisible_asset := buildAsset(issuer, assetCode, big.NewInt(0), "lemotest", "LM", "test lemo", "false", "100000", 2, false, true)
	err = issuerAcc.SetAssetCode(assetCode, notDivisible_asset)
	assert.NoError(t, err)
	// 发行资产
	err = r.IssueAssetTx(issuer, receiver, assetId, data)
	assert.NoError(t, err)
	// 验证资产的Totalsupply是否为1
	num, err = issuerAcc.GetAssetCodeTotalSupply(assetCode)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(1), num)
	// 这是erc721(第二类)资产，验证assetCode != assetId
	equity, err = receiverAcc.GetEquityState(assetId)
	assert.NoError(t, err)
	assert.NotEmpty(t, equity.AssetCode, equity.AssetId)
	assert.Equal(t, assetCode, equity.AssetCode)
	assert.Equal(t, assetId, equity.AssetId)

	// 回滚到初始状态
	// r.am.RevertToSnapshot(snapshot)
	// 2. 创建erc20+721(第三类)资产,验证发行的资产code和id不同
	thirdAsset := buildAsset(issuer, assetCode, big.NewInt(0), "lemotest", "LM", "test lemo", "false", "100000", 3, false, true)
	err = issuerAcc.SetAssetCode(assetCode, thirdAsset)
	assert.NoError(t, err)
	err = r.IssueAssetTx(issuer, receiver, assetId, data)
	assert.NoError(t, err)
	// 验证资产code和id不同
	equity, err = receiverAcc.GetEquityState(assetId)
	assert.NoError(t, err)
	assert.NotEmpty(t, equity.AssetCode, equity.AssetId)
	assert.Equal(t, assetCode, equity.AssetCode)
	assert.Equal(t, assetId, equity.AssetId)
}

// TestRunAssetEnv_ReplenishAssetTx 增发资产
func TestRunAssetEnv_ReplenishAssetTx(t *testing.T) {
	/*
		测试点：
		1. 是否能增发(此资产为不可增发资产、资产被冻结、增发资产交易sender不为资产的issuer).
		2. 增发之后资产总量和增发资产接收账户的资产余额是否相应的增加
	*/
	ClearData()
	db := newDB()
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	r := NewRunAssetEnv(am)
	issuer := common.HexToAddress("0x111111")
	assetCode := common.HexToHash("0x222222")
	// assetId := common.HexToHash("0x333333")
	receiver := common.HexToAddress("0x444444")
	issuerAcc := r.am.GetAccount(issuer)
	receiverAcc := r.am.GetAccount(receiver)

	// var snapshot = r.am.Snapshot()
	// 1. 测试是否可增发. (erc721资产(第二类)一定是不可增发的资产,不可分割的资产一定不可增发)
	// 1.1 创建一个可以分割但是不可增发的资产
	notReplenishAsset := buildAsset(issuer, assetCode, big.NewInt(0), "not replenish", "NR", "not replenish asset", "false", "5000000000", 1, true, false)
	err := issuerAcc.SetAssetCode(assetCode, notReplenishAsset)
	assert.NoError(t, err)
	// 增发这个不可发行的资产,
	replenishData := newReplenishAssetData(assetCode, assetCode, big.NewInt(10000))
	err = r.ReplenishAssetTx(issuer, receiver, replenishData)
	// 返回的错误类型为不可增发资产错误
	assert.Equal(t, ErrIsReplenishable, err)

	// 回滚到最开始状态
	// r.am.RevertToSnapshot(snapshot)
	// 1.2 测试被冻结的资产是不可以被增发的情况
	freezeAsset := buildAsset(issuer, assetCode, big.NewInt(0), "not replenish", "NR", "not replenish asset", "true", "5000000000", 1, true, true)
	err = issuerAcc.SetAssetCode(assetCode, freezeAsset)
	assert.NoError(t, err)
	// 增发被冻结的资产
	freezeData := newReplenishAssetData(assetCode, assetCode, big.NewInt(10000))
	err = r.ReplenishAssetTx(issuer, receiver, freezeData)
	// 返回资产被冻结类型的交易err
	assert.Equal(t, ErrFrozenAsset, err)

	// 1.3 验证只有资产的issuer才有增发资产的权限
	// r.am.RevertToSnapshot(snapshot)
	// 创建一个可增发的正常资产
	normalAsset := buildAsset(issuer, assetCode, big.NewInt(0), "normal", "NN", "normal asset", "false", "20000000000", 1, true, true)
	err = issuerAcc.SetAssetCode(assetCode, normalAsset)
	assert.NoError(t, err)
	// 使用非issuer进行增发资产
	err = r.ReplenishAssetTx(common.HexToAddress("0x12345"), receiver, newReplenishAssetData(assetCode, assetCode, big.NewInt(10000)))
	// 返回asset不存在的错误
	assert.Equal(t, store.ErrNotExist, err)

	// r.am.RevertToSnapshot(snapshot)
	// 2. 成功执行增发资产交易之后资产总量和交易接收者的资产余额是否增加
	// 生成一个正常的可增发的资产
	newAsset := buildAsset(issuer, assetCode, big.NewInt(0), "a", "aa", "aaa", "fasle", "100000000", 1, true, true)
	err = issuerAcc.SetAssetCode(assetCode, newAsset)
	assert.NoError(t, err)

	// 2.1 第一次增发资产
	amount01 := big.NewInt(20000) // 增发数量
	data01 := newReplenishAssetData(assetCode, assetCode, amount01)
	err = r.ReplenishAssetTx(issuer, receiver, data01)
	assert.NoError(t, err)
	// 比较当前资产总量的值
	assetTotal, err := issuerAcc.GetAssetCodeTotalSupply(assetCode)
	assert.NoError(t, err)
	assert.Equal(t, amount01, assetTotal)
	// 接收者的资产余额
	equity01, err := receiverAcc.GetEquityState(assetCode)
	assert.NoError(t, err)
	balance01 := equity01.Equity
	assert.Equal(t, amount01, balance01)

	// 2.2 第二次增发
	amount02 := big.NewInt(3000)
	data02 := newReplenishAssetData(assetCode, assetCode, amount02)
	err = r.ReplenishAssetTx(issuer, receiver, data02)
	assert.NoError(t, err)
	// 比较当前资产的总量是否为前两次增发量之和
	total, err := issuerAcc.GetAssetCodeTotalSupply(assetCode)
	assert.NoError(t, err)
	assert.Equal(t, new(big.Int).Add(amount01, amount02), total)
	// 接收者的资产余额
	equity02, err := receiverAcc.GetEquityState(assetCode)
	assert.Equal(t, new(big.Int).Add(amount01, amount02), equity02.Equity)
}

// TestRunAssetEnv_ModifyAssetProfileTx 修改资产信息测试
func TestRunAssetEnv_ModifyAssetProfileTx(t *testing.T) {
	/*
		测试点：
		1. 只有资产issuer才能修改资产info
		2. nil info的情况
		3. 修改之后的资产info字节数不能超过规定的最大字节数
		4. 正常情况
	*/
	ClearData()
	db := newDB()
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	r := NewRunAssetEnv(am)
	issuer := common.HexToAddress("0x111111")
	assetCode := common.HexToHash("0x222222")
	issuerAcc := r.am.GetAccount(issuer)

	// 创建资产
	asset01 := buildAsset(issuer, assetCode, big.NewInt(0), "a", "aa", "aaa", "fasle", "100000000", 1, true, true)
	err := issuerAcc.SetAssetCode(assetCode, asset01)
	assert.NoError(t, err)

	// var snapshot = r.am.Snapshot()
	// 1. 测试修改资产的账户不是资产的issuer
	errIssuer := common.HexToAddress("0x90999") // 拥有此资产但是不是资产issuer
	err = am.GetAccount(errIssuer).SetAssetCode(assetCode, asset01)
	assert.NoError(t, err)
	// 发起修改资产info交易
	info01 := make(types.Profile)
	info01["name"] = "lemoAsset"
	data01 := newModifyAssetData(assetCode, info01)
	err = r.ModifyAssetProfileTx(errIssuer, data01)
	// 返回交易调用者为非资产issuer的错误类型
	assert.Equal(t, ErrModifyAssetTxSender, err)

	// r.am.RevertToSnapshot(snapshot)
	// 2. 修改的info为nil的情况
	data02 := newModifyAssetData(assetCode, nil)
	err = r.ModifyAssetProfileTx(issuer, data02)
	assert.Equal(t, ErrModifyAssetInfo, err)

	// 3. 修改之后资产info字节数超过最大字节数的情况
	// 构造一个字节数过大的info
	info03 := make(types.Profile)
	info03["aaaaaaaaaaaaaaaaaaaa"] = "www.lemochain.com"
	info03["bbbbbbbbbbbbbbbbbbbb"] = "www.lemochain.com"
	info03["cccccccccccccccccccc"] = "www.lemochain.com"
	info03["dddddddddddddddddddd"] = "www.lemochain.com"
	info03["eeeeeeeeeeeeeeeeeeee"] = "www.lemochain.com"
	info03["ffffffffffffffffffff"] = "www.lemochain.com"
	info03["gggggggggggggggggggg"] = "www.lemochain.com"
	info03["hhhhhhhhhhhhhhhhhhhh"] = "www.lemochain.com"
	info03["iiiiiiiiiiiiiiiiiiii"] = "www.lemochain.com"
	info03["jjjjjjjjjjjjjjjjjjjj"] = "www.lemochain.com"
	data03 := newModifyAssetData(assetCode, info03)
	err = r.ModifyAssetProfileTx(issuer, data03)
	// 返回info超过最大值的错误
	assert.Equal(t, ErrMarshalAssetLength, err)

	// r.am.RevertToSnapshot(snapshot)
	// 4. 正常情况
	info04 := make(types.Profile)
	info04["lemo"] = "lemochain"             // 增加新字段
	info04[types.AssetName] = "newlemochain" // 修改原来的字段
	data04 := newModifyAssetData(assetCode, info04)
	err = r.ModifyAssetProfileTx(issuer, data04)
	assert.NoError(t, err)
	// 检测修改结果
	val01, err := issuerAcc.GetAssetCodeState(assetCode, "lemo")
	assert.NoError(t, err)
	assert.Equal(t, "lemochain", val01)
	val02, err := issuerAcc.GetAssetCodeState(assetCode, types.AssetName)
	assert.NoError(t, err)
	assert.Equal(t, "newlemochain", val02)
}
