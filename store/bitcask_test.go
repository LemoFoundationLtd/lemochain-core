package store

import (
	"github.com/LemoFoundationLtd/lemochain-go/store/leveldb"
	"github.com/stretchr/testify/assert"
	"testing"
)

var dns = "root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4"

func TestBitCask_Put(t *testing.T) {
	ClearData()

	bitcask, err := NewBitCask(GetStorePath(), 0, 0, nil, leveldb.NewLevelDBDatabase(GetStorePath(), 16, 16))
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

	// bitcask, err := NewBitCask(GetStorePath(), 0, 0, nil, NewMySqlDB(DRIVER_MYSQL, dns))
	bitcask, err := NewBitCask(GetStorePath(), 0, 0, nil, leveldb.NewLevelDBDatabase(GetStorePath(), 16, 16))

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

//
// func TestLmDataBase_Batch_Put1(t *testing.T) {
// 	ClearData()
//
// 	db, err := NewLmDataBase(GetStorePath())
// 	assert.NoError(t, err)
//
// 	totalCnt := 16
// 	key1, _ := NewKey1()
//
// 	keys, err := CreateBufWithNumberBatch(totalCnt, key1)
// 	assert.NoError(t, err)
//
// 	val, err := CreateBufWithNumber(512)
// 	assert.NoError(t, err)
//
// 	for index := 0; index < totalCnt; index++ {
// 		err = db.Put(keys[index], val)
// 		assert.NoError(t, err)
// 		result, err := db.Get(keys[index])
// 		assert.NoError(t, err)
// 		assert.Equal(t, val, result)
// 	}
//
// 	db, err = NewLmDataBase(GetStorePath())
// 	assert.NoError(t, err)
//
// 	for index := 0; index < totalCnt; index++ {
// 		result, err := db.Get(keys[index])
// 		if result == nil{
// 			log.Error("INDEX:")
// 		}
// 		assert.NoError(t, err)
// 		assert.Equal(t, val, result)
// 	}
// }
//
// func TestLmDataBase_Batch_Put2(t *testing.T) {
// 	ClearData()
//
// 	db, err := NewLmDataBase(GetStorePath())
// 	assert.NoError(t, err)
//
// 	totalCnt := 32
// 	key1, _ := NewKey1()
//
// 	keys, err := CreateBufWithNumberBatch(totalCnt, key1)
// 	assert.NoError(t, err)
//
// 	val, err := CreateBufWithNumber(78940000)
// 	assert.NoError(t, err)
//
// 	for index := 0; index < totalCnt; index++ {
// 		err = db.Put(keys[index], val)
// 		assert.NoError(t, err)
// 		result, err := db.Get(keys[index])
// 		assert.NoError(t, err)
// 		assert.Equal(t, val, result)
// 	}
//
// 	db, err = NewLmDataBase(GetStorePath())
// 	assert.NoError(t, err)
//
// 	for index := 0; index < totalCnt; index++ {
// 		result, err := db.Get(keys[index])
// 		if result == nil{
// 			log.Error("INDEX:")
// 		}
// 		assert.NoError(t, err)
// 		assert.Equal(t, val, result)
// 	}
// }
//
// func TestLmDataBase_CurrentBlock(t *testing.T) {
// 	ClearData()
//
// 	db, err := NewLmDataBase(GetStorePath())
// 	assert.NoError(t, err)
//
// 	// step.1
// 	block, err := CreateBufWithNumber(36)
// 	assert.NoError(t, err)
//
// 	err = db.SetCurrentBlock(block)
// 	assert.NoError(t, err)
//
// 	result := db.CurrentBlock()
// 	assert.Equal(t, block, result)
//
// 	// step.2
// 	block, err = CreateBufWithNumber(481)
// 	assert.NoError(t, err)
//
// 	err = db.SetCurrentBlock(block)
// 	assert.NoError(t, err)
//
// 	result = db.CurrentBlock()
// 	assert.Equal(t, block, result)
//
// 	// step.3
// 	block, err = CreateBufWithNumber(128)
// 	assert.NoError(t, err)
//
// 	err = db.SetCurrentBlock(block)
// 	assert.NoError(t, err)
//
// 	result = db.CurrentBlock()
// 	assert.Equal(t, block, result)
//
// 	// step.4
// 	db, err = NewLmDataBase(GetStorePath())
// 	assert.NoError(t, err)
//
// 	block, err = CreateBufWithNumber(128)
// 	assert.NoError(t, err)
//
// 	result = db.CurrentBlock()
// 	assert.Equal(t, block, result)
// }
//
// func TestLmDataBase_Has(t *testing.T) {
// 	ClearData()
//
// 	db, err := NewLmDataBase(GetStorePath())
// 	assert.NoError(t, err)
//
// 	totalCnt := 16
// 	key1, _ := NewKey1()
//
// 	keys, err := CreateBufWithNumberBatch(totalCnt, key1)
// 	assert.NoError(t, err)
//
// 	val, err := CreateBufWithNumber(58)
// 	assert.NoError(t, err)
//
// 	for index := 0; index < totalCnt; index++ {
// 		err = db.Put(keys[index], val)
// 		assert.NoError(t, err)
// 	}
//
// 	db, err = NewLmDataBase(GetStorePath())
// 	assert.NoError(t, err)
//
// 	for index := 0; index < totalCnt; index++ {
// 		isExist, err := db.Has(keys[index])
// 		if !isExist {
// 			log.Errorf("IS EXIST:", isExist)
// 		}
// 		assert.NoError(t, err)
// 		assert.Equal(t, true, isExist)
// 	}
//
// 	key2, _ := NewKey2()
// 	isExist, err := db.Has(key2)
// 	assert.NoError(t, err)
// 	assert.Equal(t, isExist, false)
// }
//
// func TestLmDataBase_Commit1(t *testing.T) {
// 	ClearData()
//
// 	db, err := NewLmDataBase(GetStorePath())
// 	assert.NoError(t, err)
//
// 	totalCnt := 4096
// 	key1, _ := NewKey1()
//
// 	keys, err := CreateBufWithNumberBatch(totalCnt, key1)
// 	assert.NoError(t, err)
//
// 	val, err := CreateBufWithNumber(6553)
// 	assert.NoError(t, err)
//
// 	items := make([]*BatchItem, totalCnt)
// 	for index := 0; index < totalCnt; index++ {
// 		item := &BatchItem{
// 			Key: make([]byte, 32),
// 			Val: val,
// 		}
// 		copy(item.Key[0:32], keys[index][0:32])
// 		items[index] = item
// 	}
//
// 	err = db.Commit(items)
// 	assert.NoError(t, err)
//
// 	keys, err = CreateBufWithNumberBatch(totalCnt, key1)
// 	for index := 0; index < totalCnt; index++ {
// 		result, err := db.Get(keys[index])
// 		assert.NoError(t, err)
// 		assert.Equal(t, val, result)
// 	}
//
// 	db, err = NewLmDataBase(GetStorePath())
// 	assert.NoError(t, err)
//
// 	keys, err = CreateBufWithNumberBatch(totalCnt, key1)
// 	for index := 0; index < totalCnt; index++ {
// 		result, err := db.Get(keys[index])
// 		assert.NoError(t, err)
// 		assert.Equal(t, val, result)
// 	}
// }
//
// func TestLmDataBase_Commit2(t *testing.T) {
// 	ClearData()
//
// 	db, err := NewLmDataBase(GetStorePath())
// 	assert.NoError(t, err)
//
// 	totalCnt := 2
// 	key1, _ := NewKey1()
//
// 	keys, err := CreateBufWithNumberBatch(totalCnt, key1)
// 	assert.NoError(t, err)
//
// 	val, err := CreateBufWithNumber(6553)
// 	assert.NoError(t, err)
//
// 	items := make([]*BatchItem, totalCnt)
// 	for index := 0; index < totalCnt; index++ {
// 		item := &BatchItem{
// 			Key: make([]byte, 32),
// 			Val: val,
// 		}
// 		copy(item.Key[0:32], keys[index][0:32])
// 		items[index] = item
// 	}
//
// 	err = db.Commit(items)
// 	assert.NoError(t, err)
//
// 	keys, err = CreateBufWithNumberBatch(totalCnt, key1)
// 	for index := 0; index < totalCnt; index++ {
// 		result, err := db.Get(keys[index])
// 		assert.NoError(t, err)
// 		assert.Equal(t, val, result)
// 	}
//
// 	db, err = NewLmDataBase(GetStorePath())
// 	assert.NoError(t, err)
//
// 	keys, err = CreateBufWithNumberBatch(totalCnt, key1)
// 	for index := 0; index < totalCnt; index++ {
// 		result, err := db.Get(keys[index])
// 		assert.NoError(t, err)
// 		assert.Equal(t, val, result)
// 	}
// }
