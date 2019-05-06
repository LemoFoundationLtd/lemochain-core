package txpool

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
)

type BlocksByTime struct {
	/* 块时间 */
	Time uint32

	/* 块列表，以高度索引块列表 */
	BlocksByHeight map[uint32]map[common.Hash]bool
}

func newBlocksByTime(block *types.Block) *BlocksByTime {
	blocksByTime := &BlocksByTime{
		Time:           block.Time(),
		BlocksByHeight: make(map[uint32]map[common.Hash]bool),
	}

	blocksByTime.add(block)
	return blocksByTime
}

func (blocks *BlocksByTime) add(block *types.Block) {
	if blocks.Time != block.Time() {
		log.Errorf("add block to time queue.err: time(%d) != block.time(%d)", blocks.Time, block.Time())
		return
	}

	height := block.Height()
	_, ok := blocks.BlocksByHeight[height]
	if !ok {
		blocks.BlocksByHeight[height] = make(map[common.Hash]bool)
	}

	blocks.BlocksByHeight[height][block.Hash()] = true
}

func (blocks *BlocksByTime) del(block *types.Block) {
	if blocks.Time != block.Time() {
		log.Errorf("add block to time queue.err: time(%d) != block.time(%d)", blocks.Time, block.Time())
		return
	}

	if len(blocks.BlocksByHeight) <= 0 {
		return
	}

	hashes := blocks.BlocksByHeight[block.Height()]
	if len(hashes) <= 0 {
		return
	} else {
		delete(hashes, block.Hash())
	}
}

func (blocks *BlocksByTime) timeOut(block *types.Block) bool {
	if blocks.Time < block.Time() {
		return true
	} else {
		return false
	}
}

func (blocks *BlocksByTime) notTimeOut(block *types.Block) bool {
	if blocks.Time == block.Time() {
		return true
	} else {
		return false
	}
}

func (blocks *BlocksByTime) before1H(block *types.Block) bool {
	if block.Time() < blocks.Time {
		return true
	} else {
		return false
	}
}

type BlocksByHash struct {
	BlocksByHash map[common.Hash]*TrieNode
}

func newBlocksByHash(block *types.Block) *BlocksByHash {
	blocks := &BlocksByHash{
		BlocksByHash: make(map[common.Hash]*TrieNode),
	}

	blocks.BlocksByHash[block.Hash()] = blocks.buildBlockNode(block)
	return blocks
}

func (blocks *BlocksByHash) buildTxsIndex(txs []*types.Transaction) map[common.Hash]bool {
	txsIndex := make(map[common.Hash]bool)
	if len(txs) <= 0 {
		return txsIndex
	}

	for index := 0; index < len(txs); index++ {
		txsIndex[txs[index].Hash()] = true
	}

	return txsIndex
}

func (blocks *BlocksByHash) buildBlockNode(block *types.Block) *TrieNode {
	if block == nil {
		return nil
	}

	return &TrieNode{
		Header:   block.Header,
		TxsIndex: blocks.buildTxsIndex(block.Txs),
	}
}

func (blocks *BlocksByHash) add(block *types.Block) {
	hash := block.Hash()
	_, ok := blocks.BlocksByHash[hash]
	if !ok {
		blocks.BlocksByHash[hash] = blocks.buildBlockNode(block)
	}
}

func (blocks *BlocksByHash) del(hash common.Hash) {
	delete(blocks.BlocksByHash, hash)
}

func (blocks *BlocksByHash) delBatch(delBlocks map[common.Hash]bool) {
	if len(blocks.BlocksByHash) <= 0 {
		return
	}
	if len(delBlocks) <= 0 {
		return
	}

	for k, _ := range delBlocks {
		delete(blocks.BlocksByHash, k)
	}
}

func (blocks *BlocksByHash) get(hash common.Hash) *TrieNode {
	node, ok := blocks.BlocksByHash[hash]
	if !ok {
		return nil
	} else {
		return node
	}
}

type TrieNode struct {
	Header *types.Header

	/* 该块打包的交易列表的索引 */
	TxsIndex map[common.Hash]bool
}

func (node *TrieNode) hashIsExist(hash common.Hash) bool {
	if len(node.TxsIndex) <= 0 {
		return false
	}

	_, ok := node.TxsIndex[hash]
	return ok
}

/* 最近一个小时的所有块 */
type BlocksTrie struct {

	/* 根据高度对块进行索引 */
	BlocksByHash map[uint32]*BlocksByHash

	/* 根据时间刻度对块Hash进行索引，用来回收块 */
	BlocksByTime []*BlocksByTime
}

func NewBlocksTrie() *BlocksTrie {
	return &BlocksTrie{
		BlocksByHash: make(map[uint32]*BlocksByHash),
		BlocksByTime: make([]*BlocksByTime, TransactionExpiration),
	}
}

func (trie *BlocksTrie) delBlocksByHashBatch(delBlocks map[uint32]map[common.Hash]bool) {
	if len(delBlocks) <= 0 || len(trie.BlocksByHash) <= 0 {
		return
	}

	for height, hashes := range delBlocks {
		blocks := trie.BlocksByHash[height]
		if blocks == nil {
			continue
		} else {
			blocks.delBatch(hashes)
		}
	}
}

func (trie *BlocksTrie) DelBlock(block *types.Block) {
	if block == nil {
		return
	}
	trie.delBlockByHash(block)
	trie.delBlockByTime(block)
}

func (trie *BlocksTrie) addBlockByHash(block *types.Block) {
	_, ok := trie.BlocksByHash[block.Height()]
	if !ok {
		trie.BlocksByHash[block.Height()] = newBlocksByHash(block)
	} else {
		trie.BlocksByHash[block.Height()].add(block)
	}
}

func (trie *BlocksTrie) delBlockByHash(block *types.Block) {
	height := block.Height()
	blocks := trie.BlocksByHash[height]
	if blocks == nil {
		return
	} else {
		blocks.del(block.Hash())
	}
}

func (trie *BlocksTrie) addBlockByTime(block *types.Block) {
	slot := block.Time() % uint32(TransactionExpiration)
	if trie.BlocksByTime[slot] == nil {
		trie.BlocksByTime[slot] = newBlocksByTime(block)
	} else {
		trie.BlocksByTime[slot].add(block)
	}
}

func (trie *BlocksTrie) delBlockByTime(block *types.Block) {
	slot := block.Time() % uint32(TransactionExpiration)
	item := trie.BlocksByTime[slot]
	if item == nil {
		return
	}
	item.del(block)
}

func (trie *BlocksTrie) resetBlockByTime(block *types.Block) {
	slot := block.Time() % uint32(TransactionExpiration)
	trie.BlocksByTime[slot] = newBlocksByTime(block)
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
		blocks := trie.BlocksByHash[pHeight]
		block := blocks.get(pHash)
		if block == nil {
			panic(fmt.Sprintf("get block is nil.hash: %s", common.ToHex(hash.Bytes())))
		}

		if pHeight <= maxHeight {
			result = append(result, block)
		}

		pHeight = block.Header.Height - 1
		pHash = block.Header.ParentHash
	}

	return result
}

/* 收到一个新块，并返回过期的块的交易列表，块过期了，块中的交易肯定也过期了 */
func (trie *BlocksTrie) PushBlock(block *types.Block) {
	if block == nil {
		return
	}

	slot := block.Time() % uint32(TransactionExpiration)
	blocks := trie.BlocksByTime[slot]
	if blocks == nil {
		trie.resetBlockByTime(block)
		trie.addBlockByHash(block)
	} else {
		if blocks.timeOut(block) {
			trie.delBlocksByHashBatch(blocks.BlocksByHeight)
			trie.resetBlockByTime(block)
			trie.addBlockByHash(block)
		}

		if blocks.notTimeOut(block) {
			trie.addBlockByHash(block)
			trie.addBlockByTime(block)
		}

		if blocks.before1H(block) {
			log.Errorf(fmt.Sprintf("item.Time(%d) > block.Time(%d)", blocks.Time, block.Time()))
		}
	}
}
