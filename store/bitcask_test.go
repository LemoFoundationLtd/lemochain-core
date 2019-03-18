package store

import (
	"github.com/LemoFoundationLtd/lemochain-core/store/leveldb"
	"github.com/stretchr/testify/assert"
	"testing"
)

var dns = "root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4"

func TestBitCask_Put(t *testing.T) {
	ClearData()
	levelDB := leveldb.NewLevelDBDatabase(GetStorePath(), 16, 16)
	defer levelDB.Close()
	bitcask, err := NewBitCask(GetStorePath(), nil, levelDB)

	assert.NoError(t, err)
	assert.NotNil(t, bitcask)

	key, err := CreateBufWithNumber(32)
	assert.NoError(t, err)

	val, err := CreateBufWithNumber(512)
	assert.NoError(t, err)

	err = bitcask.Put(CACHE_FLG_BLOCK, key, key, val)
	assert.NoError(t, err)

	result, err := bitcask.Get4Cache(key, key)
	assert.NoError(t, err)
	assert.Equal(t, val, result)
}

func TestBitCask_Commit(t *testing.T) {
	ClearData()
	levelDB := leveldb.NewLevelDBDatabase(GetStorePath(), 16, 16)
	defer levelDB.Close()
	bitcask, err := NewBitCask(GetStorePath(), nil, levelDB)

	assert.NoError(t, err)
	assert.NotNil(t, bitcask)

	route, err := CreateBufWithNumber(32)
	assert.NoError(t, err)
	batch := bitcask.NewBatch(route)

	item1 := new(BatchItem)
	item1.Key, err = CreateBufWithNumber(33)
	item1.Val, err = CreateBufWithNumber(148)
	batch.Put(CACHE_FLG_BLOCK, item1.Key, item1.Val)

	item2 := new(BatchItem)
	item2.Key, err = CreateBufWithNumber(34)
	item2.Val, err = CreateBufWithNumber(138)
	batch.Put(CACHE_FLG_BLOCK, item2.Key, item2.Val)

	item3 := new(BatchItem)
	item3.Key, err = CreateBufWithNumber(35)
	item3.Val, err = CreateBufWithNumber(192) // 192
	batch.Put(CACHE_FLG_BLOCK, item3.Key, item3.Val)

	item4 := new(BatchItem)
	item4.Key, err = CreateBufWithNumber(36)
	item4.Val, err = CreateBufWithNumber(1028) // 192
	batch.Put(CACHE_FLG_BLOCK, item4.Key, item4.Val)

	err = batch.Commit()
	assert.NoError(t, err)

	result, err := bitcask.Get4Cache(route, item1.Key)
	assert.NoError(t, err)
	assert.Equal(t, item1.Val, result)

	result, err = bitcask.Get4Cache(route, item2.Key)
	assert.NoError(t, err)
	assert.Equal(t, item2.Val, result)

	result, err = bitcask.Get4Cache(route, item3.Key)
	assert.NoError(t, err)
	assert.Equal(t, item3.Val, result)

	result, err = bitcask.Get4Cache(route, item4.Key)
	assert.NoError(t, err)
	assert.Equal(t, item4.Val, result)
}
