package store

import (
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store/leveldb"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBitCask_Put(t *testing.T) {
	ClearData()

	levelDB := leveldb.NewLevelDBDatabase(GetStorePath(), 16, 16)
	defer levelDB.Close()
	bitcask, err := NewBitCask(GetStorePath(), 0, levelDB)
	assert.NoError(t, err)
	assert.NotNil(t, bitcask)

	key, err := CreateBufWithNumber(32)
	assert.NoError(t, err)
	val, err := CreateBufWithNumber(512)
	assert.NoError(t, err)

	err = bitcask.Put(leveldb.ItemFlagBlock, key, val)
	assert.NoError(t, err)

	result, err := bitcask.Get(leveldb.ItemFlagBlock, key)
	assert.NoError(t, err)
	assert.Equal(t, val, result)
}

func TestBitCask_Put2G(t *testing.T) {
	ClearData()

	levelDB := leveldb.NewLevelDBDatabase(GetStorePath(), 16, 16)
	defer levelDB.Close()
	bitcask, err := NewBitCask(GetStorePath(), 0, levelDB)
	assert.NoError(t, err)
	assert.NotNil(t, bitcask)

	key, err := CreateBufWithNumber(32)
	assert.NoError(t, err)
	val, err := CreateBufWithNumber(1024 * 1024 * 100)
	assert.NoError(t, err)

	for index := 0; index < 32; index++ {
		log.Infof("process No.%d", index)
		err = bitcask.Put(leveldb.ItemFlagBlock, key, val)
		assert.NoError(t, err)

		result, err := bitcask.Get(leveldb.ItemFlagBlock, key)
		assert.NoError(t, err)
		assert.Equal(t, val, result)
	}
}
