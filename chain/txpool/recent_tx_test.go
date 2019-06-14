package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTxRecently_RecvTx1(t *testing.T) {
	curTime := time.Now().Unix()

	recently := NewTxRecently()
	recently.RecvTx(nil)
	assert.Equal(t, 0, len(recently.TraceMap))

	tx1 := makeTx(common.HexToAddress("0x01"), curTime)
	tx2 := makeTx(common.HexToAddress("0x02"), curTime)
	recently.RecvTx(tx1)
	recently.RecvTx(tx1)
	recently.RecvTx(tx2)

	assert.Equal(t, 2, len(recently.TraceMap))
	assert.Equal(t, 0, len(recently.TraceMap[tx1.Hash()]))
	assert.Equal(t, 0, len(recently.TraceMap[tx2.Hash()]))

	slot := tx1.Expiration() % uint64(params.TransactionExpiration)
	assert.Equal(t, 2, len(recently.TxsByTime[slot].TxIndexes))
}

func TestTxRecently_RecvTx2(t *testing.T) {
	recently := NewTxRecently()

	curTime := time.Now().Unix()
	tx := createBoxTxRandom(common.HexToAddress("0xabcde"), 5, uint64(curTime))

	recently.RecvTx(tx)
	assert.Equal(t, 6, len(recently.TraceMap))
	assert.Equal(t, 0, len(recently.TraceMap[tx.Hash()]))

	recently.RecvTx(tx)
	assert.Equal(t, 6, len(recently.TraceMap))
	assert.Equal(t, 0, len(recently.TraceMap[tx.Hash()]))
}

func TestRecentTx_IsExist(t *testing.T) {
	recently := NewTxRecently()

	// box tx
	curTime := time.Now().Unix()
	tx := createBoxTxRandom(common.HexToAddress("0xabcde"), 5, uint64(curTime)+100)

	isExist := recently.IsExist(tx)
	assert.Equal(t, false, isExist)

	recently.RecvTx(tx)
	isExist = recently.IsExist(tx)
	assert.Equal(t, true, isExist)

	// normal tx
	tx1 := makeTx(common.HexToAddress("0x01"), curTime+200)
	isExist = recently.IsExist(tx1)
	assert.Equal(t, false, isExist)

	recently.RecvTx(tx1)
	isExist = recently.IsExist(tx)
	assert.Equal(t, true, isExist)

	// box tx
	dtx1, dtx2 := createDoubleBoxTxRandom(common.HexToAddress("0xabcde"), 5, uint64(curTime)+300)
	isExist = recently.IsExist(dtx1)
	assert.Equal(t, false, isExist)

	recently.RecvTx(dtx1)
	isExist = recently.IsExist(dtx2)
	assert.Equal(t, true, isExist)
}

func TestTxRecently_RecvBlock(t *testing.T) {
	recently := NewTxRecently()
	bhash := common.HexToHash("0xabc")
	height := int64(100)
	txs := make([]*types.Transaction, 0)

	recently.RecvBlock(bhash, height, txs)
	assert.Equal(t, 0, len(recently.TraceMap))

	txs = append(txs, makeTxRandom(common.HexToAddress("0x01")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x02")))
	txs = append(txs, txs[0])
	recently.RecvBlock(bhash, height, txs)

	assert.Equal(t, 2, len(recently.TraceMap))
	assert.Equal(t, height, recently.TraceMap[txs[0].Hash()][bhash])
	assert.Equal(t, height, recently.TraceMap[txs[1].Hash()][bhash])
}

func TestTxRecently_RecvBlockTimeOut(t *testing.T) {
	recently := NewTxRecently()
	bhash := common.HexToHash("0xabc")
	height := int64(100)
	txs := make([]*types.Transaction, 0)
	expiration := time.Now().Unix()

	txs = append(txs, makeTx(common.HexToAddress("0x01"), expiration))
	txs = append(txs, makeTx(common.HexToAddress("0x02"), expiration))
	txs = append(txs, makeTx(common.HexToAddress("0x03"), expiration))
	txs = append(txs, txs[2])
	txs = append(txs, makeTx(common.HexToAddress("0x04"), expiration))
	recently.RecvBlock(bhash, height, txs)

	slot := expiration % int64(params.TransactionExpiration)
	assert.Equal(t, 4, len(recently.TxsByTime[slot].TxIndexes))
	assert.Equal(t, 4, len(recently.TraceMap))

	txs = make([]*types.Transaction, 0)
	txs = append(txs, makeTx(common.HexToAddress("0x05"), expiration+int64(params.TransactionExpiration)))
	recently.RecvBlock(bhash, height, txs)

	slot = expiration % int64(params.TransactionExpiration)
	assert.Equal(t, 1, len(recently.TxsByTime[slot].TxIndexes))
	assert.Equal(t, 1, len(recently.TraceMap))

	txs = make([]*types.Transaction, 0)
	txs = append(txs, makeTx(common.HexToAddress("0x06"), expiration))
	recently.RecvBlock(bhash, height, txs)

	slot = expiration % int64(params.TransactionExpiration)
	assert.Equal(t, 1, len(recently.TxsByTime[slot].TxIndexes))
	assert.Equal(t, 1, len(recently.TraceMap))

	txs = make([]*types.Transaction, 0)
	txs = append(txs, makeTx(common.HexToAddress("0x08"), expiration+int64(params.TransactionExpiration)))
	recently.RecvBlock(bhash, height, txs)

	slot = expiration % int64(params.TransactionExpiration)
	assert.Equal(t, 2, len(recently.TxsByTime[slot].TxIndexes))
	assert.Equal(t, 2, len(recently.TraceMap))
}

func TestTxRecently_GetPath(t *testing.T) {
	recently := NewTxRecently()
	expiration := time.Now().Unix()

	txs1 := make([]*types.Transaction, 0)
	txs1 = append(txs1, makeTx(common.HexToAddress("0x01"), expiration))
	txs1 = append(txs1, makeTx(common.HexToAddress("0x02"), expiration))
	recently.RecvBlock(common.HexToHash("0x01"), 100, txs1)

	txs2 := make([]*types.Transaction, 0)
	txs2 = append(txs2, makeTx(common.HexToAddress("0x03"), expiration))
	txs2 = append(txs2, makeTx(common.HexToAddress("0x04"), expiration))
	recently.RecvBlock(common.HexToHash("0x02"), 101, txs2)

	txs3 := make([]*types.Transaction, 0)
	txs3 = append(txs3, makeTx(common.HexToAddress("0x05"), expiration))
	txs3 = append(txs3, makeTx(common.HexToAddress("0x06"), expiration))
	recently.RecvBlock(common.HexToHash("0x03"), 102, txs3)

	txs4 := make([]*types.Transaction, 0)
	txs4 = append(txs4, makeTx(common.HexToAddress("0x07"), expiration))
	txs4 = append(txs4, makeTx(common.HexToAddress("0x08"), expiration))
	recently.RecvBlock(common.HexToHash("0x04"), 103, txs4)

	txs5 := make([]*types.Transaction, 0)
	txs5 = append(txs5, txs1[0])
	txs5 = append(txs5, txs2[0])
	txs5 = append(txs5, txs3[0])
	recently.RecvBlock(common.HexToHash("0x05"), 104, txs5)

	txs6 := make([]*types.Transaction, 0)
	txs6 = append(txs6, txs1[0])
	txs6 = append(txs6, txs2[0])
	txs6 = append(txs6, txs3[0])
	recently.RecvBlock(common.HexToHash("0x06"), 105, txs6)

	txs := make([]*types.Transaction, 0)
	txs = append(txs, txs2[0])
	txs = append(txs, txs4[0])
	txs = append(txs, makeTx(common.HexToAddress("0x10"), expiration))
	txs = append(txs, makeTx(common.HexToAddress("0x11"), expiration))

	// get path
	result := recently.GetTrace(txs)
	assert.Equal(t, 2, len(result))

	blocks, ok := result[txs4[0].Hash()]
	assert.Equal(t, true, ok)
	assert.Equal(t, 1, len(blocks))
	assert.Equal(t, int64(103), blocks[common.HexToHash("0x04")])

	blocks, ok = result[txs2[0].Hash()]
	assert.Equal(t, true, ok)
	assert.Equal(t, 3, len(blocks))
	assert.Equal(t, int64(101), blocks[common.HexToHash("0x02")])
	assert.Equal(t, int64(104), blocks[common.HexToHash("0x05")])
	assert.Equal(t, int64(105), blocks[common.HexToHash("0x06")])

	// distance
	minHeight, maxHeight := blocks.heightRange()
	assert.Equal(t, int64(101), minHeight)
	assert.Equal(t, int64(105), maxHeight)
}

func TestTxRecently_PruneBlock(t *testing.T) {
	recently := NewTxRecently()
	expiration := time.Now().Unix()

	txs1 := make([]*types.Transaction, 0)
	txs1 = append(txs1, makeTx(common.HexToAddress("0x01"), expiration))
	txs1 = append(txs1, makeTx(common.HexToAddress("0x02"), expiration))
	recently.RecvBlock(common.HexToHash("0x01"), 100, txs1)

	txs2 := make([]*types.Transaction, 0)
	txs2 = append(txs2, makeTx(common.HexToAddress("0x03"), expiration))
	txs2 = append(txs2, makeTx(common.HexToAddress("0x04"), expiration))
	recently.RecvBlock(common.HexToHash("0x02"), 101, txs2)

	txs3 := make([]*types.Transaction, 0)
	txs3 = append(txs3, makeTx(common.HexToAddress("0x05"), expiration))
	txs3 = append(txs3, makeTx(common.HexToAddress("0x06"), expiration))
	recently.RecvBlock(common.HexToHash("0x03"), 102, txs3)

	txs4 := make([]*types.Transaction, 0)
	txs4 = append(txs4, makeTx(common.HexToAddress("0x07"), expiration))
	txs4 = append(txs4, makeTx(common.HexToAddress("0x08"), expiration))
	recently.RecvBlock(common.HexToHash("0x04"), 103, txs4)

	txs5 := make([]*types.Transaction, 0)
	txs5 = append(txs5, txs1[0])
	txs5 = append(txs5, txs2[0])
	txs5 = append(txs5, txs3[0])
	recently.RecvBlock(common.HexToHash("0x05"), 104, txs5)

	txs6 := make([]*types.Transaction, 0)
	txs6 = append(txs6, txs1[0])
	txs6 = append(txs6, txs2[0])
	txs6 = append(txs6, txs3[0])
	recently.RecvBlock(common.HexToHash("0x06"), 105, txs6)
	slot := expiration % int64(params.TransactionExpiration)
	assert.Equal(t, 8, len(recently.TxsByTime[slot].TxIndexes))
	assert.Equal(t, 3, len(recently.TraceMap[txs1[0].Hash()]))
	assert.Equal(t, 3, len(recently.TraceMap[txs2[0].Hash()]))
	assert.Equal(t, 3, len(recently.TraceMap[txs3[0].Hash()]))

	recently.PruneBlock(common.HexToHash("0x05"), 104, txs5)
	slot = expiration % int64(params.TransactionExpiration)
	assert.Equal(t, 8, len(recently.TxsByTime[slot].TxIndexes))

	assert.Equal(t, 2, len(recently.TraceMap[txs1[0].Hash()]))
	assert.Equal(t, 2, len(recently.TraceMap[txs2[0].Hash()]))
	assert.Equal(t, 2, len(recently.TraceMap[txs3[0].Hash()]))

	recently.PruneBlock(common.HexToHash("0x06"), 105, txs6)
	slot = expiration % int64(params.TransactionExpiration)
	assert.Equal(t, 8, len(recently.TxsByTime[slot].TxIndexes))

	assert.Equal(t, 1, len(recently.TraceMap[txs1[0].Hash()]))
	assert.Equal(t, 1, len(recently.TraceMap[txs2[0].Hash()]))
	assert.Equal(t, 1, len(recently.TraceMap[txs3[0].Hash()]))

	// box tx
	assert.Equal(t, 8, len(recently.TraceMap))
	curTime := time.Now().Unix()
	boxTxs := make([]*types.Transaction, 1)
	boxTxs[0] = createBoxTxRandom(common.HexToAddress("0xabcde"), 5, uint64(curTime)+100)
	recently.RecvBlock(common.HexToHash("0x06"), 106, boxTxs)
	assert.Equal(t, 14, len(recently.TraceMap))

	recently.PruneBlock(common.HexToHash("0x06"), 106, boxTxs)
	assert.Equal(t, 14, len(recently.TraceMap))
}
