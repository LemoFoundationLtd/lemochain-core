package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMySqlDB_SetIndex(t *testing.T) {
	dns := "root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4"
	db := NewMySqlDB(DRIVER_MYSQL, dns)
	defer db.Close()

	err := db.SetIndex(int(CACHE_FLG_BLOCK), []byte("lm_val"), []byte("lm_key"), 100)
	assert.NoError(t, err)

	flg, val, pos, err := db.GetIndex([]byte("lm_key"))
	assert.NoError(t, err)
	assert.Equal(t, flg, int(CACHE_FLG_BLOCK))
	assert.Equal(t, val, []byte("lm_val"))
	assert.Equal(t, pos, int64(100))
}
