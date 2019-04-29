package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

func TestTxSliceByTime_AddBatch(t *testing.T) {
	var slice TxSliceByTime
	txs := make([]*types.Transaction, 0)
	txs = append(txs, makeTx(testPrivate, common.HexToAddress("0x01"), params.OrdinaryTx, new(big.Int).SetInt64(100)))
	txs = append(txs, makeTx(testPrivate, common.HexToAddress("0x02"), params.OrdinaryTx, new(big.Int).SetInt64(200)))
	slice.AddBatch(txs)

	assert.Equal(t, 2, len(slice.TxsIndexByHash))

	if txs[0].Expiration() == txs[1].Expiration() {
		slot := txs[0].Expiration() % uint64(2*TransactionExpiration)
		item := slice.TxsIndexByTime[slot]
		assert.Equal(t, txs[0].Expiration(), item.Expiration)
		assert.Equal(t, 2, len(item.Txs))
	} else {
		slot0 := txs[0].Expiration() % uint64(2*TransactionExpiration)
		slot1 := txs[1].Expiration() % uint64(2*TransactionExpiration)

		item0 := slice.TxsIndexByTime[slot0]
		assert.Equal(t, txs[0].Expiration(), item0.Expiration)
		assert.Equal(t, 1, len(item0.Txs))

		item1 := slice.TxsIndexByTime[slot1]
		assert.Equal(t, txs[1].Expiration(), item1.Expiration)
		assert.Equal(t, 1, len(item1.Txs))
	}
}

func TestTxSliceByTime_AddBatchForReturn(t *testing.T) {
	amount := new(big.Int).SetInt64(100)
	gasPrice := new(big.Int).SetInt64(100)

	curTime := uint64(time.Now().Unix())
	expiration1 := curTime + 1000

	slot := expiration1 % uint64(2*TransactionExpiration)

	var slice TxSliceByTime
	txs1 := make([]*types.Transaction, 0)
	tx1 := makeTransaction(testPrivate, common.HexToAddress("0x01"), params.OrdinaryTx, amount, gasPrice, expiration1, 1000000)
	txs1 = append(txs1, tx1)

	tx2 := makeTransaction(testPrivate, common.HexToAddress("0x02"), params.OrdinaryTx, amount, gasPrice, expiration1, 1000000)
	txs1 = append(txs1, tx2)
	slice.AddBatch(txs1)
	assert.Equal(t, 2, len(slice.TxsIndexByTime[slot].Txs))

	expiration2 := expiration1 + uint64(2*TransactionExpiration)
	txs2 := make([]*types.Transaction, 0)
	tx3 := makeTransaction(testPrivate, common.HexToAddress("0x03"), params.OrdinaryTx, amount, gasPrice, expiration2, 1000000)
	txs2 = append(txs2, tx3)

	result := slice.AddBatch(txs2)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, 1, len(slice.TxsIndexByTime[slot].Txs))

	txs3 := make([]*types.Transaction, 0)
	tx4 := makeTransaction(testPrivate, common.HexToAddress("0x04"), params.OrdinaryTx, amount, gasPrice, expiration2, 1000000)
	txs3 = append(txs3, tx4)
	result = slice.AddBatch(txs3)
	assert.Equal(t, 0, len(result))
	assert.Equal(t, 2, len(slice.TxsIndexByTime[slot].Txs))

	result = slice.AddBatch(txs1)
	assert.Equal(t, 0, len(result))
	assert.Equal(t, 2, len(slice.TxsIndexByTime[slot].Txs))
}

func TestTxSliceByTime_GetBatch(t *testing.T) {
	var slice TxSliceByTime
	txs := make([]*types.Transaction, 0)
	txs = append(txs, makeTx(testPrivate, common.HexToAddress("0x01"), params.OrdinaryTx, new(big.Int).SetInt64(100)))
	txs = append(txs, makeTx(testPrivate, common.HexToAddress("0x02"), params.OrdinaryTx, new(big.Int).SetInt64(200)))
	txs = append(txs, makeTx(testPrivate, common.HexToAddress("0x03"), params.OrdinaryTx, new(big.Int).SetInt64(300)))
	txs = append(txs, makeTx(testPrivate, common.HexToAddress("0x04"), params.OrdinaryTx, new(big.Int).SetInt64(400)))
	txs = append(txs, makeTx(testPrivate, common.HexToAddress("0x05"), params.OrdinaryTx, new(big.Int).SetInt64(500)))
	slice.AddBatch(txs)

	result1 := slice.GetBatch(3)
	assert.Equal(t, 3, len(result1))
	assert.Equal(t, txs[0].Hash(), result1[0].Hash())
	assert.Equal(t, txs[2].Hash(), result1[2].Hash())

	result2 := slice.GetBatch(3)
	assert.Equal(t, 3, len(result2))
	assert.Equal(t, result1, result2)
	assert.Equal(t, txs[0].Hash(), result1[0].Hash())
	assert.Equal(t, txs[2].Hash(), result1[2].Hash())

	assert.Equal(t, 5, len(slice.TxsIndexByHash))
}

func TestTxSliceByTime_DelBatchByTx(t *testing.T) {
	amount := new(big.Int).SetInt64(100)
	gasPrice := new(big.Int).SetInt64(100)

	curTime := uint64(time.Now().Unix())
	expiration := curTime + 1000

	var slice TxSliceByTime
	txs := make([]*types.Transaction, 0)
	txs = append(txs, makeTransaction(testPrivate, common.HexToAddress("0x01"), params.OrdinaryTx, amount, gasPrice, expiration, 1000000))
	txs = append(txs, makeTransaction(testPrivate, common.HexToAddress("0x02"), params.OrdinaryTx, amount, gasPrice, expiration, 1000000))
	txs = append(txs, makeTransaction(testPrivate, common.HexToAddress("0x03"), params.OrdinaryTx, amount, gasPrice, expiration, 1000000))
	txs = append(txs, makeTransaction(testPrivate, common.HexToAddress("0x04"), params.OrdinaryTx, amount, gasPrice, expiration, 1000000))
	txs = append(txs, makeTransaction(testPrivate, common.HexToAddress("0x05"), params.OrdinaryTx, amount, gasPrice, expiration, 1000000))
	slice.AddBatch(txs)

	isExist := slice.IsExist(txs[0])
	assert.Equal(t, true, isExist)
	isExist = slice.IsExist(txs[1])
	assert.Equal(t, true, isExist)

	slot := expiration % uint64(2*TransactionExpiration)
	assert.Equal(t, 5, len(slice.TxsIndexByTime[slot].Txs))

	del := make([]*types.Transaction, 2)
	del[0] = txs[0]
	del[1] = txs[1]
	slice.DelBatchByTx(del)

	isExist = slice.IsExist(txs[0])
	assert.Equal(t, false, isExist)
	isExist = slice.IsExist(txs[1])
	assert.Equal(t, false, isExist)

	assert.Equal(t, 3, len(slice.TxsIndexByTime[slot].Txs))
}
