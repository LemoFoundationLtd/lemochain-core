package txpool

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_NewTxGuard(t *testing.T) {
	guard := NewTxGuard(0)
	assert.Equal(t, uint32(0), guard.blockBuckets.TimeBase)

	guard = NewTxGuard(uint32(params.MaxTxLifeTime) + BucketDuration + 1)
	assert.Equal(t, BucketDuration, guard.blockBuckets.TimeBase)
}

func TestTxGuard_SaveBlock(t *testing.T) {
	cur := uint32(time.Now().Unix())
	guard := NewTxGuard(cur + uint32(params.MaxTxLifeTime))

	guard.SaveBlock(nil)

	// expired block
	tx1 := makeTx(101, 0)
	block1 := makeBlock(101, cur-BucketDuration, tx1)
	guard.SaveBlock(block1)
	assert.Equal(t, 0, len(guard.blockCache))
	assert.Equal(t, (*BlockNode)(nil), guard.blockCache.Get(block1.Hash()))
	assert.Equal(t, (HashList)(nil), guard.blockBuckets.buckets[0])
	assert.Equal(t, 0, len(guard.txTracer))

	// no tx block
	block2 := makeBlock(102, cur)
	guard.SaveBlock(block2)
	assert.Equal(t, 1, len(guard.blockCache))
	assert.Equal(t, block2.Hash(), guard.blockBuckets.buckets[0][0])
	assert.Equal(t, 0, len(guard.txTracer))

	// 2 txs block and in different time bucket
	tx2 := makeTx(102, 0)
	block3 := makeBlock(103, cur+BucketDuration, tx1, tx2)
	guard.SaveBlock(block3)
	assert.Equal(t, 2, len(guard.blockCache))
	assert.Equal(t, block3.Hash(), guard.blockBuckets.buckets[1][0])
	assert.Equal(t, 2, len(guard.txTracer))

	// same block
	guard.SaveBlock(block3)
	assert.Equal(t, 2, len(guard.blockCache))
	assert.Equal(t, (HashList)(nil), guard.blockBuckets.buckets[2])
	assert.Equal(t, 2, len(guard.blockBuckets.buckets[1])) // redundant. they are both block3
	assert.Equal(t, block3.Hash(), guard.blockBuckets.buckets[1][0])
	assert.Equal(t, block3.Hash(), guard.blockBuckets.buckets[1][1])
	assert.Equal(t, 2, len(guard.txTracer))

	// same tx in different blocks
	block4 := makeBlock(103, cur+BucketDuration, tx1)
	guard.SaveBlock(block4)
	assert.Equal(t, 3, len(guard.blockCache))
	assert.Equal(t, (HashList)(nil), guard.blockBuckets.buckets[2])
	assert.Equal(t, 2, len(guard.txTracer))
	tx1Trace := guard.txTracer.LoadTraces(types.Transactions{tx1})
	assert.Equal(t, 2, len(tx1Trace))
	assert.Equal(t, true, tx1Trace.Has(block3.Hash()))
	assert.Equal(t, true, tx1Trace.Has(block4.Hash()))
}

func TestTxGuard_ExistTxs(t *testing.T) {
	// block forks like this: (the number in bracket is transaction name)
	//          ┌─2(c)
	// 0───1(a)─┼─3(b)───6(c)
	//          ├─4────┬─7───9(bc)
	//          │      └─8
	//          └─5(box[cd])
	blocks := generateBlocks()
	txa := blocks[1].Txs[0]
	txb := blocks[3].Txs[0]
	txc := blocks[2].Txs[0]
	boxTx := blocks[5].Txs[0]
	txd := getSubTxs(boxTx)[1]
	txe := makeTx(0x10e, 100)
	guard := NewTxGuard(0)
	for _, b := range blocks {
		guard.SaveBlock(b)
	}

	test := func(expect bool, blockIndex int, txs ...*types.Transaction) {
		assert.Equal(t, expect, guard.ExistTxs(blocks[blockIndex].Hash(), txs))
	}

	test(false, 0, txa)
	test(true, 1, txa)
	test(false, 1, txb)
	test(false, 1, txe)
	test(true, 2, txa)
	test(false, 2, txb)
	test(true, 2, txc)
	test(true, 2, txb, txc)
	test(true, 2, txa, txc)
	test(true, 2, boxTx)
	test(false, 2, txd)
	test(false, 3, txc)
	test(false, 3, boxTx)
	test(true, 3, txc, txa)
	test(true, 4, txa)
	test(false, 4, txb)
	test(false, 4, boxTx)
	test(true, 5, txa)
	test(false, 5, txb)
	test(true, 5, txc)
	test(true, 5, boxTx)
	test(false, 8, txb)
	test(false, 8, boxTx)
	test(true, 9, txc)
	test(true, 9, boxTx)
}

func TestTxGuard_DelOldBlocks(t *testing.T) {
	// block forks like this: (the number in bracket is transaction name)
	//          ┌─2(c)
	// 0───1(a)─┼─3(b)───6(c)
	//          ├─4────┬─7───9(bc)
	//          │      └─8
	//          └─5(box[cd])
	// the blocks' time bucket belonging like this: (the number in bracket is index of time bucket)
	//             ┌─2(1)
	// 0(0)───1(1)─┼─3(2)───6(3)
	//             ├─4(1)─┬─7(4)───9(4)
	//             │      └─8(3)
	//             └─5(2)
	blocks := generateBlocks()
	txa := blocks[1].Txs[0]
	txb := blocks[3].Txs[0]
	txc := blocks[2].Txs[0]
	boxTx := blocks[5].Txs[0]
	txd := getSubTxs(boxTx)[1]
	cur := uint32(time.Now().Unix())
	guard := NewTxGuard(cur + uint32(params.MaxTxLifeTime))
	for _, b := range blocks {
		guard.SaveBlock(b)
	}

	// time is smaller than bucket base time
	assert.PanicsWithValue(t, ErrInvalidBaseTime, func() {
		guard.DelOldBlocks(0)
	})

	// expire nothing
	guard.DelOldBlocks(cur + uint32(params.MaxTxLifeTime))
	assert.NotNil(t, guard.blockCache.Get(blocks[0].Hash()))

	// expire bucket 0. delete block 0, no transactions
	guard.DelOldBlocks(cur + uint32(params.MaxTxLifeTime) + BucketDuration)
	assert.Nil(t, guard.blockCache.Get(blocks[0].Hash()))
	assert.NotNil(t, guard.blockCache.Get(blocks[1].Hash()))
	assert.Equal(t, true, guard.ExistTx(blocks[9].Hash(), txa))

	// expire bucket 1. delete block 1,2,4
	guard.DelOldBlocks(cur + uint32(params.MaxTxLifeTime) + BucketDuration*2)
	assert.Nil(t, guard.blockCache.Get(blocks[1].Hash()))
	assert.Nil(t, guard.blockCache.Get(blocks[2].Hash()))
	assert.Nil(t, guard.blockCache.Get(blocks[4].Hash()))
	assert.NotNil(t, guard.blockCache.Get(blocks[3].Hash()))
	assert.NotNil(t, guard.blockCache.Get(blocks[5].Hash()))
	assert.Equal(t, true, guard.ExistTx(blocks[6].Hash(), txb))
	assert.Equal(t, false, guard.ExistTx(blocks[9].Hash(), txa))
	assert.Equal(t, false, guard.ExistTx(blocks[9].Hash(), txc)) // txc is exist in block 9, but not detected. because the newest stable block is later than txc's expiration time, txc is expired and no longer be packaged by a new block. It wouldn't be happen that somewhere test txc if it exist
	assert.Equal(t, true, guard.ExistTxs(blocks[9].Hash(), types.Transactions{txb, txd}))
	assert.Equal(t, false, guard.ExistTx(blocks[8].Hash(), txc))

	// reset guard
	guard = NewTxGuard(cur + uint32(params.MaxTxLifeTime))
	for _, b := range blocks {
		guard.SaveBlock(b)
	}
	// expire bucket 0,1,2. only block 6,7,8,9 left
	guard.DelOldBlocks(cur + uint32(params.MaxTxLifeTime) + BucketDuration*3)
	assert.Nil(t, guard.blockCache.Get(blocks[0].Hash()))
	assert.Nil(t, guard.blockCache.Get(blocks[1].Hash()))
	assert.Nil(t, guard.blockCache.Get(blocks[2].Hash()))
	assert.Nil(t, guard.blockCache.Get(blocks[3].Hash()))
	assert.Nil(t, guard.blockCache.Get(blocks[4].Hash()))
	assert.Nil(t, guard.blockCache.Get(blocks[5].Hash()))
	assert.NotNil(t, guard.blockCache.Get(blocks[6].Hash()))
	assert.NotNil(t, guard.blockCache.Get(blocks[7].Hash()))
	assert.NotNil(t, guard.blockCache.Get(blocks[8].Hash()))
	assert.NotNil(t, guard.blockCache.Get(blocks[9].Hash()))
	assert.Equal(t, false, guard.ExistTx(blocks[6].Hash(), txa))
	assert.Equal(t, false, guard.ExistTx(blocks[6].Hash(), txb))
	assert.Equal(t, false, guard.ExistTx(blocks[6].Hash(), txc))
	assert.Equal(t, false, guard.ExistTx(blocks[9].Hash(), txd))
	assert.Equal(t, false, guard.ExistTx(blocks[9].Hash(), boxTx))
	assert.Equal(t, false, guard.ExistTx(blocks[9].Hash(), txc))
	assert.Equal(t, false, guard.ExistTx(blocks[8].Hash(), txc))

	// expire all buckets. nothing left
	guard.DelOldBlocks(cur + uint32(params.MaxTxLifeTime) + BucketDuration*10)
	assert.Nil(t, guard.blockCache.Get(blocks[8].Hash()))
	assert.Nil(t, guard.blockCache.Get(blocks[9].Hash()))
	assert.Equal(t, 0, len(guard.blockCache))
	assert.Equal(t, 0, len(guard.txTracer))
}

func TestTxGuard_GetTxsByBranch(t *testing.T) {
	// block forks like this: (the number in bracket is transaction name)
	//          ┌─2(c)
	// 0───1(a)─┼─3(b)───6(c)
	//          ├─4────┬─7───9(bc)
	//          │      └─8
	//          └─5(box[cd])
	blocks := generateBlocks()
	txb := blocks[3].Txs[0]
	txc := blocks[2].Txs[0]
	boxTx := blocks[5].Txs[0]
	guard := NewTxGuard(0)
	for _, b := range blocks {
		guard.SaveBlock(b)
	}

	type testConfig struct {
		blockIndex1 int
		blockIndex2 int
		txs1        types.Transactions
		txs2        types.Transactions
	}
	tests := []testConfig{
		{9, 9, types.Transactions{}, types.Transactions{}},
		{9, 8, types.Transactions{txb, txc}, types.Transactions{}},
		{9, 6, types.Transactions{txb, txc}, types.Transactions{txc, txb}},
		{7, 6, types.Transactions{}, types.Transactions{txc, txb}},
		{7, 5, types.Transactions{}, types.Transactions{boxTx}},
		{3, 5, types.Transactions{txb}, types.Transactions{boxTx}},
		{3, 2, types.Transactions{txb}, types.Transactions{txc}},
		{3, 1, types.Transactions{txb}, types.Transactions{}},
		{3, 6, types.Transactions{}, types.Transactions{txc}},
	}

	for i, test := range tests {
		caseName := fmt.Sprintf("case %d. fork%d vs fork%d", i, test.blockIndex1, test.blockIndex2)
		t.Run(caseName, func(t *testing.T) {
			test := test // capture range variable
			t.Parallel()

			txs1, txs2, err := guard.GetTxsByBranch(blocks[test.blockIndex1], blocks[test.blockIndex2])
			assert.NoError(t, err)
			assert.Equal(t, len(test.txs1), len(txs1))
			for i, tx := range test.txs1 {
				assert.Equal(t, tx, txs1[i], "index", i)
			}
			assert.Equal(t, len(test.txs2), len(txs2))
			for i, tx := range test.txs2 {
				assert.Equal(t, tx, txs2[i], "index", i)
			}
		})
	}
}

func TestTxGuard_GetTxsByBranch_Error(t *testing.T) {
	guard := NewTxGuard(0)
	_, _, err := guard.GetTxsByBranch(makeBlock(1, 1), makeBlock(2, 2))
	assert.Equal(t, ErrNotFoundBlockCache, err)
}
