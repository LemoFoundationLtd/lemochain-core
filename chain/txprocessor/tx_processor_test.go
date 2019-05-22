package txprocessor

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
	"testing"
	"time"
)

func TestNewTxProcessor(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	bc := newTestChain(db)
	p := NewTxProcessor(config, bc, am, db)
	assert.Equal(t, chainID, p.ChainID)
	assert.Equal(t, config.RewardManager, p.cfg.RewardManager)
	assert.False(t, p.cfg.Debug)
}

// test valid block processing
// func TestTxProcessor_Process(t *testing.T) {
// 	ClearData()
// 	db := newDbForTest()
// 	defer db.Close()
// 	am := account.NewManager(testBlockHash, db)
// 	bc := newTestChain(db)
// 	p := NewTxProcessor(config, bc, am, db)
//
// 	p.am.GetAccount(testAddr)
// 	// last not stable block
// 	block := defaultBlocks[2]
// 	gasUsed, err := p.Process(block.Header, block.Txs)
// 	assert.NoError(t, err)
// 	assert.Equal(t, block.Header.GasUsed, gasUsed)
// 	err = p.am.Finalise()
// 	assert.NoError(t, err)
// 	assert.Equal(t, block.Header.VersionRoot, p.am.GetVersionRoot())
// 	assert.Equal(t, len(block.ChangeLogs), len(p.am.GetChangeLogs()))
// 	p.am.GetAccount(testAddr)
//
// 	// block not in db
// 	block = defaultBlocks[3]
// 	gasUsed, err = p.Process(block.Header, block.Txs)
// 	assert.NoError(t, err)
// 	err = p.am.Finalise()
// 	assert.NoError(t, err)
// 	assert.Equal(t, block.Header.GasUsed, gasUsed)
// 	// TODO these test is fail because Account.GetNextVersion always +1, so that the ChangeLog is not continuous. This will be fixed if we refactor the logic of ChangeLog merging to makes all version continuous in account.
// 	assert.Equal(t, block.Header.VersionRoot, p.am.GetVersionRoot())
// 	assert.Equal(t, len(block.ChangeLogs), len(p.am.GetChangeLogs()))
// 	p.am.GetAccount(testAddr)
//
// 	// genesis block
// 	block = defaultBlocks[0]
// 	_, err = p.Process(block.Header, block.Txs)
// 	assert.Equal(t, err, ErrInvalidGenesis)
//
// 	// block on fork branch
// 	block = createNewBlock(db)
// 	gasUsed, err = p.Process(block.Header, block.Txs)
// 	assert.NoError(t, err)
// 	err = p.am.Finalise()
// 	assert.NoError(t, err)
// 	assert.Equal(t, block.Header.GasUsed, gasUsed)
// 	assert.Equal(t, block.Header.VersionRoot, p.am.GetVersionRoot())
// 	assert.Equal(t, len(block.ChangeLogs), len(p.am.GetChangeLogs()))
// }

// test invalid block processing
// func TestTxProcessor_Process2(t *testing.T) {
// 	ClearData()
// 	db := newDB()
// 	defer db.Close()
// 	latestBlock, err := db.LoadLatestBlock()
// 	assert.NoError(t, err)
// 	am := account.NewManager(latestBlock.Hash(), db)
// 	p := NewTxProcessor(config, NewTestChain(db), am, db)
//
// 	// tamper with amount
// 	block := createNewBlock(db)
// 	rawTx, _ := rlp.EncodeToBytes(block.Txs[0])
// 	rawTx[29]++ // amount++
// 	cpy := new(types.Transaction)
// 	err = rlp.DecodeBytes(rawTx, cpy)
// 	assert.NoError(t, err)
// 	assert.Equal(t, new(big.Int).Add(block.Txs[0].Amount(), big.NewInt(1)), cpy.Amount())
// 	block.Txs[0] = cpy
// 	_, err = p.Process(block.Header, block.Txs)
// 	assert.Equal(t, ErrInvalidTxInBlock, err)
//
// 	// invalid signature
// 	block = createNewBlock(db)
// 	rawTx, _ = rlp.EncodeToBytes(block.Txs[0])
// 	rawTx[43] = 0 // invalid S
// 	cpy = new(types.Transaction)
// 	err = rlp.DecodeBytes(rawTx, cpy)
// 	assert.NoError(t, err)
// 	block.Txs[0] = cpy
// 	_, err = p.Process(block.Header, block.Txs)
// 	assert.Equal(t, ErrInvalidTxInBlock, err)
//
// 	// not enough gas (resign by another address)
// 	block = createNewBlock(db)
// 	private, _ := crypto.GenerateKey()
// 	origFrom, _ := block.Txs[0].From()
// 	block.Txs[0] = signTransaction(block.Txs[0], private)
// 	newFrom, _ := block.Txs[0].From()
// 	assert.NotEqual(t, origFrom, newFrom)
// 	block.Header.TxRoot = block.Txs.MerkleRootSha()
// 	_, err = p.Process(block.Header, block.Txs)
// 	assert.Equal(t, ErrInvalidTxInBlock, err)
//
// 	// exceed block gas limit
// 	block = createNewBlock(db)
// 	block.Header.GasLimit = 1
// 	_, err = p.Process(block.Header, block.Txs)
// 	assert.Equal(t, ErrInvalidTxInBlock, err)
//
// 	// used gas reach limit in some tx
// 	block = createNewBlock(db)
// 	block.Txs[0] = makeTransaction(testPrivate, defaultAccounts[1], params.OrdinaryTx, big.NewInt(100), common.Big1, 0, 1)
// 	block.Header.TxRoot = block.Txs.MerkleRootSha()
// 	_, err = p.Process(block.Header, block.Txs)
// 	assert.Equal(t, ErrInvalidTxInBlock, err)
//
// 	// balance not enough
// 	block = createNewBlock(db)
// 	balance := p.am.GetAccount(testAddr).GetBalance()
// 	block.Txs[0] = makeTx(testPrivate, defaultAccounts[1], params.OrdinaryTx, new(big.Int).Add(balance, big.NewInt(1)))
// 	block.Header.TxRoot = block.Txs.MerkleRootSha()
// 	_, err = p.Process(block.Header, block.Txs)
// 	assert.Equal(t, ErrInvalidTxInBlock, err)
//
// 	// TODO test create or call contract fail
// }

// test tx picking logic
// func TestTxProcessor_ApplyTxs(t *testing.T) {
// 	ClearData()
// 	db := newDB()
// 	defer db.Close()
// 	latestBlock, err := db.LoadLatestBlock()
// 	assert.NoError(t, err)
// 	am := account.NewManager(latestBlock.Hash(), db)
// 	p := NewTxProcessor(config, NewTestChain(db), am, db)
//
// 	// 1 txs
// 	header := defaultBlocks[2].Header
// 	txs := defaultBlocks[2].Txs
// 	emptyHeader := &types.Header{
// 		ParentHash:   header.ParentHash,
// 		MinerAddress: header.MinerAddress,
// 		Height:       header.Height,
// 		GasLimit:     header.GasLimit,
// 		GasUsed:      header.GasUsed,
// 		Time:         header.Time,
// 	}
// 	selectedTxs, invalidTxs, gasUsed := p.ApplyTxs(emptyHeader, txs, int64(10000))
// 	p.am.MergeChangeLogs()
// 	p.am.Finalise()
// 	assert.Equal(t, header.GasUsed, gasUsed)
// 	assert.Equal(t, header.VersionRoot, p.am.GetVersionRoot())
// 	assert.Equal(t, defaultBlocks[2].ChangeLogs, p.am.GetChangeLogs())
// 	assert.Equal(t, len(txs), len(selectedTxs))
// 	assert.Equal(t, 0, len(invalidTxs))
//
// 	// 2 txs
// 	header = defaultBlocks[3].Header
// 	txs = defaultBlocks[3].Txs
// 	emptyHeader = &types.Header{
// 		ParentHash:   header.ParentHash,
// 		MinerAddress: header.MinerAddress,
// 		Height:       header.Height,
// 		GasLimit:     header.GasLimit,
// 		GasUsed:      header.GasUsed,
// 		Time:         header.Time,
// 	}
// 	selectedTxs, invalidTxs, gasUsed = p.ApplyTxs(emptyHeader, txs, int64(10000))
// 	p.am.MergeChangeLogs()
// 	p.am.Finalise()
// 	assert.Equal(t, header.GasUsed, gasUsed)
// 	assert.Equal(t, header.VersionRoot, p.am.GetVersionRoot())
// 	assert.Equal(t, defaultBlocks[3].ChangeLogs, p.am.GetChangeLogs())
// 	assert.Equal(t, len(txs), len(selectedTxs))
// 	assert.Equal(t, 0, len(invalidTxs))
//
// 	// 0 txs
// 	header = defaultBlocks[3].Header
// 	emptyHeader = &types.Header{
// 		ParentHash:   header.ParentHash,
// 		MinerAddress: header.MinerAddress,
// 		Height:       header.Height,
// 		GasLimit:     header.GasLimit,
// 		GasUsed:      header.GasUsed,
// 		Time:         header.Time,
// 	}
// 	p.am.Reset(emptyHeader.ParentHash)
// 	author := p.am.GetAccount(header.MinerAddress)
// 	origBalance := author.GetBalance()
// 	selectedTxs, invalidTxs, gasUsed = p.ApplyTxs(emptyHeader, nil, int64(10000))
// 	p.am.MergeChangeLogs()
// 	p.am.Finalise()
// 	assert.Equal(t, uint64(0), gasUsed)
// 	assert.Equal(t, defaultBlocks[2].VersionRoot(), p.am.GetVersionRoot()) // last block version root
// 	assert.Equal(t, 0, len(selectedTxs))
// 	assert.Equal(t, *origBalance, *author.GetBalance())
// 	assert.Equal(t, 0, len(p.am.GetChangeLogs()))
//
// 	// too many txs
// 	header = defaultBlocks[3].Header
// 	txs = defaultBlocks[3].Txs
// 	emptyHeader = &types.Header{
// 		ParentHash:   header.ParentHash,
// 		MinerAddress: header.MinerAddress,
// 		Height:       header.Height,
// 		GasLimit:     45000, // Every transaction's gasLimit is 30000. So the block only contains one transaction.
// 		GasUsed:      header.GasUsed,
// 		Time:         header.Time,
// 	}
// 	selectedTxs, invalidTxs, gasUsed = p.ApplyTxs(emptyHeader, txs, int64(10000))
// 	p.am.MergeChangeLogs()
// 	p.am.Finalise()
// 	assert.NotEqual(t, header.GasUsed, gasUsed)
// 	assert.NotEqual(t, header.VersionRoot, p.am.GetVersionRoot())
// 	assert.NotEqual(t, true, len(defaultBlocks[3].ChangeLogs), len(p.am.GetChangeLogs()))
// 	assert.NotEqual(t, len(txs), len(selectedTxs))
// 	assert.Equal(t, 0, len(invalidTxs))
//
// 	// balance not enough
// 	header = defaultBlocks[3].Header
// 	txs = defaultBlocks[3].Txs
// 	emptyHeader = &types.Header{
// 		ParentHash:   header.ParentHash,
// 		MinerAddress: header.MinerAddress,
// 		Height:       header.Height,
// 		GasLimit:     header.GasLimit,
// 		GasUsed:      header.GasUsed,
// 		Time:         header.Time,
// 	}
// 	p.am.Reset(emptyHeader.ParentHash)
// 	balance := p.am.GetAccount(testAddr).GetBalance()
// 	txs = types.Transactions{
// 		txs[0],
// 		makeTx(testPrivate, defaultAccounts[1], params.OrdinaryTx, new(big.Int).Add(balance, big.NewInt(1))),
// 		txs[1],
// 	}
// 	selectedTxs, invalidTxs, _ = p.ApplyTxs(emptyHeader, txs, int64(10000))
// 	p.am.MergeChangeLogs()
// 	p.am.Finalise()
// 	assert.Equal(t, len(txs)-1, len(selectedTxs))
// 	assert.Equal(t, 1, len(invalidTxs))
// }

// TODO move these cases to evm
// test different transactions
// func TestTxProcessor_ApplyTxs2(t *testing.T) {
// 	ClearData()
// 	db := newDB()
// 	defer db.Close()
// 	latestBlock, err := db.LoadLatestBlock()
// 	assert.NoError(t, err)
// 	am := account.NewManager(latestBlock.Hash(), db)
// 	p := NewTxProcessor(config, NewTestChain(db), am, db)
//
// 	// transfer to other
// 	header := defaultBlocks[3].Header
// 	emptyHeader := &types.Header{
// 		ParentHash:   header.ParentHash,
// 		MinerAddress: header.MinerAddress,
// 		Height:       header.Height,
// 		GasLimit:     header.GasLimit,
// 		GasUsed:      header.GasUsed,
// 		Time:         header.Time,
// 	}
// 	p.am.Reset(emptyHeader.ParentHash)
// 	senderBalance := p.am.GetAccount(testAddr).GetBalance()
// 	minerBalance := p.am.GetAccount(defaultAccounts[0]).GetBalance()
// 	recipientBalance := p.am.GetAccount(defaultAccounts[1]).GetBalance()
// 	txs := types.Transactions{
// 		makeTx(testPrivate, defaultAccounts[1], params.OrdinaryTx, common.Big1),
// 	}
// 	_, _, GasUsed := p.ApplyTxs(emptyHeader, txs, int64(10000))
// 	assert.Equal(t, params.TxGas, GasUsed)
// 	newSenderBalance := p.am.GetAccount(testAddr).GetBalance()
// 	newMinerBalance := p.am.GetAccount(defaultAccounts[0]).GetBalance()
// 	newRecipientBalance := p.am.GetAccount(defaultAccounts[1]).GetBalance()
// 	cost := txs[0].GasPrice().Mul(txs[0].GasPrice(), big.NewInt(int64(params.TxGas)))
// 	senderBalance.Sub(senderBalance, cost)
// 	senderBalance.Sub(senderBalance, common.Big1)
// 	assert.Equal(t, senderBalance, newSenderBalance)
// 	assert.Equal(t, minerBalance.Add(minerBalance, cost), newMinerBalance)
// 	assert.Equal(t, recipientBalance.Add(recipientBalance, common.Big1), newRecipientBalance)
//
// 	// transfer to self, only cost gas
// 	header = defaultBlocks[3].Header
// 	emptyHeader = &types.Header{
// 		ParentHash:   header.ParentHash,
// 		MinerAddress: header.MinerAddress,
// 		Height:       header.Height,
// 		GasLimit:     header.GasLimit,
// 		GasUsed:      header.GasUsed,
// 		Time:         header.Time,
// 	}
// 	p.am.Reset(emptyHeader.ParentHash)
// 	senderBalance = p.am.GetAccount(testAddr).GetBalance()
// 	txs = types.Transactions{
// 		makeTx(testPrivate, testAddr, params.OrdinaryTx, common.Big1),
// 	}
// 	_, _, GasUsed = p.ApplyTxs(emptyHeader, txs, int64(10000))
// 	assert.Equal(t, params.TxGas, GasUsed)
// 	newSenderBalance = p.am.GetAccount(testAddr).GetBalance()
// 	cost = txs[0].GasPrice().Mul(txs[0].GasPrice(), big.NewInt(int64(params.TxGas)))
// 	assert.Equal(t, senderBalance.Sub(senderBalance, cost), newSenderBalance)
// }

func TestApplyTxsTimeoutTime(t *testing.T) {
	ClearData()
	db, genesisHash := newCoverGenesisDB()
	defer db.Close()
	am := account.NewManager(genesisHash, db)

	p := NewTxProcessor(config, newTestChain(db), am, db)

	parentBlock, err := db.LoadLatestBlock()
	assert.NoError(t, err)
	header := &types.Header{
		ParentHash:   parentBlock.Hash(),
		MinerAddress: parentBlock.MinerAddress(),
		Height:       parentBlock.Height() + 1,
		GasLimit:     parentBlock.GasLimit(),
	}

	tx01 := makeTx(godPrivate, common.HexToAddress("0x11223"), params.OrdinaryTx, common.Big1)
	txNum := 500
	txs := make([]*types.Transaction, 0, txNum)
	for i := 0; i < txNum; i++ {
		txs = append(txs, tx01)
	}
	selectedTxs01, _, _ := p.ApplyTxs(header, txs, int64(0))
	assert.NotEqual(t, len(selectedTxs01), txNum)
	selectedTxs02, _, _ := p.ApplyTxs(header, txs, int64(2))
	assert.NotEqual(t, len(selectedTxs02), txNum)
	selectedTxs03, _, _ := p.ApplyTxs(header, txs, int64(100))
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
		Tx01               = makeTx(godPrivate, gasPayerAddr, params.OrdinaryTx, params.RegisterCandidateNodeFees) // 转账1000LEMO给gasPayerAddr
		Tx02               = makeTx(godPrivate, senderAddr, params.OrdinaryTx, params.RegisterCandidateNodeFees)   // 转账1000LEMO给senderAddr

		amountReceiver = common.HexToAddress("0x1234")
		TxV01          = types.NewReimbursementTransaction(amountReceiver, gasPayerAddr, params.RegisterCandidateNodeFees, []byte{}, params.OrdinaryTx, chainID, uint64(time.Now().Unix()+300), "", "")
	)
	ClearData()
	db, genesisHash := newCoverGenesisDB()
	defer db.Close()
	am := account.NewManager(genesisHash, db)
	p := NewTxProcessor(config, newTestChain(db), am, db)

	// create a block contains two account which used to make reimbursement transaction
	_ = addBlockToDB(1, types.Transactions{Tx01, Tx02}, am, db)

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
	_ = addBlockToDB(2, types.Transactions{lastSignTxV}, am, db)

	// check their balance
	endGasPayerBalance := p.am.GetCanonicalAccount(gasPayerAddr).GetBalance()
	endTxSenderBalance := p.am.GetCanonicalAccount(senderAddr).GetBalance()
	assert.Equal(t, big.NewInt(0), endTxSenderBalance)
	assert.Equal(t, endGasPayerBalance, new(big.Int).Sub(initGasPayerBalance, big.NewInt(int64(params.TxGas))))
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

	gasUsed, err := IntrinsicGas(data, false)
	assert.NoError(t, err)
	t.Logf("max gasUsed : %d", gasUsed)
}

// Test_setRewardTx 设置矿工奖励交易
func Test_setRewardTx(t *testing.T) {
	ClearData()
	db, genesisHash := newCoverGenesisDB()
	defer db.Close()
	am := account.NewManager(genesisHash, db)
	p := NewTxProcessor(config, newTestChain(db), am, db)

	// 设置第0届的矿工奖励
	data := setRewardTxData(0, new(big.Int).Div(params.TermRewardPoolTotal, common.Big2))
	// private, err := crypto.HexToECDSA("c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa")
	// assert.NoError(t, err)
	TxV01 := types.NewReimbursementTransaction(params.TermRewardContract, godAddr, nil, data, params.OrdinaryTx, chainID, uint64(time.Now().Unix()+300), "", "")
	firstSignTxV, err := types.MakeReimbursementTxSigner().SignTx(TxV01, godPrivate)
	assert.NoError(t, err)
	firstSignTxV = types.GasPayerSignatureTx(firstSignTxV, common.Big1, uint64(60000))
	lastSignTxV, err := types.MakeGasPayerSigner().SignTx(firstSignTxV, godPrivate)
	assert.NoError(t, err)
	txs := types.Transactions{lastSignTxV}

	Block02 := addBlockToDB(1, txs, am, db)
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
// 		gas := params.TxGas + params.TxDataNonZeroGas*uint64(len("abc"))
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
