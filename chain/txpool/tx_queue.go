package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

type TxQueue struct {
	/* 交易状态标记，true：正常；false：已删除 */
	TxsStatus map[common.Hash]bool

	TxsQueue []*types.Transaction
}

func NewTxQueue() *TxQueue {
	return &TxQueue{
		TxsStatus: make(map[common.Hash]bool),
		TxsQueue:  make([]*types.Transaction, 0),
	}
}

func (queue *TxQueue) isExist(hash common.Hash) bool {
	if len(queue.TxsStatus) <= 0 {
		return false
	}

	isExist, ok := queue.TxsStatus[hash]
	return ok && isExist
}

func (queue *TxQueue) hardDel(index int) {
	tx := queue.TxsQueue[index]
	delete(queue.TxsStatus, tx.Hash())

	queue.TxsQueue = append(queue.TxsQueue[:index], queue.TxsQueue[index+1:]...)
}

func (queue *TxQueue) softDel(hash common.Hash) {
	queue.TxsStatus[hash] = false
}

func (queue *TxQueue) isTimeOut(tx *types.Transaction, time uint32) bool {
	// TODO: box tx
	if tx.Expiration() < uint64(time) {
		return true
	} else {
		return false
	}
}

func (queue *TxQueue) Del(hash common.Hash) {
	if !queue.isExist(hash) {
		return
	} else {
		queue.softDel(hash)
		txpoolTotalNumberCounter.Dec(1) // 删除交易池中的交易
	}
}

func (queue *TxQueue) DelBatch(hashes []common.Hash) {
	if len(hashes) <= 0 {
		return
	}

	for _, hash := range hashes {
		queue.Del(hash)
		invalidTxMeter.Mark(1)
	}
}

// IsExistCanPackageTx 存在可以打包的交易
func (queue *TxQueue) IsExistCanPackageTx(time uint32) bool {
	for _, tx := range queue.TxsQueue {
		// 此交易没有超时并且没有被软删除掉
		if queue.isExist(tx.Hash()) && !queue.isTimeOut(tx, time) {
			return true
		}
	}
	return false
}

func (queue *TxQueue) Pop(time uint32, size int) []*types.Transaction {
	result := make([]*types.Transaction, 0)
	if size <= 0 {
		return result
	}

	count := 0
	index := 0
	for count < size && index < len(queue.TxsQueue) {
		if !queue.isExist(queue.TxsQueue[index].Hash()) { // 标记为删除
			queue.hardDel(index)
			continue
		}

		if queue.isTimeOut(queue.TxsQueue[index], time) {
			queue.hardDel(index)
			continue
		}

		result = append(result, queue.TxsQueue[index])
		count++
		index++
	}
	return result
}

func (queue *TxQueue) Push(tx *types.Transaction) {
	if tx == nil {
		return
	}

	hash := tx.Hash()
	isExist, ok := queue.TxsStatus[hash]
	if !ok { // 没有该交易
		queue.TxsStatus[hash] = true
		queue.TxsQueue = append(queue.TxsQueue, tx)
		return
	}

	if !isExist { // 有该交易，但是处于删除状态
		queue.TxsStatus[hash] = true
		return
	}
}

func (queue *TxQueue) PushBatch(txs []*types.Transaction) {
	if len(txs) <= 0 {
		return
	}

	for _, tx := range txs {
		queue.Push(tx)
	}
}
