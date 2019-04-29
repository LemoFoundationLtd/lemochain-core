package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestTxRecently_RecvTx(t *testing.T) {
	var recently TxRecently
	tx1 := makeTx(testPrivate, common.HexToAddress("0x01"), params.OrdinaryTx, new(big.Int).SetInt64(100))
	tx2 := makeTx(testPrivate, common.HexToAddress("0x02"), params.OrdinaryTx, new(big.Int).SetInt64(200))
	tx3 := makeTx(testPrivate, common.HexToAddress("0x03"), params.OrdinaryTx, new(big.Int).SetInt64(300))
	recently.RecvTx(tx1)
	recently.RecvTx(tx2)
	recently.RecvTx(tx3)
	assert.Equal(t, 3, len(recently.TxsByHash))

	isExist := recently.IsExist(tx1.Hash())
	assert.Equal(t, true, isExist)

	isExist = recently.IsExist(common.HexToHash("0xABC"))
	assert.Equal(t, false, isExist)

	recently.RecvTx(tx1)
	assert.Equal(t, 3, len(recently.TxsByHash))

	val := recently.TxsByHash[tx1.Hash()]
	assert.Equal(t, int64(-1), val)

	recently.add(5, tx1)
	val = recently.TxsByHash[tx1.Hash()]
	assert.Equal(t, int64(5), val)

	recently.add(3, tx1)
	val = recently.TxsByHash[tx1.Hash()]
	assert.Equal(t, int64(3), val)

	recently.add(6, tx1)
	val = recently.TxsByHash[tx1.Hash()]
	assert.Equal(t, int64(3), val)
}

func TestTxRecently_RecvBlock(t *testing.T) {
	var recently TxRecently
	txs := make([]*types.Transaction, 0)
	txs = append(txs, makeTx(testPrivate, common.HexToAddress("0x01"), params.OrdinaryTx, new(big.Int).SetInt64(100)))
	txs = append(txs, makeTx(testPrivate, common.HexToAddress("0x02"), params.OrdinaryTx, new(big.Int).SetInt64(200)))
	txs = append(txs, makeTx(testPrivate, common.HexToAddress("0x03"), params.OrdinaryTx, new(big.Int).SetInt64(300)))
	recently.RecvBlock(10, txs)
	assert.Equal(t, 3, len(recently.TxsByHash))

	txs = append(txs, txs[0])
	recently.RecvBlock(10, txs)
	assert.Equal(t, 3, len(recently.TxsByHash))

	val := recently.TxsByHash[txs[0].Hash()]
	assert.Equal(t, int64(10), val)
}

func TestTxRecently_DelBatch(t *testing.T) {
	var recently TxRecently
	txs := make([]*types.Transaction, 0)
	txs = append(txs, makeTx(testPrivate, common.HexToAddress("0x01"), params.OrdinaryTx, new(big.Int).SetInt64(100)))
	txs = append(txs, makeTx(testPrivate, common.HexToAddress("0x02"), params.OrdinaryTx, new(big.Int).SetInt64(200)))
	txs = append(txs, makeTx(testPrivate, common.HexToAddress("0x03"), params.OrdinaryTx, new(big.Int).SetInt64(300)))
	recently.RecvBlock(10, txs)

	del := make([]common.Hash, 0, 2)
	del = append(del, txs[0].Hash())
	del = append(del, txs[1].Hash())
	recently.DelBatch(del)
	assert.Equal(t, 1, len(recently.TxsByHash))

	isExist := recently.IsExist(txs[0].Hash())
	assert.Equal(t, false, isExist)

	isExist = recently.IsExist(txs[1].Hash())
	assert.Equal(t, false, isExist)

	isExist = recently.IsExist(txs[2].Hash())
	assert.Equal(t, true, isExist)
}

func TestTxRecently_GetPath(t *testing.T) {
	var recently TxRecently
	tx1 := makeTx(testPrivate, common.HexToAddress("0x01"), params.OrdinaryTx, new(big.Int).SetInt64(100))
	recently.add(11, tx1)

	tx2 := makeTx(testPrivate, common.HexToAddress("0x02"), params.OrdinaryTx, new(big.Int).SetInt64(200))
	recently.add(12, tx2)

	tx3 := makeTx(testPrivate, common.HexToAddress("0x03"), params.OrdinaryTx, new(big.Int).SetInt64(300))
	recently.add(13, tx3)

	tx4 := makeTx(testPrivate, common.HexToAddress("0x04"), params.OrdinaryTx, new(big.Int).SetInt64(400))
	recently.add(14, tx4)

	txs := make([]*types.Transaction, 0)
	txs = append(txs, tx1)
	txs = append(txs, tx2)
	txs = append(txs, tx3)

	tx5 := makeTx(testPrivate, common.HexToAddress("0x05"), params.OrdinaryTx, new(big.Int).SetInt64(500))
	txs = append(txs, tx5)

	minHeight, maxHeight, result := recently.GetPath(txs)
	assert.Equal(t, int64(11), minHeight)
	assert.Equal(t, int64(13), maxHeight)
	assert.Equal(t, 3, len(result))
}
