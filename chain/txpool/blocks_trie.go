package txpool

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
)

type BlockTimeBucket struct {
	/* 块时间 */
	Time uint32

	/* 块列表，以高度索引块列表 */
	BlocksByHeight map[uint32]HashSet
}

func newBlockTimeBucket(block *types.Block) *BlockTimeBucket {
	blockBucket := &BlockTimeBucket{
		Time:           block.Time(),
		BlocksByHeight: make(map[uint32]HashSet),
	}

	blockBucket.add(block)
	return blockBucket
}

func (blockBucket *BlockTimeBucket) add(block *types.Block) {
	if blockBucket.Time != block.Time() {
		log.Errorf("add block to time queue.err: time(%d) != block.time(%d)", blockBucket.Time, block.Time())
		return
	}

	height := block.Height()
	_, ok := blockBucket.BlocksByHeight[height]
	if !ok {
		blockBucket.BlocksByHeight[height] = make(HashSet)
	}

	blockBucket.BlocksByHeight[height].Add(block.Hash())
}

func (blockBucket *BlockTimeBucket) del(block *types.Block) {
	if blockBucket.Time != block.Time() {
		log.Errorf("add block to time queue.err: time(%d) != block.time(%d)", blockBucket.Time, block.Time())
		return
	}

	if len(blockBucket.BlocksByHeight) <= 0 {
		return
	}

	blockSet, ok := blockBucket.BlocksByHeight[block.Height()]
	if ok {
		blockSet.Del(block.Hash())
	}
}

func (blockBucket *BlockTimeBucket) timeOut(block *types.Block) bool {
	if blockBucket.Time < block.Time() {
		return true
	} else {
		return false
	}
}

func (blockBucket *BlockTimeBucket) notTimeOut(block *types.Block) bool {
	if blockBucket.Time == block.Time() {
		return true
	} else {
		return false
	}
}

func (blockBucket *BlockTimeBucket) before1H(block *types.Block) bool {
	if block.Time() < blockBucket.Time {
		return true
	} else {
		return false
	}
}

type NodeByHash map[common.Hash]*TrieNode

func newNodeByHash(block *types.Block) NodeByHash {
	nodeByHash := make(map[common.Hash]*TrieNode)

	nodeByHash[block.Hash()] = buildBlockNode(block)
	return nodeByHash
}

func buildTxSet(txs []*types.Transaction) HashSet {
	txSet := make(HashSet)
	if len(txs) <= 0 {
		return txSet
	}

	for index := 0; index < len(txs); index++ {
		txSet.Add(txs[index].Hash())
	}

	return txSet
}

func buildBlockNode(block *types.Block) *TrieNode {
	if block == nil {
		return nil
	}

	return &TrieNode{
		Header:    block.Header,
		TxHashSet: buildTxSet(block.Txs),
	}
}

func (nodeByHash NodeByHash) add(block *types.Block) {
	hash := block.Hash()
	_, ok := nodeByHash[hash]
	if !ok {
		nodeByHash[hash] = buildBlockNode(block)
	}
}

func (nodeByHash NodeByHash) del(hash common.Hash) {
	delete(nodeByHash, hash)
}

func (nodeByHash NodeByHash) delBatch(delBlocks HashSet) {
	if len(nodeByHash) <= 0 {
		return
	}
	if len(delBlocks) <= 0 {
		return
	}

	for k, _ := range delBlocks {
		delete(nodeByHash, k)
	}
}

func (nodeByHash NodeByHash) get(hash common.Hash) *TrieNode {
	node, ok := nodeByHash[hash]
	if !ok {
		return nil
	} else {
		return node
	}
}

type TrieNode struct {
	Header *types.Header

	/* 该块打包的交易列表的索引 */
	TxHashSet HashSet
}

func (node *TrieNode) hashIsExist(hash common.Hash) bool {
	if len(node.TxHashSet) <= 0 {
		return false
	}

	_, ok := node.TxHashSet[hash]
	return ok
}

/* 最近一个小时的所有块 */
type BlocksTrie struct {

	/* 根据高度对块进行索引 */
	HeightBuckets map[uint32]NodeByHash

	/* 根据时间刻度对块Hash进行索引，用来回收块 */
	TimeBuckets []*BlockTimeBucket
}

func NewBlocksTrie() *BlocksTrie {
	return &BlocksTrie{
		HeightBuckets: make(map[uint32]NodeByHash),
		TimeBuckets:   make([]*BlockTimeBucket, TransactionExpiration),
	}
}

func (trie *BlocksTrie) delFromHeightBucketBatch(delBlocks map[uint32]HashSet) {
	if len(delBlocks) <= 0 || len(trie.HeightBuckets) <= 0 {
		return
	}

	for height, hashSet := range delBlocks {
		blocks := trie.HeightBuckets[height]
		if blocks == nil {
			continue
		} else {
			blocks.delBatch(hashSet)
		}
	}
}

func (trie *BlocksTrie) DelBlock(block *types.Block) {
	if block == nil {
		return
	}
	trie.delFromHeightBucket(block)
	trie.delFromTimeBucket(block)
}

func (trie *BlocksTrie) addToHeightBucket(block *types.Block) {
	_, ok := trie.HeightBuckets[block.Height()]
	if !ok {
		trie.HeightBuckets[block.Height()] = newNodeByHash(block)
	} else {
		trie.HeightBuckets[block.Height()].add(block)
	}
}

func (trie *BlocksTrie) delFromHeightBucket(block *types.Block) {
	height := block.Height()
	blocks := trie.HeightBuckets[height]
	if blocks == nil {
		return
	} else {
		blocks.del(block.Hash())
	}
}

func (trie *BlocksTrie) addToTimeBucket(block *types.Block) {
	slot := block.Time() % uint32(TransactionExpiration)
	if trie.TimeBuckets[slot] == nil {
		trie.TimeBuckets[slot] = newBlockTimeBucket(block)
	} else {
		trie.TimeBuckets[slot].add(block)
	}
}

func (trie *BlocksTrie) delFromTimeBucket(block *types.Block) {
	slot := block.Time() % uint32(TransactionExpiration)
	bucket := trie.TimeBuckets[slot]
	if bucket == nil {
		return
	}
	bucket.del(block)
}

func (trie *BlocksTrie) resetTimeBucket(block *types.Block) {
	slot := block.Time() % uint32(TransactionExpiration)
	trie.TimeBuckets[slot] = newBlockTimeBucket(block)
}

/* 从指定块开始，收集该块所在链指定高度区间的块[minHeight, maxHeight] */
func (trie *BlocksTrie) Path(hash common.Hash, height uint32, minHeight uint32, maxHeight uint32) []*TrieNode {
	if hash == (common.Hash{}) || (minHeight > maxHeight) {
		return make([]*TrieNode, 0)
	}

	result := make([]*TrieNode, 0, maxHeight-minHeight+1)

	pHash := hash
	pHeight := height
	for pHeight >= minHeight {
		nodes := trie.HeightBuckets[pHeight]
		node := nodes.get(pHash)
		if node == nil {
			panic(fmt.Sprintf("get block is nil. hash: %#x", hash.Bytes()))
		}

		if pHeight <= maxHeight {
			result = append(result, node)
		}

		pHeight = node.Header.Height - 1
		pHash = node.Header.ParentHash
	}

	return result
}

/* 收到一个新块，并返回过期的块的交易列表，块过期了，块中的交易肯定也过期了 */
func (trie *BlocksTrie) PushBlock(block *types.Block) {
	if block == nil {
		return
	}

	slot := block.Time() % uint32(TransactionExpiration)
	timeBucket := trie.TimeBuckets[slot]
	if timeBucket == nil {
		trie.resetTimeBucket(block)
		trie.addToHeightBucket(block)
	} else {
		if timeBucket.timeOut(block) {
			trie.delFromHeightBucketBatch(timeBucket.BlocksByHeight)
			trie.resetTimeBucket(block)
			trie.addToHeightBucket(block)
		}

		if timeBucket.notTimeOut(block) {
			trie.addToHeightBucket(block)
			trie.addToTimeBucket(block)
		}

		if timeBucket.before1H(block) {
			log.Errorf(fmt.Sprintf("item.Time(%d) > block.Time(%d)", timeBucket.Time, block.Time()))
		}
	}
}
