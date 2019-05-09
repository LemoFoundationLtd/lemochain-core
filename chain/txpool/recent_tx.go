package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
)

type TxTimeBucket struct {
	/* 交易超时时间，UTC */
	Expiration uint64

	/* 该超时时间下的所有交易列表 */
	TxIndexes HashSet
}

func newTxTimeBucket(tx *types.Transaction) *TxTimeBucket {
	bucket := &TxTimeBucket{
		Expiration: tx.Expiration(),
		TxIndexes:  make(HashSet),
	}

	bucket.TxIndexes.Add(tx.Hash())
	return bucket
}

func (bucket *TxTimeBucket) timeOut(time uint64) bool {
	if bucket.Expiration < time {
		return true
	} else {
		return false
	}
}

func (bucket *TxTimeBucket) notTimeOut(time uint64) bool {
	if bucket.Expiration == time {
		return true
	} else {
		return false
	}
}

func (bucket *TxTimeBucket) halfHourAgo(time uint64) bool {
	if bucket.Expiration > time {
		return true
	} else {
		return false
	}
}

func (bucket *TxTimeBucket) add(tx *types.Transaction) {
	if bucket.Expiration != tx.Expiration() {
		log.Errorf("add tx to time.expiration(%d) != tx.expiration(%d)", bucket.Expiration, tx.Expiration())
	} else {
		bucket.TxIndexes.Add(tx.Hash())
	}
}

// 交易在区块中出现过的记录。key是区块hash，value是区块高度
type TxTrace map[common.Hash]int64

func NewEmptyTxTrace() TxTrace {
	return make(TxTrace)
}

func NewTxTrace(hash common.Hash, height int64) TxTrace {
	txTrace := make(TxTrace)
	txTrace[hash] = height
	return txTrace
}

func (txTrace TxTrace) add(hash common.Hash, height int64) {
	txTrace[hash] = height
}

func (txTrace TxTrace) heightRange() (int64, int64) {
	minHeight := int64(^uint64(0) >> 1)
	maxHeight := int64(-1)
	if len(txTrace) <= 0 {
		return 0, 0
	}

	for _, v := range txTrace {
		if minHeight > int64(v) {
			minHeight = int64(v)
		}

		if maxHeight < int64(v) {
			maxHeight = int64(v)
		}
	}

	return minHeight, maxHeight
}

func (txTrace TxTrace) del(hash common.Hash) {
	delete(txTrace, hash)
}

/* 近一个小时收到的所有交易的集合，用于防止交易重放 */
type RecentTx struct {
	// key是交易hash，value是交易所在块的记录
	TraceMap map[common.Hash]TxTrace

	TxsByTime []*TxTimeBucket
}

func NewTxRecently() *RecentTx {
	return &RecentTx{
		TraceMap:  make(map[common.Hash]TxTrace),
		TxsByTime: make([]*TxTimeBucket, TransactionExpiration),
	}
}

func (recent *RecentTx) delBatch4Txs(hashes []common.Hash) {
	if len(hashes) <= 0 || len(recent.TraceMap) <= 0 {
		return
	}

	for _, hash := range hashes {
		delete(recent.TraceMap, hash)
	}
}

func (recent *RecentTx) del4block(bHash common.Hash, tHash common.Hash) {
	trace, ok := recent.TraceMap[tHash]
	if ok {
		trace.del(bHash)
	}
}

func (recent *RecentTx) delBatch4Block(bHash common.Hash, thashes []common.Hash) {
	if len(thashes) <= 0 {
		return
	}
	for _, tHash := range thashes {
		recent.del4block(bHash, tHash)
	}
}

// GetTrace 获取交易在区块中出现过的记录
func (recent *RecentTx) GetTrace(txs []*types.Transaction) map[common.Hash]TxTrace {
	result := make(map[common.Hash]TxTrace)
	if len(txs) <= 0 || len(recent.TraceMap) <= 0 {
		return result
	}

	for _, tx := range txs {
		hash := tx.Hash()
		val, ok := recent.TraceMap[hash]
		if !ok || (val == nil) {
			continue
		} else {
			result[hash] = val
		}
	}

	return result
}

func (recent *RecentTx) IsExist(hash common.Hash) bool {
	if len(recent.TraceMap) <= 0 {
		return false
	}

	_, ok := recent.TraceMap[hash]
	if !ok {
		return false
	} else {
		return true
	}
}

func (recent *RecentTx) add2Time(tx *types.Transaction) error {
	expiration := tx.Expiration()
	slot := expiration % uint64(TransactionExpiration)
	bucket := recent.TxsByTime[slot]
	if bucket == nil {
		recent.TxsByTime[slot] = newTxTimeBucket(tx)
		return nil
	}

	if bucket.timeOut(expiration) {
		recent.delBatch4Txs(bucket.TxIndexes.Collect())
		recent.TxsByTime[slot] = newTxTimeBucket(tx)
		return nil
	}

	if bucket.notTimeOut(expiration) {
		bucket.add(tx)
		return nil
	}

	if bucket.halfHourAgo(expiration) {
		log.Errorf("tx is already time out.expiration: %d", expiration)
		return ErrTxPoolBlockExpired
	}

	return nil
}

func (recent *RecentTx) add2Hash(hash common.Hash, height int64, tx *types.Transaction) {
	_, ok := recent.TraceMap[tx.Hash()]
	if !ok {
		if height == -1 { // 新收到的交易，还没有被打包过
			recent.TraceMap[tx.Hash()] = NewEmptyTxTrace()
		} else {
			recent.TraceMap[tx.Hash()] = NewTxTrace(hash, height)
		}
	} else {
		if height == -1 { // 其他节点已经把该交易打包了，本节点同步该交易后，才收到该交易(不会发生)
			return
		} else {
			recent.TraceMap[tx.Hash()].add(hash, height)
		}
	}
}

func (recent *RecentTx) add(hash common.Hash, height int64, tx *types.Transaction) {
	if (height < -1) || (tx == nil) {
		return
	}

	err := recent.add2Time(tx)
	if err == nil {
		recent.add2Hash(hash, height, tx)
	}
}

func (recent *RecentTx) addBatch(hash common.Hash, height int64, txs []*types.Transaction) {
	for index := 0; index < len(txs); index++ {
		recent.add(hash, height, txs[index])
	}
}

/* 收到一条新的交易，放入最近交易列表 */
func (recent *RecentTx) RecvTx(tx *types.Transaction) {
	if tx == nil {
		return
	}

	if recent.IsExist(tx.Hash()) {
		return
	} else {
		recent.add(common.Hash{}, -1, tx) // 没在块上的交易， 设置其高度为-1
	}
}

/* 收到一个新块，把该块种的交易放入最近交易列表 */
func (recent *RecentTx) RecvBlock(hash common.Hash, height int64, txs []*types.Transaction) {
	if (height < -1) || (len(txs) <= 0) {
		return
	}

	recent.addBatch(hash, height, txs)
}

func (recent *RecentTx) PruneBlock(hash common.Hash, height int64, txs []*types.Transaction) {
	if len(txs) <= 0 {
		return
	}

	hashes := make([]common.Hash, 0, len(txs))
	for _, v := range txs {
		hashes = append(hashes, v.Hash())
	}

	recent.delBatch4Block(hash, hashes)
}
