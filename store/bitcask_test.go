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
	bitcask, err := NewBitCask(GetStorePath(), 0, levelDB, make(chan struct{}))

	Done := make(chan *Inject)
	Err := make(chan *Inject)
	go bitcask.Start(Done, Err)

	assert.NoError(t, err)
	assert.NotNil(t, bitcask)

	key, err := CreateBufWithNumber(32)
	assert.NoError(t, err)

	val, err := CreateBufWithNumber(512)
	assert.NoError(t, err)

	bitcask.Put(leveldb.ItemFlagBlock, key, val)
	select {
	case op := <-Done:
		log.Errorf("done: %d", op.Flg)
		break
	case op := <-Err:
		log.Error("Err: %d", op.Flg)
		break
	}

	result, err := bitcask.Get(leveldb.ItemFlagBlock, key)
	assert.NoError(t, err)
	assert.Equal(t, val, result)
}
