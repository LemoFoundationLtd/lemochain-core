package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
)

func map2slice(src map[common.Hash]bool) []common.Hash {
	if len(src) <= 0 {
		return make([]common.Hash, 0)
	}

	result := make([]common.Hash, 0, len(src))
	for k, _ := range src {
		result = append(result, k)
	}
	return result
}

type TxTimeItem struct {

	/* 交易超时时间 */
	Expiration uint64

	/* 该超时时间下的所有交易 */
	Txs map[common.Hash]bool
}

func newTxTimeItem(tx *types.Transaction) *TxTimeItem {
	txs := make(map[common.Hash]bool)
	txs[tx.Hash()] = true
	return &TxTimeItem{
		Expiration: tx.Expiration(),
		Txs:        txs,
	}
}

func (item *TxTimeItem) add(tx *types.Transaction) {
	if tx == nil {
		return
	}

	if item.Expiration != tx.Expiration() {
		log.Errorf("add tx to time.expiration(%d) != tx.expiration(%d)", item.Expiration, tx.Expiration())
		return
	}

	if item.Txs == nil {
		item.Txs = make(map[common.Hash]bool)
	}

	item.Txs[tx.Hash()] = true
}

/* 一个交易队列，具有根据交易Hash查询和先进先出的特性*/
type TxSliceByTime struct {
	TxsIndexByHash map[common.Hash]int
	TxsIndexByTime []*TxTimeItem
	Txs            []*types.Transaction
}

func (slice *TxSliceByTime) IsExist(tx *types.Transaction) bool {
	if (len(slice.Txs) <= 0) ||
		(len(slice.TxsIndexByHash) <= 0) ||
		(len(slice.TxsIndexByTime) <= 0) {
		return false
	}

	_, ok := slice.TxsIndexByHash[tx.Hash()]
	if !ok {
		return false
	} else {
		return true
	}
}

func (slice *TxSliceByTime) Get(hash common.Hash) *types.Transaction {
	if (len(slice.Txs) <= 0) ||
		(len(slice.TxsIndexByHash) <= 0) ||
		(len(slice.TxsIndexByTime) <= 0) {
		return nil
	}

	index, ok := slice.TxsIndexByHash[hash]
	if !ok {
		return nil
	} else {
		return slice.Txs[index]
	}
}

func (slice *TxSliceByTime) add2Hash(tx *types.Transaction) {
	hash := tx.Hash()
	slice.Txs = append(slice.Txs, tx)
	slice.TxsIndexByHash[hash] = len(slice.Txs) - 1
}

func (slice *TxSliceByTime) add2time(tx *types.Transaction) []common.Hash {
	expiration := tx.Expiration()
	slot := expiration % uint64(2*TransactionExpiration)
	items := slice.TxsIndexByTime[slot]
	if items == nil {
		slice.TxsIndexByTime[slot] = newTxTimeItem(tx)
		return make([]common.Hash, 0)
	}

	if items.Expiration < expiration {
		result := map2slice(items.Txs)
		slice.DelBatch(result)
		slice.TxsIndexByTime[slot] = newTxTimeItem(tx)
		return result
	}

	if items.Expiration == expiration {
		items.Txs[tx.Hash()] = true
		slice.TxsIndexByTime[slot] = items
		return make([]common.Hash, 0)
	}

	if items.Expiration > expiration {
		log.Errorf("tx is already time out.expiration: %d", expiration)
	}

	return make([]common.Hash, 0)
}

func (slice *TxSliceByTime) add(tx *types.Transaction) []common.Hash {
	if slice.IsExist(tx) {
		return make([]common.Hash, 0)
	}

	slice.add2Hash(tx)
	return slice.add2time(tx)
}

func (slice *TxSliceByTime) Del(hash common.Hash) {
	if (slice.Txs == nil) || (slice.TxsIndexByHash == nil) || (slice.TxsIndexByTime == nil) {
		return
	}

	index, ok := slice.TxsIndexByHash[hash]
	if !ok {
		return
	}

	delete(slice.TxsIndexByHash, hash)
	slot := slice.Txs[index].Expiration() % uint64(2*TransactionExpiration)
	delete(slice.TxsIndexByTime[slot].Txs, hash)
	slice.Txs = append(slice.Txs[:index], slice.Txs[index+1:]...)
}

func (slice *TxSliceByTime) DelBatch(hashs []common.Hash) {
	if len(hashs) <= 0 {
		return
	}

	for index := 0; index < len(hashs); index++ {
		slice.Del(hashs[index])
	}
}

func (slice *TxSliceByTime) DelBatchByTx(txs []*types.Transaction) {
	if len(txs) <= 0 {
		return
	}

	for index := 0; index < len(txs); index++ {
		slice.Del(txs[index].Hash())
	}
}

func (slice *TxSliceByTime) GetBatch(size int) []*types.Transaction {
	if (size <= 0) || (slice.Txs == nil) || (slice.TxsIndexByHash == nil) || (slice.TxsIndexByTime == nil) {
		return make([]*types.Transaction, 0)
	}

	length := size
	if len(slice.Txs) <= size {
		length = len(slice.Txs)
	}

	result := make([]*types.Transaction, 0, length)
	result = append(result[:], slice.Txs[0:length]...)
	return result
}

/* 添加新的交易进入交易池，返回超时的交易列表 */
func (slice *TxSliceByTime) AddBatch(txs []*types.Transaction) []common.Hash {
	if (slice.Txs == nil) || (slice.TxsIndexByHash == nil) || (slice.TxsIndexByTime == nil) {
		slice.Txs = make([]*types.Transaction, 0)
		slice.TxsIndexByHash = make(map[common.Hash]int)
		slice.TxsIndexByTime = make([]*TxTimeItem, 2*TransactionExpiration)
	}

	result := make([]common.Hash, 0)
	if len(txs) <= 0 {
		return result
	}

	for index := 0; index < len(txs); index++ {
		result = append(result, slice.add(txs[index])...)
	}

	return result
}
