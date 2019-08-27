package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"strconv"
	"testing"
	"time"
)

func TestNewValidator(t *testing.T) {
	dm := deputynode.NewManager(5, testBlockLoader{})

	fm := NewValidator(1000, testBlockLoader{}, dm, txPoolForValidator{}, testCandidateLoader{})
	assert.Equal(t, uint64(1000), fm.timeoutTime)
}

func Test_verifyParentHash(t *testing.T) {
	// no parent
	loader := createUnconfirmBlockLoader([]int{})
	parent, err := verifyParentHash(testBlocks[0], loader)
	assert.Equal(t, ErrVerifyBlockFailed, err)

	// exist parent
	loader = createUnconfirmBlockLoader([]int{0, 1})
	parent, err = verifyParentHash(testBlocks[1], loader)
	assert.NoError(t, err)
	assert.Equal(t, testBlocks[0], parent)
}

func Test_verifySigner(t *testing.T) {
	dm := deputynode.NewManager(5, loader{
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
	block01.Header.SignData = common.FromHex("0x123456") // 赋值错误的sign data
	assert.Equal(t, ErrVerifyHeaderFailed, verifySigner(block01, dm))
	// 3. 验证区块签名者不是出块节点的情况
	block02 := newBlockForVerifySigner(0, private02)
	assert.Error(t, ErrVerifyHeaderFailed, verifySigner(block02, dm))
	// 4. 验证minerAddress是否正确
	block03 := newBlockForVerifySigner(0, minerPrivate)
	block03.Header.MinerAddress = addr02 // 修改block的minerAddress
	assert.Error(t, ErrVerifyHeaderFailed, verifySigner(block03, dm))
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
	assert.Error(t, ErrVerifyBlockFailed, verifyTxRoot(incorrectBlock))
}

func Test_verifyTxs(t *testing.T) {
	txs := types.Transactions{
		makeTx(common.HexToAddress("0x11"), common.HexToAddress("0x12"), uint64(90000)),
	}
	txPool := txPoolForValidator{true} // 交易池中返回的状态为true
	// 1. 正确情况
	block01 := newBlockForVerifyTxs(txs, uint32(80000)) // block的时间小于tx的时间
	assert.NoError(t, verifyTxs(block01, txPool))

	// 2. 交易池返回状态为false的情况
	assert.Error(t, ErrVerifyBlockFailed, verifyTxs(block01, txPoolForValidator{false}))

	// 3. 交易时间小于block时间的情况
	block02 := newBlockForVerifyTxs(txs, uint32(90001))
	assert.Error(t, ErrVerifyBlockFailed, verifyTxs(block02, txPool))
}

func Test_verifyHeight(t *testing.T) {
	for i := 0; i < 10; i++ {
		// 1. 正确情况
		assert.NoError(t, verifyHeight(newBlockForVerifyHeight(uint32(i+1)), newBlockForVerifyHeight(uint32(i))))
		// 2. 错误情况
		assert.Error(t, ErrVerifyHeaderFailed, verifyHeight(newBlockForVerifyHeight(uint32(i+2)), newBlockForVerifyHeight(uint32(i))))
	}
}

func Test_verifyTime(t *testing.T) {
	now := uint32(time.Now().Unix())
	// 1. 正确情况
	assert.NoError(t, verifyTime(newBlockForVerifyTime(now)))
	// 2. 验证误差为1s
	assert.NoError(t, verifyTime(newBlockForVerifyTime(now+1)))
	// 3. 异常情况
	assert.Error(t, ErrVerifyHeaderFailed, verifyTime(newBlockForVerifyTime(now+2)))
}

func Test_verifyDeputy(t *testing.T) {
	// 1. 验证不是deputyNodes快照高度的情况
	height01 := params.TermDuration + 1
	block01 := newBlockForVerifyDeputy(height01, nil, nil)
	assert.NoError(t, verifyDeputy(block01, testCandidateLoader{}))

	// 区块为deputyNodes快照块
	height := params.TermDuration * 10
	deputies := types.DeputyNodes{
		&types.DeputyNode{
			MinerAddress: minerAddr,
			NodeID:       minerNodeId,
			Rank:         0,
			Votes:        big.NewInt(10000),
		},
		&types.DeputyNode{
			MinerAddress: addr02,
			NodeID:       nodeId02,
			Rank:         1,
			Votes:        big.NewInt(1000),
		},
		&types.DeputyNode{
			MinerAddress: addr02,
			NodeID:       nodeId02,
			Rank:         2,
			Votes:        big.NewInt(100),
		},
	}
	// 2. 验证快照块中的deputyNodes是我们预期的nodes
	block02 := newBlockForVerifyDeputy(height, deputies, deputies.MerkleRootSha().Bytes())
	assert.NoError(t, verifyDeputy(block02, testCandidateLoader(deputies)))
	// 3. block中的deputyNodeRoot不正确的情况
	block03 := newBlockForVerifyDeputy(height, deputies, common.FromHex("0x99999999999999999999999999"))
	assert.Error(t, ErrVerifyBlockFailed, verifyDeputy(block03, testCandidateLoader(deputies)))
	// 4. 验证block中的deputyNodes和链上直接获取的deputyNodes不相等的情况
	block04 := newBlockForVerifyDeputy(height, deputies, deputies.MerkleRootSha().Bytes())
	assert.Error(t, ErrVerifyBlockFailed, verifyDeputy(block04, testCandidateLoader(deputies[:1]))) // 链上获取到的deputyNodes为deputies[:1]
}

func Test_verifyExtraData(t *testing.T) {
	// 验证block中的额外数据长度
	for i := 1; i <= params.MaxExtraDataLen*2; i++ {
		block := newBlockForVerifyExtraData(make([]byte, i))
		if i > params.MaxExtraDataLen {
			assert.Error(t, ErrVerifyHeaderFailed, verifyExtraData(block))
		} else {
			assert.NoError(t, verifyExtraData(block))
		}
	}
}

func Test_VerifyMineSlot(t *testing.T) {
	timeoutTime := uint32(10) // unit: s
	deputyCount := 100
	// 创建100个miner地址
	minerAddrs := make([]common.Address, 0, deputyCount)
	for i := 0; i < 100; i++ {
		minerAddrs = append(minerAddrs, common.HexToAddress("0x1"+strconv.Itoa(i)))
	}
	// 创建100个deputy node
	deputyNodes := make(types.DeputyNodes, 0, deputyCount)
	for i := 0; i < deputyCount; i++ {
		deputy := &types.DeputyNode{
			MinerAddress: minerAddrs[i],
			NodeID:       nil,
			Rank:         uint32(i),
			Votes:        big.NewInt(int64(1000 / (i + 1))),
		}
		deputyNodes = append(deputyNodes, deputy)
	}

	dm := deputynode.NewManager(len(deputyNodes), loader{
		Nodes: deputyNodes,
	})
	// 一轮时间
	oneLoopTime := uint32(len(deputyNodes)) * timeoutTime // 单位： s
	// 测试两个区块的出块者不同的distance的出块时间间隔情况
	for i := 0; i < deputyCount; i++ {
		for j := i + 1; j < deputyCount; j++ {
			// 1. 验证正确的区块出块时间间隔
			correctPassTime := uint32(j-i-1)*timeoutTime + (timeoutTime - uint32(1)) // parentBlock和block的正确的时间差为： (j-i-1)*timeoutTime < passTime < (j-i)*timeoutTime
			parentBlock, currentBlock := assembleBlockForVerifyMineSlot(correctPassTime, oneLoopTime, minerAddrs[i], minerAddrs[j])
			assert.NoError(t, VerifyMineSlot(currentBlock, parentBlock, uint64(timeoutTime*1000), dm))

			// 2. 验证时间间隔小于规定的最小时间间隔的情况
			underPassTime := uint32(j-i-1)*timeoutTime - 1
			parentBlock, currentBlock = assembleBlockForVerifyMineSlot(underPassTime, oneLoopTime, minerAddrs[i], minerAddrs[j])
			assert.Error(t, ErrVerifyHeaderFailed, VerifyMineSlot(currentBlock, parentBlock, uint64(timeoutTime*1000), dm))

			// 3. 验证时间间隔大于规定的最大出块间隔时间
			oversizePassTime := uint32(j-i)*timeoutTime + 1
			parentBlock, currentBlock = assembleBlockForVerifyMineSlot(oversizePassTime, oneLoopTime, minerAddrs[i], minerAddrs[j])
			assert.Error(t, ErrVerifyHeaderFailed, VerifyMineSlot(currentBlock, parentBlock, uint64(timeoutTime*1000), dm))
		}
	}
}
