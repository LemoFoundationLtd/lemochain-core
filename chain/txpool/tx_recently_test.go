package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTxRecently_RecvTx(t *testing.T) {
	recently := NewTxRecently()
	recently.RecvTx(nil)
	assert.Equal(t, 0, len(recently.TxsByHash))

	tx1 := makeTxRandom(common.HexToAddress("0x01"))
	tx2 := makeTxRandom(common.HexToAddress("0x02"))
	recently.RecvTx(tx1)
	recently.RecvTx(tx2)

	assert.Equal(t, 2, len(recently.TxsByHash))
	assert.Equal(t, 0, len(recently.TxsByHash[tx1.Hash()].BlocksHash))
	assert.Equal(t, 0, len(recently.TxsByHash[tx2.Hash()].BlocksHash))

	if tx1.Expiration() == tx2.Expiration() {
		slot := tx1.Expiration() % uint64(TransactionExpiration)
		assert.Equal(t, 2, len(recently.TxsByTime[slot].TxIndexes))
	} else {
		slot1 := tx1.Expiration() % uint64(TransactionExpiration)
		assert.Equal(t, 1, len(recently.TxsByTime[slot1].TxIndexes))

		slot2 := tx2.Expiration() % uint64(TransactionExpiration)
		assert.Equal(t, 1, len(recently.TxsByTime[slot2].TxIndexes))
	}
}

func TestTxRecently_RecvBlock(t *testing.T) {
	recently := NewTxRecently()
	bhash := common.HexToHash("0xabc")
	height := int64(100)
	txs := make([]*types.Transaction, 0)

	recently.RecvBlock(bhash, height, txs)
	assert.Equal(t, 0, len(recently.TxsByHash))

	txs = append(txs, makeTxRandom(common.HexToAddress("0x01")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x02")))
	recently.RecvBlock(bhash, height, txs)

	assert.Equal(t, 2, len(recently.TxsByHash))
	assert.Equal(t, height, recently.TxsByHash[txs[0].Hash()].BlocksHash[bhash])
	assert.Equal(t, height, recently.TxsByHash[txs[1].Hash()].BlocksHash[bhash])
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

	slot := expiration % int64(TransactionExpiration)
	assert.Equal(t, 4, len(recently.TxsByTime[slot].TxIndexes))
	assert.Equal(t, 4, len(recently.TxsByHash))

	txs = make([]*types.Transaction, 0)
	txs = append(txs, makeTx(common.HexToAddress("0x05"), expiration+int64(TransactionExpiration)))
	recently.RecvBlock(bhash, height, txs)

	slot = expiration % int64(TransactionExpiration)
	assert.Equal(t, 1, len(recently.TxsByTime[slot].TxIndexes))
	assert.Equal(t, 1, len(recently.TxsByHash))

	txs = make([]*types.Transaction, 0)
	txs = append(txs, makeTx(common.HexToAddress("0x06"), expiration))
	recently.RecvBlock(bhash, height, txs)

	slot = expiration % int64(TransactionExpiration)
	assert.Equal(t, 1, len(recently.TxsByTime[slot].TxIndexes))
	assert.Equal(t, 1, len(recently.TxsByHash))

	txs = make([]*types.Transaction, 0)
	txs = append(txs, makeTx(common.HexToAddress("0x08"), expiration+int64(TransactionExpiration)))
	recently.RecvBlock(bhash, height, txs)

	slot = expiration % int64(TransactionExpiration)
	assert.Equal(t, 2, len(recently.TxsByTime[slot].TxIndexes))
	assert.Equal(t, 2, len(recently.TxsByHash))
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
	result := recently.GetPath(txs)
	assert.Equal(t, 2, len(result))

	blocks, ok := result[txs4[0].Hash()]
	assert.Equal(t, true, ok)
	assert.Equal(t, 1, len(blocks.BlocksHash))
	assert.Equal(t, int64(103), blocks.BlocksHash[common.HexToHash("0x04")])

	blocks, ok = result[txs2[0].Hash()]
	assert.Equal(t, true, ok)
	assert.Equal(t, 3, len(blocks.BlocksHash))
	assert.Equal(t, int64(101), blocks.BlocksHash[common.HexToHash("0x02")])
	assert.Equal(t, int64(104), blocks.BlocksHash[common.HexToHash("0x05")])
	assert.Equal(t, int64(105), blocks.BlocksHash[common.HexToHash("0x06")])

	// distance
	minHeight, maxHeight, hashes := blocks.distance()
	assert.Equal(t, int64(101), minHeight)
	assert.Equal(t, int64(105), maxHeight)
	assert.Equal(t, 3, len(hashes))
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
	slot := expiration % int64(TransactionExpiration)
	assert.Equal(t, 8, len(recently.TxsByTime[slot].TxIndexes))
	assert.Equal(t, 3, len(recently.TxsByHash[txs1[0].Hash()].BlocksHash))
	assert.Equal(t, 3, len(recently.TxsByHash[txs2[0].Hash()].BlocksHash))
	assert.Equal(t, 3, len(recently.TxsByHash[txs3[0].Hash()].BlocksHash))

	recently.PruneBlock(common.HexToHash("0x05"), 104, txs5)
	slot = expiration % int64(TransactionExpiration)
	assert.Equal(t, 8, len(recently.TxsByTime[slot].TxIndexes))

	assert.Equal(t, 2, len(recently.TxsByHash[txs1[0].Hash()].BlocksHash))
	assert.Equal(t, 2, len(recently.TxsByHash[txs2[0].Hash()].BlocksHash))
	assert.Equal(t, 2, len(recently.TxsByHash[txs3[0].Hash()].BlocksHash))

	recently.PruneBlock(common.HexToHash("0x06"), 105, txs6)
	slot = expiration % int64(TransactionExpiration)
	assert.Equal(t, 8, len(recently.TxsByTime[slot].TxIndexes))

	assert.Equal(t, 1, len(recently.TxsByHash[txs1[0].Hash()].BlocksHash))
	assert.Equal(t, 1, len(recently.TxsByHash[txs2[0].Hash()].BlocksHash))
	assert.Equal(t, 1, len(recently.TxsByHash[txs3[0].Hash()].BlocksHash))
}
