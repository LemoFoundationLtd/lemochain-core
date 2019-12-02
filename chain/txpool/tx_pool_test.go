package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTxPool_PushTx1(t *testing.T) {
	curTime := time.Now().Unix()
	pool := NewTxPool()

	tx1 := makeTxRandom(common.HexToAddress("0x01"))
	tx2 := makeTxRandom(common.HexToAddress("0x02"))
	tx3 := makeTxRandom(common.HexToAddress("0x03"))
	pool.PushTx(tx1)
	pool.PushTx(tx2)
	pool.PushTx(tx3)

	result := pool.Get(uint32(curTime), 10)
	assert.Equal(t, 3, len(result))

	result = pool.Get(uint32(curTime), 10)
	assert.Equal(t, 3, len(result))
}

func TestTxPool_PushTx2(t *testing.T) {
	curTime := time.Now().Unix()
	pool := NewTxPool()

	txs := make([]*types.Transaction, 0)
	tx1 := makeTxRandom(common.HexToAddress("0x01"))
	pool.PushTx(tx1)
	result := pool.Get(uint32(curTime), 10)
	assert.Equal(t, 1, len(result))

	txs = append(txs, tx1)
	pool.DelInvalidTxs(txs)
	pool.PushTx(tx1)

	result = pool.Get(uint32(curTime), 10)
	assert.Equal(t, 0, len(result))
}

func TestTxPool_ExistPendingTx(t *testing.T) {
	curTime := time.Now().Unix()
	pool := NewTxPool()

	tx1 := makeTxRandom(common.HexToAddress("0x01"))
	tx2 := makeTxRandom(common.HexToAddress("0x02"))
	tx3 := makeTxRandom(common.HexToAddress("0x03"))
	pool.PushTx(tx1)
	pool.PushTx(tx2)
	pool.PushTx(tx3)

	isExist := pool.ExistPendingTx(uint32(curTime))
	assert.Equal(t, true, isExist)

	isExist = pool.ExistPendingTx(uint32(curTime) + 30*60*60 + 1)
	assert.Equal(t, false, isExist)
}

func TestTxPool_DelInvalidTxs(t *testing.T) {
	curTime := time.Now().Unix()
	pool := NewTxPool()

	tx1 := makeTxRandom(common.HexToAddress("0x01"))
	tx2 := makeTxRandom(common.HexToAddress("0x02"))
	tx3 := makeTxRandom(common.HexToAddress("0x03"))
	tx4 := makeTxRandom(common.HexToAddress("0x04"))
	pool.PushTx(tx1)
	pool.PushTx(tx2)
	pool.PushTx(tx3)
	pool.PushTx(tx4)

	delTxs := make([]*types.Transaction, 0, 3)
	delTxs = append(delTxs, tx1)
	delTxs = append(delTxs, tx2)
	delTxs = append(delTxs, tx3)
	pool.DelInvalidTxs(delTxs)

	result := pool.Get(uint32(curTime), 10)
	assert.Equal(t, 1, len(result))
}

func TestTxPool_RecvBlock(t *testing.T) {
	curTime := time.Now().Unix()

	pool := NewTxPool()
	tx1 := makeTxRandom(common.HexToAddress("0x01"))
	tx2 := makeTxRandom(common.HexToAddress("0x02"))
	tx3 := makeTxRandom(common.HexToAddress("0x03"))
	tx4 := makeTxRandom(common.HexToAddress("0x04"))
	tx5 := makeTxRandom(common.HexToAddress("0x05"))
	tx6 := makeTxRandom(common.HexToAddress("0x06"))
	tx7 := makeTxRandom(common.HexToAddress("0x07"))
	tx8 := makeTxRandom(common.HexToAddress("0x08"))
	tx9 := makeTxRandom(common.HexToAddress("0x09"))
	pool.PushTx(tx1)
	pool.PushTx(tx2)
	pool.PushTx(tx3)
	pool.PushTx(tx4)
	pool.PushTx(tx5)
	pool.PushTx(tx6)
	pool.PushTx(tx7)
	pool.PushTx(tx8)
	pool.PushTx(tx9)

	block1 := store.GetBlock1()
	block1.Header.Time = uint32(curTime)
	block1.Txs = append(block1.Txs, tx1)
	block1.Txs = append(block1.Txs, tx2)
	block1.Txs = append(block1.Txs, tx3)
	block1.Txs = append(block1.Txs, tx4)
	block1.Txs = append(block1.Txs, tx5)
	block1.Txs = append(block1.Txs, tx6)
	block1.Txs = append(block1.Txs, tx7)
	pool.RecvBlock(block1)

	result := pool.Get(uint32(curTime), 10)
	assert.Equal(t, 2, len(result))
}

func TestTxPool_Box(t *testing.T) {
	log.Setup(log.LevelDebug, false, false)

	curTime := time.Now().Unix()
	tx := createBoxTxRandom(common.HexToAddress("0xabcde"), 5, uint64(curTime))

	pool := NewTxPool()
	pool.PushTx(tx)

	result := pool.Get(uint32(curTime), 10)
	assert.Equal(t, 1, len(result))

	pool.PushTx(tx)
	result = pool.Get(uint32(curTime), 10)
	assert.Equal(t, 1, len(result))
}

func TestTxPool_SetTxsFlag1(t *testing.T) {
	curTime := time.Now().Unix()
	pool := NewTxPool()

	txs := make([]*types.Transaction, 0)
	txs = append(txs, makeTxRandom(common.HexToAddress("0x01")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x02")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x03")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x04")))

	pool.SetTxsFlag(txs, false)
	isExist := pool.ExistPendingTx(uint32(curTime))
	assert.Equal(t, false, isExist)
}

func TestTxPool_SetTxsFlag2(t *testing.T) {
	curTime := time.Now().Unix()
	pool := NewTxPool()

	txs := make([]*types.Transaction, 0)
	txs = append(txs, makeTxRandom(common.HexToAddress("0x01")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x02")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x03")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x04")))

	pool.SetTxsFlag(txs, true)
	isExist := pool.ExistPendingTx(uint32(curTime))
	assert.Equal(t, true, isExist)
}
