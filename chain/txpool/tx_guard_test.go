package txpool

import (
	"crypto/rand"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func newBlock(height, timestamp uint32, txs types.Transactions) *types.Block {
	return &types.Block{
		Header: &types.Header{
			TxRoot: txs.MerkleRootSha(),
			Time:   timestamp,
			Height: height,
		},
		Txs: txs,
	}
}

// TestBlocksInTime_SaveBlock
func TestBlocksInTime_SaveBlock(t *testing.T) {
	/**
	测试要点：
	1、测试保存的block时间戳小于lastMaxTime的异常情况
	2、测试保存的block时间戳等于lastMaxTime的正常情况
	3、测试保存的block时间区间能落到目前已经保存的时间区间里的正常情况
	*/
	// 1. 测试保存的block时间戳小于lastMaxTime的异常情况
	blocksInTime01 := newBlocksInTime(999999)
	block01 := newBlock(1, 8888, nil)
	err := blocksInTime01.SaveBlock(block01)
	// 比较blockSet中是否有数据
	assert.Equal(t, len(blocksInTime01.BlockSet), 0)
	assert.Error(t, err)
	// 2. 测试保存的block时间戳等于lastMaxTime的正常情况
	blocksInTime02 := newBlocksInTime(999999)
	block02 := newBlock(1, 999999, nil)
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
	hasTxsBlock02 := newBlock(1, 999999, txs)
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
	block0 := newBlock(1, 150, types.Transactions{tx0})
	blocksInTime03.SaveBlock(block0)

	// 区块时间戳为[160,220)区间保存在BlockSet的index == 1 中
	tx1 := makeTxRandom(common.HexToAddress("0x44"))
	block1 := newBlock(1, 160, types.Transactions{tx1})
	blocksInTime03.SaveBlock(block1)

	// 区块时间戳为[220,280)区间保存在BlockSet的index == 2中
	tx2 := makeTxRandom(common.HexToAddress("0x55"))
	block2 := newBlock(1, 221, types.Transactions{tx2})
	blocksInTime03.SaveBlock(block2)

	// 区块时间戳为[280, 340)区间保存在BlockSet的index == 3中，注意，此时BlockSet的最大index为2，所以这里会扩展BlockSet切片的长度到最大index为3
	tx3 := makeTxRandom(common.HexToAddress("0x66"))
	block3 := newBlock(1, 300, types.Transactions{tx3})
	blocksInTime03.SaveBlock(block3)

	// 区块时间戳为[460, 520)区间保存在BlockSet的index == 6中，注意，此时BlockSet的最大index为3，所以这里会扩展BlockSet切片到最大index为6
	tx6 := makeTxRandom(common.HexToAddress("0x77"))
	block6 := newBlock(1, 500, types.Transactions{tx6})
	blocksInTime03.SaveBlock(block6)

	/* 验证结果 **/
	// 验证BlockSet长度
	assert.Equal(t, len(blocksInTime03.BlockSet), 7)

	// 0. 验证BlockSet中index == 0保存的数据
	newBlock0, ok := blocksInTime03.BlockSet[0][block0.Hash()]
	assert.True(t, ok)
	assert.Equal(t, newBlock0.Header, block0.Header)
	// 验证交易存在，和保存的数量是否一致
	_, ok = newBlock0.TxHashes[tx0.Hash()]
	assert.True(t, ok)
	assert.Equal(t, len(newBlock0.TxHashes), 1)

	// 1. 验证BlockSet中index == 1保存的数据
	newBlock1, ok := blocksInTime03.BlockSet[1][block1.Hash()]
	assert.True(t, ok)
	assert.Equal(t, newBlock1.Header, block1.Header)
	// 验证交易存在，和保存的数量是否一致
	_, ok = newBlock1.TxHashes[tx1.Hash()]
	assert.True(t, ok)
	assert.Equal(t, len(newBlock1.TxHashes), 1)

	// 2. 验证BlockSet中index == 2保存的数据
	newBlock2, ok := blocksInTime03.BlockSet[2][block2.Hash()]
	assert.True(t, ok)
	assert.Equal(t, newBlock2.Header, block2.Header)
	// 验证交易存在，和保存的数量是否一致
	_, ok = newBlock2.TxHashes[tx2.Hash()]
	assert.True(t, ok)
	assert.Equal(t, len(newBlock2.TxHashes), 1)

	// 3. 验证BlockSet中index == 3保存的数据
	newBlock3, ok := blocksInTime03.BlockSet[3][block3.Hash()]
	assert.True(t, ok)
	assert.Equal(t, newBlock3.Header, block3.Header)
	// 验证交易存在，和保存的数量是否一致
	_, ok = newBlock3.TxHashes[tx3.Hash()]
	assert.True(t, ok)
	assert.Equal(t, len(newBlock3.TxHashes), 1)

	// 4. BlockSet中index == 4和5中都没有存储数据
	assert.Equal(t, 0, len(blocksInTime03.BlockSet[4]))
	assert.Equal(t, 0, len(blocksInTime03.BlockSet[5]))

	// 5. 验证BlockSet中index == 6保存的数据
	newBlock6, ok := blocksInTime03.BlockSet[6][block6.Hash()]
	assert.True(t, ok)
	assert.Equal(t, newBlock6.Header, block6.Header)
	// 验证交易存在，和保存的数量是否一致
	_, ok = newBlock6.TxHashes[tx6.Hash()]
	assert.True(t, ok)
	assert.Equal(t, len(newBlock6.TxHashes), 1)
}

// TestBlocksInTime_DelBlock
func TestBlocksInTime_DelBlock(t *testing.T) {
	/**
	测试要点:
	1. 删除的block时间戳小于lastMaxTime的异常情况
	2. 删除的block时间戳大于存储的时间区间的最大值的异常情况
	3. 删除不存在的block
	4. 删除存在的block的正常情况
	*/

	// 0. init初始化一个存储了block的blocksInTime的结构体
	blocksInTime := newBlocksInTime(1000) // 构造一个lastMaxTime == 1000的对象实例
	// 保存区块
	block01 := newBlock(1, 1000, nil) // 保存进第一个时间片段[1000-1060)
	blocksInTime.SaveBlock(block01)
	block02 := newBlock(1, 1061, nil) // 保存进第二个时间片段[1060-1120)
	blocksInTime.SaveBlock(block02)
	block03 := newBlock(1, 1179, nil) // 保存进第三个时间片段[1120-1180)
	blocksInTime.SaveBlock(block03)
	// 检测这三个block是否成功保存进去
	assert.Equal(t, 3, len(blocksInTime.BlockSet[0])+len(blocksInTime.BlockSet[1])+len(blocksInTime.BlockSet[2]))

	// 1. 删除的block时间戳小于lastMaxTime的异常情况
	delBlock01 := newBlock(1, 999, nil) // 构造一个时间戳小于lastMaxTime的block
	blocksInTime.DelBlock(delBlock01)
	// 验证没有删除成功
	assert.Equal(t, 3, len(blocksInTime.BlockSet[0])+len(blocksInTime.BlockSet[1])+len(blocksInTime.BlockSet[2]))

	// 2. 删除的block时间戳大于存储的时间区间的最大值的异常情况
	delBlock02 := newBlock(1, 1180, nil)
	blocksInTime.DelBlock(delBlock02)
	// 验证没有删除成功
	assert.Equal(t, 3, len(blocksInTime.BlockSet[0])+len(blocksInTime.BlockSet[1])+len(blocksInTime.BlockSet[2]))

	// 3. 删除不存在的block
	delBlock03 := newBlock(1, 1070, nil) // 构造一个满足时间区间但是没有被保存进去的区块
	blocksInTime.DelBlock(delBlock03)
	// 验证没有删除成功
	assert.Equal(t, 3, len(blocksInTime.BlockSet[0])+len(blocksInTime.BlockSet[1])+len(blocksInTime.BlockSet[2]))

	// 4. 删除存在的block的正常情况
	blocksInTime.DelBlock(block01)
	assert.Equal(t, 0, len(blocksInTime.BlockSet[0]))
	assert.Equal(t, 1, len(blocksInTime.BlockSet[1]))
	assert.Equal(t, 1, len(blocksInTime.BlockSet[2]))
	blocksInTime.DelBlock(block02)
	assert.Equal(t, 0, len(blocksInTime.BlockSet[0]))
	assert.Equal(t, 0, len(blocksInTime.BlockSet[1]))
	assert.Equal(t, 1, len(blocksInTime.BlockSet[2]))
	blocksInTime.DelBlock(block03)
	assert.Equal(t, 0, len(blocksInTime.BlockSet[0]))
	assert.Equal(t, 0, len(blocksInTime.BlockSet[1]))
	assert.Equal(t, 0, len(blocksInTime.BlockSet[2]))
}

// TestBlocksInTime_DelOldBlocks
func TestBlocksInTime_DelOldBlocks(t *testing.T) {
	/**
	测试要点：
	1. 传入的时间戳小于lastMaxTime + 60则不作改变并返回nil
	2. 如果传入的时间戳大于最大的时间片段的值，则返回存储的所有block，并修改lastMaxTime和内存中存储的block会变为空
	3. 正常情况
	*/
	// 0. init初始化一个存储了block的blocksInTime的结构体
	tx01 := makeTxRandom(common.HexToAddress("0x111"))
	tx02 := makeTxRandom(common.HexToAddress("0x222"))
	tx03 := makeTxRandom(common.HexToAddress("0x333"))
	tx04 := makeTxRandom(common.HexToAddress("0x444"))
	blocksInTime := newBlocksInTime(1000) // 构造一个lastMaxTime == 1000的对象实例
	// 保存区块
	block01 := newBlock(1, 1000, types.Transactions{tx01, tx02}) // 保存进第一个时间片段[1000-1060)
	blocksInTime.SaveBlock(block01)
	block02 := newBlock(1, 1061, types.Transactions{tx03}) // 保存进第二个时间片段[1060-1120)
	blocksInTime.SaveBlock(block02)
	block03 := newBlock(1, 1179, types.Transactions{tx04}) // 保存进第三个时间片段[1120-1180)
	blocksInTime.SaveBlock(block03)

	// 1. 传入的时间戳小于lastMaxTime + 60则不作改变并返回nil
	maxTime01 := blocksInTime.LastMaxTime + 59
	getBlocks := blocksInTime.DelOldBlocks(maxTime01)
	assert.Equal(t, []*Block(nil), getBlocks)

	// 2. 如果传入的时间戳大于最大的时间片段的值，则返回存储的所有block，并修改lastMaxTime和内存中存储的block会变为空
	max := blocksInTime.LastMaxTime + uint32(len(blocksInTime.BlockSet)*60) - 1 // 时间片段的最大时间
	maxTime02 := max + 1                                                        // 设置一个大于max的值
	getBlocks = blocksInTime.DelOldBlocks(maxTime02)
	// 查看是否返回了缓存中的所有block
	assert.Equal(t, 3, len(getBlocks))
	// 比较区块头相等
	assert.Equal(t, block01.Header, getBlocks[0].Header)
	assert.Equal(t, block02.Header, getBlocks[1].Header)
	assert.Equal(t, block03.Header, getBlocks[2].Header)
	// 比较交易hash相等
	assert.True(t, cmpTxHash(block01, getBlocks[0]))
	assert.True(t, cmpTxHash(block02, getBlocks[1]))
	assert.True(t, cmpTxHash(block03, getBlocks[2]))
	// 查看lastMaxTime是否已经改变
	assert.Equal(t, maxTime02, blocksInTime.LastMaxTime)
	// 查看BlockSet是否为空
	assert.Equal(t, 0, len(blocksInTime.BlockSet))
	// 3. 正常情况
	blocksInTime = newBlocksInTime(1000) // 构造一个lastMaxTime == 1000的对象实例
	// 保存区块
	block011 := newBlock(1, 1000, types.Transactions{tx01}) // 保存进第一个时间片段[1000-1060)
	blocksInTime.SaveBlock(block011)
	block012 := newBlock(1, 1059, nil) // 保存进第一个时间片段[1000-1060)
	blocksInTime.SaveBlock(block012)
	block021 := newBlock(1, 1060, types.Transactions{tx03, tx04}) // 保存进第二个时间片段[1060-1120)
	blocksInTime.SaveBlock(block021)
	// 第三个时间片段为nil
	block033 := newBlock(1, 1200, types.Transactions{tx01, tx02, tx03, tx04}) // 保存进第四个时间片段[1180-1240)
	blocksInTime.SaveBlock(block033)
	assert.Equal(t, 4, len(blocksInTime.BlockSet))

	// 删除第一个时间片段
	maxTime := uint32(1065)                        // 时间要落在第二个时间片段中
	getBlocks = blocksInTime.DelOldBlocks(maxTime) // 返回包括block011和block012
	assert.Equal(t, 2, len(getBlocks))
	if cmpTxHash(block011, getBlocks[0]) {
		assert.True(t, cmpTxHash(block012, getBlocks[1]))
	} else {
		assert.True(t, cmpTxHash(block012, getBlocks[0]))
		assert.True(t, cmpTxHash(block011, getBlocks[1]))
	}
	assert.Equal(t, 3, len(blocksInTime.BlockSet))
	assert.Equal(t, uint32(1060), blocksInTime.LastMaxTime)

	// 现在只剩下第2,3,4时间片段
	// 删除第二个和第三个时间片段
	maxTime = 1180                                 // 时间要落在第四个时间片段中
	getBlocks = blocksInTime.DelOldBlocks(maxTime) // 返回block021，第三个时间片段是空所以没有区块返回
	assert.Equal(t, 1, len(getBlocks))
	assert.True(t, cmpTxHash(block021, getBlocks[0]))
	assert.Equal(t, uint32(1180), blocksInTime.LastMaxTime)
	assert.Equal(t, 1, len(blocksInTime.BlockSet))

	// 删除第四个时间片段
	maxTime = 99999                                // 设置一个远大于最大时间片段的时间值
	getBlocks = blocksInTime.DelOldBlocks(maxTime) // 返回第四个时间片段中保存的block033
	assert.Equal(t, 1, len(getBlocks))
	assert.True(t, cmpTxHash(block033, getBlocks[0]))
	assert.Equal(t, maxTime, blocksInTime.LastMaxTime) // maxTime赋值给lastMaxTime
	assert.Equal(t, 0, len(blocksInTime.BlockSet))
}

// 比较保存之前和之后的block中的交易是否相等
func cmpTxHash(block *types.Block, cacheBlock *Block) bool {
	blockTxs := make([]common.Hash, 0)
	for _, tx := range block.Txs {
		blockTxs = append(blockTxs, tx.Hash())
	}
	// 数量必须相等
	if len(blockTxs) != len(cacheBlock.TxHashes) {
		return false
	}
	// 遍历blockTxs中的交易hash一定能在cacheBlock.TxHashes中找到
	for _, h := range blockTxs {
		if _, ok := cacheBlock.TxHashes[h]; !ok {
			return false
		}
	}
	return true
}

func TestTxGuard_getMaxAndMinHeight(t *testing.T) {
	trace := make(Trace)
	trace[common.HexToHash("0x111")] = 0
	trace[common.HexToHash("0x222")] = 999

	for i := 0; i < 10; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(999))
		trace[common.HexToHash("0xaa"+n.String())] = uint32(n.Int64())
	}
	minHeight, maxHeight := trace.getMaxAndMinHeight()
	assert.Equal(t, uint32(0), minHeight)
	assert.Equal(t, uint32(999), maxHeight)
}

func TestTxGuard_SaveBlock(t *testing.T) {
	/**
	测试要点：
	1. 保存一个没有交易的block
	2. 保存一个有交易的block
	3. 重复保存相同的block
	4. 保存高度相同的block,但是block中的交易不同
	5. 保存高度相同的block,相同block中存在相同的交易
	6. 不同高度的区块中包含相同的tx
	*/
	guard := NewTxGuard(100)
	// 1. 保存一个没有交易的block
	block11 := newBlock(8888, 100, nil)
	guard.SaveBlock(block11)
	// 判断保存结果
	assert.Equal(t, 0, len(guard.Traces))        // 交易索引traces中没有交易
	assert.Equal(t, 1, len(guard.HeightBuckets)) // block桶中目前只存了这一个block
	assert.Equal(t, 1, len(guard.HeightBuckets[8888]))
	assert.Equal(t, block11.Header, guard.HeightBuckets[8888][block11.Hash()].Header) // 比较保存的区块的header
	assert.Equal(t, 0, len(guard.HeightBuckets[8888][block11.Hash()].TxHashes))       // 保存的区块中的交易数量为0

	// 2. 保存一个有交易的block
	guard = NewTxGuard(100)
	tx01 := makeTxRandom(common.HexToAddress("0x111"))
	tx02 := makeTxRandom(common.HexToAddress("0x222"))
	block21 := newBlock(9999, 100, types.Transactions{tx01, tx02})
	guard.SaveBlock(block21)
	// 验证
	assert.Equal(t, 2, len(guard.Traces))        // 交易索引traces中有两条交易
	assert.Equal(t, 1, len(guard.HeightBuckets)) // block桶中只存了这一个block
	assert.Equal(t, 1, len(guard.HeightBuckets[9999]))
	assert.Equal(t, block21.Header, guard.HeightBuckets[9999][block21.Hash()].Header) // 比较保存的区块的header
	assert.Equal(t, 2, len(guard.HeightBuckets[9999][block21.Hash()].TxHashes))       // 保存的区块中的交易数量为2

	// 3. 重复保存相同的block
	// 再次保存block21
	guard.SaveBlock(block21)
	// 验证，由于都是通过hash map来保存的，所以保存相同的block是会出现幂等的现象的
	// 所以结果会和上面的验证结果相同
	assert.Equal(t, 2, len(guard.Traces))        // 交易索引traces中有两条交易
	assert.Equal(t, 1, len(guard.HeightBuckets)) // block桶中只存了这一个block
	assert.Equal(t, 1, len(guard.HeightBuckets[9999]))
	assert.Equal(t, block21.Header, guard.HeightBuckets[9999][block21.Hash()].Header) // 比较保存的区块的header
	assert.Equal(t, 2, len(guard.HeightBuckets[9999][block21.Hash()].TxHashes))       // 保存的区块中的交易数量为2

	// 4. 保存高度相同的block,但是block中的交易不同
	tx03 := makeTxRandom(common.HexToAddress("0x333"))
	tx04 := makeTxRandom(common.HexToAddress("0x444"))
	block41 := newBlock(9999, 150, types.Transactions{tx03, tx04})
	guard.SaveBlock(block41)
	// 验证
	// Traces中有四笔交易，tx01和tx02对应着区块block21,tx03和tx04对应着block41。
	assert.Equal(t, 4, len(guard.Traces))
	// 目前每个交易只对应一个block
	assert.Equal(t, 1, len(guard.Traces[tx01.Hash()]))
	assert.Equal(t, 1, len(guard.Traces[tx02.Hash()]))
	assert.Equal(t, 1, len(guard.Traces[tx03.Hash()]))
	assert.Equal(t, 1, len(guard.Traces[tx04.Hash()]))
	// 每个block的高度都为9999
	assert.Equal(t, block21.Height(), guard.Traces[tx01.Hash()][block21.Hash()])
	assert.Equal(t, block21.Height(), guard.Traces[tx02.Hash()][block21.Hash()])
	assert.Equal(t, block41.Height(), guard.Traces[tx03.Hash()][block41.Hash()])
	assert.Equal(t, block41.Height(), guard.Traces[tx04.Hash()][block41.Hash()])
	// HeightBuckets中高度为9999会有两个区块。
	assert.Equal(t, 1, len(guard.HeightBuckets))                                      // block桶中只存了一个高度中的block
	assert.Equal(t, 2, len(guard.HeightBuckets[9999]))                                // 这个高度中存在两个区块
	assert.Equal(t, block21.Header, guard.HeightBuckets[9999][block21.Hash()].Header) // 比较保存的区块的header
	assert.Equal(t, block41.Header, guard.HeightBuckets[9999][block41.Hash()].Header)
	// 比较保存之前与保存之后的区块中的交易是否相等
	assert.True(t, cmpTxHash(block21, guard.HeightBuckets[9999][block21.Hash()]))
	assert.True(t, cmpTxHash(block41, guard.HeightBuckets[9999][block41.Hash()]))

	// 5. 保存高度相同的block,相同block中存在相同的交易
	block51 := newBlock(9999, 150, types.Transactions{tx01, tx04}) // tx01和block21重合，tx04和block41重合
	guard.SaveBlock(block51)
	// 验证
	// 验证Traces中的tx01和tx04指向两个block
	assert.Equal(t, 2, len(guard.Traces[tx01.Hash()]))
	assert.Equal(t, 2, len(guard.Traces[tx04.Hash()]))
	// 验证tx01和tx04指向的具体block
	assert.Equal(t, block21.Height(), guard.Traces[tx01.Hash()][block21.Hash()])
	assert.Equal(t, block51.Height(), guard.Traces[tx01.Hash()][block51.Hash()])
	assert.Equal(t, block41.Height(), guard.Traces[tx04.Hash()][block41.Hash()])
	assert.Equal(t, block51.Height(), guard.Traces[tx04.Hash()][block51.Hash()])
	// 验证HeightBuckets中9999高度存在三个block
	assert.Equal(t, 3, len(guard.HeightBuckets[9999]))

	// 6. 不同高度的区块中包含相同的tx
	block61 := newBlock(6666, 200, types.Transactions{tx02, tx03})
	guard.SaveBlock(block61)
	// 验证
	// 验证Traces中tx02和tx03指向两个block
	assert.Equal(t, 2, len(guard.Traces[tx02.Hash()]))
	assert.Equal(t, 2, len(guard.Traces[tx03.Hash()]))
	// 验证tx02和tx03指向的具体block
	assert.Equal(t, block21.Height(), guard.Traces[tx02.Hash()][block21.Hash()])
	assert.Equal(t, block61.Height(), guard.Traces[tx02.Hash()][block61.Hash()])
	assert.Equal(t, block41.Height(), guard.Traces[tx03.Hash()][block41.Hash()])
	assert.Equal(t, block61.Height(), guard.Traces[tx03.Hash()][block61.Hash()])

	// 验证HeightBuckets中6666高度的区块
	assert.Equal(t, 1, len(guard.HeightBuckets[block61.Height()]))
	assert.True(t, cmpTxHash(block61, guard.HeightBuckets[block61.Height()][block61.Hash()]))
}

func TestTxGuard_DelOldBlocks(t *testing.T) {
	/**
	测试要点：
	1. 测试删除的block是不同高度，tx没有重复的情况
	2. 测试删除的block是存在相同高度，tx存在重复的交易
	*/
	guard := NewTxGuard(100)
	// 初始化测试数据
	tx11 := makeTxRandom(common.HexToAddress("0x111"))
	tx12 := makeTxRandom(common.HexToAddress("0x122"))
	tx21 := makeTxRandom(common.HexToAddress("0x211"))
	tx22 := makeTxRandom(common.HexToAddress("0x222"))

	block11 := newBlock(1, 100, types.Transactions{tx11}) // 第一时间片段[100,160)
	guard.SaveBlock(block11)
	block12 := newBlock(1, 159, types.Transactions{tx12})
	guard.SaveBlock(block12)

	block21 := newBlock(2, 160, types.Transactions{tx21}) // 第二时间片段[160,220)
	guard.SaveBlock(block21)
	block22 := newBlock(2, 219, types.Transactions{tx22})
	guard.SaveBlock(block22)

	// 验证初始化之后的数据
	assert.Equal(t, 4, len(guard.Traces))
	assert.Equal(t, 2, len(guard.HeightBuckets))

	// 1. 测试删除的block是不同高度，tx没有重复的情况
	maxTime := uint32(165) // 时间戳落在第二时间片段则删除第一时间片段
	guard.DelOldBlocks(maxTime)
	// 验证结果
	// 第一个时间片段的blocks中存储有block11，block12要被删除,HeightBuckets中高度为1的记录也会被删除掉
	assert.Equal(t, 1, len(guard.HeightBuckets))
	assert.Equal(t, block21.Header, guard.HeightBuckets[2][block21.Hash()].Header)
	assert.Equal(t, block22.Header, guard.HeightBuckets[2][block22.Hash()].Header)
	assert.True(t, cmpTxHash(block21, guard.HeightBuckets[2][block21.Hash()]))
	assert.True(t, cmpTxHash(block22, guard.HeightBuckets[2][block22.Hash()]))
	// 验证交易
	assert.Equal(t, 2, len(guard.Traces))
	assert.Equal(t, block21.Height(), guard.Traces[tx21.Hash()][block21.Hash()])
	assert.Equal(t, block22.Height(), guard.Traces[tx22.Hash()][block22.Hash()])

	// 2. 测试删除的block是存在相同高度，tx存在重复的交易
	// 接着上面的保存blocks进去
	block31 := newBlock(2, 230, types.Transactions{tx21, tx22}) // 第三时间片段[220,280),其中这个片段中的blocks保存的交易和区块高度都有和第二时间片段的相同
	guard.SaveBlock(block31)
	// 验证删除之前的数据
	assert.Equal(t, 1, len(guard.HeightBuckets))                               // 现在内存中只存了高度为2的区块
	assert.Equal(t, 3, len(guard.HeightBuckets[2]))                            // 高度为2的区块一共有3个
	assert.True(t, cmpTxHash(block31, guard.HeightBuckets[2][block31.Hash()])) // 验证新增加的block31保存进去的交易是否正确
	assert.Equal(t, 2, len(guard.Traces))                                      // 目前内存中只保存了两笔交易，tx21和tx22
	assert.Equal(t, uint32(2), guard.Traces[tx21.Hash()][block21.Hash()])      // tx21存在block21区块中
	assert.Equal(t, uint32(2), guard.Traces[tx21.Hash()][block31.Hash()])      // tx21存在block31区块中
	assert.Equal(t, uint32(2), guard.Traces[tx22.Hash()][block22.Hash()])      // tx22存在block22区块中
	assert.Equal(t, uint32(2), guard.Traces[tx22.Hash()][block31.Hash()])      // tx22存在block31区块中

	// 删除第二时间片段的blocks
	maxTime = 230
	guard.DelOldBlocks(maxTime)
	// 验证
	// 删除了第二时间片段的区块block21和block22
	assert.Equal(t, 1, len(guard.HeightBuckets[2]))                                // 高度为2的区块只剩下block31一个了
	assert.Equal(t, block31.Header, guard.HeightBuckets[2][block31.Hash()].Header) // 验证剩下的是block31
	assert.Equal(t, 2, len(guard.Traces))                                          // traces中还是存储了两笔交易tx21和tx22，因为block31中存在着两笔交易
	assert.Equal(t, 1, len(guard.Traces[tx21.Hash()]))                             // 此时tx21只存在一个区块中
	assert.Equal(t, uint32(2), guard.Traces[tx21.Hash()][block31.Hash()])          // 验证tx21只存在block31中
	assert.Equal(t, 1, len(guard.Traces[tx22.Hash()]))                             // 此时tx22只存在一个区块中
	assert.Equal(t, uint32(2), guard.Traces[tx22.Hash()][block31.Hash()])          // 验证tx22只存在block31中
}

func TestTxGuard_IsTxExist(t *testing.T) {

}

func TestTxGuard_IsTxsExist(t *testing.T) {

}

func TestTxGuard_GetTxsByBranch(t *testing.T) {

}
