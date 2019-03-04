package store

import (
	"github.com/LemoFoundationLtd/lemochain-go/store/leveldb"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBeansDB_Put(t *testing.T) {
	ClearData()
	levelDB := leveldb.NewLevelDBDatabase(GetStorePath(), 16, 16)
	defer levelDB.Close()

	beansdb := NewBeansDB(GetStorePath(), 2, levelDB, nil)

	assert.NotNil(t, beansdb)

	key, err := CreateBufWithNumber(32)
	assert.NoError(t, err)

	val, err := CreateBufWithNumber(512)
	assert.NoError(t, err)

	err = beansdb.Put(CACHE_FLG_BLOCK, key, key, val)
	assert.NoError(t, err)

	result, err := beansdb.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, val, result)
}

func TestBeansDB_Commit(t *testing.T) {
	ClearData()
	levelDB := leveldb.NewLevelDBDatabase(GetStorePath(), 16, 16)
	defer levelDB.Close()

	beansdb := NewBeansDB(GetStorePath(), 2, levelDB, nil)
	assert.NotNil(t, beansdb)

	route, err := CreateBufWithNumber(32)
	assert.NoError(t, err)
	batch := beansdb.NewBatch(route)

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

	result, err := beansdb.Get(item1.Key)
	assert.NoError(t, err)
	assert.Equal(t, item1.Val, result)

	result, err = beansdb.Get(item2.Key)
	assert.NoError(t, err)
	assert.Equal(t, item2.Val, result)

	result, err = beansdb.Get(item3.Key)
	assert.NoError(t, err)
	assert.Equal(t, item3.Val, result)

	result, err = beansdb.Get(item4.Key)
	assert.NoError(t, err)
	assert.Equal(t, item4.Val, result)
}
