package chain

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

const (
	block01LemoBase = "0x015780F8456F9c1532645087a19DcF9a7e0c7F97"
	deputy01Privkey = "0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"
	block02LemoBase = "0x016ad4Fc7e1608685Bf5fe5573973BF2B1Ef9B8A"
	deputy02Privkey = "0x9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb"
	block03LemoBase = "0x01f98855Be9ecc5c23A28Ce345D2Cc04686f2c61"
	deputy03Privkey = "0xba9b51e59ec57d66b30b9b868c76d6f4d386ce148d9c6c1520360d92ef0f27ae"
	block04LemoBase = "0x0112fDDcF0C08132A5dcd9ED77e1a3348ff378D2"
	deputy04Privkey = "0xb381bad69ad4b200462a0cc08fcb8ba64d26efd4f49933c2c2448cb23f2cd9d0"
	block05LemoBase = "0x016017aF50F4bB67101CE79298ACBdA1A3c12C15"
	deputy05Privkey = "0x56b5fe1b8c40f0dec29b621a16ffcbc7a1bb5c0b0f910c5529f991273cd0569c"
)

// loadDpovp 加载一个Dpovp实例
func loadDpovp() *Dpovp {
	store.ClearData()
	db, err := store.NewCacheChain(store.GetStorePath())
	if err != nil {
		panic(err)
	}
	d := NewDpovp(10*1000, db)
	return d
}

// 初始化代理节点,numNode为选择共识节点数量，取值为[1,5],height为发放奖励高度
func initDeputyNode(numNode int, height uint32) error {
	manager := deputynode.Instance()

	privarte01, err := crypto.ToECDSA(common.FromHex(deputy01Privkey))
	if err != nil {
		return err
	}
	privarte02, err := crypto.ToECDSA(common.FromHex(deputy02Privkey))
	if err != nil {
		return err
	}
	privarte03, err := crypto.ToECDSA(common.FromHex(deputy03Privkey))
	if err != nil {
		return err
	}
	privarte04, err := crypto.ToECDSA(common.FromHex(deputy04Privkey))
	if err != nil {
		return err
	}
	privarte05, err := crypto.ToECDSA(common.FromHex(deputy05Privkey))
	if err != nil {
		return err
	}

	var nodes = make([]*deputynode.DeputyNode, 5)
	nodes[0] = &deputynode.DeputyNode{
		LemoBase: common.HexToAddress(block01LemoBase),
		NodeID:   (crypto.FromECDSAPub(&privarte01.PublicKey))[1:],
		IP:       nil,
		Port:     7001,
		Rank:     1,
		Votes:    120,
	}
	nodes[1] = &deputynode.DeputyNode{
		LemoBase: common.HexToAddress(block02LemoBase),
		NodeID:   (crypto.FromECDSAPub(&privarte02.PublicKey))[1:],
		IP:       nil,
		Port:     7002,
		Rank:     2,
		Votes:    110,
	}
	nodes[2] = &deputynode.DeputyNode{
		LemoBase: common.HexToAddress(block03LemoBase),
		NodeID:   (crypto.FromECDSAPub(&privarte03.PublicKey))[1:],
		IP:       nil,
		Port:     7003,
		Rank:     3,
		Votes:    100,
	}
	nodes[3] = &deputynode.DeputyNode{
		LemoBase: common.HexToAddress(block04LemoBase),
		NodeID:   (crypto.FromECDSAPub(&privarte04.PublicKey))[1:],
		IP:       nil,
		Port:     7004,
		Rank:     4,
		Votes:    90,
	}
	nodes[4] = &deputynode.DeputyNode{
		LemoBase: common.HexToAddress(block05LemoBase),
		NodeID:   (crypto.FromECDSAPub(&privarte05.PublicKey))[1:],
		IP:       nil,
		Port:     7005,
		Rank:     5,
		Votes:    80,
	}

	if numNode > 5 || numNode == 0 {
		return fmt.Errorf("overflow index. numNode must be [1,5]")
	}
	manager.Add(height, nodes[:numNode])

	return nil
}

// 对区块进行签名的函数
func signTestBlock(deputyPrivate string, block *types.Block) ([]byte, error) {
	// 对区块签名
	private, err := crypto.ToECDSA(common.FromHex(deputyPrivate))
	if err != nil {
		return nil, err
	}
	signBlock, err := crypto.Sign(block.Hash().Bytes(), private)
	if err != nil {
		return nil, err
	}
	return signBlock, nil
}

// newTestBlock 创建一个函数，专门用来生成符合测试用例所用的区块
func newTestBlock(dpovp *Dpovp, parentHash common.Hash, height uint32, lemoBase common.Address, timeStamp *big.Int, signPrivate string, save bool) (*types.Block, error) {
	testBlock := makeBlock(dpovp.db, blockInfo{
		hash:        common.Hash{},
		parentHash:  parentHash,
		height:      height,
		author:      lemoBase,
		versionRoot: common.Hash{},
		txRoot:      common.Hash{},
		logsRoot:    common.Hash{},
		txList:      nil,
		gasLimit:    0,
		time:        timeStamp,
	}, save)
	// 对区块进行签名
	signBlock, err := signTestBlock(signPrivate, testBlock)
	if err != nil {
		return nil, err
	}
	testBlock.Header.SignData = signBlock
	return testBlock, nil
}

// Test_verifyHeaderTime 测试验证区块时间戳函数是否正确
func Test_verifyHeaderTime(t *testing.T) {
	blocks := []types.Block{
		{
			Header: &types.Header{
				ParentHash:  common.Hash{},
				LemoBase:    common.Address{},
				VersionRoot: common.Hash{},
				TxRoot:      common.Hash{},
				LogsRoot:    common.Hash{},
				EventRoot:   common.Hash{},
				Bloom:       types.Bloom{},
				Height:      0,
				GasLimit:    0,
				GasUsed:     0,
				Time:        big.NewInt(time.Now().Unix() - 2), // 正确时间
				SignData:    nil,
				Extra:       nil,
			},
			Txs:            nil,
			ChangeLogs:     nil,
			Events:         nil,
			ConfirmPackage: nil,
		},
		{
			Header: &types.Header{
				ParentHash:  common.Hash{},
				LemoBase:    common.Address{},
				VersionRoot: common.Hash{},
				TxRoot:      common.Hash{},
				LogsRoot:    common.Hash{},
				EventRoot:   common.Hash{},
				Bloom:       types.Bloom{},
				Height:      0,
				GasLimit:    0,
				GasUsed:     0,
				Time:        big.NewInt(time.Now().Unix() - 1), // 临界点时间
				SignData:    nil,
				Extra:       nil,
			},
			Txs:            nil,
			ChangeLogs:     nil,
			Events:         nil,
			ConfirmPackage: nil,
		},
		{
			Header: &types.Header{
				ParentHash:  common.Hash{},
				LemoBase:    common.Address{},
				VersionRoot: common.Hash{},
				TxRoot:      common.Hash{},
				LogsRoot:    common.Hash{},
				EventRoot:   common.Hash{},
				Bloom:       types.Bloom{},
				Height:      0,
				GasLimit:    0,
				GasUsed:     0,
				Time:        big.NewInt(time.Now().Unix() + 2), // 不正确时间
				SignData:    nil,
				Extra:       nil,
			},
			Txs:            nil,
			ChangeLogs:     nil,
			Events:         nil,
			ConfirmPackage: nil,
		},
	}

	err01 := verifyHeaderTime(&blocks[0])
	assert.Equal(t, nil, err01)
	err02 := verifyHeaderTime(&blocks[1])
	assert.Equal(t, nil, err02)
	err03 := verifyHeaderTime(&blocks[2])
	assert.Equal(t, errors.New("verify block's header error"), err03)

}

// Test_verifyHeaderSignData 测试验证区块签名数据函数是否正确
func Test_verifyHeaderSignData(t *testing.T) {
	err := initDeputyNode(3, 0) // 选择前三个共识节点
	if err != nil {
		assert.Error(t, err)
	}
	dpovp := loadDpovp()
	// 创建一个块并用另一个节点来对此区块进行签名
	block01, err := newTestBlock(dpovp, common.Hash{}, 1, common.HexToAddress(block01LemoBase), big.NewInt(time.Now().Unix()), deputy02Privkey, false)
	if err != nil {
		t.Error(err)
	}
	// header := block01.Header
	assert.Equal(t, ErrVerifyHeaderFailed, verifyHeaderSignData(block01))
	store.ClearData()
}

// TestDpovp_VerifyHeader01 对共识中共识区块与父块关联情况共识的测试
func TestDpovp_VerifyHeader01(t *testing.T) {
	err := initDeputyNode(3, 0)
	if err != nil {
		t.Error(err)
	}
	dpovp := loadDpovp()

	// 验证不存在父区块的情况
	testBlock00, err := newTestBlock(dpovp, common.Hash{}, 0, common.HexToAddress(block01LemoBase), big.NewInt(time.Now().Unix()-10), deputy01Privkey, true)
	if err != nil {
		t.Error(err)
	}
	// header := testBlock00.Header
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyHeader(testBlock00))

	// 验证父区块的高度为0，也就是父区块为创世区块情况
	testBlock01, err := newTestBlock(dpovp, testBlock00.Hash(), 1, common.HexToAddress(block02LemoBase), big.NewInt(time.Now().Unix()-5), deputy02Privkey, true)
	if err != nil {
		t.Error(err)
	}

	t.Log("parent block is genesis block, return ", dpovp.VerifyHeader(testBlock01))

	// // 与父块的时间间隔小于blockInternal(3000ms) // 设置与父区块出块时间相差1000ms
	// testBlock02, err := newTestBlock(dpovp, testBlock01.Hash(), 2, common.HexToAddress(block03LemoBase), testBlock01.Time().Add(testBlock01.Time(), big.NewInt(1)), deputy03Privkey, false)
	// if err != nil {
	// 	t.Error(err)
	// }
	//
	// assert.Equal(t, fmt.Errorf("verifyHeader: block is not enough newer than it's parent"), dpovp.VerifyHeader(testBlock02))

}

// 对共识中代理节点各种情况共识的测试
// TestDpovp_VerifyHeader02 验证共识节点只有一个节点的情况(if nodeCount == 1)
func TestDpovp_VerifyHeader02(t *testing.T) {
	err := initDeputyNode(1, 0) // 更改为一个代理节点参加共识
	if err != nil {
		assert.Nil(t, err)
	}

	dpovp := loadDpovp()
	// 创世块
	block00, err := newTestBlock(dpovp, common.Hash{}, 0, common.HexToAddress(block01LemoBase), big.NewInt(995), deputy01Privkey, true)
	if err != nil {
		t.Error(err)
	}
	// parent is genesis block
	block01, err := newTestBlock(dpovp, block00.Hash(), 1, common.HexToAddress(block01LemoBase), big.NewInt(1000), deputy01Privkey, true)
	if err != nil {
		t.Error(err)
	}
	// 验证共识节点只有一个的情况(if nodeCount == 1)
	block02, err := newTestBlock(dpovp, block01.Hash(), 2, common.HexToAddress(block01LemoBase), big.NewInt(1005), deputy01Privkey, true)
	if err != nil {
		t.Error(err)
	}
	// 获取当前的共识节点数
	// nodeCount := deputynode.Instance().GetDeputiesCount()
	// t.Log("deputy node number is ", nodeCount) // 打印当前代理节点数,结果应该为1
	// t.Log(dpovp.VerifyHeader(block02))

	// 通过调试打断点的方式来验证测试结果的正确性
	assert.Equal(t, nil, dpovp.VerifyHeader(block02))
}

// TestDpovp_VerifyHeader03 测试slot == 0,slot == 1,slot > 1的情况
func TestDpovp_VerifyHeader03(t *testing.T) {
	// 创建5个代理节点
	err := initDeputyNode(5, 0)
	if err != nil {
		t.Error(err)
	}
	dpovp := loadDpovp()
	// 创世块,随便哪个节点出块在这里没有影响
	block00, err := newTestBlock(dpovp, common.Hash{}, 0, common.HexToAddress(block01LemoBase), big.NewInt(1995), deputy01Privkey, true)
	if err != nil {
		t.Error(err)
	}
	// parent is genesis block,由第一个节点出块,此区块是作为测试区块的参照区块
	block01, err := newTestBlock(dpovp, block00.Hash(), 1, common.HexToAddress(block01LemoBase), big.NewInt(2000), deputy01Privkey, true)
	if err != nil {
		t.Error(err)
	}
	// if slot == 0 :
	// 还是由第一个节点出块,模拟 (if slot == 0 ) 的情况 ,与block01时间差为44s,满足条件(if timeSpan >= oneLoopTime-d.timeoutTime),此区块共识验证通过会返回nil
	block02, err := newTestBlock(dpovp, block01.Hash(), 2, common.HexToAddress(block01LemoBase), big.NewInt(2044), deputy01Privkey, false)
	if err != nil {
		t.Error(err)
	}
	t.Log("if slot == 0 && timeSpan >= oneLoopTime-d.timeoutTime , return nil", dpovp.VerifyHeader(block02))
	// 同上 else 情况, 与block01时间差为33s,小于40s
	block03, err := newTestBlock(dpovp, block01.Hash(), 2, common.HexToAddress(block01LemoBase), big.NewInt(2033), deputy01Privkey, false)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyHeader(block03))
	// 测试一个临界值，与block01时间差等于40s的情况
	block04, err := newTestBlock(dpovp, block01.Hash(), 2, common.HexToAddress(block01LemoBase), big.NewInt(2040), deputy01Privkey, false)
	if err != nil {
		t.Error(err)
	}
	t.Log("if slot == 0 && timeSpan == 40s, return nil", dpovp.VerifyHeader(block04))

	// else if slot == 1 :
	// 都与block01作为父块, 设置出块代理节点为第二个节点，满足slot == 1,时间差设为第一种小于一轮(50s)的情况,
	// block05时间满足(timeSpan >= d.blockInternal && timeSpan < d.timeoutTime)的正常情况
	block05, err := newTestBlock(dpovp, block01.Hash(), 2, common.HexToAddress(block02LemoBase), big.NewInt(2005), deputy02Privkey, false)
	if err != nil {
		t.Error(err)
	}
	t.Log("solt == 1 && timeSpan < oneLoopTime && timeSpan >= d.blockInternal && timeSpan < d.timeoutTime ,return nil", dpovp.VerifyHeader(block05))
	// block06 不满足(timeSpan >= d.blockInternal && timeSpan < d.timeoutTime)的情况,timeSpan == 11 > 10
	block06, err := newTestBlock(dpovp, block01.Hash(), 2, common.HexToAddress(block02LemoBase), big.NewInt(2011), deputy02Privkey, false)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyHeader(block06))
	// if slot == 1 else 的情况，此情况是timeSpan >= oneLoopTime,时间间隔超过一轮
	// 首先测试 timeSpan % oneLoopTime < timeoutTime 的正常情况
	block07, err := newTestBlock(dpovp, block01.Hash(), 2, common.HexToAddress(block02LemoBase), big.NewInt(2051), deputy02Privkey, false)
	if err != nil {
		t.Error(err)
	}
	t.Log("timeSpan % oneLoopTime < timeoutTime is normal,return nil", dpovp.VerifyHeader(block07))
	// 异常情况,timeSpan % oneLoopTime = 20 > timeoutTime
	block08, err := newTestBlock(dpovp, block01.Hash(), 2, common.HexToAddress(block02LemoBase), big.NewInt(2070), deputy02Privkey, false)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyHeader(block08))

	// else :
	// slot > 1的情况分析
	// timeSpan/d.timeoutTime == int64(slot-1) , timeSpan与timeoutTime的除数正好是间隔的代理节点数，为正常情况
	// 设置block09为第四个节点出块，与block01出块节点中间相隔2个节点,设置时间timeSpan == 20-29都是符合出块的时间
	block09, err := newTestBlock(dpovp, block01.Hash(), 2, common.HexToAddress(block04LemoBase), big.NewInt(2025), deputy04Privkey, false)
	if err != nil {
		t.Error(err)
	}
	t.Log("else: slot > 1 ,timeSpan/d.timeoutTime == int64(slot-1), return nil ", dpovp.VerifyHeader(block09))
	// 不符合情况,设置timeSpan >=30 || timeSpan < 20
	// timeSpan >=30 情况
	block10, err := newTestBlock(dpovp, block01.Hash(), 2, common.HexToAddress(block04LemoBase), big.NewInt(2030), deputy04Privkey, false)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyHeader(block10))
	// timeSpan < 20 情况
	block11, err := newTestBlock(dpovp, block01.Hash(), 2, common.HexToAddress(block04LemoBase), big.NewInt(2019), deputy04Privkey, false)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, ErrVerifyHeaderFailed, dpovp.VerifyHeader(block11))

}

// TestDpovp_Seal
func TestDpovp_Seal(t *testing.T) {
	dpovp := loadDpovp()
	// 创世块
	block00, err := newTestBlock(dpovp, common.Hash{}, 0, common.HexToAddress(block01LemoBase), big.NewInt(995), deputy01Privkey, true)
	if err != nil {
		t.Error(err)
	}
	// parent is genesis block,此区块是作为测试区块的参照区块
	block01, err := newTestBlock(dpovp, block00.Hash(), 1, common.HexToAddress(block01LemoBase), big.NewInt(1000), deputy01Privkey, true)
	if err != nil {
		t.Error(err)
	}

	TestBlockHeader := block01.Header // 得到block01头，为生成TestBlock所用
	txs := []*types.Transaction{
		signTransaction(types.NewTransaction(defaultAccounts[0], common.Big1, 2000000, common.Big2, []byte{12}, chainID, 1538210391, "aa", "aaa"), testPrivate),
		makeTransaction(testPrivate, defaultAccounts[1], common.Big1, common.Big2, 1538210491, 2000000),
	}
	block01.Txs = txs // 添加bock01交易
	TestBlockChangeLog := block01.ChangeLogs
	TestBlockEvents := block01.Events
	TestBlock, err := dpovp.Seal(TestBlockHeader, txs, TestBlockChangeLog, TestBlockEvents)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, block01.Hash(), TestBlock.Hash())

}

// // TestDpovp_Finalize todo
// func TestDpovp_Finalize(t *testing.T) {
//
// 	// 添加第一个共识节点列表,设置共识的节点为前两个节点
// 	err := initDeputyNode(2, 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	// 添加第二个共识节点列表,设置共识的节点为前三个节点,并设置发放奖励高度为10000,在挖出高度为10000+1000+1的区块的时候为上一轮共识节点发放奖励
// 	err = initDeputyNode(3, 10000)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	// 添加第三个共识节点列表,并设置发放奖励高度为20000,在挖出高度为20000+1000+1的区块的时候为上一轮共识节点发放奖励
// 	err = initDeputyNode(5, 20000)
// 	if err != nil {
// 		t.Error(err)
// 	}
//
// 	dpovp := loadDpovp()
// 	// 测试挖出的块高度不满足发放奖励高度的时候
// 	dpovp.Finalize(&types.Header{Height: 9999}, account.NewManager(common.Hash{}, dpovp.db))
// 	// dpovp.handOutRewards(9999)
// 	dpovp.Finalize(&types.Header{Height: 19998}, account.NewManager(common.Hash{}, dpovp.db))
// 	// dpovp.handOutRewards(19998)
// 	account01, err := dpovp.db.GetAccount(common.Hash{}, common.HexToAddress(block01LemoBase))
// 	assert.Nil(t, nil)
// 	t.Log("When there is no reward,node01Balance = ", account01.Balance)
// 	account02, err := dpovp.db.GetAccount(common.Hash{}, common.HexToAddress(block02LemoBase))
// 	assert.Nil(t, nil)
// 	t.Log("When there is no reward,node01Balance = ", account02.Balance)
// 	// 测试挖出的块高度满足发放奖励高度的时候
// 	// dpovp.handOutRewards(11001)
// 	dpovp.Finalize(&types.Header{Height: 11001}, account.NewManager(common.Hash{}, dpovp.db))
// 	t.Log("When it comes to giving out rewards,node01Balance = ", account01.Balance)
// 	t.Log("When it comes to giving out rewards,node01Balance = ", account02.Balance)
// 	// 第二轮发放奖励
// 	// dpovp.handOutRewards(21001)
// 	dpovp.Finalize(&types.Header{Height: 21001}, account.NewManager(common.Hash{}, dpovp.db))
// 	t.Log("When it comes to giving out rewards,node01Balance = ", account01.Balance)
// 	t.Log("When it comes to giving out rewards,node01Balance = ", account02.Balance)
//
// }
