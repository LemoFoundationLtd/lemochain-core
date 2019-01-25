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
	db := NewMySqlDB(DRIVER_MYSQL, DNS_MYSQL)
	defer db.Close()

	err := db.TxSet("hash", "addr1", "addr2", []byte("paul"), 1, 100)
	err = db.TxSet("hash1", "addr1", "addr2", []byte("paul"), 2, 101)
	err = db.TxSet("hash2", "addr1", "addr2", []byte("paul"), 3, 102)
	err = db.TxSet("hash3", "addr1", "addr2", []byte("paul"), 4, 103)
	err = db.TxSet("hash4", "addr1", "addr2", []byte("paul"), 5, 104)
	assert.NoError(t, err)

	val, st, err := db.TxGet8Hash("hash")
	assert.NoError(t, err)
	assert.Equal(t, []byte("paul"), val)
	assert.Equal(t, int64(100), st)

	// next
	results, sts, maxVer, err := db.TxGet8AddrNext("addr1", 0, 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
	assert.Equal(t, 2, len(sts))
	assert.Equal(t, int64(2), maxVer)

	results, sts, maxVer, err = db.TxGet8AddrNext("addr1", 2, 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
	assert.Equal(t, 2, len(sts))
	assert.Equal(t, int64(4), maxVer)

	results, sts, maxVer, err = db.TxGet8AddrNext("addr1", 4, 2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, 1, len(sts))
	assert.Equal(t, int64(5), maxVer)

	results, sts, maxVer, err = db.TxGet8AddrNext("addr1", 5, 2)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results))
	assert.Equal(t, 0, len(sts))
	assert.Equal(t, int64(5), maxVer)

	// pre
	results, sts, maxVer, err = db.TxGet8AddrPre("addr1", 4, 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
	assert.Equal(t, 2, len(sts))
	assert.Equal(t, int64(2), maxVer)

	results, sts, maxVer, err = db.TxGet8AddrPre("addr1", 2, 2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, 1, len(sts))
	assert.Equal(t, int64(1), maxVer)

	results, sts, maxVer, err = db.TxGet8AddrPre("addr1", 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results))
	assert.Equal(t, 0, len(sts))
	assert.Equal(t, int64(1), maxVer)
}
