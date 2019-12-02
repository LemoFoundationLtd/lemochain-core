package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func newBlock(timestamp uint32, txs types.Transactions) *types.Block {
	return &types.Block{
		Header: &types.Header{
			TxRoot: txs.MerkleRootSha(),
			Time:   timestamp,
		},
		Txs: txs,
	}
}

func TestBlocksInTime_SaveBlock(t *testing.T) {
	/**
	测试要点：
	1、测试保存的block时间戳小于lastMaxTime的异常情况
	2、测试保存的block时间戳等于lastMaxTime的正常情况
	3、测试保存的block时间区间能落到目前已经保存的时间区间里的正常情况
	*/
	// 1. 测试保存的block时间戳小于lastMaxTime的异常情况
	blocksInTime01 := newBlocksInTime(999999)
	block01 := newBlock(8888, nil)
	blocksInTime01.SaveBlock(block01)
	// 比较blockSet中是否有数据
	assert.Equal(t, len(blocksInTime01.BlockSet), 0)

	// 2. 测试保存的block时间戳等于lastMaxTime的正常情况
	blocksInTime02 := newBlocksInTime(999999)
	block02 := newBlock(999999, nil)
	blocksInTime02.SaveBlock(block02)
	// 预期能保存成功
	assert.Equal(t, len(blocksInTime02.BlockSet[0]), 1)
	assert.Equal(t, blocksInTime02.BlockSet[0][block02.Hash()].Header, block02.Header)
	// 测试保存带交易的block
	tx01 := makeTxRandom(common.HexToAddress("0x111"))
	tx02 := makeTxRandom(common.HexToAddress("0x222"))
	tx03 := makeTxRandom(common.HexToAddress("0x333"))
	tx04 := makeTxRandom(common.HexToAddress("0x444"))
	txs := types.Transactions{tx01, tx02, tx03, tx04}
	hasTxsBlock02 := newBlock(999999, txs)
	blocksInTime02.SaveBlock(hasTxsBlock02)
	// 第一个时间区间保存了两个区块
	assert.Equal(t, len(blocksInTime02.BlockSet[0]), 2)
	// 比较第二个区块
	assert.Equal(t, blocksInTime02.BlockSet[0][hasTxsBlock02.Hash()].Header, hasTxsBlock02.Header)
	txHashes := blocksInTime02.BlockSet[0][hasTxsBlock02.Hash()].TxHashes
	assert.Equal(t, len(txHashes), 4)
	_, ok := txHashes[tx01.Hash()]
	assert.True(t, ok)
	_, ok = txHashes[tx02.Hash()]
	assert.True(t, ok)
	_, ok = txHashes[tx03.Hash()]
	assert.True(t, ok)
	_, ok = txHashes[tx04.Hash()]
	assert.True(t, ok)

	// 3. 测试保存的block时间区间能落到目前已经保存的时间区间里的正常情况
	blocksInTime03 := &BlocksInTime{
		LastMaxTime: 100,
		BlockSet:    make([]BlocksByHash, 3),
	}
	// 区块时间戳为[100,160)区间保存在BlockSet的index == 0中
	tx0 := makeTxRandom(common.HexToAddress("0x33"))
	block0 := newBlock(150, types.Transactions{tx0})
	blocksInTime03.SaveBlock(block0)

	// 区块时间戳为[160,220)区间保存在BlockSet的index == 1 中
	tx1 := makeTxRandom(common.HexToAddress("0x44"))
	block1 := newBlock(160, types.Transactions{tx1})
	blocksInTime03.SaveBlock(block1)

	// 区块时间戳为[220,280)区间保存在BlockSet的index == 2中
	tx2 := makeTxRandom(common.HexToAddress("0x55"))
	block2 := newBlock(221, types.Transactions{tx2})
	blocksInTime03.SaveBlock(block2)

	// 区块时间戳为[280, 340)区间保存在BlockSet的index == 3中，注意，此时BlockSet的最大index为2，所以这里会扩展BlockSet切片的长度到最大index为3
	tx3 := makeTxRandom(common.HexToAddress("0x66"))
	block3 := newBlock(300, types.Transactions{tx3})
	blocksInTime03.SaveBlock(block3)

	// 区块时间戳为[460, 520)区间保存在BlockSet的index == 6中，注意，此时BlockSet的最大index为3，所以这里会扩展BlockSet切片到最大index为6
	tx4 := makeTxRandom(common.HexToAddress("0x77"))
	block4 := newBlock(500, types.Transactions{tx4})
	blocksInTime03.SaveBlock(block4)

	/* 验证结果 **/
	// 验证BlockSet长度
	assert.Equal(t, len(blocksInTime03.BlockSet), 7)

	// 0. 验证BlockSet中index == 0保存的数据
	newBlock03, ok := blocksInTime03.BlockSet[0][block0.Hash()]
	assert.True(t, ok)
	assert.Equal(t, newBlock03.Header, block0.Header)
	// 验证交易存在，和保存的数量是否一致
	_, ok = newBlock03.TxHashes[tx0.Hash()]
	assert.True(t, ok)
	assert.Equal(t, len(newBlock03.TxHashes), 1)

	// // 1. 验证BlockSet中index == 1保存的数据
	// newBlock03,ok := blocksInTime03.BlockSet[0][block0.Hash()]
	// assert.True(t, ok)
	// assert.Equal(t, newBlock03.Header, block0.Header)
	// // 验证交易存在，和保存的数量是否一致
	// _,ok = newBlock03.TxHashes[tx0.Hash()]
	// assert.True(t, ok)
	// assert.Equal(t, len(newBlock03.TxHashes), 1)

}
