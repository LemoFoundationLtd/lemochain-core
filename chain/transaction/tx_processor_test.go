package transaction

import (
	"encoding/json"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/stretchr/testify/assert"
	"math/big"
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
	tx := makeTx(randPrivate, crypto.PubkeyToAddress(randPrivate.PublicKey), godAddr, params.OrdinaryTx, big.NewInt(4000000))
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
		tx := makeTx(godPrivate, godAddr, common.HexToAddress("0x9910"+strconv.Itoa(i)), params.OrdinaryTx, big.NewInt(50000))
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

	tx01 := makeTx(godPrivate, godAddr, common.HexToAddress("0x11223"), params.OrdinaryTx, common.Big1)
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
		Tx01               = makeTx(godPrivate, godAddr, gasPayerAddr, params.OrdinaryTx, params.RegisterCandidateNodeFees) // 转账1000LEMO给gasPayerAddr
		Tx02               = makeTx(godPrivate, godAddr, senderAddr, params.OrdinaryTx, params.RegisterCandidateNodeFees)   // 转账1000LEMO给senderAddr

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
	for i := 0; i <= 8; i++ {
		gas, _ = IntrinsicGas(uint16(i), false, nil)
		switch i {
		case 0:
			assert.Equal(t, params.OrdinaryTxGas, gas)
		case 1:
			assert.Equal(t, params.VoteTxGas, gas)
		case 2:
			assert.Equal(t, params.RegisterTxGas, gas)
		case 3:
			assert.Equal(t, params.CreateAssetTxGas, gas)
		case 4:
			assert.Equal(t, params.IssueAssetTxGas, gas)
		case 5:
			assert.Equal(t, params.ReplenishAssetTxGas, gas)
		case 6:
			assert.Equal(t, params.ModifyAssetTxGas, gas)
		case 7:
			assert.Equal(t, params.TransferAssetTxGas, gas)
		case 8:
			assert.Equal(t, params.SetMultisigAccountTxGas, gas)
		default:
			panic(i)
		}
	}
	// 创建合约
	gas, _ = IntrinsicGas(params.OrdinaryTx, true, nil)
	assert.Equal(t, params.TxGasContractCreation, gas)
	// 测试data中字节全为0
	zeroData := make([]byte, 10)
	zeroData = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	gas, _ = IntrinsicGas(params.OrdinaryTx, false, zeroData)
	assert.Equal(t, params.OrdinaryTxGas+10*params.TxDataZeroGas, gas)
	// 测试data 中字节全不为0的情况
	notZeroData := make([]byte, 10)
	notZeroData = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	gas, _ = IntrinsicGas(params.OrdinaryTx, false, notZeroData)
	assert.Equal(t, params.OrdinaryTxGas+10*params.TxDataNonZeroGas, gas)
	// 测试data一半为0的情况
	halfZeroData := make([]byte, 10)
	halfZeroData = []byte{1, 0, 2, 0, 3, 0, 4, 0, 5, 0}
	gas, _ = IntrinsicGas(params.OrdinaryTx, false, halfZeroData)
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

	gasUsed, err := IntrinsicGas(params.OrdinaryTx, false, data)
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
