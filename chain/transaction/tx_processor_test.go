package transaction

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"
)

func TestNewTxProcessor(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	bc := newTestChain(db)
	p := NewTxProcessor(config.RewardManager, config.ChainID, bc, am, db)
	assert.Equal(t, chainID, p.ChainID)
	assert.Equal(t, config.RewardManager, p.cfg.RewardManager)
	assert.False(t, p.cfg.Debug)
}

// TestTxProcessor_Process 测试process异常情况
func TestTxProcessor_Process(t *testing.T) {
	ClearData()
	db, genesisHash := newCoverGenesisDB()
	defer db.Close()
	am := account.NewManager(genesisHash, db)
	p := NewTxProcessor(config.RewardManager, config.ChainID, newTestChain(db), am, db)

	// 测试执行创世块panic的情况
	genesisBlock, err := db.LoadLatestBlock()
	assert.NoError(t, err)
	assert.PanicsWithValue(t, ErrInvalidGenesis, func() {
		p.Process(genesisBlock.Header, nil)
	})

	// 测试执行错误交易返回错误的情况
	block01 := newBlockForTest(1, nil, am, db, false)
	// 创建一个余额不足的交易
	randPrivate, _ := crypto.GenerateKey()
	tx := makeTx(randPrivate, crypto.PubkeyToAddress(randPrivate.PublicKey), godAddr, nil, params.OrdinaryTx, big.NewInt(4000000))
	_, err = p.Process(block01.Header, types.Transactions{tx})
	assert.Equal(t, ErrInvalidTxInBlock, err)
}

// TestTxProcessor_Process_applyTxs 测试 process 和 applyTxs结果一致性问题
func TestTxProcessor_Process_applyTxs(t *testing.T) {
	ClearData()
	db, genesisHash := newCoverGenesisDB()
	defer db.Close()
	am := account.NewManager(genesisHash, db)
	p := NewTxProcessor(config.RewardManager, config.ChainID, newTestChain(db), am, db)

	// 创建5笔普通交易交易
	txs := make(types.Transactions, 0)
	for i := 0; i < 5; i++ {
		tx := makeTx(godPrivate, godAddr, common.HexToAddress("0x9910"+strconv.Itoa(i)), nil, params.OrdinaryTx, big.NewInt(50000))
		txs = append(txs, tx)
	}
	// 打包交易进区块
	block01 := newBlockForTest(1, txs, am, db, false)
	applyTxsLogs := block01.ChangeLogs
	gasUsed := block01.GasUsed()
	applyTxsVersionRoot := block01.VersionRoot()
	// 执行交易
	newGasUsed, err := p.Process(block01.Header, txs)
	assert.NoError(t, err)
	am.Finalise()
	processLogs := am.GetChangeLogs()
	assert.Equal(t, applyTxsLogs, processLogs)                // 验证changlogs一致性
	assert.Equal(t, gasUsed, newGasUsed)                      // 验证消耗的gas一致性
	assert.Equal(t, applyTxsVersionRoot, am.GetVersionRoot()) // 验证versionRoot一致性

}

// Test_ApplyTxs_TimeoutTime 测试执行交易超时情况
func Test_ApplyTxs_TimeoutTime(t *testing.T) {
	ClearData()
	db, genesisHash := newCoverGenesisDB()
	defer db.Close()
	am := account.NewManager(genesisHash, db)

	p := NewTxProcessor(config.RewardManager, config.ChainID, newTestChain(db), am, db)

	parentBlock, err := db.LoadLatestBlock()
	assert.NoError(t, err)
	header := &types.Header{
		ParentHash:   parentBlock.Hash(),
		MinerAddress: parentBlock.MinerAddress(),
		Height:       parentBlock.Height() + 1,
		GasLimit:     parentBlock.GasLimit(),
	}

	tx01 := makeTx(godPrivate, godAddr, common.HexToAddress("0x11223"), nil, params.OrdinaryTx, common.Big1)
	txNum := 500
	txs := make([]*types.Transaction, 0, txNum)
	for i := 0; i < txNum; i++ {
		txs = append(txs, tx01)
	}
	selectedTxs01, _, _ := p.ApplyTxs(header, txs, int64(0))
	assert.NotEqual(t, len(selectedTxs01), txNum)
	selectedTxs02, _, _ := p.ApplyTxs(header, txs, int64(2))
	assert.NotEqual(t, len(selectedTxs02), txNum)
	selectedTxs03, _, _ := p.ApplyTxs(header, txs, int64(300))
	assert.Equal(t, len(selectedTxs03), txNum)
}

// Test_CreatRegisterTxData 构造注册候选节点所用交易data
func Test_CreatRegisterTxData(t *testing.T) {
	pro1 := make(types.Profile)
	pro1[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	pro1[types.CandidateKeyPort] = "1111"
	pro1[types.CandidateKeyNodeID] = "5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"
	pro1[types.CandidateKeyIncomeAddress] = "Lemo83JZRYPYF97CFSZBBQBH4GW42PD8CFHT5ARN"
	pro1[types.CandidateKeyHost] = "1111"
	marPro1, _ := json.Marshal(pro1)
	fmt.Println("txData1:", common.ToHex(marPro1))

	pro2 := make(types.Profile)
	pro2[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	pro2[types.CandidateKeyPort] = "2222"
	pro2[types.CandidateKeyNodeID] = "34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"
	pro2[types.CandidateKeyIncomeAddress] = common.HexToAddress("0x222").String()
	pro2[types.CandidateKeyHost] = "2222"
	marPro2, _ := json.Marshal(pro2)
	fmt.Println("txData2:", common.ToHex(marPro2))

	pro3 := make(types.Profile)
	pro3[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	pro3[types.CandidateKeyPort] = "3333"
	pro3[types.CandidateKeyNodeID] = "5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"
	pro3[types.CandidateKeyIncomeAddress] = common.HexToAddress("0x333").String()
	pro3[types.CandidateKeyHost] = "3333"
	marPro3, _ := json.Marshal(pro3)
	fmt.Println("txData3:", common.ToHex(marPro3))

	pro4 := make(types.Profile)
	pro4[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	pro4[types.CandidateKeyPort] = "4444"
	pro4[types.CandidateKeyNodeID] = "ddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0"
	pro4[types.CandidateKeyIncomeAddress] = common.HexToAddress("0x444").String()
	pro4[types.CandidateKeyHost] = "4444"
	marPro4, _ := json.Marshal(pro4)
	fmt.Println("txData4:", common.ToHex(marPro4))

	pro5 := make(types.Profile)
	pro5[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	pro5[types.CandidateKeyPort] = "5555"
	pro5[types.CandidateKeyNodeID] = "7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2d0347692f36ef736f342fb5ce4710f7e337f062cc2110d134b63a9575f78cb167bfae2f43"
	pro5[types.CandidateKeyIncomeAddress] = common.HexToAddress("0x555").String()
	pro5[types.CandidateKeyHost] = "5555"
	marPro5, _ := json.Marshal(pro5)
	fmt.Println("txData5:", common.ToHex(marPro5))
}

// TestReimbursement_transaction 打包并验证代付gas交易测试
func TestReimbursement_transaction(t *testing.T) {
	var (
		senderPrivate, _   = crypto.HexToECDSA("c8fa12aa54fbcc249611e5fefa0967658a7ca06022e9d50b53ef6f5b050b697f")
		senderAddr         = crypto.PubkeyToAddress(senderPrivate.PublicKey)
		gasPayerPrivate, _ = crypto.HexToECDSA("57a0b0be5616e74c4315882e3649ade12c775db3b5023dcaa168d01825612c9b")
		gasPayerAddr       = crypto.PubkeyToAddress(gasPayerPrivate.PublicKey)
		Tx01               = makeTx(godPrivate, godAddr, gasPayerAddr, nil, params.OrdinaryTx, params.RegisterCandidateNodeFees) // 转账1000LEMO给gasPayerAddr
		Tx02               = makeTx(godPrivate, godAddr, senderAddr, nil, params.OrdinaryTx, params.RegisterCandidateNodeFees)   // 转账1000LEMO给senderAddr

		amountReceiver = common.HexToAddress("0x1234")
		TxV01          = types.NewReimbursementTransaction(senderAddr, amountReceiver, gasPayerAddr, params.RegisterCandidateNodeFees, []byte{}, params.OrdinaryTx, chainID, uint64(time.Now().Unix()+300), "", "")
	)
	ClearData()
	db, genesisHash := newCoverGenesisDB()
	defer db.Close()
	am := account.NewManager(genesisHash, db)
	p := NewTxProcessor(config.RewardManager, config.ChainID, newTestChain(db), am, db)

	// create a block contains two account which used to make reimbursement transaction
	_ = newBlockForTest(1, types.Transactions{Tx01, Tx02}, am, db, true)

	// p.am.Reset(Block01.Hash())
	// check their balance
	gasPayerAcc := p.am.GetAccount(gasPayerAddr)
	senderAcc := p.am.GetAccount(senderAddr)
	initGasPayerBalance := gasPayerAcc.GetBalance()
	initTxSenderBalance := senderAcc.GetBalance()
	assert.Equal(t, params.RegisterCandidateNodeFees, initGasPayerBalance)
	assert.Equal(t, params.RegisterCandidateNodeFees, initTxSenderBalance)

	// sender transfer LEMO to receiver, payer pay for that transaction
	firstSignTxV, err := types.MakeReimbursementTxSigner().SignTx(TxV01, senderPrivate)
	assert.NoError(t, err)
	firstSignTxV = types.GasPayerSignatureTx(firstSignTxV, common.Big1, uint64(60000))
	lastSignTxV, err := types.MakeGasPayerSigner().SignTx(firstSignTxV, gasPayerPrivate)
	assert.NoError(t, err)
	_ = newBlockForTest(2, types.Transactions{lastSignTxV}, am, db, true)

	// check their balance
	endGasPayerBalance := p.am.GetCanonicalAccount(gasPayerAddr).GetBalance()
	endTxSenderBalance := p.am.GetCanonicalAccount(senderAddr).GetBalance()
	assert.Equal(t, big.NewInt(0), endTxSenderBalance)
	assert.Equal(t, endGasPayerBalance, new(big.Int).Sub(initGasPayerBalance, big.NewInt(int64(params.OrdinaryTxGas))))
	assert.Equal(t, params.RegisterCandidateNodeFees, p.am.GetAccount(amountReceiver).GetBalance())
}

// TestBlockChain_txData 构造生成调用设置换届奖励的预编译合约交易的data
func TestBlockChain_data(t *testing.T) {
	re := params.RewardJson{
		Term:  3,
		Value: big.NewInt(3330),
	}
	by, _ := json.Marshal(re)
	fmt.Println("tx data", common.ToHex(by))
	fmt.Println("预编译合约地址", common.BytesToAddress([]byte{9}).String())
}

func TestIntrinsicGas(t *testing.T) {
	var gas uint64
	mm := make(map[uint16]uint64) // k == txType; v == 正确的gas花费
	mm[params.OrdinaryTx] = params.OrdinaryTxGas
	mm[params.VoteTx] = params.VoteTxGas
	mm[params.RegisterTx] = params.RegisterTxGas
	mm[params.CreateAssetTx] = params.CreateAssetTxGas
	mm[params.IssueAssetTx] = params.IssueAssetTxGas
	mm[params.ReplenishAssetTx] = params.ReplenishAssetTxGas
	mm[params.ModifyAssetTx] = params.ModifyAssetTxGas
	mm[params.TransferAssetTx] = params.TransferAssetTxGas
	mm[params.ModifySignersTx] = params.ModifySigsTxGas

	for k, v := range mm {
		gas, _ = IntrinsicGas(k, nil, "")
		assert.Equal(t, v, gas)
	}

	// 创建合约
	gas, _ = IntrinsicGas(params.CreateContractTx, nil, "")
	assert.Equal(t, params.TxGasContractCreation, gas)

	// 测试交易message消耗的gas
	message := "test message spend gas"
	messLen := uint64(len(message))
	gas, _ = IntrinsicGas(params.OrdinaryTx, nil, message)
	assert.Equal(t, params.OrdinaryTxGas+messLen*params.TxMessageGas, gas)

	// 测试data中字节全为0
	zeroData := make([]byte, 10)
	zeroData = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	gas, _ = IntrinsicGas(params.OrdinaryTx, zeroData, "")
	assert.Equal(t, params.OrdinaryTxGas+10*params.TxDataZeroGas, gas)
	// 测试data 中字节全不为0的情况
	notZeroData := make([]byte, 10)
	notZeroData = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	gas, _ = IntrinsicGas(params.OrdinaryTx, notZeroData, "")
	assert.Equal(t, params.OrdinaryTxGas+10*params.TxDataNonZeroGas, gas)
	// 测试data一半为0的情况
	halfZeroData := make([]byte, 10)
	halfZeroData = []byte{1, 0, 2, 0, 3, 0, 4, 0, 5, 0}
	gas, _ = IntrinsicGas(params.OrdinaryTx, halfZeroData, "")
	assert.Equal(t, params.OrdinaryTxGas+5*(params.TxDataNonZeroGas+params.TxDataZeroGas), gas)
}

// 测试获取资产data最大字节数标准值
func TestMaxAssetProfile(t *testing.T) {
	profile := make(types.Profile)
	profile["aaaaaaaaaaaaaaaaaaaa"] = "www.lemochain.com"
	profile["bbbbbbbbbbbbbbbbbbbb"] = "www.lemochain.com"
	profile["cccccccccccccccccccc"] = "www.lemochain.com"
	profile["dddddddddddddddddddd"] = "www.lemochain.com"
	profile["eeeeeeeeeeeeeeeeeeee"] = "www.lemochain.com"
	profile["ffffffffffffffffffff"] = "www.lemochain.com"
	profile["gggggggggggggggggggg"] = "www.lemochain.com"
	profile["hhhhhhhhhhhhhhhhhhhh"] = "www.lemochain.com"
	profile["iiiiiiiiiiiiiiiiiiii"] = "www.lemochain.com"
	profile["jjjjjjjjjjjjjjjjjjjj"] = "www.lemochain.com"
	asset := &types.Asset{
		Category:        1,
		IsDivisible:     false,
		AssetCode:       common.StringToHash("702aff687d34228aa696d32cf702844c4cbe619411250e864ea45826d8df6751"),
		Decimal:         18,
		TotalSupply:     big.NewInt(111111111111111111),
		IsReplenishable: false,
		Issuer:          common.HexToAddress("0x702aff687d34228aa69619411250e864ea45826d8df6751"),
		Profile:         profile,
	}
	data, err := json.Marshal(asset)
	assert.NoError(t, err)
	t.Logf("data length : %d", len(data))

	gasUsed, err := IntrinsicGas(params.OrdinaryTx, data, "")
	assert.NoError(t, err)
	t.Logf("max gasUsed : %d", gasUsed)
}

// Test_setRewardTx 设置矿工奖励交易
func Test_setRewardTx(t *testing.T) {
	ClearData()
	db, genesisHash := newCoverGenesisDB()
	defer db.Close()
	am := account.NewManager(genesisHash, db)
	p := NewTxProcessor(config.RewardManager, config.ChainID, newTestChain(db), am, db)

	// 设置第0届的矿工奖励
	data := setRewardTxData(0, new(big.Int).Div(params.TermRewardPoolTotal, common.Big2))
	// private, err := crypto.HexToECDSA("c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa")
	// assert.NoError(t, err)
	TxV01 := types.NewReimbursementTransaction(godAddr, params.TermRewardContract, godAddr, nil, data, params.OrdinaryTx, chainID, uint64(time.Now().Unix()+300), "", "")
	firstSignTxV, err := types.MakeReimbursementTxSigner().SignTx(TxV01, godPrivate)
	assert.NoError(t, err)
	firstSignTxV = types.GasPayerSignatureTx(firstSignTxV, common.Big1, uint64(60000))
	lastSignTxV, err := types.MakeGasPayerSigner().SignTx(firstSignTxV, godPrivate)
	assert.NoError(t, err)
	txs := types.Transactions{lastSignTxV}

	Block02 := newBlockForTest(1, txs, am, db, true)
	assert.NotEmpty(t, Block02)
	Acc := p.am.GetAccount(params.TermRewardContract)
	key := params.TermRewardContract.Hash()
	v, err := Acc.GetStorageState(key)
	assert.NoError(t, err)
	rewardMap := make(params.RewardsMap)
	err = json.Unmarshal(v, &rewardMap)
	assert.NoError(t, err)
	assert.Equal(t, uint32(0), rewardMap[0].Term)
	assert.Equal(t, new(big.Int).Div(params.TermRewardPoolTotal, common.Big2), rewardMap[0].Value)
	assert.Equal(t, uint32(1), rewardMap[0].Times)
}

func setRewardTxData(term uint32, value *big.Int) []byte {
	re := params.RewardJson{
		Term:  term,
		Value: value,
	}
	by, err := json.Marshal(re)
	if err != nil {
		log.Warn(err.Error())
		return nil
	}
	return by
}

// TestBlockChain_txData 生成调用设置换届奖励的预编译合约交易的data
func TestBlockChain_txData(t *testing.T) {
	value, _ := new(big.Int).SetString("99999999999999999999", 10)
	re := params.RewardJson{
		Term:  0,
		Value: value,
	}
	by, _ := json.Marshal(re)
	fmt.Println("tx data", common.ToHex(by))
	fmt.Println("预编译合约地址", common.BytesToAddress([]byte{9}).String())
}

// 生成设置换届奖励交易data
func Test_createRewardTxData(t *testing.T) {
	value, _ := new(big.Int).SetString("500000000000000000000000", 10)
	for i := 11; i < 26; i++ {
		re := params.RewardJson{
			Term:  uint32(i),
			Value: value,
		}
		by, _ := json.Marshal(re)
		fmt.Println(i, "term: ", common.ToHex(by))
	}
}

func Test_rlpBlock(t *testing.T) {
	ClearData()
	db, _ := newCoverGenesisDB()
	defer db.Close()
	block, _ := db.GetBlockByHeight(0)
	buf, err := rlp.EncodeToBytes(block)
	assert.NoError(t, err)
	t.Log("rlp length:", len(buf))
	var decBlock types.Block

	err = rlp.DecodeBytes(buf, &decBlock)
	assert.NoError(t, err)
	assert.Equal(t, block.Header.Hash(), decBlock.Header.Hash())
}

// getContractFunctionCode 计算合约函数code
func getContractFunctionCode(funcName string) []byte {
	h := crypto.Keccak256Hash([]byte(funcName))
	return h.Bytes()[:4]
}

// formatArgs 把参数转换成[32]byte的数组类型
func formatArgs(args string) []byte {
	b := common.FromHex(args)
	var h [32]byte
	if len(b) > len(h) {
		b = b[len(b)-32:]
	}
	copy(h[32-len(b):], b)
	return h[:]
}

// Test_Contract 合约测试
func Test_Contract(t *testing.T) {
	ClearData()
	db, genesisHash := newCoverGenesisDB()
	defer db.Close()
	am := account.NewManager(genesisHash, db)
	p := NewTxProcessor(config.RewardManager, config.ChainID, newTestChain(db), am, db)

	// 创建一个发行erc20代币的合约
	/*
		构造函数参数为：
		token_supply: 500
		token_name: LemoCoin
		token_symbol: Lemo
	*/
	// 文件中读取合约code
	filePath, _ := filepath.Abs("../transaction/test_data.txt")
	f, err := os.Open(filePath)
	assert.NoError(t, err)
	defer f.Close()
	by, err := ioutil.ReadAll(f)
	assert.NoError(t, err)

	code := string(by)
	data := common.FromHex(code)
	createContractTx := signTransaction(types.NewContractCreation(godAddr, nil, uint64(5000000), common.Big1, data, params.CreateContractTx, chainID, uint64(time.Now().Unix()+30*60), "", ""), godPrivate)
	txs := types.Transactions{createContractTx}
	block01 := newBlockForTest(1, txs, am, db, true)
	// godAcc := am.GetAccount(godAddr)
	contractAddr := crypto.CreateContractAddress(godAddr, createContractTx.Hash()) // 合约地址

	// 1. 读取部署的智能合约
	// 1.1 读取合约发行代币的name
	funcName := "name()"
	funcCode := getContractFunctionCode(funcName)
	ret, _, err := readContraction(p, db, block01.Header, &contractAddr, funcCode)
	assert.NoError(t, err)
	// 通过正则匹配，筛选出返回的name
	name := regexMatchLetter(string(ret))
	assert.Equal(t, "LemoCoin", name)

	// 1.2 读取合约的symbol
	funcName = "symbol()"
	funcCode = getContractFunctionCode(funcName)
	ret, _, err = readContraction(p, db, block01.Header, &contractAddr, funcCode)
	symbol := regexMatchLetter(string(ret))
	assert.Equal(t, "Lemo", symbol)

	// 1.3 读取合约的totalSupply
	funcName = "totalSupply()"
	funcCode = getContractFunctionCode(funcName)
	ret, _, err = readContraction(p, db, block01.Header, &contractAddr, funcCode)
	totalSpply, err := strconv.ParseInt(common.ToHex(ret), 0, 64)
	assert.NoError(t, err)
	assert.Equal(t, int64(500), totalSpply)

	// 1.4 读取合约拥有者owner
	funcName = "owner()"
	funcCode = getContractFunctionCode(funcName)
	ret, _, err = readContraction(p, db, block01.Header, &contractAddr, funcCode)
	assert.Equal(t, godLemoAddr, common.BytesToAddress(ret).String())

	// 1.5 读取合约拥有者的持有代币balance
	funcName = "balanceOf(address)"
	funcCode = getContractFunctionCode(funcName)
	ownerAddr := "0x015780f8456f9c1532645087a19dcf9a7e0c7f97"
	codeBytes := make([]byte, 0)
	codeBytes = append(funcCode, formatArgs(ownerAddr)...)
	ret, _, err = readContraction(p, db, block01.Header, &contractAddr, codeBytes)
	balance := new(big.Int).SetBytes(ret)
	assert.Equal(t, big.NewInt(500), balance)

	// 1.6 读取合约的authority账户
	funcName = "authority()"
	funcCode = getContractFunctionCode(funcName)
	ret, _, err = readContraction(p, db, block01.Header, &contractAddr, funcCode)
	expectAuthorityAddr := crypto.CreateContractAddress(contractAddr, createContractTx.Hash())
	assert.Equal(t, expectAuthorityAddr, common.BytesToAddress(ret))

	// 2. 调用合约中的方法
	// 2.1 批量调用转账方法
	/*
		合约发送者godAddr 的初始代币数量为500
		向10个账户每个账户转10个代币
		只有前面五个账户转账才会成功，后面五个账户由于余额不足转账会失败
	*/
	funcName = "transfer(address,uint256)"
	funcCode = getContractFunctionCode(funcName)
	amount01 := formatArgs(fmt.Sprintf("%x", 10)) // 转币数量10
	// transferFun := "0xa9059cbb"
	// amount := "0000000000000000000000000000000000000000000000000000000000000064" // 转币数量100

	txs = make(types.Transactions, 0)
	// 存储随机的100个账户的私钥
	randKey := make(map[int]*ecdsa.PrivateKey, 0)
	// 给100个随机账户转代币10，由于总代币只有500，所以只有前50个随机账户才会转成功
	for i := 1; i <= 100; i++ {
		input := make([]byte, 0)
		private, _ := crypto.GenerateKey()
		randKey[i] = private

		rawAddrBytes := formatArgs(crypto.PubkeyToAddress(private.PublicKey).Hex())
		input = append(append(funcCode, rawAddrBytes...), amount01...)
		// 转代币给这100个随机账户
		tx := makeTx(godPrivate, godAddr, contractAddr, input, params.OrdinaryTx, nil)
		txs = append(txs, tx)
	}

	// 给前50个账户转lemo用于后面发送交易消耗gas
	for index, key := range randKey {
		if index <= 50 {
			to := crypto.PubkeyToAddress(key.PublicKey)
			tx := makeTx(godPrivate, godAddr, to, nil, params.OrdinaryTx, big.NewInt(5000000)) // 转lemo交易
			txs = append(txs, tx)
		}
	}

	// 把前50个转成功的账户分别转5个代币给后面转失败的50个账户中，这样到最后这一百个账户都有5个代币
	amount02 := formatArgs(fmt.Sprintf("%x", 5)) // 转5个代币
	for i := 1; i <= 50; i++ {
		input := make([]byte, 0)

		fromPriv := randKey[i]
		fromAddr := crypto.PubkeyToAddress(fromPriv.PublicKey)
		toPriv := randKey[100-i+1]
		toAddrBytes := formatArgs(crypto.PubkeyToAddress(toPriv.PublicKey).Hex())

		input = append(append(funcCode, toAddrBytes...), amount02...)
		tx := makeTx(fromPriv, fromAddr, contractAddr, input, params.OrdinaryTx, nil)
		txs = append(txs, tx)
	}

	// 通过打包区块来执行交易
	block02 := newBlockForTest(2, txs, am, db, true)
	// event changlog
	changeLogs := block02.ChangeLogs
	count := 0
	for _, log := range changeLogs {
		if log.LogType == account.AddEventLog {
			count++
		}
	}
	assert.Equal(t, 150, count)
	// 验证只打包交易条数
	assert.Equal(t, len(txs), len(block02.Txs))

	// 从合约中读取转账之后发送者(godAddr)的代币余额 (期望值为0)
	godAddrGetBalanceFunc := common.FromHex("0x70a08231000000000000000000000000015780f8456f9c1532645087a19dcf9a7e0c7f97")
	ret, _, err = readContraction(p, db, block02.Header, &contractAddr, godAddrGetBalanceFunc)
	balance = new(big.Int).SetBytes(ret)
	assert.Equal(t, big.NewInt(0).String(), balance.String())

	// 读取这100个账户拥有的代币数量，期望值是都为5个
	funcName = "balanceOf(address)"
	funcCode = getContractFunctionCode(funcName)
	for _, key := range randKey {
		codeBytes := make([]byte, 0)
		owner := formatArgs(crypto.PubkeyToAddress(key.PublicKey).Hex())
		codeBytes = append(funcCode, owner...)
		ret, _, err = readContraction(p, db, block02.Header, &contractAddr, codeBytes)
		balance = new(big.Int).SetBytes(ret)
		// 每个地址代币数都为5
		assert.Equal(t, big.NewInt(5).String(), balance.String())
	}

	// 2.2 测试合约stop功能
	funcName = "stop()"
	stopContractData := getContractFunctionCode(funcName)
	tx03 := makeTx(godPrivate, godAddr, contractAddr, stopContractData, params.OrdinaryTx, nil)
	// 打包区块来执行交易
	block03 := newBlockForTest(3, types.Transactions{tx03}, am, db, true)
	assert.Equal(t, types.Transactions{tx03}, block03.Txs)
	// 通过测试合约内转账功能是否能成功来判断stop合约是否成功
	funcName = "transfer(address,uint256)"
	funcCode = getContractFunctionCode(funcName)
	coinReceiver := "0x016ad4fc7e1608685bf5fe5573973bf2b1ef9b8a"
	num := formatArgs(fmt.Sprintf("%x", 5)) // 转账5个代币
	input := make([]byte, 0)
	input = append(append(funcCode, formatArgs(coinReceiver)...), num...)

	// 创建transfer交易
	fromPriv := randKey[1] // 交易发送者为上面100个随机地址中的第一个，因为已经有足够的lemo和5个代币
	from := crypto.PubkeyToAddress(fromPriv.PublicKey)
	tx04 := makeTx(fromPriv, from, contractAddr, input, params.OrdinaryTx, nil)
	block04 := newBlockForTest(4, types.Transactions{tx04}, am, db, true)
	assert.Equal(t, types.Transactions{tx04}, block04.Txs)
	assert.Equal(t, 1, len(block04.Txs))
	// 从合约中读取转账之后代币接收者的代币余额(期望值是没变的)
	funcName = "balanceOf(address)"
	funcCode = getContractFunctionCode(funcName)
	readInput := make([]byte, 0)
	readInput = append(funcCode, formatArgs(coinReceiver)...)
	ret, _, err = readContraction(p, db, block04.Header, &contractAddr, readInput)
	balance = new(big.Int).SetBytes(ret)
	assert.Equal(t, big.NewInt(0).String(), balance.String())
	// 读取发送者的代币余额
	ret, _, err = readContraction(p, db, block04.Header, &contractAddr, append(funcCode, formatArgs(from.Hex())...))
	number := new(big.Int).SetBytes(ret)
	assert.Equal(t, big.NewInt(5).String(), number.String())
}

// regexMatchLetter 正则配置出字符串中的字母字符串
func regexMatchLetter(src string) string {
	reg := regexp.MustCompile(`[a-zA-Z0-9]+`)
	result := reg.FindAllString(src, -1)
	return result[0]
}

// readContraction 读取合约中的数据
func readContraction(p *TxProcessor, db protocol.ChainDB, currentHeader *types.Header, contractAddr *common.Address, data []byte) (reslut []byte, gasUsed uint64, err error) {
	ctx := context.Background()
	accM := account.NewReadOnlyManager(db, false)
	return p.PreExecutionTransaction(ctx, accM, currentHeader, contractAddr, params.OrdinaryTx, data, common.Hash{}, 5*time.Second)
}

// func BenchmarkApplyTxs(b *testing.B) {
// 	ClearData()
// 	bc := newChain()
// 	defer bc.db.Close()
// 	p := NewTxProcessor(bc)
//
// 	// prepare account and balance
// 	blockHash, accountKeys := createAccounts(b.N, bc.db)
// 	bc.am.Reset(blockHash)
// 	header := &types.Header{
// 		ParentHash:   blockHash,
// 		MinerAddress: defaultAccounts[0],
// 		Height:       4,
// 		GasLimit:     2100000000,
// 		Time:         1538209762,
// 	}
// 	// make txs
// 	txs := make(types.Transactions, b.N, b.N)
// 	r := rand.New(rand.NewSource(time.Now().UnixNano()))
// 	for i := 0; i < b.N; i++ {
// 		fromKey := accountKeys[r.Intn(b.N)]
// 		fromPrivate, _ := crypto.ToECDSA(common.FromHex(fromKey.Private))
// 		to := accountKeys[r.Intn(b.N)].Address
// 		fromBalance := p.am.GetAccount(fromKey.Address).GetBalance() // maybe 0
// 		amount := new(big.Int).Rand(r, fromBalance)                  // maybe too many if we make transaction more than twice from same address
// 		txs[i] = makeTx(fromPrivate, to, params.OrdinaryTx, amount)
// 	}
//
// 	start := time.Now().UnixNano()
// 	b.ResetTimer()
// 	selectedTxs, invalidTxs, _ := p.ApplyTxs(header, txs, int64(10000))
// 	fmt.Printf("BenchmarkApplyTxs cost %dms\n", (time.Now().UnixNano()-start)/1000000)
// 	fmt.Printf("%d transactions success, %d transactions fail\n", len(selectedTxs), len(invalidTxs))
// }

// func BenchmarkMakeBlock(b *testing.B) {
// 	ClearData()
// 	bc := newChain()
// 	defer bc.db.Close()
// 	p := NewTxProcessor(bc)
// 	balanceRecord := make(map[common.Address]*big.Int)
//
// 	// prepare account and balance
// 	blockHash, accountKeys := createAccounts(b.N, bc.db)
// 	bc.am.Reset(blockHash)
// 	bc.db.SetStableBlock(blockHash)
// 	// make txs
// 	txs := make(types.Transactions, b.N, b.N)
// 	r := rand.New(rand.NewSource(time.Now().UnixNano()))
// 	for i := 0; i < b.N; i++ {
// 		fromKey := accountKeys[r.Intn(b.N)]
// 		fromPrivate, _ := crypto.ToECDSA(common.FromHex(fromKey.Private))
// 		to := accountKeys[r.Intn(b.N)].Address
// 		fromBalance, ok := balanceRecord[fromKey.Address]
// 		if !ok {
// 			fromBalance = p.am.GetAccount(fromKey.Address).GetBalance() // maybe 0
// 		}
// 		amount := new(big.Int).Rand(r, fromBalance)
// 		balanceRecord[fromKey.Address] = new(big.Int).Sub(fromBalance, amount)
// 		txs[i] = makeTx(fromPrivate, to, params.OrdinaryTx, amount)
// 	}
//
// 	start := time.Now().UnixNano()
// 	b.ResetTimer()
// 	newBlock := makeBlock(bc.db, blockInfo{
// 		height:     4,
// 		parentHash: blockHash,
// 		author:     defaultAccounts[0],
// 		time:       1538209762,
// 		txList:     txs,
// 		gasLimit:   2100000000,
// 	}, true)
// 	fmt.Printf("BenchmarkMakeBlock cost %dms\n", (time.Now().UnixNano()-start)/1000000)
// 	fmt.Printf("%d transactions success, %d transactions fail\n", len(newBlock.Txs), b.N-len(newBlock.Txs))
//
// 	startSave := time.Now().UnixNano()
// 	bc.db.SetStableBlock(newBlock.Hash())
// 	fmt.Printf("Saving stable to disk cost %dms\n", (time.Now().UnixNano()-startSave)/1000000)
// 	time.Sleep(3 * time.Second)
// }

// func BenchmarkSetBalance(b *testing.B) {
// 	fromAddr := testAddr
// 	fromBalance := new(big.Int)
// 	toBalance := new(big.Int)
// 	salary := new(big.Int)
// 	amount, _ := new(big.Int).SetString("1234857462837462918237", 10)
// 	tx := makeTx(testPrivate, common.HexToAddress("0x123"), params.OrdinaryTx, amount)
// 	for i := 0; i < b.N; i++ {
// 		gas := params.OrdinaryTxGas + params.TxDataNonZeroGas*uint64(len("abc"))
// 		// fromAddr, err := tx.From()
// 		// if err != nil {
// 		// 	panic(err)
// 		// }
// 		// from := manager.GetAccount(fromAddr)
// 		fee := new(big.Int).Mul(new(big.Int).SetUint64(gas), tx.GasPrice())
// 		cost := new(big.Int).Add(tx.Amount(), fee)
// 		// to := manager.GetAccount(*tx.To())
// 		// make sure the change log has right order
// 		if fromAddr.Hex() < tx.To().Hex() {
// 			fromBalance.Set(new(big.Int).Sub(fromBalance, cost))
// 			toBalance.Set(new(big.Int).Add(toBalance, tx.Amount()))
// 		} else {
// 			toBalance.Set(new(big.Int).Add(toBalance, tx.Amount()))
// 			fromBalance.Set(new(big.Int).Sub(fromBalance, cost))
// 		}
// 		salary.Add(salary, fee)
// 	}
// // }
