package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

var (
	TestChainID  uint16 = 1
	minerAddr, _        = common.StringToAddress("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG")
	minerPrivate        = "c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"
	minerNodeId         = common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0")

	addr02, _ = common.StringToAddress("Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY")
	private02 = "9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb"
)

func TestNewValidator(t *testing.T) {
	dm := deputynode.NewManager(5, &testBlockLoader{})

	fm := NewValidator(1000, &testBlockLoader{}, dm, txPoolForValidator{}, testCandidateLoader{})
	assert.Equal(t, uint64(1000), fm.timeoutTime)
}

func Test_verifyParentHash(t *testing.T) {
	// no parent
	loader := createBlockLoader([]int{}, -1)
	parent, err := verifyParentHash(testBlocks[0], loader)
	assert.Equal(t, ErrVerifyBlockFailed, err)

	// exist parent
	loader = createBlockLoader([]int{0, 1}, 0)
	parent, err = verifyParentHash(testBlocks[1], loader)
	assert.NoError(t, err)
	assert.Equal(t, testBlocks[0], parent)
}

// newBlockForVerifySigner 需要构造出区块签名数据、MinerAddress、区块高度
func newBlockForVerifySigner(height uint32, private string) *types.Block {
	privateKey, _ := crypto.HexToECDSA(private)
	minerAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	header := &types.Header{
		MinerAddress: minerAddress,
		Height:       height,
	}
	block := &types.Block{
		Header: header,
	}
	hash := block.Hash()
	signData, _ := crypto.Sign(hash[:], privateKey)
	block.Header.SignData = signData
	return block
}

// 用于New deputyNode manager初始化deputy
type snapshotLoader struct {
	Nodes types.DeputyNodes
}

func (l snapshotLoader) GetBlockByHeight(height uint32) (*types.Block, error) {
	if height >= params.TermDuration {
		return nil, store.ErrNotExist
	}
	return &types.Block{
		DeputyNodes: l.Nodes,
	}, nil
}

func Test_verifySigner(t *testing.T) {
	dm := deputynode.NewManager(5, snapshotLoader{
		Nodes: types.DeputyNodes{&types.DeputyNode{
			MinerAddress: minerAddr,
			NodeID:       minerNodeId,
			Rank:         0,
			Votes:        big.NewInt(100),
		}},
	})
	// 1. 验证block 正确的签名
	block01 := newBlockForVerifySigner(0, minerPrivate)
	assert.NoError(t, verifySigner(block01, dm))
	// 2. 验证block的签名数据不正确的情况
	block01 = newBlockForVerifySigner(0, minerPrivate)
	block01.Header.SignData = common.FromHex("0x123456") // 赋值错误的sign data
	assert.Equal(t, ErrVerifyHeaderFailed, verifySigner(block01, dm))
	// 3. 验证区块签名者不是出块节点的情况
	block02 := newBlockForVerifySigner(0, private02)
	assert.Equal(t, ErrVerifyHeaderFailed, verifySigner(block02, dm))
	// 4. 验证minerAddress是否正确
	block03 := newBlockForVerifySigner(0, minerPrivate)
	block03.Header.MinerAddress = addr02 // 修改block的minerAddress
	assert.Equal(t, ErrVerifyHeaderFailed, verifySigner(block03, dm))
}

// newBlockForVerifyTxRoot
func newBlockForVerifyTxRoot(txs types.Transactions, txRoot common.Hash) *types.Block {
	header := &types.Header{
		TxRoot: txRoot,
	}
	return &types.Block{
		Header: header,
		Txs:    txs,
	}
}

func makeTx(from, to common.Address, txTime uint64) *types.Transaction {
	return types.NewTransaction(from, to, big.NewInt(100), uint64(1000), big.NewInt(100), nil, 0, TestChainID, txTime, "", "")
}

// 验证block中的txs和txRoot
func Test_verifyTxRoot(t *testing.T) {
	// 构造txs
	txs := make(types.Transactions, 0, 10)
	for i := 0; i < 10; i++ {
		tx := makeTx(common.HexToAddress("0x"+strconv.Itoa(i)), common.HexToAddress("0x88"), uint64(time.Now().Unix()))
		txs = append(txs, tx)
	}
	correctTxRoot := txs.MerkleRootSha()
	// 正确的情况
	correctBlock := newBlockForVerifyTxRoot(txs, correctTxRoot)
	assert.NoError(t, verifyTxRoot(correctBlock))
	// 错误的情况
	incorrectRoot := common.HexToHash("0x111111111111111111111111")
	incorrectBlock := newBlockForVerifyTxRoot(txs, incorrectRoot)
	assert.Equal(t, ErrVerifyBlockFailed, verifyTxRoot(incorrectBlock))
}

func newBlockForVerifyTxs(txs types.Transactions, time uint32) *types.Block {
	header := &types.Header{
		Time: time,
	}
	return &types.Block{
		Header: header,
		Txs:    txs,
	}
}

func Test_verifyTxs(t *testing.T) {
	txs := types.Transactions{
		makeTx(common.HexToAddress("0x11"), common.HexToAddress("0x12"), uint64(90)),
	}
	txPool := txPoolForValidator{true} // 交易池中返回的状态为true
	// 1. 正确情况
	block01 := newBlockForVerifyTxs(txs, uint32(80)) // block的时间小于tx的时间
	assert.NoError(t, verifyTxs(block01, txPool, TestChainID))

	// 2. 交易池返回状态为false的情况
	assert.Equal(t, ErrVerifyBlockFailed, verifyTxs(block01, txPoolForValidator{false}, TestChainID))

	// 3. 交易时间小于block时间的情况
	block02 := newBlockForVerifyTxs(txs, uint32(91))
	assert.Equal(t, ErrVerifyBlockFailed, verifyTxs(block02, txPool, TestChainID))
}

func newBlockForVerifyHeight(height uint32) *types.Block {
	header := &types.Header{
		Height: height,
	}
	return &types.Block{
		Header: header,
	}
}

func Test_verifyHeight(t *testing.T) {
	for i := 0; i < 10; i++ {
		// 1. 正确情况
		assert.NoError(t, verifyHeight(newBlockForVerifyHeight(uint32(i+1)), newBlockForVerifyHeight(uint32(i))))
		// 2. 错误情况
		assert.Equal(t, ErrVerifyHeaderFailed, verifyHeight(newBlockForVerifyHeight(uint32(i+2)), newBlockForVerifyHeight(uint32(i))))
	}
}

func newBlockForVerifyTime(time uint32) *types.Block {
	header := &types.Header{
		Time: time,
	}
	return &types.Block{
		Header: header,
	}
}

func Test_verifyTime(t *testing.T) {
	now := uint32(time.Now().Unix())
	// 1. 正确情况
	assert.NoError(t, verifyTime(newBlockForVerifyTime(now)))
	// 2. 验证误差为1s
	assert.NoError(t, verifyTime(newBlockForVerifyTime(now+1)))
	// 3. 异常情况
	assert.Equal(t, ErrVerifyHeaderFailed, verifyTime(newBlockForVerifyTime(now+2)))
}

func newBlockForVerifyDeputy(height uint32, deputyNodes types.DeputyNodes, deputyRoot []byte) *types.Block {
	header := &types.Header{
		Height:     height,
		DeputyRoot: deputyRoot,
	}
	return &types.Block{
		Header:      header,
		DeputyNodes: deputyNodes,
	}
}

func Test_verifyDeputy(t *testing.T) {
	// 1. 验证不是deputyNodes快照高度的情况
	height01 := params.TermDuration + 1
	block01 := newBlockForVerifyDeputy(height01, nil, nil)
	assert.NoError(t, verifyDeputy(block01, testCandidateLoader{}))

	// 区块为deputyNodes快照块
	height := params.TermDuration * 10
	deputies := pickNodes(0, 1, 2, 3, 4)
	// 2. 验证快照块中的deputyNodes是我们预期的nodes
	block02 := newBlockForVerifyDeputy(height, deputies, deputies.MerkleRootSha().Bytes())
	assert.NoError(t, verifyDeputy(block02, createCandidateLoader(0, 1, 2, 3, 4)))
	// 3. block中的deputyNodeRoot不正确的情况
	block03 := newBlockForVerifyDeputy(height, deputies, common.FromHex("0x99999999999999999999999999"))
	assert.Equal(t, ErrVerifyBlockFailed, verifyDeputy(block03, createCandidateLoader(0, 1, 2, 3, 4)))
	// 4. 验证block中的deputyNodes和链上直接获取的deputyNodes不相等的情况
	block04 := newBlockForVerifyDeputy(height, deputies, deputies.MerkleRootSha().Bytes())
	assert.Equal(t, ErrVerifyBlockFailed, verifyDeputy(block04, createCandidateLoader(0, 1, 2))) // 链上获取到的deputyNodes为deputies中的一半
}

func newBlockForVerifyExtraData(extraData []byte) *types.Block {
	return &types.Block{
		Header: &types.Header{
			Extra: extraData,
		},
	}
}

func Test_verifyExtraData(t *testing.T) {
	// 验证block中的额外数据长度
	block := newBlockForVerifyExtraData(make([]byte, 0))
	assert.NoError(t, verifyExtraData(block))

	block = newBlockForVerifyExtraData(make([]byte, params.MaxExtraDataLen-1))
	assert.NoError(t, verifyExtraData(block))

	block = newBlockForVerifyExtraData(make([]byte, params.MaxExtraDataLen))
	assert.NoError(t, verifyExtraData(block))

	block = newBlockForVerifyExtraData(make([]byte, params.MaxExtraDataLen+1))
	assert.Equal(t, ErrVerifyHeaderFailed, verifyExtraData(block))
}

// time单位:s
func newBlockForVerifyMineSlot(height uint32, minerAddress common.Address, time uint32) *types.Block {
	return &types.Block{
		Header: &types.Header{
			MinerAddress: minerAddress,
			Height:       height,
			Time:         time,
		},
	}
}

// assembleBlockForVerifyMineSlot
func assembleBlockForVerifyMineSlot(passTime, oneLoopTime uint32, parentMiner, currentMiner common.Address) (parentBlock *types.Block, currentBlock *types.Block) {
	rand.Seed(time.Now().UnixNano())
	parentTime := uint32(rand.Intn(500)) + 1
	blockTime := parentTime + passTime + oneLoopTime*uint32(rand.Intn(5)) // blockTime为parentTime + 正确的相差时间 + 随机的轮数
	parentBlock = newBlockForVerifyMineSlot(1, parentMiner, parentTime)
	currentBlock = newBlockForVerifyMineSlot(2, currentMiner, blockTime)
	return
}

func Test_VerifyMineSlot(t *testing.T) {
	timeoutTime := uint32(10) // unit: s
	timeoutStamp := uint64(timeoutTime * 1000)
	deputyCount := 17
	dm := initDeputyManager(deputyCount)
	firstTerm, err := dm.GetTermByHeight(1)
	assert.NoError(t, err)
	getMiner := func(index int) common.Address {
		return firstTerm.Nodes[index].MinerAddress
	}
	// 一轮时间
	oneLoopTime := uint32(deputyCount) * timeoutTime // 单位： s
	// 测试两个区块的出块者不同的distance的出块时间间隔情况
	for i := 0; i < deputyCount; i++ {
		for j := 0; j < deputyCount; j++ {
			distance, err := GetMinerDistance(2, getMiner(i), getMiner(j), dm)
			assert.NoError(t, err)
			minPassTime := uint32(distance-1) * timeoutTime
			maxPassTime := uint32(distance) * timeoutTime
			// parentBlock和block的正确的时间差为： minPassTime < passTime < maxPassTime

			// 1. 验证时间间隔小于规定的最小时间间隔的情况
			parentBlock, currentBlock := assembleBlockForVerifyMineSlot(minPassTime-1, oneLoopTime, getMiner(i), getMiner(j))
			assert.Equal(t, ErrVerifyHeaderFailed, VerifyMineSlot(currentBlock.Header, parentBlock.Header, timeoutStamp, dm))
			// 2. 验证正确的区块出块时间间隔
			parentBlock, currentBlock = assembleBlockForVerifyMineSlot(minPassTime, oneLoopTime, getMiner(i), getMiner(j))
			assert.NoError(t, VerifyMineSlot(currentBlock.Header, parentBlock.Header, timeoutStamp, dm))
			parentBlock, currentBlock = assembleBlockForVerifyMineSlot(minPassTime+1, oneLoopTime, getMiner(i), getMiner(j))
			assert.NoError(t, VerifyMineSlot(currentBlock.Header, parentBlock.Header, timeoutStamp, dm))
			parentBlock, currentBlock = assembleBlockForVerifyMineSlot(maxPassTime, oneLoopTime, getMiner(i), getMiner(j))
			assert.Equal(t, ErrVerifyHeaderFailed, VerifyMineSlot(currentBlock.Header, parentBlock.Header, timeoutStamp, dm))
			// 3. 验证时间间隔大于规定的最大出块间隔时间
			parentBlock, currentBlock = assembleBlockForVerifyMineSlot(maxPassTime+1, oneLoopTime, getMiner(i), getMiner(j))
			assert.Equal(t, ErrVerifyHeaderFailed, VerifyMineSlot(currentBlock.Header, parentBlock.Header, timeoutStamp, dm))
		}
	}
}

func newBlockForVerifyChangeLog(logs types.ChangeLogSlice, logRoot common.Hash) *types.Block {
	block := &types.Block{
		Header: &types.Header{
			LogRoot: logRoot,
		},
	}
	block.SetChangeLogs(logs)
	return block
}

func Test_verifyChangeLog(t *testing.T) {
	// 1. 验证block的changeLogs为null时候的正常情况
	nullchangeLogs := make(types.ChangeLogSlice, 0)
	nullLogRoot := nullchangeLogs.MerkleRootSha()
	block01 := newBlockForVerifyChangeLog(nullchangeLogs, nullLogRoot)
	assert.NoError(t, verifyChangeLog(block01, nullchangeLogs))
	// 2. new changeLogs
	logs := make(types.ChangeLogSlice, 0, 10)
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 10; i++ {
		log := &types.ChangeLog{
			LogType: types.ChangeLogType(uint32(i)),
			Address: common.HexToAddress("0x11" + strconv.Itoa(i)),
			Version: uint32(i % 3),
			OldVal:  rand.Intn(100),
			NewVal:  rand.Intn(100),
			Extra:   strconv.Itoa(i),
		}
		logs = append(logs, log)
	}

	// 3. 验证正常情况
	block02 := newBlockForVerifyChangeLog(logs, logs.MerkleRootSha())
	computedLogs := logs // 验证节点执行区块交易生成的changeLogs与需要验证的区块中的changeLogs相等
	assert.NoError(t, verifyChangeLog(block02, computedLogs))

	// 4. 需要验证的区块中的changeLogs算出的默克尔hash与区块头中的logRoot不相等的情况
	incorrectLogRoot := common.HexToHash("0x9999999999")          // 构造一个错误的logRoot
	block03 := newBlockForVerifyChangeLog(logs, incorrectLogRoot) // 构造一个logRoot错误的block
	assert.Equal(t, ErrVerifyBlockFailed, verifyChangeLog(block03, computedLogs))

	// 5. 验证区块中的changeLogs与验证节点执行区块之后产生的changeLogs不相等的情况
	block04 := newBlockForVerifyChangeLog(logs, logs.MerkleRootSha())
	computedLogs = logs[:5] // 计算出来的logs
	assert.Equal(t, ErrVerifyBlockFailed, verifyChangeLog(block04, computedLogs))

	// 6. 验证区块changeLogs为空，但是logRoot是预期的正确值
	block05 := newBlockForVerifyChangeLog(nil, logs.MerkleRootSha())
	assert.NoError(t, verifyChangeLog(block05, logs))
}

func TestValidator_VerifyAfterTxProcess(t *testing.T) {
	dm := deputynode.NewManager(5, createBlockLoader([]int{}, -1))
	v := NewValidator(1000, createBlockLoader([]int{}, -1), dm, txPoolForValidator{}, testCandidateLoader{})
	// 计算changeLogs 为null的logRoot
	nullchangeLogs := make(types.ChangeLogSlice, 0)
	nullLogRoot := nullchangeLogs.MerkleRootSha()

	block := &types.Block{
		Header: &types.Header{
			ParentHash:   common.Hash{},
			MinerAddress: common.HexToAddress("0x111"),
			LogRoot:      nullLogRoot,
		},
	}
	// 1. 验证正常情况
	assert.NoError(t, v.VerifyAfterTxProcess(block, block))

	// 2. 验证verifyChangeLog()返回err, 待验证的区块中的changeLogs和重新计算出来的block中的changeLogs不相等
	computedBlock := &types.Block{
		Header: block.Header,
		ChangeLogs: types.ChangeLogSlice{
			&types.ChangeLog{
				LogType: account.BalanceLog,
				Address: common.HexToAddress("0x11" + strconv.Itoa(1)),
				Version: uint32(100),
			},
		},
	}
	assert.Equal(t, ErrVerifyBlockFailed, v.VerifyAfterTxProcess(block, computedBlock))

	// 3. 验证两个block计算出的hash不等的情况，这里构造两个块的MinerAddress不同
	computedBlock = &types.Block{
		Header: &types.Header{
			ParentHash:   common.Hash{},
			MinerAddress: common.HexToAddress("0x222"),
			LogRoot:      nullLogRoot,
		},
	}
	assert.Equal(t, ErrVerifyBlockFailed, v.VerifyAfterTxProcess(block, computedBlock))
}

// newBlockForJudgeDeputy extra用于改变交易的hash
func newBlockForJudgeDeputy(height uint32, private, extra string) *types.Block {
	block := &types.Block{
		Header: &types.Header{
			Height: height,
			Extra:  []byte(extra),
		},
	}
	hash := block.Hash()
	privateKey, _ := crypto.HexToECDSA(private)
	signData, _ := crypto.Sign(hash.Bytes(), privateKey)
	block.Header.SignData = signData
	return block
}

func TestValidator_JudgeDeputy(t *testing.T) {
	private01 := "c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"
	private02 := "9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb"
	dm := deputynode.NewManager(5, createBlockLoader([]int{}, -1))
	v1 := NewValidator(1000, createBlockLoader([]int{}, -1), dm, txPoolForValidator{}, testCandidateLoader{})

	// 1. 测试newBlock.SignerNodeID()返回error的情况
	block01 := newBlockForJudgeDeputy(0, private01, "")
	// 修改block的signData长度不合法
	block01.Header.SignData = common.FromHex("11111")
	assert.False(t, v1.JudgeDeputy(block01))

	// 2. 测试同一高度的两个不同的区块是由同一个节点签名的情况
	block02 := newBlockForJudgeDeputy(1, private01, "我签名了高度为1的区块")
	// 构造一个testBlockLoader中存储着block02的validator对象
	v2 := NewValidator(1000, createUnstableLoader(block02), dm, txPoolForValidator{}, testCandidateLoader{})
	block03 := newBlockForJudgeDeputy(1, private01, "我又签名了高度为1的区块")
	// 返回true
	assert.True(t, v2.JudgeDeputy(block03))

	// 3. 测试非稳定块中没有同一个节点签名同一高度的区块的情况
	block04 := newBlockForJudgeDeputy(100, private01, "我是private01，我签名了高度为100的区块")
	// 构造一个testBlockLoader中存储着block04的validator对象
	v3 := NewValidator(1000, createUnstableLoader(block04), dm, txPoolForValidator{}, testCandidateLoader{})
	block05 := newBlockForJudgeDeputy(100, private02, "我是private02，我签名了高度为100的区块")
	// 返回false
	assert.False(t, v3.JudgeDeputy(block05))

	// 4. 测试v.blockLoader.IterateUnConfirms迭代器还原nodeId出错的情况
	errBlock := block05
	errBlock.Header.SignData = common.FromHex("122") // 签名长度不为65位
	v4 := NewValidator(1000, createUnstableLoader(errBlock), dm, txPoolForValidator{}, testCandidateLoader{})
	assert.False(t, v4.JudgeDeputy(block03)) // block03中的signData是正常的,但是迭代器中迭代出的block的signData有误,直接返回
}

func newBlockForVerifyNewConfirms(private string) *types.Block {
	privateKey, _ := crypto.HexToECDSA(private)
	minerAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	block := &types.Block{
		Header: &types.Header{
			MinerAddress: minerAddress,
			Height:       0,
		},
	}
	hash := block.Hash()
	signData, _ := crypto.Sign(hash[:], privateKey)
	block.Header.SignData = signData
	return block
}

func signBlock(block *types.Block, private string) types.SignData {
	privateKey, _ := crypto.HexToECDSA(private)
	hash := block.Hash()
	signData, _ := crypto.Sign(hash[:], privateKey)
	var sig types.SignData
	copy(sig[:], signData)
	return sig
}

func TestValidator_VerifyNewConfirms(t *testing.T) {
	// 三个deputy
	private01 := "c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"
	nodeId01 := common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0")
	private02 := "9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb"
	nodeId02 := common.FromHex("0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0")
	private03 := "ba9b51e59ec57d66b30b9b868c76d6f4d386ce148d9c6c1520360d92ef0f27ae"
	nodeId03 := common.FromHex("0x7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2d0347692f36ef736f342fb5ce4710f7e337f062cc2110d134b63a9575f78cb167bfae2f43")
	// 创建deputyNodes
	deputyNodes := types.DeputyNodes{
		&types.DeputyNode{
			MinerAddress: common.Address{},
			NodeID:       nodeId01,
			Rank:         0,
			Votes:        big.NewInt(10000),
		},
		&types.DeputyNode{
			MinerAddress: common.Address{},
			NodeID:       nodeId02,
			Rank:         1,
			Votes:        big.NewInt(1000),
		},
		&types.DeputyNode{
			MinerAddress: common.Address{},
			NodeID:       nodeId03,
			Rank:         2,
			Votes:        big.NewInt(100),
		},
	}

	dm := deputynode.NewManager(3, snapshotLoader{
		Nodes: deputyNodes,
	})
	v := NewValidator(1000, createBlockLoader([]int{}, -1), dm, txPoolForValidator{}, testCandidateLoader{})
	// 1. 验证正常情况
	block01 := newBlockForVerifyNewConfirms(private01) // 创建一个第一个代理节点出的区块并且区块中的确认包为空
	sig02 := signBlock(block01, private02)
	sig03 := signBlock(block01, private03)
	sigList01 := []types.SignData{sig02, sig03}
	validConfirms, err := v.VerifyNewConfirms(block01, sigList01, dm)
	assert.NoError(t, err)
	assert.Equal(t, sigList01, validConfirms) // 返回的确认包列表与输入验证的确认包列表相同

	// 2. 测试验证的确认包中有相同的确认包信息
	sigList02 := []types.SignData{sig02, sig02, sig03, sig03} // 验证的确认包列表中有两个相同的确认包
	expectReturnList := []types.SignData{sig02, sig03}        // 预期返回的确认包列表为查重之后的确认包列表
	validConfirms, err = v.VerifyNewConfirms(block01, sigList02, dm)
	assert.Equal(t, ErrExistedConfirm, err)
	assert.Equal(t, expectReturnList, validConfirms)

	// 3. 验证区块中的Confirms中存在需要验证的确认包
	sig01 := signBlock(block01, private01) // 验证minerAddress签名的确认信息
	validConfirms, err = v.VerifyNewConfirms(block01, []types.SignData{sig01}, dm)
	assert.Equal(t, ErrInvalidConfirmSigner, err)
	length := len(validConfirms) // 返回值长度为0
	assert.Equal(t, 0, length)

	block01.Confirms = []types.SignData{sig02}  // 把第二个deputy的区块签名信息放在区块的confirms中
	sigList03 := []types.SignData{sig02, sig03} // 验证确认包中包含了block的confirms中的sig2
	expectReturn := []types.SignData{sig03}     // 期望的返回确认包会除去重复的sig2
	validConfirms, err = v.VerifyNewConfirms(block01, sigList03, dm)
	assert.Equal(t, ErrInvalidConfirmSigner, err)
	assert.Equal(t, expectReturn, validConfirms)

	// 4. 验证确认包中包含非deputy node的签名确认信息的情况
	ordinaryPrivate := "7b1b260cd1c40a44b05bf231a5804691a2fffeec4b4e9bf79ccddaf613cfc053"
	errSign := signBlock(block01, ordinaryPrivate) // 普通账户私钥对block进行的签名
	_, err = v.VerifyNewConfirms(block01, []types.SignData{errSign}, dm)
	assert.Equal(t, ErrInvalidConfirmSigner, err)

	// 5. 确认包不是对该block进行的签名
	block02 := newBlockForVerifyNewConfirms(private02)
	block03 := newBlockForVerifyNewConfirms(private03)
	// 使用private01对block02进行签名
	sigBlock02 := signBlock(block02, private01)
	// 拿对block02的确认包去对block03中验证，在解析签名后会返回一个新的nodeId，此nodeId是不会在deputy nodes中的
	_, err = v.VerifyNewConfirms(block03, []types.SignData{sigBlock02}, dm)
	assert.Equal(t, ErrInvalidConfirmSigner, err)
}

func TestValidator_VerifyConfirmPacket(t *testing.T) {
	dm := deputynode.NewManager(5, createBlockLoader([]int{}, -1))
	// 1. 测试通过blockHash得不到block的情况
	v1 := NewValidator(1000, createBlockLoader([]int{}, -1), dm, txPoolForValidator{}, testCandidateLoader{})
	hash := testBlocks[0].Hash()
	confirms, err := v1.VerifyConfirmPacket(0, hash, nil)
	assert.Nil(t, confirms)
	assert.Equal(t, ErrBlockNotExist, err)
	// 2. 测试区块高度不对的情况
	block := testBlocks[1]
	v2 := NewValidator(1000, createBlockLoader([]int{0, 1}, 0), dm, txPoolForValidator{}, testCandidateLoader{})
	confirms, err = v2.VerifyConfirmPacket(block.Height()+1, block.Hash(), nil)
	assert.Nil(t, confirms)
	assert.Equal(t, ErrInvalidSignedConfirmInfo, err)
}
