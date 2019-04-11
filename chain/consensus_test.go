package chain

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"math/big"
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
func loadDpovp(dm *deputynode.Manager) *Dpovp {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
	d := NewDpovp(10*1000, dm, db)
	return d
}

// 初始化代理节点,numNode为选择共识节点数量，取值为[1,5],height为发放奖励高度
func initDeputyNode(numNode int, height uint32) *deputynode.Manager {
	manager := deputynode.NewManager(5)

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

	var nodes = make([]*deputynode.DeputyNode, 5)
	nodes[0] = &deputynode.DeputyNode{
		MinerAddress: common.HexToAddress(block01MinerAddress),
		NodeID:       (crypto.FromECDSAPub(&privarte01.PublicKey))[1:],
		IP:           nil,
		Port:         7001,
		Rank:         0,
		Votes:        big.NewInt(120),
	}
	nodes[1] = &deputynode.DeputyNode{
		MinerAddress: common.HexToAddress(block02MinerAddress),
		NodeID:       (crypto.FromECDSAPub(&privarte02.PublicKey))[1:],
		IP:           nil,
		Port:         7002,
		Rank:         1,
		Votes:        big.NewInt(110),
	}
	nodes[2] = &deputynode.DeputyNode{
		MinerAddress: common.HexToAddress(block03MinerAddress),
		NodeID:       (crypto.FromECDSAPub(&privarte03.PublicKey))[1:],
		IP:           nil,
		Port:         7003,
		Rank:         2,
		Votes:        big.NewInt(100),
	}
	nodes[3] = &deputynode.DeputyNode{
		MinerAddress: common.HexToAddress(block04MinerAddress),
		NodeID:       (crypto.FromECDSAPub(&privarte04.PublicKey))[1:],
		IP:           nil,
		Port:         7004,
		Rank:         3,
		Votes:        big.NewInt(90),
	}
	nodes[4] = &deputynode.DeputyNode{
		MinerAddress: common.HexToAddress(block05MinerAddress),
		NodeID:       (crypto.FromECDSAPub(&privarte05.PublicKey))[1:],
		IP:           nil,
		Port:         7005,
		Rank:         4,
		Votes:        big.NewInt(80),
	}

	if numNode > 5 || numNode == 0 {
		panic(fmt.Errorf("overflow index. numNode must be [1,5]"))
	}
	manager.SaveSnapshot(height, nodes[:numNode])

	return manager
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
func newSignedBlock(dpovp *Dpovp, parentHash common.Hash, author common.Address, txs []*types.Transaction, time uint32, signPrivate string, save bool) *types.Block {
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

	err01 := verifyHeaderTime(&blocks[0])
	assert.Equal(t, nil, err01)
	err02 := verifyHeaderTime(&blocks[1])
	assert.Equal(t, nil, err02)
	err03 := verifyHeaderTime(&blocks[2])
	assert.Equal(t, ErrVerifyHeaderFailed, err03)

}

// Test_verifyHeaderSignData 测试验证区块签名数据函数是否正确
func Test_verifyHeaderSignData(t *testing.T) {
	dm := initDeputyNode(3, 0) // 选择前三个共识节点
	dpovp := loadDpovp(dm)
	defer dpovp.db.Close()
	// 创建一个块并用另一个节点来对此区块进行签名
	block01 := newSignedBlock(dpovp, common.Hash{}, common.HexToAddress(block01MinerAddress), nil, uint32(time.Now().Unix()), deputy02Privkey, false)
	// header := block01.Header
	assert.Equal(t, ErrVerifyHeaderFailed, verifyHeaderSignData(dm, block01))
}

// // TestDpovp_nodeCount1 nodeCount = 1 的情况下直接返回nil
func TestDpovp_nodeCount1(t *testing.T) {
	dm := initDeputyNode(1, 0)
	dpovp := loadDpovp(dm)
	defer dpovp.db.Close()

	t.Log(dm.GetDeputiesCount(1))
	assert.Equal(t, nil, dpovp.VerifyHeader(&types.Block{Header: &types.Header{Height: 1}}))
}

// 验证区块头Extra字段长度是否正确
func Test_headerExtra(t *testing.T) {
	dm := initDeputyNode(3, 0)
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
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyHeader(testBlcok))
}

// TestDpovp_VerifyHeader01 对共识中共识区块与父块关联情况共识的测试
func TestDpovp_VerifyHeader01(t *testing.T) {
	dm := initDeputyNode(5, 0)
	t.Log(dm.GetDeputiesCount(1))
	dpovp := loadDpovp(dm)
	defer dpovp.db.Close()
	// 验证不存在父区块的情况
	testBlock00 := newSignedBlock(dpovp, common.Hash{}, common.HexToAddress(block01MinerAddress), nil, uint32(time.Now().Unix()-10), deputy01Privkey, true)
	// header := testBlock00.Header
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyHeader(testBlock00))

	// 验证父区块的高度为0，也就是父区块为创世区块情况
	testBlock01 := newSignedBlock(dpovp, testBlock00.Hash(), common.HexToAddress(block02MinerAddress), nil, uint32(time.Now().Unix()-5), deputy02Privkey, false)

	assert.Equal(t, nil, dpovp.VerifyHeader(testBlock01))
}

// TestDpovp_VerifyHeader03 测试slot == 0,slot == 1,slot > 1的情况
func TestDpovp_VerifyHeader02(t *testing.T) {
	ClearData()
	// 创建5个代理节点
	dm := initDeputyNode(5, 0)
	dpovp := loadDpovp(dm)
	defer dpovp.db.Close()
	// 创世块,随便哪个节点出块在这里没有影响
	block00 := newSignedBlock(dpovp, common.Hash{}, common.HexToAddress(block01MinerAddress), nil, 1995, deputy01Privkey, true)
	// parent is genesis block,由第一个节点出块,此区块是作为测试区块的参照区块
	block01 := newSignedBlock(dpovp, block00.Hash(), common.HexToAddress(block01MinerAddress), nil, 2000, deputy01Privkey, true)

	// if slot == 0 :
	// 还是由第一个节点出块,模拟 (if slot == 0 ) 的情况 ,与block01时间差为44s,满足条件(if timeSpan >= oneLoopTime-d.timeoutTime),此区块共识验证通过会返回nil
	block02 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block01MinerAddress), nil, 2044, deputy01Privkey, false)
	assert.Equal(t, nil, dpovp.VerifyHeader(block02))
	// 与block01时间差为33s,小于40s,验证不通过的情况
	block03 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block01MinerAddress), nil, 2033, deputy01Privkey, false)
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyHeader(block03))
	// 测试一个临界值，与block01时间差等于40s的情况
	block04 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block01MinerAddress), nil, 2040, deputy01Privkey, false)
	assert.Equal(t, nil, dpovp.VerifyHeader(block04))

	// else if slot == 1 :
	// 都与block01作为父块, 设置出块代理节点为第二个节点，满足slot == 1,时间差设为第一种小于一轮(50s)的情况,
	// block05时间满足(timeSpan >= d.blockInterval && timeSpan < d.timeoutTime)的正常情况
	block05 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block02MinerAddress), nil, 2005, deputy02Privkey, false)
	assert.Equal(t, nil, dpovp.VerifyHeader(block05))
	// block06 不满足(timeSpan >= d.blockInterval && timeSpan < d.timeoutTime)的情况,timeSpan == 11 > 10
	block06 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block02MinerAddress), nil, 2011, deputy02Privkey, false)
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyHeader(block06))
	// if slot == 1 else 的情况，此情况是timeSpan >= oneLoopTime,时间间隔超过一轮
	// 首先测试 timeSpan % oneLoopTime < timeoutTime 的正常情况
	block07 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block02MinerAddress), nil, 2051, deputy02Privkey, false)
	assert.Equal(t, nil, dpovp.VerifyHeader(block07))
	// 异常情况,timeSpan % oneLoopTime = 20 > timeoutTime
	block08 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block02MinerAddress), nil, 2070, deputy02Privkey, false)
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyHeader(block08))

	// else :
	// slot > 1的情况分析
	// timeSpan/d.timeoutTime == int64(slot-1) , timeSpan与timeoutTime的除数正好是间隔的代理节点数，为正常情况
	// 设置block09为第四个节点出块，与block01出块节点中间相隔2个节点,设置时间timeSpan == 20--29都是符合出块的时间
	block09 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block04MinerAddress), nil, 2025, deputy04Privkey, false)
	assert.Equal(t, nil, dpovp.VerifyHeader(block09))
	// 不符合情况,设置timeSpan >=30 || timeSpan < 20
	// timeSpan >=30 情况， 满足条件 timeSpan/d.timeoutTime != int64(slot-1)
	block10 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block04MinerAddress), nil, 2030, deputy04Privkey, false)
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyHeader(block10))
	// timeSpan < 20 情况， 满足条件 timeSpan/d.timeoutTime != int64(slot-1)
	block11 := newSignedBlock(dpovp, block01.Hash(), common.HexToAddress(block04MinerAddress), nil, 2019, deputy04Privkey, false)
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyHeader(block11))

}

// TestDpovp_Seal
func TestDpovp_Seal(t *testing.T) {
	// 创建5个代理节点
	dm := initDeputyNode(5, 0)
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
	assert.Equal(t, deputynode.DeputyNodes(nil), TestBlock.DeputyNodes)
}

// TestDpovp_Finalize todo
func TestDpovp_Finalize(t *testing.T) {
	// 添加第一个共识节点列表,设置共识的节点为前两个节点
	dm := initDeputyNode(2, 0)
	// 添加第二个共识节点列表,设置共识的节点为前三个节点,并设置发放奖励高度为10000,在挖出高度为10000+1000+1的区块的时候为上一轮共识节点发放奖励
	dm = initDeputyNode(3, 10000)
	// 添加第三个共识节点列表,并设置发放奖励高度为20000,在挖出高度为20000+1000+1的区块的时候为上一轮共识节点发放奖励
	dm = initDeputyNode(5, 20000)

	dpovp := loadDpovp(dm)
	defer dpovp.db.Close()
	am := account.NewManager(common.Hash{}, dpovp.db)
	// 测试挖出的块高度不满足发放奖励高度的时候
	err := dpovp.Finalize(9999, am)
	assert.NoError(t, err)
	// dpovp.handOutRewards(9999)
	err = dpovp.Finalize(19998, am)
	assert.NoError(t, err)
	// dpovp.handOutRewards(19998)
	addr1, err := common.StringToAddress(block01MinerAddress)
	assert.NoError(t, err)
	account01 := am.GetAccount(addr1)
	t.Log("When there is no reward,node01Balance = ", account01.GetBalance())

	addr2, err := common.StringToAddress(block02MinerAddress)
	assert.NoError(t, err)
	account02 := am.GetAccount(addr2)
	t.Log("When there is no reward,node01Balance = ", account02.GetBalance())
	// 测试挖出的块高度满足发放奖励高度的时候
	// dpovp.handOutRewards(11001)
	err = dpovp.Finalize(11001, account.NewManager(common.Hash{}, dpovp.db))
	assert.NoError(t, err)
	t.Log("When it comes to giving out rewards,node01Balance = ", account01.GetBalance())
	t.Log("When it comes to giving out rewards,node01Balance = ", account02.GetBalance())
	// 第二轮发放奖励
	// dpovp.handOutRewards(21001)
	err = dpovp.Finalize(21001, account.NewManager(common.Hash{}, dpovp.db))
	assert.NoError(t, err)
	t.Log("When it comes to giving out rewards,node01Balance = ", account01.GetBalance())
	t.Log("When it comes to giving out rewards,node01Balance = ", account02.GetBalance())
}
