package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
)

type TxsOneSecond struct {
	/* 交易超时时间，UTC */
	Expiration uint64

	/* 该超时时间下的所有交易列表 */
	TxIndexes map[common.Hash]bool
}

func newTxsOneSecond(tx *types.Transaction) *TxsOneSecond {
	txsOneSecond := &TxsOneSecond{
		Expiration: tx.Expiration(),
		TxIndexes:  make(map[common.Hash]bool),
	}

	txsOneSecond.TxIndexes[tx.Hash()] = true
	return txsOneSecond
}

func (txsOneSecond *TxsOneSecond) timeOut(time uint64) bool {
	if txsOneSecond.Expiration < time {
		return true
	} else {
		return false
	}
}

func (txsOneSecond *TxsOneSecond) unTimeOut(time uint64) bool {
	if txsOneSecond.Expiration == time {
		return true
	} else {
		return false
	}
}

func (txsOneSecond *TxsOneSecond) beforeEx(time uint64) bool {
	if txsOneSecond.Expiration > time {
		return true
	} else {
		return false
	}
}

func (txsOneSecond *TxsOneSecond) add(tx *types.Transaction) {
	if tx == nil {
		return
	}

	if txsOneSecond.Expiration != tx.Expiration() {
		log.Errorf("add tx to time.expiration(%d) != tx.expiration(%d)", txsOneSecond.Expiration, tx.Expiration())
	} else {
		txsOneSecond.TxIndexes[tx.Hash()] = true
	}
}

func (txsOneSecond *TxsOneSecond) get() []common.Hash {
	if len(txsOneSecond.TxIndexes) <= 0 {
		return make([]common.Hash, 0)
	}

	result := make([]common.Hash, 0, len(txsOneSecond.TxIndexes))
	for k, _ := range txsOneSecond.TxIndexes {
		result = append(result, k)
	}
	return result
}

type TxInBlocks struct {
	BlocksHash map[common.Hash]int64
}

func NewEmptyTxInBlocks() *TxInBlocks {
	return &TxInBlocks{
		BlocksHash: make(map[common.Hash]int64),
	}
}

func NewTxInBlocks(hash common.Hash, height int64) *TxInBlocks {
	txInBlocks := &TxInBlocks{
		BlocksHash: make(map[common.Hash]int64),
	}
	txInBlocks.BlocksHash[hash] = height
	return txInBlocks
}

func (txInBlocks *TxInBlocks) add(hash common.Hash, height int64) {
	txInBlocks.BlocksHash[hash] = height
}

func (txInBlocks *TxInBlocks) distance() (int64, int64, map[common.Hash]int64) {
	minHeight := int64(^uint64(0) >> 1)
	maxHeight := int64(-1)
	if len(txInBlocks.BlocksHash) <= 0 {
		return 0, 0, make(map[common.Hash]int64)
	}

	for _, v := range txInBlocks.BlocksHash {
		if minHeight > int64(v) {
			minHeight = int64(v)
		}

		if maxHeight < int64(v) {
			maxHeight = int64(v)
		}
	}

	return minHeight, maxHeight, txInBlocks.BlocksHash
}

func (txInBlocks *TxInBlocks) del(hash common.Hash) {
	delete(txInBlocks.BlocksHash, hash)
}

/* 近一个小时收到的所有交易的集合，用于防止交易重放 */
type TxRecently struct {
	TxsByHash map[common.Hash]*TxInBlocks

	TxsByTime []*TxsOneSecond
}

func NewTxRecently() *TxRecently {
	return &TxRecently{
		TxsByHash: make(map[common.Hash]*TxInBlocks),
		TxsByTime: make([]*TxsOneSecond, TransactionExpiration),
	}
}

func (recently *TxRecently) delBatch4Txs(hashs []common.Hash) {
	if len(hashs) <= 0 || len(recently.TxsByHash) <= 0 {
		return
	}

	for index := 0; index < len(hashs); index++ {
		delete(recently.TxsByHash, hashs[index])
	}
}

func (recently *TxRecently) del4block(bhash common.Hash, thash common.Hash) {
	txInBlocks, ok := recently.TxsByHash[thash]
	if ok {
		txInBlocks.del(bhash)
	}
}

func (recently *TxRecently) delBatch4Block(bhash common.Hash, thashes []common.Hash) {
	if len(thashes) <= 0 {
		return
	}
	for _, thash := range thashes {
		recently.del4block(bhash, thash)
	}
}

func (recently *TxRecently) GetPath(txs []*types.Transaction) map[common.Hash]*TxInBlocks {
	result := make(map[common.Hash]*TxInBlocks)
	if len(txs) <= 0 || len(recently.TxsByHash) <= 0 {
		return result
	}

	for index := 0; index < len(txs); index++ {
		hash := txs[index].Hash()
		val, ok := recently.TxsByHash[hash]
		if !ok || (val == nil) {
			continue
		} else {
			result[hash] = val
		}
	}

	return result
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

func (recently *TxRecently) add2Time(tx *types.Transaction) bool {
	if tx == nil {
		return false
	}

	expiration := tx.Expiration()
	slot := expiration % uint64(TransactionExpiration)
	txsOneSecond := recently.TxsByTime[slot]
	if txsOneSecond == nil {
		recently.TxsByTime[slot] = newTxsOneSecond(tx)
		return true
	} else {
		if txsOneSecond.timeOut(expiration) {
			recently.delBatch4Txs(txsOneSecond.get())
			recently.TxsByTime[slot] = newTxsOneSecond(tx)
			return true
		}

		if txsOneSecond.unTimeOut(expiration) {
			txsOneSecond.add(tx)
			return true
		}

		if txsOneSecond.beforeEx(expiration) {
			log.Errorf("tx is already time out.expiration: %d", expiration)
			return false
		}

		return true
	}
}

func (recently *TxRecently) add2Hash(hash common.Hash, height int64, tx *types.Transaction) {
	if (height < -1) || (tx == nil) {
		return
	}

	_, ok := recently.TxsByHash[tx.Hash()]
	if !ok {
		if height == -1 { // 该交易还没有被打包
			recently.TxsByHash[tx.Hash()] = NewEmptyTxInBlocks()
		} else {
			recently.TxsByHash[tx.Hash()] = NewTxInBlocks(hash, height)
		}
	} else {
		recently.TxsByHash[tx.Hash()].add(hash, height)
	}
}

func (recently *TxRecently) add(hash common.Hash, height int64, tx *types.Transaction) {
	if recently.add2Time(tx) {
		recently.add2Hash(hash, height, tx)
	}
}

func (recently *TxRecently) addBatch(hash common.Hash, height int64, txs []*types.Transaction) {
	for index := 0; index < len(txs); index++ {
		recently.add(hash, height, txs[index])
	}
}

/* 收到一条新的交易，放入最近交易列表 */
func (recently *TxRecently) RecvTx(tx *types.Transaction) {
	if tx == nil {
		return
	}

	if recently.IsExist(tx.Hash()) {
		return
	} else {
		recently.add(common.Hash{}, -1, tx) // 没在块上的交易， 设置其高度为-1
	}
}

/* 收到一个新块，把该块种的交易放入最近交易列表 */
func (recently *TxRecently) RecvBlock(hash common.Hash, height int64, txs []*types.Transaction) {
	if (height < -1) || (len(txs) <= 0) {
		return
	}

	recently.addBatch(hash, height, txs)
}

func (recently *TxRecently) PruneBlock(hash common.Hash, height int64, txs []*types.Transaction) {
	if len(txs) <= 0 {
		return
	}

	hashes := make([]common.Hash, 0, len(txs))
	for _, v := range txs {
		hashes = append(hashes, v.Hash())
	}

	recently.delBatch4Block(hash, hashes)
}
