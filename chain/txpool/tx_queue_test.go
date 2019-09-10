package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

func TestTxQueue_Pop(t *testing.T) {
	queue := NewTxQueue(big.NewInt(0))
	txs := make([]*types.Transaction, 0)
	txs = append(txs, makeTxRandom(common.HexToAddress("0x01")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x02")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x03")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x04")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x05")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x06")))
	queue.PushBatch(txs)

	// 正常情况
	result := queue.Pop(uint32(time.Now().Unix()), 2)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, 6, len(queue.TxsQueue))
	assert.Equal(t, 6, len(queue.TxsStatus))

	// 超时交易
	queue = NewTxQueue(big.NewInt(0))
	txs = make([]*types.Transaction, 0)
	txs = append(txs, makeExpirationTx(common.HexToAddress("0x01")))
	txs = append(txs, makeExpirationTx(common.HexToAddress("0x02")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x03")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x04")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x05")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x06")))
	queue.PushBatch(txs)
	result = queue.Pop(uint32(time.Now().Unix()), 2)
	assert.Equal(t, 2, len(result))

	assert.Equal(t, 4, len(queue.TxsQueue))
	assert.Equal(t, 4, len(queue.TxsStatus))

	_, ok := queue.TxsStatus[txs[0].Hash()]
	assert.Equal(t, false, ok)
	_, ok = queue.TxsStatus[txs[1].Hash()]
	assert.Equal(t, false, ok)

	// 删除交易
	queue = NewTxQueue(big.NewInt(0))
	txs = make([]*types.Transaction, 0)
	txs = append(txs, makeExpirationTx(common.HexToAddress("0x01")))
	txs = append(txs, makeExpirationTx(common.HexToAddress("0x02")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x03")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x04")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x05")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x06")))
	queue.PushBatch(txs)

	delTxs := make([]common.Hash, 0)
	delTxs = append(delTxs, txs[2].Hash())
	delTxs = append(delTxs, txs[3].Hash())
	queue.DelBatch(delTxs)

	result = queue.Pop(uint32(time.Now().Unix()), 10)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, 2, len(queue.TxsQueue))
	assert.Equal(t, 2, len(queue.TxsStatus))
	_, ok = queue.TxsStatus[txs[0].Hash()]
	assert.Equal(t, false, ok)
	_, ok = queue.TxsStatus[txs[1].Hash()]
	assert.Equal(t, false, ok)
	_, ok = queue.TxsStatus[txs[2].Hash()]
	assert.Equal(t, false, ok)
	_, ok = queue.TxsStatus[txs[3].Hash()]
	assert.Equal(t, false, ok)
}

func TestTxQueue_PushBatch(t *testing.T) {
	queue := NewTxQueue(big.NewInt(0))
	txs := make([]*types.Transaction, 0)
	queue.PushBatch(txs)
	assert.Equal(t, 0, len(queue.TxsQueue))
	assert.Equal(t, 0, len(queue.TxsStatus))

	// 正常情况
	txs = append(txs, makeTxRandom(common.HexToAddress("0x01")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x02")))
	queue.PushBatch(txs)
	assert.Equal(t, 2, len(queue.TxsQueue))
	assert.Equal(t, 2, len(queue.TxsStatus))
	assert.Equal(t, true, queue.TxsStatus[txs[0].Hash()])
	assert.Equal(t, true, queue.TxsStatus[txs[1].Hash()])

	// 删除后添加 && 重复添加
	queue.Del(txs[0].Hash())
	assert.Equal(t, false, queue.TxsStatus[txs[0].Hash()])
	assert.Equal(t, true, queue.TxsStatus[txs[1].Hash()])
	assert.Equal(t, 2, len(queue.TxsQueue))
	assert.Equal(t, 2, len(queue.TxsStatus))

	queue.PushBatch(txs)
	assert.Equal(t, 2, len(queue.TxsQueue))
	assert.Equal(t, 2, len(queue.TxsStatus))
	assert.Equal(t, true, queue.TxsStatus[txs[0].Hash()])
	assert.Equal(t, true, queue.TxsStatus[txs[1].Hash()])
}

func TestTxQueue_Del(t *testing.T) {
	queue := NewTxQueue(big.NewInt(0))
	tx := makeTxRandom(common.HexToAddress("0x01"))
	queue.Push(tx)
	assert.Equal(t, 1, len(queue.TxsQueue))
	assert.Equal(t, 1, len(queue.TxsStatus))
	assert.Equal(t, true, queue.TxsStatus[tx.Hash()])

	queue.Del(tx.Hash())
	assert.Equal(t, 1, len(queue.TxsQueue))
	assert.Equal(t, 1, len(queue.TxsStatus))
	assert.Equal(t, false, queue.TxsStatus[tx.Hash()])

	queue.Del(tx.Hash())
	assert.Equal(t, 1, len(queue.TxsQueue))
	assert.Equal(t, 1, len(queue.TxsStatus))
	assert.Equal(t, false, queue.TxsStatus[tx.Hash()])
}

func TestTxQueue_DelBatch(t *testing.T) {
	queue := NewTxQueue(big.NewInt(0))
	txs := make([]*types.Transaction, 0)
	txs = append(txs, makeTxRandom(common.HexToAddress("0x01")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x02")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x03")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x04")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x05")))
	txs = append(txs, makeTxRandom(common.HexToAddress("0x06")))
	queue.PushBatch(txs)

	delTxs := make([]common.Hash, 0)
	delTxs = append(delTxs, txs[1].Hash())
	delTxs = append(delTxs, txs[3].Hash())
	queue.DelBatch(delTxs)

	assert.Equal(t, 6, len(queue.TxsQueue))
	assert.Equal(t, 6, len(queue.TxsStatus))
	assert.Equal(t, true, queue.TxsStatus[txs[0].Hash()])
	assert.Equal(t, false, queue.TxsStatus[txs[1].Hash()])
	assert.Equal(t, true, queue.TxsStatus[txs[2].Hash()])
	assert.Equal(t, false, queue.TxsStatus[txs[3].Hash()])
	assert.Equal(t, true, queue.TxsStatus[txs[4].Hash()])
}
