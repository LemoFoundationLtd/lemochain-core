package chain

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
)

var TransactionExpiration = 30 * 60

var (
	TxPoolErrExist = errors.New("transaction is exist")
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

type BlocksByTimeItem struct {
	Time    uint32
	Indexes map[uint32][]common.Hash
}

type BlockNode struct {
	Header *types.Header

	/* 该块打包的交易的索引 */
	TxsIndex map[common.Hash]bool
}

/* 以当前高度为基准，高度差为3600的所有块 */
type BlocksTrie struct {
	BaseHeight uint32

	/* 根据高度对块进行索引 */
	BlocksByHash []map[common.Hash]*BlockNode

	/* 根据时间刻度对块Hash进行索引，用来回收块 */
	BlocksByTime []*BlocksByTimeItem
}

func (trie *BlocksTrie) buildTxsIndex(txs []*types.Transaction) map[common.Hash]bool {
	txsIndex := make(map[common.Hash]bool)
	if len(txs) <= 0 {
		return txsIndex
	}

	for index := 0; index < len(txs); index++ {
		txsIndex[txs[index].Hash()] = true
	}

	return txsIndex
}

func (trie *BlocksTrie) buildBlockNode(block *types.Block) *BlockNode {
	if block == nil {
		return nil
	}

	return &BlockNode{
		Header:   block.Header,
		TxsIndex: trie.buildTxsIndex(block.Txs),
	}
}

func (trie *BlocksTrie) newBlockByHashItem(block *types.Block) {
	if (trie.BlocksByTime == nil) || (trie.BlocksByHash == nil) {
		trie.make()
	}
	if block == nil {
		return
	}

	hash := block.Hash()
	slot := block.Height() % uint32(2*TransactionExpiration)
	nodes := trie.BlocksByHash[slot]
	if nodes == nil {
		trie.BlocksByHash[slot] = make(map[common.Hash]*BlockNode)
		trie.BlocksByHash[slot][hash] = trie.buildBlockNode(block)
	} else {
		trie.BlocksByHash[slot][hash] = trie.buildBlockNode(block)
	}
}

func (trie *BlocksTrie) newBlockByTimeItem(block *types.Block) *BlocksByTimeItem {
	item := &BlocksByTimeItem{
		Time:    block.Time(),
		Indexes: make(map[uint32][]common.Hash),
	}

	item.Indexes[block.Height()] = make([]common.Hash, 1)
	item.Indexes[block.Height()][0] = block.Hash()
	return item
}

func (trie *BlocksTrie) delBlocksByHash(result map[uint32][]common.Hash) {
	if len(result) <= 0 {
		return
	}
	if len(trie.BlocksByHash) <= 0 {
		return
	}

	for height, hashs := range result {
		slot := height % uint32(2*TransactionExpiration)
		blocks := trie.BlocksByHash[slot]
		if len(blocks) <= 0 {
			continue
		}

		for index := 0; index < len(hashs); index++ {
			delete(blocks, hashs[index])
		}
	}
}

func (trie *BlocksTrie) DelBlock(block *types.Block) {
	if block == nil {
		return
	}

	height := block.Height()
	hash := block.Hash()
	slot := height % uint32(2*TransactionExpiration)
	items := trie.BlocksByHash[slot]
	if len(items) <= 0 {
		return
	}

	delete(items, hash)
}

/* 从指定块开始，收集该块所在链指定高度区间的块[minHeight, maxHeight] */
func (trie *BlocksTrie) Path(hash common.Hash, height uint32, minHeight uint32, maxHeight uint32) []*BlockNode {
	if hash == (common.Hash{}) || (minHeight > maxHeight) {
		return make([]*BlockNode, 0)
	}

	result := make([]*BlockNode, 0, maxHeight-minHeight+1)

	pHash := hash
	pHeight := height
	for pHeight >= minHeight && pHeight <= maxHeight {
		block := trie.BlocksByHash[pHeight][pHash]
		if block == nil {
			panic(fmt.Sprintf("get block is nil.hash: %s", common.ToHex(hash.Bytes())))
		} else {
			result = append(result, block)
			pHeight = block.Header.Height - 1
			pHash = block.Header.ParentHash
		}
	}

	return result
}

func (trie *BlocksTrie) make() {
	trie.BlocksByHash = make([]map[common.Hash]*BlockNode, 2*TransactionExpiration)
	trie.BlocksByTime = make([]*BlocksByTimeItem, 2*TransactionExpiration)
}

func (trie *BlocksTrie) merge(src map[uint32][]common.Hash) map[common.Hash]bool {
	result := make(map[common.Hash]bool)

	if len(src) <= 0 {
		return result
	}

	for _, v := range src {
		for index := 0; index < len(v); index++ {
			result[v[index]] = true
		}
	}

	return result
}

func (trie *BlocksTrie) PushBlock(block *types.Block) map[common.Hash]bool {
	if block == nil {
		return make(map[common.Hash]bool)
	}
	if (trie.BlocksByTime == nil) || (trie.BlocksByHash == nil) {
		trie.make()
	}

	slot := block.Time() % uint32(2*TransactionExpiration)
	item := trie.BlocksByTime[slot]
	if item == nil {
		trie.BlocksByTime[slot] = trie.newBlockByTimeItem(block)
		trie.newBlockByHashItem(block)
		return make(map[common.Hash]bool)
	}

	if item.Time < block.Time() {
		trie.delBlocksByHash(item.Indexes)
		trie.BlocksByTime[slot] = trie.newBlockByTimeItem(block)
		trie.newBlockByHashItem(block)
		return trie.merge(item.Indexes)
	}

	if item.Time == block.Time() {
		trie.newBlockByHashItem(block)
		return make(map[common.Hash]bool)
	}

	if item.Time > block.Time() {
		panic(fmt.Sprintf("item.Time(%d) > block.Time(%d)", item.Time, block.Time()))
	}

	return make(map[common.Hash]bool)
}

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

type TxTimeItem struct {
	/* 交易超时时间 */
	Expiration uint64

	/* 该超时时间下的所有交易 */
	Txs map[common.Hash]bool
}

/* 一个交易队列，具有根据交易Hash查询和先进先出的特性*/
type TxSliceByTime struct {
	TxsIndexByHash map[common.Hash]int
	TxsIndexByTime []*TxTimeItem
	Txs            []*types.Transaction
}

func (slice *TxSliceByTime) IsExist(tx *types.Transaction) bool {
	if (len(slice.Txs) <= 0) || (len(slice.TxsIndexByHash) <= 0) || (len(slice.TxsIndexByTime) <= 0) {
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
	if (len(slice.Txs) <= 0) || (len(slice.TxsIndexByHash) <= 0) || (len(slice.TxsIndexByTime) <= 0) {
		return nil
	}

	index, ok := slice.TxsIndexByHash[hash]
	if !ok {
		return nil
	} else {
		return slice.Txs[index]
	}
}

func (slice *TxSliceByTime) newTxTimeItem(tx *types.Transaction) *TxTimeItem {
	txs := make(map[common.Hash]bool)
	txs[tx.Hash()] = true
	return &TxTimeItem{
		Expiration: tx.Expiration(),
		Txs:        txs,
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
		slice.TxsIndexByTime[slot] = slice.newTxTimeItem(tx)
		return make([]common.Hash, 0)
	}

	if items.Expiration < expiration {
		result := map2slice(items.Txs)
		slice.DelBatch(result)
		slice.TxsIndexByTime[slot] = slice.newTxTimeItem(tx)
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

type TxPool struct {
	/* 还未被打包进块的交易 */
	TxCache TxSliceByTime

	/* 最近1个小时的所有交易 */
	TxRecently TxRecently

	/* 从当前高度向后的3600个块 */
	BlockCache *BlocksTrie
}

/* 本节点出块时，从交易池中取出交易进行打包，但并不从交易池中删除 */
func (pool *TxPool) Get(size int) []*types.Transaction {
	return pool.TxCache.GetBatch(size)
}

/* 本节点出块时，执行交易后，发现错误的交易通过该接口进行删除 */
func (pool *TxPool) DelErrTxs(txs []*types.Transaction) {
	if len(txs) <= 0 {
		return
	} else {
		pool.TxCache.DelBatchByTx(txs)
	}
}

func (pool *TxPool) isInBlocks(hashs []common.Hash, blocks []*BlockNode) bool {
	for index := 0; index < len(hashs); index++ {
		hash := hashs[index]
		for _, v := range blocks {
			_, ok := v.TxsIndex[hash]
			if !ok {
				continue
			} else {
				return false
			}
		}
	}

	return true
}

/* 新收一个块时，验证块中的交易是否被同一条分叉上的其他块打包了 */
func (pool *TxPool) BlockIsValid(block *types.Block) bool {
	if block == nil {
		return false
	}
	if len(block.Txs) <= 0 {
		return true
	}

	minHeight, maxHeight, hashs := pool.TxRecently.GetPath(block.Txs)
	nodes := pool.BlockCache.Path(block.Hash(), block.Height(), uint32(minHeight), uint32(maxHeight))
	return pool.isInBlocks(hashs, nodes)
}

/* 新收到一个通过验证的新块（包括本节点出的块），需要从交易池中删除该块中已打包的交易 */
func (pool *TxPool) RecvBlock(block *types.Block) {
	if block == nil || len(block.Txs) <= 0 {
		return
	}

	pool.TxCache.DelBatchByTx(block.Txs)
	pool.TxRecently.RecvBlock(int64(block.Height()), block.Txs)
	result := pool.BlockCache.PushBlock(block)
	pool.TxRecently.DelBatch(map2slice(result))
}

/* 收到一笔新的交易 */
func (pool *TxPool) RecvTx(tx *types.Transaction) {
	if tx == nil {
		return
	}

	pool.TxRecently.RecvTx(tx)
	result := pool.TxCache.add(tx)
	pool.TxRecently.DelBatch(result)
}

/* 对链进行剪枝，剪下的块中的交易需要回归交易池 */
func (pool *TxPool) PruneBlock(block *types.Block) {
	if block == nil || len(block.Txs) <= 0 {
		return
	}

	result := pool.TxCache.AddBatch(block.Txs)
	pool.TxRecently.DelBatch(result)
	pool.BlockCache.DelBlock(block)
}
