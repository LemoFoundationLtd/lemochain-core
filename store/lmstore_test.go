package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLmDataBase_Put(t *testing.T) {
	ClearData()

	db, err := NewLmDataBase(GetStorePath())
	assert.NoError(t, err)

	key, err := CreateBufWithNumber(32)
	assert.NoError(t, err)

	val, err := CreateBufWithNumber(1024)
	assert.NoError(t, err)

	err = db.Put(key, val)
	assert.NoError(t, err)

	result, err := db.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, val, result)
}

func TestLmDataBase_Batch_Put1(t *testing.T) {
	ClearData()

	db, err := NewLmDataBase(GetStorePath())
	assert.NoError(t, err)

	totalCnt := 16
	key1, _ := NewKey1()

	keys, err := CreateBufWithNumberBatch(totalCnt, key1)
	assert.NoError(t, err)

	val, err := CreateBufWithNumber(65536)
	assert.NoError(t, err)

	for index := 0; index < totalCnt; index++ {
		err = db.Put(keys[index], val)
		assert.NoError(t, err)
		result, err := db.Get(keys[index])
		assert.NoError(t, err)
		assert.Equal(t, val, result)
	}

	db, err = NewLmDataBase(GetStorePath())
	assert.NoError(t, err)

	for index := 0; index < totalCnt; index++ {
		result, err := db.Get(keys[index])
		assert.NoError(t, err)
		assert.Equal(t, val, result)
	}
}

func TestLmDataBase_Batch_Put2(t *testing.T) {
	ClearData()

	db, err := NewLmDataBase(GetStorePath())
	assert.NoError(t, err)

	totalCnt := 32
	key1, _ := NewKey1()

	keys, err := CreateBufWithNumberBatch(totalCnt, key1)
	assert.NoError(t, err)

	val, err := CreateBufWithNumber(78940000)
	assert.NoError(t, err)

	for index := 0; index < totalCnt; index++ {
		err = db.Put(keys[index], val)
		assert.NoError(t, err)
		result, err := db.Get(keys[index])
		assert.NoError(t, err)
		assert.Equal(t, val, result)
	}

	db, err = NewLmDataBase(GetStorePath())
	assert.NoError(t, err)

	for index := 0; index < totalCnt; index++ {
		result, err := db.Get(keys[index])
		assert.NoError(t, err)
		assert.Equal(t, val, result)
	}
}

func TestLmDataBase_CurrentBlock(t *testing.T) {
	ClearData()

	db, err := NewLmDataBase(GetStorePath())
	assert.NoError(t, err)

	// step.1
	block, err := CreateBufWithNumber(36)
	assert.NoError(t, err)

	err = db.SetCurrentBlock(block)
	assert.NoError(t, err)

	result := db.CurrentBlock()
	assert.Equal(t, block, result)

	// step.2
	block, err = CreateBufWithNumber(481)
	assert.NoError(t, err)

	err = db.SetCurrentBlock(block)
	assert.NoError(t, err)

	result = db.CurrentBlock()
	assert.Equal(t, block, result)

	// step.3
	block, err = CreateBufWithNumber(128)
	assert.NoError(t, err)

	err = db.SetCurrentBlock(block)
	assert.NoError(t, err)

	result = db.CurrentBlock()
	assert.Equal(t, block, result)

	// step.4
	db, err = NewLmDataBase(GetStorePath())
	assert.NoError(t, err)

	block, err = CreateBufWithNumber(128)
	assert.NoError(t, err)

	result = db.CurrentBlock()
	assert.Equal(t, block, result)
}

func TestLmDataBase_Has(t *testing.T) {
	ClearData()

	db, err := NewLmDataBase(GetStorePath())
	assert.NoError(t, err)

	totalCnt := 16
	key1, _ := NewKey1()

	keys, err := CreateBufWithNumberBatch(totalCnt, key1)
	assert.NoError(t, err)

	val, err := CreateBufWithNumber(58)
	assert.NoError(t, err)

	for index := 0; index < totalCnt; index++ {
		err = db.Put(keys[index], val)
		assert.NoError(t, err)
	}

	db, err = NewLmDataBase(GetStorePath())
	assert.NoError(t, err)

	for index := 0; index < totalCnt; index++ {
		isExist, err := db.Has(keys[index])
		assert.NoError(t, err)
		assert.Equal(t, isExist, true)
	}

	key2, _ := NewKey2()
	isExist, err := db.Has(key2)
	assert.NoError(t, err)
	assert.Equal(t, isExist, false)
}
