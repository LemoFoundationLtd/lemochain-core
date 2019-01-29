package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMySqlDB_SetIndex(t *testing.T) {
	db := NewMySqlDB(DRIVER_MYSQL, DNS_MYSQL)
	defer db.Close()

	err := db.SetIndex(int(CACHE_FLG_BLOCK), []byte("lm_val"), []byte("lm_key"), 100)
	assert.NoError(t, err)

	flg, val, pos, err := db.GetIndex([]byte("lm_key"))
	assert.NoError(t, err)
	assert.Equal(t, flg, int(CACHE_FLG_BLOCK))
	assert.Equal(t, val, []byte("lm_val"))
	assert.Equal(t, pos, int64(100))
}

func TestMySqlDB_Tx(t *testing.T) {
	ClearData()

	db := NewMySqlDB(DRIVER_MYSQL, DNS_MYSQL)
	defer db.Close()

	err := db.TxSet("hash", "block hash", "addr1", "addr2", []byte("paul"), 1, 100)
	err = db.TxSet("hash1", "block hash", "addr1", "addr2", []byte("paul"), 2, 101)
	err = db.TxSet("hash2", "block hash", "addr1", "addr2", []byte("paul"), 3, 102)
	err = db.TxSet("hash3", "block hash", "addr1", "addr2", []byte("paul"), 4, 103)
	err = db.TxSet("hash4", "block hash", "addr1", "addr2", []byte("paul"), 5, 104)
	assert.NoError(t, err)

	hash, val, st, err := db.TxGetByHash("hash")
	assert.NoError(t, err)
	assert.Equal(t, []byte("paul"), val)
	assert.Equal(t, "block hash", hash)
	assert.Equal(t, int64(100), st)

	// next
	hashes, results, sts, maxVer, err := db.TxGetByAddr("addr1", 0, 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(hashes))
	assert.Equal(t, 2, len(results))
	assert.Equal(t, 2, len(sts))
	assert.Equal(t, int64(2), maxVer)

	hashes, results, sts, maxVer, err = db.TxGetByAddr("addr1", 2, 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(hashes))
	assert.Equal(t, 2, len(results))
	assert.Equal(t, 2, len(sts))
	assert.Equal(t, int64(4), maxVer)

	hashes, results, sts, maxVer, err = db.TxGetByAddr("addr1", 4, 2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(hashes))
	assert.Equal(t, 1, len(results))
	assert.Equal(t, 1, len(sts))
	assert.Equal(t, int64(5), maxVer)

	hashes, results, sts, maxVer, err = db.TxGetByAddr("addr1", 5, 2)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(hashes))
	assert.Equal(t, 0, len(results))
	assert.Equal(t, 0, len(sts))
	assert.Equal(t, int64(5), maxVer)
}
