package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

/* 近一个小时收到的所有交易的集合，用于防止交易重放 */
type TxRecently struct {
	/**
	 * 根据交易Hash索引该交易所在的块的高度
	 * (1) -1：还未打包的交易
	 * (2) 非-1：表示该交易所在高度最低的块的高度（一条交易可能同时存在几个块中）
	 */
	TxsByHash map[common.Hash]int64
}

func (recently *TxRecently) DelBatch(hashs []common.Hash) {
	if len(hashs) <= 0 || len(recently.TxsByHash) <= 0 {
		return
	}

	for index := 0; index < len(hashs); index++ {
		delete(recently.TxsByHash, hashs[index])
	}
}

func (recently *TxRecently) getPath(txs []*types.Transaction) map[common.Hash]int64 {
	result := make(map[common.Hash]int64)
	if len(txs) <= 0 || len(recently.TxsByHash) <= 0 {
		return result
	}

	for index := 0; index < len(txs); index++ {
		hash := txs[index].Hash()
		val, ok := recently.TxsByHash[hash]
		if !ok || (val == -1) { /* 未打包的交易，则肯定不在链上 */
			continue
		} else {
			result[hash] = val
		}
	}

	return result
}

/* 根据一批交易，返回这批交易在链上的交易列表，并返回他们的最低及最高的高度 */
func (recently *TxRecently) GetPath(txs []*types.Transaction) (int64, int64, []common.Hash) {
	hashs := recently.getPath(txs)
	if len(hashs) <= 0 {
		return -1, -1, make([]common.Hash, 0)
	}

	result := make([]common.Hash, 0, len(hashs))
	minHeight := int64(^uint64(0) >> 1)
	maxHeight := int64(-1)
	for k, v := range hashs {
		if v < minHeight {
			minHeight = v
		}

		if v > maxHeight {
			maxHeight = v
		}

		result = append(result, k)
	}

	return minHeight, maxHeight, result
}

func (recently *TxRecently) IsExist(hash common.Hash) bool {
	if len(recently.TxsByHash) <= 0 {
		return false
	}

	_, ok := recently.TxsByHash[hash]
	if !ok {
		return false
	} else {
		return true
	}
}

func (recently *TxRecently) add(height int64, tx *types.Transaction) {
	if (height < -1) || (tx == nil) {
		return
	}

	if recently.TxsByHash == nil {
		recently.TxsByHash = make(map[common.Hash]int64)
	}

	hash := tx.Hash()
	val, ok := recently.TxsByHash[hash]
	if !ok || (val == -1) || (val > height) {
		recently.TxsByHash[hash] = height
	}
}

func (recently *TxRecently) addBatch(height int64, txs []*types.Transaction) {
	if (height < -1) || (len(txs) <= 0) {
		return
	}

	if recently.TxsByHash == nil {
		recently.TxsByHash = make(map[common.Hash]int64)
	}

	for index := 0; index < len(txs); index++ {
		recently.add(height, txs[index])
	}
}

func (recently *TxRecently) RecvTx(tx *types.Transaction) {
	if tx == nil {
		return
	}

	if recently.TxsByHash == nil {
		recently.TxsByHash = make(map[common.Hash]int64)
	}

	if recently.IsExist(tx.Hash()) {
		return
	} else {
		recently.add(-1, tx)
	}
}

func (recently *TxRecently) RecvBlock(height int64, txs []*types.Transaction) {
	if (height < -1) || (len(txs) <= 0) {
		return
	}

	if recently.TxsByHash == nil {
		recently.TxsByHash = make(map[common.Hash]int64)
	}

	recently.addBatch(height, txs)
}
