package consensus

import (
	"encoding/json"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/txpool"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

const (
	block01MinerAddress = "0x015780F8456F9c1532645087a19DcF9a7e0c7F97"
	deputy01Privkey     = "0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"
	block02MinerAddress = "0x016ad4Fc7e1608685Bf5fe5573973BF2B1Ef9B8A"
	deputy02Privkey     = "0x9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb"
	block03MinerAddress = "0x01f98855Be9ecc5c23A28Ce345D2Cc04686f2c61"
	deputy03Privkey     = "0xba9b51e59ec57d66b30b9b868c76d6f4d386ce148d9c6c1520360d92ef0f27ae"
	block04MinerAddress = "0x0112fDDcF0C08132A5dcd9ED77e1a3348ff378D2"
	deputy04Privkey     = "0xb381bad69ad4b200462a0cc08fcb8ba64d26efd4f49933c2c2448cb23f2cd9d0"
	block05MinerAddress = "0x016017aF50F4bB67101CE79298ACBdA1A3c12C15"
	deputy05Privkey     = "0x56b5fe1b8c40f0dec29b621a16ffcbc7a1bb5c0b0f910c5529f991273cd0569c"
)

// loadDpovp 加载一个Dpovp实例
func loadDpovp(dm *deputynode.Manager) *DPoVP {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
	latestBlock, err := db.LoadLatestBlock()
	if err != nil {
		return nil
	}
	d := NewDpovp(config, db, dm, account.NewManager(latestBlock.Hash(), db), NewTestChain(db), txpool.NewTxPool(), latestBlock)
	return d
}

// 初始化代理节点,numNode为选择共识节点数量，取值为[1,5],height为发放奖励高度
func initDeputyNode(numNode int, height uint32, dm *deputynode.Manager) *deputynode.Manager {
	// manager := deputynode.NewManager(5)

	privarte01, err := crypto.ToECDSA(common.FromHex(deputy01Privkey))
	if err != nil {
		panic(err)
	}
	privarte02, err := crypto.ToECDSA(common.FromHex(deputy02Privkey))
	if err != nil {
		panic(err)
	}
	privarte03, err := crypto.ToECDSA(common.FromHex(deputy03Privkey))
	if err != nil {
		panic(err)
	}
	privarte04, err := crypto.ToECDSA(common.FromHex(deputy04Privkey))
	if err != nil {
		panic(err)
	}
	privarte05, err := crypto.ToECDSA(common.FromHex(deputy05Privkey))
	if err != nil {
		panic(err)
	}

	var nodes = make([]*types.DeputyNode, 5)
	nodes[0] = &types.DeputyNode{
		MinerAddress: common.HexToAddress(block01MinerAddress),
		NodeID:       (crypto.FromECDSAPub(&privarte01.PublicKey))[1:],
		Rank:         0,
		Votes:        big.NewInt(120),
	}
	nodes[1] = &types.DeputyNode{
		MinerAddress: common.HexToAddress(block02MinerAddress),
		NodeID:       (crypto.FromECDSAPub(&privarte02.PublicKey))[1:],
		Rank:         1,
		Votes:        big.NewInt(110),
	}
	nodes[2] = &types.DeputyNode{
		MinerAddress: common.HexToAddress(block03MinerAddress),
		NodeID:       (crypto.FromECDSAPub(&privarte03.PublicKey))[1:],
		Rank:         2,
		Votes:        big.NewInt(100),
	}
	nodes[3] = &types.DeputyNode{
		MinerAddress: common.HexToAddress(block04MinerAddress),
		NodeID:       (crypto.FromECDSAPub(&privarte04.PublicKey))[1:],
		Rank:         3,
		Votes:        big.NewInt(90),
	}
	nodes[4] = &types.DeputyNode{
		MinerAddress: common.HexToAddress(block05MinerAddress),
		NodeID:       (crypto.FromECDSAPub(&privarte05.PublicKey))[1:],
		Rank:         4,
		Votes:        big.NewInt(80),
	}

	if numNode > 5 || numNode == 0 {
		panic(fmt.Errorf("overflow index. numNode must be [1,5]"))
	}
	dm.SaveSnapshot(height, nodes[:numNode])

	return dm
}

// 对区块进行签名的函数
func signTestBlock(deputyPrivate string, block *types.Block) {
	// 对区块签名
	private, err := crypto.ToECDSA(common.FromHex(deputyPrivate))
	if err != nil {
		panic(err)
	}
	signBlock, err := crypto.Sign(block.Hash().Bytes(), private)
	if err != nil {
		panic(err)
	}
	block.Header.SignData = signBlock
}

// newSignedBlock 用来生成符合测试用例所用的区块
func newSignedBlock(dpovp *DPoVP, parentHash common.Hash, author common.Address, txs []*types.Transaction, time uint32, signPrivate string, save bool) *types.Block {
	newBlockInfo := blockInfo{
		parentHash: parentHash,
		author:     author,
		txList:     txs,
		gasLimit:   500000000,
		time:       time,
	}
	parent, err := dpovp.db.GetBlockByHash(parentHash)
	if err != nil {
		// genesis block
		newBlockInfo.height = 0
	} else {
		newBlockInfo.height = parent.Height() + 1
	}
	testBlock := makeBlock(dpovp.db, newBlockInfo, save)
	if save {
		if err := dpovp.db.SetStableBlock(testBlock.Hash()); err != nil {
			panic(err)
		}
	}
	// 对区块进行签名
	signTestBlock(signPrivate, testBlock)
	return testBlock
}

// Test_verifyHeaderTime 测试验证区块时间戳函数是否正确
func Test_verifyHeaderTime(t *testing.T) {
	blocks := []types.Block{
		{
			Header: &types.Header{
				ParentHash:   common.Hash{},
				MinerAddress: common.Address{},
				VersionRoot:  common.Hash{},
				TxRoot:       common.Hash{},
				LogRoot:      common.Hash{},
				Height:       0,
				GasLimit:     0,
				GasUsed:      0,
				Time:         uint32(time.Now().Unix() - 2), // 正确时间
				SignData:     nil,
				Extra:        nil,
			},
			Txs:        nil,
			ChangeLogs: nil,
			Confirms:   nil,
		},
		{
			Header: &types.Header{
				ParentHash:   common.Hash{},
				MinerAddress: common.Address{},
				VersionRoot:  common.Hash{},
				TxRoot:       common.Hash{},
				LogRoot:      common.Hash{},
				Height:       0,
				GasLimit:     0,
				GasUsed:      0,
				Time:         uint32(time.Now().Unix() - 1), // 临界点时间
				SignData:     nil,
				Extra:        nil,
			},
			Txs:        nil,
			ChangeLogs: nil,
			Confirms:   nil,
		},
		{
			Header: &types.Header{
				ParentHash:   common.Hash{},
				MinerAddress: common.Address{},
				VersionRoot:  common.Hash{},
				TxRoot:       common.Hash{},
				LogRoot:      common.Hash{},
				Height:       0,
				GasLimit:     0,
				GasUsed:      0,
				Time:         uint32(time.Now().Unix() + 2), // 不正确时间
				SignData:     nil,
				Extra:        nil,
			},
			Txs:        nil,
			ChangeLogs: nil,
			Confirms:   nil,
		},
	}

	err01 := verifyTime(&blocks[0])
	assert.Equal(t, nil, err01)
	err02 := verifyTime(&blocks[1])
	assert.Equal(t, nil, err02)
	err03 := verifyTime(&blocks[2])
	assert.Equal(t, ErrVerifyHeaderFailed, err03)

}

// Test_verifyHeaderSignData 测试验证区块签名数据函数是否正确
func Test_verifyHeaderSignData(t *testing.T) {
	dm := deputynode.NewManager(5)
	dm = initDeputyNode(3, 0, dm) // 选择前三个共识节点
	dpovp := loadDpovp(dm)
	defer dpovp.db.Close()
	// 创建一个块并用另一个节点来对此区块进行签名
	block01 := newSignedBlock(dpovp, common.Hash{}, common.HexToAddress(block01MinerAddress), nil, uint32(time.Now().Unix()), deputy02Privkey, false)
	// header := block01.Header
	assert.Equal(t, ErrVerifyHeaderFailed, verifySigner(dm, block01))
}

// // TestDpovp_nodeCount1 nodeCount = 1 的情况下直接返回nil
func TestDpovp_nodeCount1(t *testing.T) {
	dm := deputynode.NewManager(5)
	dm = initDeputyNode(1, 0, dm)
	dpovp := loadDpovp(dm)
	defer dpovp.db.Close()

	t.Log(dm.GetDeputiesCount(1))
	assert.Equal(t, nil, dpovp.VerifyBeforeTxProcess(&types.Block{Header: &types.Header{Height: 1}}))
}

// 验证区块头Extra字段长度是否正确
func Test_headerExtra(t *testing.T) {
	dm := deputynode.NewManager(5)
	dm = initDeputyNode(3, 0, dm)
	dpovp := loadDpovp(dm)
	defer dpovp.db.Close()
	// 创建一个标准的区块
	testBlcok := newSignedBlock(dpovp, common.Hash{}, common.HexToAddress(block01MinerAddress), nil, uint32(time.Now().Unix()-10), deputy01Privkey, false)
	// 设置区块头中的extra字段长度大于标准长度
	extra := make([]byte, 257)
	testBlcok.Header.Extra = extra
	// 重新对区块进行签名
	signTestBlock(deputy01Privkey, testBlcok)
	// 验证
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyBeforeTxProcess(testBlcok))
}

// TestDpovp_VerifyHeader01 对共识中共识区块与父块关联情况共识的测试
func TestDpovp_VerifyHeader01(t *testing.T) {
	dm := deputynode.NewManager(5)
	dm = initDeputyNode(5, 0, dm)
	t.Log(dm.GetDeputiesCount(1))
	dpovp := loadDpovp(dm)
	defer dpovp.db.Close()
	// 验证不存在父区块的情况
	testBlock00 := newSignedBlock(dpovp, common.Hash{}, common.HexToAddress(block01MinerAddress), nil, uint32(time.Now().Unix()-10), deputy01Privkey, true)
	// header := testBlock00.Header
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyBeforeTxProcess(testBlock00))

	// 验证父区块的高度为0，也就是父区块为创世区块情况
	testBlock01 := newSignedBlock(dpovp, testBlock00.Hash(), common.HexToAddress(block02MinerAddress), nil, uint32(time.Now().Unix()-5), deputy02Privkey, false)

	assert.Equal(t, nil, dpovp.VerifyBeforeTxProcess(testBlock01))
}

// TestDpovp_VerifyHeader03 测试slot == 0,slot == 1,slot > 1的情况
func TestDpovp_VerifyHeader02(t *testing.T) {
	ClearData()
	// 创建5个代理节点
	dm := deputynode.NewManager(5)
	dm = initDeputyNode(5, 0, dm)
	dpovp := loadDpovp(dm)
	defer dpovp.db.Close()
	// 创世块,随便哪个节点出块在这里没有影响
	block00 := newSignedBlock(dpovp, common.Hash{}, common.HexToAddress(block01MinerAddress), nil, 1995, deputy01Privkey, true)
	// parent is genesis block,由第一个节点出块,此区块是作为测试区块的参照区块
	block01 := newSignedBlock(dpovp, block00.Hash(), common.HexToAddress(block01MinerAddress), nil, 2000, deputy01Privkey, true)

	// if slot == 0 :
	// 还是由第一个节点出块,模拟 (if slot == 0 ) 的情况 ,与block01时间差为44s,满足条件(if timeSpan >= oneLoopTime-d.timeoutTime),此区块共识验证通过会返回nil
	block02 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block01MinerAddress), nil, 2044, deputy01Privkey, false)
	assert.Equal(t, nil, dpovp.VerifyBeforeTxProcess(block02))
	// 与block01时间差为33s,小于40s,验证不通过的情况
	block03 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block01MinerAddress), nil, 2033, deputy01Privkey, false)
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyBeforeTxProcess(block03))
	// 测试一个临界值，与block01时间差等于40s的情况
	block04 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block01MinerAddress), nil, 2040, deputy01Privkey, false)
	assert.Equal(t, nil, dpovp.VerifyBeforeTxProcess(block04))

	// else if slot == 1 :
	// 都与block01作为父块, 设置出块代理节点为第二个节点，满足slot == 1,时间差设为第一种小于一轮(50s)的情况,
	// block05时间满足(timeSpan >= d.blockInterval && timeSpan < d.timeoutTime)的正常情况
	block05 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block02MinerAddress), nil, 2005, deputy02Privkey, false)
	assert.Equal(t, nil, dpovp.VerifyBeforeTxProcess(block05))
	// block06 不满足(timeSpan >= d.blockInterval && timeSpan < d.timeoutTime)的情况,timeSpan == 11 > 10
	block06 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block02MinerAddress), nil, 2011, deputy02Privkey, false)
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyBeforeTxProcess(block06))
	// if slot == 1 else 的情况，此情况是timeSpan >= oneLoopTime,时间间隔超过一轮
	// 首先测试 timeSpan % oneLoopTime < timeoutTime 的正常情况
	block07 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block02MinerAddress), nil, 2051, deputy02Privkey, false)
	assert.Equal(t, nil, dpovp.VerifyBeforeTxProcess(block07))
	// 异常情况,timeSpan % oneLoopTime = 20 > timeoutTime
	block08 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block02MinerAddress), nil, 2070, deputy02Privkey, false)
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyBeforeTxProcess(block08))

	// else :
	// slot > 1的情况分析
	// timeSpan/d.timeoutTime == int64(slot-1) , timeSpan与timeoutTime的除数正好是间隔的代理节点数，为正常情况
	// 设置block09为第四个节点出块，与block01出块节点中间相隔2个节点,设置时间timeSpan == 20--29都是符合出块的时间
	block09 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block04MinerAddress), nil, 2025, deputy04Privkey, false)
	assert.Equal(t, nil, dpovp.VerifyBeforeTxProcess(block09))
	// 不符合情况,设置timeSpan >=30 || timeSpan < 20
	// timeSpan >=30 情况， 满足条件 timeSpan/d.timeoutTime != int64(slot-1)
	block10 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block04MinerAddress), nil, 2030, deputy04Privkey, false)
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyBeforeTxProcess(block10))
	// timeSpan < 20 情况， 满足条件 timeSpan/d.timeoutTime != int64(slot-1)
	block11 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block04MinerAddress), nil, 2019, deputy04Privkey, false)
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyBeforeTxProcess(block11))

}

// TestDpovp_Seal
func TestDpovp_Seal(t *testing.T) {
	// 创建5个代理节点
	dm := deputynode.NewManager(5)
	dm = initDeputyNode(5, 0, dm)
	dpovp := loadDpovp(dm)
	defer dpovp.db.Close()
	// 创世块
	block00 := newSignedBlock(dpovp, common.Hash{}, common.HexToAddress(block01MinerAddress), nil, 995, deputy01Privkey, true)

	txs := []*types.Transaction{
		signTransaction(types.NewTransaction(defaultAccounts[0], common.Big1, 2000000, common.Big2, []byte{12}, 0, chainID, 1538210391, "aa", "aaa"), testPrivate),
		makeTransaction(testPrivate, defaultAccounts[1], params.OrdinaryTx, common.Big1, common.Big2, 1538210491, 2000000),
	}
	// parent is genesis block,此区块是作为测试区块的参照区块
	block01 := newSignedBlock(dpovp, block00.Hash(), common.HexToAddress(block01MinerAddress), txs, 1000, deputy01Privkey, true)

	TestBlockHeader := block01.Header // 得到block01头，为生成TestBlock所用
	product := &account.TxsProduct{
		Txs:         txs,
		GasUsed:     block01.GasUsed(),
		ChangeLogs:  block01.ChangeLogs,
		VersionRoot: block01.VersionRoot(),
	}
	TestBlock, err := dpovp.Seal(TestBlockHeader, product, block01.Confirms, nil)
	assert.NoError(t, err)
	assert.Equal(t, block01.Hash(), TestBlock.Hash())
	assert.Equal(t, block01.Txs, TestBlock.Txs)
	assert.Equal(t, block01.ChangeLogs, TestBlock.ChangeLogs)
	assert.Equal(t, types.DeputyNodes(nil), TestBlock.DeputyNodes)
}

// TestDpovp_Finalize
func TestDpovp_Finalize(t *testing.T) {
	dm := deputynode.NewManager(5)
	// 第0届 一个deputy node
	dm = initDeputyNode(1, 0, dm)
	// 第一届
	dm = initDeputyNode(3, params.TermDuration, dm)
	// 第二届
	dm = initDeputyNode(5, 2*params.TermDuration, dm)

	dpovp := loadDpovp(dm)
	defer dpovp.db.Close()
	am := account.NewManager(common.Hash{}, dpovp.db)

	// 设置前0,1,2届的矿工换届奖励
	rewardMap := make(params.RewardsMap)
	num00, _ := new(big.Int).SetString("55555555555555555555", 10)
	num01, _ := new(big.Int).SetString("66666666666666666666", 10)
	num02, _ := new(big.Int).SetString("77777777777777777777", 10)
	rewardMap[0] = &params.Reward{
		Term:  0,
		Value: num00,
		Times: 1,
	}
	rewardMap[1] = &params.Reward{
		Term:  1,
		Value: num01,
		Times: 1,
	}
	rewardMap[2] = &params.Reward{
		Term:  2,
		Value: num02,
		Times: 1,
	}
	data, err := json.Marshal(rewardMap)
	assert.NoError(t, err)
	rewardAcc := am.GetAccount(params.TermRewardContract)
	rewardAcc.SetStorageState(params.TermRewardContract.Hash(), data)
	// 设置deputy node的income address
	term00, err := dm.GetTermByHeight(0)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(term00.Nodes))

	term01, err := dm.GetTermByHeight(params.TermDuration + params.InterimDuration + 1)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(term01.Nodes))

	term02, err := dm.GetTermByHeight(2*params.TermDuration + params.InterimDuration + 1)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(term02.Nodes))

	// miner
	minerAddr00 := term02.Nodes[0].MinerAddress
	minerAddr01 := term02.Nodes[1].MinerAddress
	minerAddr02 := term02.Nodes[2].MinerAddress
	minerAddr03 := term02.Nodes[3].MinerAddress
	minerAddr04 := term02.Nodes[4].MinerAddress
	minerAcc00 := am.GetAccount(minerAddr00)
	minerAcc01 := am.GetAccount(minerAddr01)
	minerAcc02 := am.GetAccount(minerAddr02)
	minerAcc03 := am.GetAccount(minerAddr03)
	minerAcc04 := am.GetAccount(minerAddr04)
	// 设置income address
	incomeAddr00 := common.HexToAddress("0x10000")
	incomeAddr01 := common.HexToAddress("0x10001")
	incomeAddr02 := common.HexToAddress("0x10002")
	incomeAddr03 := common.HexToAddress("0x10003")
	incomeAddr04 := common.HexToAddress("0x10004")
	profile := make(map[string]string)
	profile[types.CandidateKeyIncomeAddress] = incomeAddr00.String()
	minerAcc00.SetCandidate(profile)
	profile[types.CandidateKeyIncomeAddress] = incomeAddr01.String()
	minerAcc01.SetCandidate(profile)
	profile[types.CandidateKeyIncomeAddress] = incomeAddr02.String()
	minerAcc02.SetCandidate(profile)
	profile[types.CandidateKeyIncomeAddress] = incomeAddr03.String()
	minerAcc03.SetCandidate(profile)
	profile[types.CandidateKeyIncomeAddress] = incomeAddr04.String()
	minerAcc04.SetCandidate(profile)

	// 为第0届发放奖励
	err = dpovp.Finalize(params.InterimDuration+params.TermDuration+1, am)
	assert.NoError(t, err)
	// 查看第0届的deputy node 收益地址的balance. 只有第一个deputy node
	incomeAcc00 := am.GetAccount(incomeAddr00)
	value1, _ := new(big.Int).SetString("55000000000000000000", 10)
	assert.Equal(t, value1, incomeAcc00.GetBalance())

	// 	为第二届发放奖励
	err = dpovp.Finalize(2*params.TermDuration+params.InterimDuration+1, am)
	assert.NoError(t, err)
	// 查看第二届的deputy node 收益地址的balance.前三个deputy node
	value2, _ := new(big.Int).SetString("79000000000000000000", 10)
	assert.Equal(t, value2, incomeAcc00.GetBalance())

	incomeAcc01 := am.GetAccount(incomeAddr01)
	value3, _ := new(big.Int).SetString("22000000000000000000", 10)
	assert.Equal(t, value3, incomeAcc01.GetBalance())

	incomeAcc02 := am.GetAccount(incomeAddr02)
	value4, _ := new(big.Int).SetString("20000000000000000000", 10)
	assert.Equal(t, value4, incomeAcc02.GetBalance())

	// 	为第三届的deputy nodes 发放奖励 5个deputy node
	err = dpovp.Finalize(3*params.TermDuration+params.InterimDuration+1, am)
	assert.NoError(t, err)
	//
	value5, _ := new(big.Int).SetString("97000000000000000000", 10)
	assert.Equal(t, value5, incomeAcc00.GetBalance())

	value6, _ := new(big.Int).SetString("39000000000000000000", 10)
	assert.Equal(t, value6, incomeAcc01.GetBalance())

	value7, _ := new(big.Int).SetString("35000000000000000000", 10)
	assert.Equal(t, value7, incomeAcc02.GetBalance())

	incomeAcc03 := am.GetAccount(incomeAddr03)
	value8, _ := new(big.Int).SetString("13000000000000000000", 10)
	assert.Equal(t, value8, incomeAcc03.GetBalance())

	incomeAcc04 := am.GetAccount(incomeAddr04)
	value9, _ := new(big.Int).SetString("12000000000000000000", 10)
	assert.Equal(t, value9, incomeAcc04.GetBalance())

}

func Test_calculateSalary(t *testing.T) {
	tests := []struct {
		Expect, TotalSalary, DeputyVotes, TotalVotes, Precision int64
	}{
		// total votes=100
		{0, 100, 0, 100, 1},
		{1, 100, 1, 100, 1},
		{2, 100, 2, 100, 1},
		{100, 100, 100, 100, 1},
		// total votes=100, precision=10
		{0, 100, 1, 100, 10},
		{10, 100, 10, 100, 10},
		{10, 100, 11, 100, 10},
		// total votes=1000
		{0, 100, 1, 1000, 1},
		{0, 100, 9, 1000, 1},
		{1, 100, 10, 1000, 1},
		{1, 100, 11, 1000, 1},
		{100, 100, 1000, 1000, 1},
		// total votes=1000, precision=10
		{10, 100, 100, 1000, 10},
		{10, 100, 120, 1000, 10},
		{20, 100, 280, 1000, 10},
		// total votes=10
		{0, 100, 0, 10, 1},
		{10, 100, 1, 10, 1},
		{100, 100, 10, 10, 1},
		// total votes=10, precision=10
		{10, 100, 1, 10, 10},
		{100, 100, 10, 10, 10},
	}
	for _, test := range tests {
		expect := big.NewInt(test.Expect)
		totalSalary := big.NewInt(test.TotalSalary)
		deputyVotes := big.NewInt(test.DeputyVotes)
		totalVotes := big.NewInt(test.TotalVotes)
		precision := big.NewInt(test.Precision)
		assert.Equalf(t, 0, calculateSalary(totalSalary, deputyVotes, totalVotes, precision).Cmp(expect), "calculateSalary(%v, %v, %v, %v)", totalSalary, deputyVotes, totalVotes, precision)
	}
}

// Test_DivideSalary test total salary with random data
func Test_DivideSalary(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)

	r := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < 100; i++ {
		nodeCount := r.Intn(49) + 1 // [1, 50]
		nodes := GenerateDeputies(nodeCount, am)
		for _, node := range nodes {
			node.Votes = randomBigInt(r)
		}

		totalSalary := randomBigInt(r)
		term := &deputynode.TermRecord{TermIndex: 0, Nodes: nodes}

		salaries := DivideSalary(totalSalary, am, term)
		assert.Len(t, salaries, nodeCount)

		// 验证income是否相同
		for j := 0; j < len(nodes); j++ {
			if getIncomeAddressFromDeputyNode(am, nodes[j]) != salaries[j].Address {
				panic("income address no equal")
			}
		}
		actualTotal := new(big.Int)
		for _, s := range salaries {
			actualTotal.Add(actualTotal, s.Salary)
		}
		totalVotes := new(big.Int)
		for _, v := range nodes {
			totalVotes.Add(totalVotes, v.Votes)
		}
		// 比较每个deputy node salary
		for k := 0; k < len(nodes); k++ {
			if salaries[k].Salary.Cmp(calculateSalary(totalSalary, nodes[k].Votes, totalVotes, minPrecision)) != 0 {
				panic("deputy node salary no equal")
			}
		}

		// errRange = nodeCount * minPrecision
		// actualTotal must be in range [totalSalary - errRange, totalSalary]
		errRange := new(big.Int).Mul(big.NewInt(int64(nodeCount)), minPrecision)
		assert.Equal(t, true, actualTotal.Cmp(new(big.Int).Sub(totalSalary, errRange)) >= 0)
		assert.Equal(t, true, actualTotal.Cmp(totalSalary) <= 0)
	}
}

// GenerateDeputies generate random deputy nodes
func GenerateDeputies(num int, am *account.Manager) types.DeputyNodes {
	var result []*types.DeputyNode
	for i := 0; i < num; i++ {
		private, _ := crypto.GenerateKey()
		node := &types.DeputyNode{
			MinerAddress: crypto.PubkeyToAddress(private.PublicKey),
			NodeID:       (crypto.FromECDSAPub(&private.PublicKey))[1:],
			Rank:         uint32(i),
			Votes:        big.NewInt(int64(10000000000 - i)),
		}
		result = append(result, node)
		private, _ = crypto.GenerateKey()
		incomeAddress := crypto.PubkeyToAddress(private.PublicKey)
		setIncomeAddress(am, node, incomeAddress)
	}
	return result
}

func setIncomeAddress(am *account.Manager, node *types.DeputyNode, incomeAddress common.Address) {
	profile := make(map[string]string)
	// 设置deputy node 的income address
	minerAcc := am.GetAccount(node.MinerAddress)
	// 设置income address 为minerAddress
	profile[types.CandidateKeyIncomeAddress] = incomeAddress.String()
	minerAcc.SetCandidate(profile)

}

func randomBigInt(r *rand.Rand) *big.Int {
	return new(big.Int).Mul(big.NewInt(r.Int63()), big.NewInt(r.Int63()))
}
