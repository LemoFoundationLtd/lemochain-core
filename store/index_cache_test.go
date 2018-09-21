package store

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func NewBucket(index uint32) *CacheBucket {
	return NewCacheBucket(index, 256*256, 8)
}

func NewByte(size int) []byte {
	return make([]byte, size)
}

func NewPool(size uint32) *CachePool {
	return NewCachePool(256*256, uint8(size))
}

func NewByteWithData(size int, data string) []byte {
	var buf []byte = []byte(data)
	result := NewByte(size)
	copy(result[0:], buf)
	return result
}

func TestIndex_Remind(t *testing.T) {
	bucket := NewBucket(0)
	remind := bucket.Remind()
	assert.Equal(t, uint32(65536), remind)

	buf := make([]byte, 16*8)
	pos, err := bucket.SetUnit(buf)
	assert.NoError(t, err)
	assert.Equal(t, uint32(0), pos)

	remind = bucket.Remind()
	assert.Equal(t, uint32(65536-16), remind)

	buf = make([]byte, 16*8)
	pos, err = bucket.SetUnit(buf)
	assert.NoError(t, err)
	assert.Equal(t, uint32(16), pos)

	remind = bucket.Remind()
	assert.Equal(t, uint32(65536-32), remind)

	err = bucket.UpdateUint(pos, buf)
	assert.NoError(t, err)

	remind = bucket.Remind()
	assert.Equal(t, uint32(65536-32), remind)
}

func TestIndexCache_Set(t *testing.T) {
	bucket := NewBucket(0)

	bufLen := 16*8 - 1
	buf := NewByte(bufLen)
	_, err := bucket.SetUnit(buf)
	assert.Equal(t, ErrArgInvalid, err)

	// pos = 0
	bufLen = 16 * 8
	buf = NewByte(bufLen)
	pos, err := bucket.SetUnit(buf)
	assert.NoError(t, err)
	assert.Equal(t, uint32(0), pos)

	// pos = 16
	bufLen = 16 * 8
	buf = NewByte(bufLen)
	pos, err = bucket.SetUnit(buf)
	assert.NoError(t, err)
	assert.Equal(t, uint32(16), pos)

	data := NewByteWithData(16*8, "111chain111")
	err = bucket.UpdateUint(pos, data)
	assert.NoError(t, err)

	result, err := bucket.GetUnit(pos)
	assert.NoError(t, err)
	assert.Equal(t, data, result)

	// pos = 32
	data = NewByteWithData(16*8, "chain")
	pos, err = bucket.SetUnit(data)
	assert.NoError(t, err)
	assert.Equal(t, uint32(32), pos)

	result, err = bucket.GetUnit(pos)
	assert.NoError(t, err)
	assert.Equal(t, data, result)

	//
	bufLen = 256 * 256 * 8
	buf = NewByte(bufLen)
	_, err = bucket.SetUnit(buf)
	assert.Equal(t, ErrArgInvalid, err)
}

func TestCachePool(t *testing.T) {
	pool := NewPool(8)

	buf := NewByteWithData(16*8, "CHINA")
	tmp := NewByteWithData(16*8, "LEMO-CHINA")
	for bucketIndex := 0; bucketIndex < 8; bucketIndex++ {
		max := 256 * 16
		for index := 0; index < max; index++ {
			// set
			pos, err := pool.SetUnit(buf)
			assert.NoError(t, err)
			assert.Equal(t, GetPos(uint32(bucketIndex), uint32(index)*16), pos)

			// get
			result, err := pool.GetUnit(pos)
			assert.NoError(t, err)
			assert.Equal(t, buf, result)

			// update
			err = pool.UpdateUnit(pos, tmp)
			assert.NoError(t, err)
			result, err = pool.GetUnit(pos)
			assert.NotEqual(t, buf, result)
			assert.Equal(t, tmp, result)
		}
	}
}

func NewNodes(depth uint32, header uint32) []*Node {
	nodes := make([]*Node, 16)
	for index := 0; index < 16; index++ {
		nodes[index] = &Node{
			IsNode:  true,
			Depth:   depth,
			MaxCnt:  uint8(index),
			UsedCnt: 0,
			Header:  header,
		}
	}
	return nodes
}

func NewItems(pos uint32) []*TItem {
	items := make([]*TItem, 16)
	key := common.HexToHash("0x5fa2358263196dbbf23d1ca7a509451f7a2f64c15837bfbb81298b1e3e24e4fa")
	for index := 0; index < 16; index++ {
		items[index] = &TItem{
			Flg: 0,
			Num: uint32(index),
			Pos: pos,
			Key: key.Bytes(),
		}
	}
	return items
}

func Test_GetNodes(t *testing.T) {
	pool := NewPool(nodeSize)

	// set
	nodes := NewNodes(1, 0)
	pos, err := SetNodes(pool, nodes)
	assert.NoError(t, err)

	// get
	result, err := GetNodes(pool, pos)
	assert.NoError(t, err)
	assert.Equal(t, result[0].Depth, uint32(1))
	assert.Equal(t, result[0].Header, uint32(0))
	assert.Equal(t, result[15].Header, uint32(0))
	assert.Equal(t, result[15].Depth, uint32(1))

	// update
	nodes = NewNodes(2, 1)
	err = UpdateNodes(pool, pos, nodes)
	assert.NoError(t, err)

	result, err = GetNodes(pool, pos)
	assert.Equal(t, result[0].Depth, uint32(2))
	assert.Equal(t, result[0].Header, uint32(1))
	assert.Equal(t, result[15].Header, uint32(1))
	assert.Equal(t, result[15].Depth, uint32(2))
}

func Test_GetItems(t *testing.T) {
	pool := NewPool(itemHeaderSize + keySize)

	// set
	items := NewItems(1)
	pos, err := SetItems(pool, items)
	assert.NoError(t, err)

	// get
	result, err := GetItems(pool, pos)
	assert.NoError(t, err)
	assert.Equal(t, result[0].Pos, uint32(1))
	assert.Equal(t, result[15].Pos, uint32(1))
	assert.Equal(t, result[0].Num, uint32(0))
	assert.Equal(t, result[15].Num, uint32(15))

	//update
	items = NewItems(2)
	err = UpdateItems(pool, pos, items)
	assert.NoError(t, err)

	result, err = GetItems(pool, pos)
	assert.Equal(t, result[0].Pos, uint32(2))
	assert.Equal(t, result[15].Pos, uint32(2))
	assert.Equal(t, result[0].Num, uint32(0))
	assert.Equal(t, result[15].Num, uint32(15))
}
