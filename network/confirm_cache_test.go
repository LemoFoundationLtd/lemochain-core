package network

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	hash_1  = common.Hash{0x01, 0x02, 0x03}
	hash_21 = common.Hash{0x02, 0x03, 0x04}
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

func Test_Push(t *testing.T) {
	cache := NewConfirmCache()
	simulateData(cache)

	assert.Len(t, cache.cache, 2)            // count of height
	assert.Len(t, cache.cache[1], 2)         // count of block in special height
	assert.Len(t, cache.cache[1][hash_1], 3) // count of confirms in special block
	assert.Len(t, cache.cache[2], 1)
	assert.Len(t, cache.cache[2][hash_32], 4)
}

func Test_Pop(t *testing.T) {
	cache := NewConfirmCache()
	simulatePop(cache)

	assert.Len(t, cache.cache[2], 2)

	confirms := cache.Pop(2, hash_32)
	assert.Len(t, confirms, 4)

	assert.Len(t, cache.cache[2], 1)

	confirms = cache.Pop(2, hash_42)
	assert.Len(t, confirms, 4)
	assert.Len(t, cache.cache[2], 0)
}

func Test_Clear(t *testing.T) {
	cache := NewConfirmCache()
	simulatePop(cache)
	cache.Clear(1)

	assert.Len(t, cache.cache, 1)
	assert.Len(t, cache.cache[1], 0)
	assert.Len(t, cache.cache[2], 2)
}
