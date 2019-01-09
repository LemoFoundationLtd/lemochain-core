package store

//
// import (
// 	"testing"
// 	"github.com/stretchr/testify/assert"
// 	"strconv"
// )
//
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
