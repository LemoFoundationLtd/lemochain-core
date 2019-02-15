package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateDB(t *testing.T) {
	db, _ := Open(DRIVER_MYSQL, DNS_MYSQL)
	_, err := CreateDB(db)
	assert.NoError(t, err)
}

// func TestSet(t *testing.T) {
// 	db, err := Open(DRIVER_MYSQL, DNS_MYSQL)
// 	assert.NoError(t, err)
//
// 	err = TxSet(db, "hash", "addr1", "addr2", []byte("paul"), 1, 100)
// 	err = TxSet(db, "hash1", "addr1", "addr2", []byte("paul"), 2, 101)
// 	err = TxSet(db, "hash2", "addr1", "addr2", []byte("paul"), 3, 102)
// 	err = TxSet(db, "hash3", "addr1", "addr2", []byte("paul"), 4, 103)
// 	err = TxSet(db, "hash4", "addr1", "addr2", []byte("paul"), 5, 104)
// 	assert.NoError(t, err)
//
// 	val, st, err := TxGet8Hash(db, "hash")
// 	assert.NoError(t, err)
// 	assert.Equal(t, []byte("paul"), val)
// 	assert.Equal(t, int64(100), st)
//
// 	// next
// 	results, sts, maxVer, err := TxGet8AddrNext(db, "addr1", 0, 2)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 2, len(results))
// 	assert.Equal(t, 2, len(sts))
// 	assert.Equal(t, int64(2), maxVer)
//
// 	results, sts, maxVer, err = TxGet8AddrNext(db, "addr1", 2, 2)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 2, len(results))
// 	assert.Equal(t, 2, len(sts))
// 	assert.Equal(t, int64(4), maxVer)
//
// 	results, sts, maxVer, err = TxGet8AddrNext(db, "addr1", 4, 2)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 1, len(results))
// 	assert.Equal(t, 1, len(sts))
// 	assert.Equal(t, int64(5), maxVer)
//
// 	results, sts, maxVer, err = TxGet8AddrNext(db, "addr1", 5, 2)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 0, len(results))
// 	assert.Equal(t, 0, len(sts))
// 	assert.Equal(t, int64(5), maxVer)
//
// 	// pre
// 	results, sts, maxVer, err = TxGet8AddrPre(db, "addr1", 4, 2)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 2, len(results))
// 	assert.Equal(t, 2, len(sts))
// 	assert.Equal(t, int64(2), maxVer)
//
// 	results, sts, maxVer, err = TxGet8AddrPre(db, "addr1", 2, 2)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 1, len(results))
// 	assert.Equal(t, 1, len(sts))
// 	assert.Equal(t, int64(1), maxVer)
//
// 	results, sts, maxVer, err = TxGet8AddrPre(db, "addr1", 1, 2)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 0, len(results))
// 	assert.Equal(t, 0, len(sts))
// 	assert.Equal(t, int64(1), maxVer)
// }

//
// func TestDoUtils_Set(t *testing.T) {
// 	dns := "root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4"
// 	db, err := Open(DRIVER_MYSQL, dns)
// 	defer db.Close()
// 	assert.NoError(t, err)
//
// 	err = clear(db)
// 	assert.NoError(t, err)
//
// 	key := "tx"
// 	val := NewByteWithData(10, "LEMOCHAIN")
// 	err = Set(db, key, val)
// 	assert.NoError(t, err)
//
// 	result, err := Get(db, key)
// 	assert.NoError(t, err)
// 	assert.Equal(t, val, result)
//
// 	err = Del(db, key)
// 	assert.NoError(t, err)
//
// 	key1, _ := NewKey1()
// 	keys, err := CreateBufWithNumberBatch(15000, key1)
// 	assert.NoError(t, err)
//
// 	vals := make(map[string][]byte, 15000)
// 	for index := 0; index < 15000; index++{
// 		vals[strconv.Itoa(index)] = keys[index]
// 	}
// 	err = SetBatch(db, vals)
// 	assert.NoError(t, err)
//
// 	err = UpdateBatch(db, vals)
// 	assert.NoError(t, err)
// }
//
// func TestDoUtils_Update(t *testing.T) {
// 	dns := "root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4"
// 	db, err := Open(DRIVER_MYSQL, dns)
// 	defer db.Close()
// 	assert.NoError(t, err)
//
// 	// err = clear(db)
// 	// assert.NoError(t, err)
//
// 	// key := "tx"
// 	// val := NewByteWithData(10, "LEMOCHAIN")
// 	// err = Set(db, key, val)
// 	// assert.NoError(t, err)
// 	//
// 	// result, err := Get(db, key)
// 	// assert.NoError(t, err)
// 	// assert.Equal(t, val, result)
// 	//
// 	// err = Del(db, key)
// 	// assert.NoError(t, err)
//
// 	key1, _ := NewKey1()
// 	keys, err := CreateBufWithNumberBatch(15000, key1)
// 	assert.NoError(t, err)
//
// 	vals := make(map[string][]byte, 15000)
// 	for index := 0; index < 15000; index++{
// 		vals[strconv.Itoa(index)] = keys[index]
// 	}
// 	// err = SetBatch(db, vals)
// 	// assert.NoError(t, err)
//
// 	err = UpdateBatch(db, vals)
// 	assert.NoError(t, err)
// }
