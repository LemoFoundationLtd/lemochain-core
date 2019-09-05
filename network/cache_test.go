package network

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"sync"
	"testing"
	"time"
)

var (
	hash_1  = common.Hash{0x01, 0x02, 0x03}
	hash_11 = common.Hash{0x01, 0x02, 0x04}
	hash_21 = common.Hash{0x02, 0x03, 0x04}
	hash_22 = common.Hash{0x02, 0x04, 0x04}
	hash_32 = common.Hash{0x03, 0x04, 0x05}
	hash_42 = common.Hash{0x04, 0x05, 0x06}
)

func simulateData(cache *ConfirmCache) {

	data_1 := &BlockConfirmData{
		Hash:     hash_1,
		Height:   1,
		SignInfo: types.SignData{0x11},
	}
	data_2 := &BlockConfirmData{
		Hash:     hash_1,
		Height:   1,
		SignInfo: types.SignData{0x12},
	}
	data_3 := &BlockConfirmData{
		Hash:     hash_1,
		Height:   1,
		SignInfo: types.SignData{0x13},
	}
	cache.Push(data_1)
	cache.Push(data_2)
	cache.Push(data_3)

	data_21 := &BlockConfirmData{
		Hash:     hash_21,
		Height:   1,
		SignInfo: types.SignData{0x21},
	}
	data_22 := &BlockConfirmData{
		Hash:     hash_21,
		Height:   1,
		SignInfo: types.SignData{0x22},
	}
	data_23 := &BlockConfirmData{
		Hash:     hash_21,
		Height:   1,
		SignInfo: types.SignData{0x23},
	}
	cache.Push(data_21)
	cache.Push(data_22)
	cache.Push(data_23)

	data_32 := &BlockConfirmData{
		Hash:     hash_32,
		Height:   2,
		SignInfo: types.SignData{0x31},
	}
	data_33 := &BlockConfirmData{
		Hash:     hash_32,
		Height:   2,
		SignInfo: types.SignData{0x32},
	}
	data_34 := &BlockConfirmData{
		Hash:     hash_32,
		Height:   2,
		SignInfo: types.SignData{0x33},
	}
	data_35 := &BlockConfirmData{
		Hash:     hash_32,
		Height:   2,
		SignInfo: types.SignData{0x34},
	}
	cache.Push(data_32)
	cache.Push(data_33)
	cache.Push(data_34)
	cache.Push(data_35)
}

func simulatePop(cache *ConfirmCache) {
	simulateData(cache)

	data_42 := &BlockConfirmData{
		Hash:     hash_42,
		Height:   2,
		SignInfo: types.SignData{0x41},
	}
	data_43 := &BlockConfirmData{
		Hash:     hash_42,
		Height:   2,
		SignInfo: types.SignData{0x42},
	}
	data_44 := &BlockConfirmData{
		Hash:     hash_42,
		Height:   2,
		SignInfo: types.SignData{0x43},
	}
	data_45 := &BlockConfirmData{
		Hash:     hash_42,
		Height:   2,
		SignInfo: types.SignData{0x44},
	}
	cache.Push(data_42)
	cache.Push(data_43)
	cache.Push(data_44)
	cache.Push(data_45)
}

func Test_Confirm_Push(t *testing.T) {
	cache := NewConfirmCache()
	simulateData(cache)
	assert.Equal(t, 10, cache.Size())

	assert.Len(t, cache.cache, 2)            // count of height
	assert.Len(t, cache.cache[1], 2)         // count of block in special height
	assert.Len(t, cache.cache[1][hash_1], 3) // count of confirms in special block
	assert.Len(t, cache.cache[2], 1)
	assert.Len(t, cache.cache[2][hash_32], 4)
}

func Test_Confirm_Pop(t *testing.T) {
	cache := NewConfirmCache()
	simulatePop(cache)

	assert.Len(t, cache.cache[2], 2)

	confirms := cache.Pop(2, hash_32)
	assert.Len(t, confirms, 4)

	assert.Len(t, cache.cache[2], 1)

	assert.Nil(t, cache.Pop(10, common.Hash{}))
	assert.Nil(t, cache.Pop(2, common.Hash{}))

	confirms = cache.Pop(2, hash_42)
	assert.Len(t, confirms, 4)
	assert.Equal(t, 6, cache.Size())
	assert.Len(t, cache.cache[2], 0)
}

func Test_Confirm_Clear(t *testing.T) {
	cache := NewConfirmCache()
	simulatePop(cache)
	cache.Clear(1)

	assert.Len(t, cache.cache, 1)
	assert.Len(t, cache.cache[1], 0)
	assert.Len(t, cache.cache[2], 2)
}

func createBlocks() types.Blocks {
	b1 := new(types.Block)
	b1.Header = new(types.Header)
	b1.Header.Height = 3
	b1.Header.Time = uint32(time.Now().Unix()) + rand.Uint32()

	b2 := new(types.Block)
	b2.Header = new(types.Header)
	b2.Header.Height = 3
	b2.Header.Time = uint32(time.Now().Unix()) + rand.Uint32()

	b3 := new(types.Block)
	b3.Header = new(types.Header)
	b3.Header.Height = 4
	b3.Header.Time = uint32(time.Now().Unix()) + rand.Uint32()

	b4 := new(types.Block)
	b4.Header = new(types.Header)
	b4.Header.Height = 4
	b4.Header.Time = uint32(time.Now().Unix()) + rand.Uint32()

	b5 := new(types.Block)
	b5.Header = new(types.Header)
	b5.Header.Height = 5
	b5.Header.Time = uint32(time.Now().Unix()) + rand.Uint32()

	b6 := new(types.Block)
	b6.Header = new(types.Header)
	b6.Header.Height = 6
	b6.Header.Time = uint32(time.Now().Unix()) + rand.Uint32()

	b7 := new(types.Block)
	b7.Header = new(types.Header)
	b7.Header.Height = 9
	b7.Header.Time = uint32(time.Now().Unix()) + rand.Uint32()

	b8 := new(types.Block)
	b8.Header = new(types.Header)
	b8.Header.Height = 12
	b8.Header.Time = uint32(time.Now().Unix()) + rand.Uint32()

	blocks := make([]*types.Block, 8)
	blocks[0] = b1
	blocks[1] = b2
	blocks[2] = b3
	blocks[3] = b4
	blocks[4] = b5
	blocks[5] = b6
	blocks[6] = b7
	blocks[7] = b8
	return blocks
}

func Test_Block_Add(t *testing.T) {
	cache := NewBlockCache()
	blocks := createBlocks()
	for _, b := range blocks {
		cache.Add(b)
	}
	assert.Equal(t, 6, len(cache.cache))
	assert.Equal(t, len(blocks), cache.Size())

	b1 := new(types.Block)
	b1.Header = new(types.Header)
	b1.Header.Height = 1
	b1.Header.Time = uint32(time.Now().Unix()) + rand.Uint32()
	b2 := new(types.Block)
	b2.Header = new(types.Header)
	b2.Header.Height = 21
	b2.Header.Time = uint32(time.Now().Unix()) + rand.Uint32()
	cache.Add(b1)
	cache.Add(b2)
	assert.Equal(t, uint32(1), cache.FirstHeight())
	assert.Equal(t, uint32(21), cache.cache[len(cache.cache)-1].Height)

	b3 := new(types.Block)
	b3.Header = new(types.Header)
	b3.Header.Height = 21
	b3.Header.Time = uint32(time.Now().Unix()) + rand.Uint32()
	cache.Add(b3)
	assert.Equal(t, 11, cache.Size())

	b4 := new(types.Block)
	b4.Header = new(types.Header)
	b4.Header.Height = 20
	b4.Header.Time = uint32(time.Now().Unix()) + rand.Uint32()
	cache.Add(b4)
	assert.Equal(t, 12, cache.Size())
}

func Test_Block_Iterate(t *testing.T) {
	cache := NewBlockCache()
	blocks := createBlocks()
	for _, b := range blocks {
		cache.Add(b)
	}

	cache.Iterate(func(b *types.Block) bool {
		if b.Height() == uint32(4) {
			return true
		}
		return false
	})

	assert.Equal(t, 6, cache.Size())
}

func Test_Block_Clear(t *testing.T) {
	cache := NewBlockCache()
	blocks := createBlocks()
	for _, b := range blocks {
		cache.Add(b)
	}

	cache.Clear(uint32(5))

	assert.Equal(t, 3, cache.Size())
	assert.Equal(t, uint32(6), cache.FirstHeight())
}

func TestBlockBlackCache_IsBlackBlock(t *testing.T) {
	blackHash := common.HexToHash("0x111")
	cache := make(map[common.Hash]struct{})
	cache[blackHash] = struct{}{}
	bbc := &invalidBlockCache{
		HashSet: HashSet{
			cache: cache,
			Mutex: sync.Mutex{},
		},
	}
	// 1. 验证当前区块为黑名单的情况
	assert.True(t, bbc.IsBlackBlock(blackHash, common.HexToHash("0x222")))
	// 2. 验证父块为黑名单块
	assert.True(t, bbc.IsBlackBlock(common.HexToHash("0x222"), blackHash))
	// 此时common.HexToHash("0x222")也会被加入到黑名单列表中
	_, ok := bbc.cache[common.HexToHash("0x222")]
	assert.True(t, ok)
	// 3. 验证都不在黑名单列表中的情况
	assert.False(t, bbc.IsBlackBlock(common.HexToHash("0x333"), common.HexToHash("0x444")))
	// 4. 验证黑名单为空的情况
	bbc.cache = make(map[common.Hash]struct{})
	assert.False(t, bbc.IsBlackBlock(blackHash, common.HexToHash("0x222")))
}
