package store

import (
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-core/store/leveldb"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBeansDB_Put(t *testing.T) {
	ClearData()
	levelDB := leveldb.NewLevelDBDatabase(GetStorePath(), 16, 16)
	defer levelDB.Close()

	beansdb := NewBeansDB(GetStorePath(), levelDB)
	defer beansdb.Close()
	beansdb.Start()

	assert.NotNil(t, beansdb)

	key, err := CreateBufWithNumber(32)
	assert.NoError(t, err)

	val, err := CreateBufWithNumber(512)
	assert.NoError(t, err)

	err = beansdb.Put(leveldb.ItemFlagBlock, key, val)
	assert.NoError(t, err)

	result, err := beansdb.Get(leveldb.ItemFlagBlock, key)
	assert.NoError(t, err)
	assert.Equal(t, val, result)

	// time.Sleep(500 * time.Millisecond)
}

func TestBeansDB_Commit(t *testing.T) {
	ClearData()
	levelDB := leveldb.NewLevelDBDatabase(GetStorePath(), 16, 16)
	defer levelDB.Close()

	beansdb := NewBeansDB(GetStorePath(), levelDB)
	defer beansdb.Close()
	beansdb.Start()

	assert.NotNil(t, beansdb)

	batch := beansdb.NewBatch()
	block0 := GetBlock0()
	block1 := GetBlock1()
	block2 := GetBlock2()
	block3 := GetBlock3()
	buf0, _ := rlp.EncodeToBytes(&block0)
	batch.Put(leveldb.ItemFlagBlock, block0.Hash().Bytes(), buf0)

	buf1, _ := rlp.EncodeToBytes(&block1)
	batch.Put(leveldb.ItemFlagBlock, block1.Hash().Bytes(), buf1)

	buf2, _ := rlp.EncodeToBytes(&block2)
	batch.Put(leveldb.ItemFlagBlock, block2.Hash().Bytes(), buf2)

	buf3, _ := rlp.EncodeToBytes(&block3)
	batch.Put(leveldb.ItemFlagBlock, block3.Hash().Bytes(), buf3)

	err := batch.Commit()
	assert.NoError(t, err)

	result, err := beansdb.Get(leveldb.ItemFlagBlock, block0.Hash().Bytes())
	assert.NoError(t, err)
	assert.Equal(t, buf0, result)

	result, err = beansdb.Get(leveldb.ItemFlagBlock, block1.Hash().Bytes())
	assert.NoError(t, err)
	assert.Equal(t, buf1, result)

	result, err = beansdb.Get(leveldb.ItemFlagBlock, block2.Hash().Bytes())
	assert.NoError(t, err)
	assert.Equal(t, buf2, result)

	result, err = beansdb.Get(leveldb.ItemFlagBlock, block3.Hash().Bytes())
	assert.NoError(t, err)
	assert.Equal(t, buf3, result)

	batch.Commit()
}
