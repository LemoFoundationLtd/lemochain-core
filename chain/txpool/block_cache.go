package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"math"
)

// 简化的区块数据
type BlockNode struct {
	Header *types.Header

	/* 该块打包的交易列表 */
	Txs types.Transactions
}

func buildBlockNode(block *types.Block) *BlockNode {
	if block == nil {
		return nil
	}

	return &BlockNode{
		Header: block.Header,
		Txs:    block.Txs,
	}
}

type BlockNodes []*BlockNode

// getHeightRange
func (nodes BlockNodes) getHeightRange() (minHeight, maxHeight uint32) {
	if len(nodes) == 0 {
		return
	}
	minHeight = math.MaxUint32
	maxHeight = uint32(0)
	for _, block := range nodes {
		if block.Header.Height < minHeight {
			minHeight = block.Header.Height
		}
		if block.Header.Height > maxHeight {
			maxHeight = block.Header.Height
		}
	}
	return
}

// 根据区块Hash索引区块
type BlockCache map[common.Hash]*BlockNode

func (cache BlockCache) Add(block *types.Block) {
	hash := block.Hash()
	_, ok := cache[hash]
	if !ok {
		cache[hash] = buildBlockNode(block)
	}
}

func (cache BlockCache) Del(hash common.Hash) {
	delete(cache, hash)
}

func (cache BlockCache) Get(hash common.Hash) *BlockNode {
	node, ok := cache[hash]
	if !ok {
		return nil
	} else {
		return node
	}
}

// IsAppearedOnFork tests if the blocks are appeared on the fork from startBlock
func (cache BlockCache) IsAppearedOnFork(blockHashes HashSet, startBlockHash common.Hash) bool {
	if len(blockHashes) == 0 {
		return false
	}

	// 1. 找出trace中的最大高度和最小高度
	blocks, err := cache.CollectBlocks(blockHashes)
	if err != nil {
		panic(err)
	}
	minHeight, maxHeight := blocks.getHeightRange()
	// 2. 在当前分支上截取一段区块
	forkSlice, err := cache.SliceOnFork(startBlockHash, minHeight, maxHeight)
	if err != nil {
		// 在前面的流程中已经验证过区块的父块一定存在且不是叔块，因此再出现找不到的问题就是bug了
		panic(err)
	}
	// 3. 在这段区块中检测是否包含了trace中的块
	for _, hash := range forkSlice {
		if blockHashes.Has(hash) {
			log.Warnf("Some txs in block %s appear again", hash.String())
			return true
		}
	}
	return false
}

// SliceOnFork collect blocks' hashes from specific fork and specific height range [minHeight, maxHeight]
func (cache BlockCache) CollectBlocks(blockHashSet HashSet) (BlockNodes, error) {
	if len(blockHashSet) == 0 {
		return make(BlockNodes, 0), nil
	}

	hashes := blockHashSet.Collect()
	result := make(BlockNodes, len(hashes))
	for i, hash := range hashes {
		block := cache.Get(hash)
		if block == nil {
			log.Errorf("Not found block in CollectBlocks. hash: %s", hash.String())
			return nil, ErrNotFoundBlockCache
		}
		result[i] = block
	}
	return result, nil
}

// SliceOnFork collect blocks' hashes from specific fork and specific height range [minHeight, maxHeight].
// The result array is from leaf block hash to root block hash
func (cache BlockCache) SliceOnFork(startBlockHash common.Hash, minHeight uint32, maxHeight uint32) (blockHashes []common.Hash, err error) {
	if startBlockHash == (common.Hash{}) || (minHeight > maxHeight) {
		return
	}

	pBlock := cache.Get(startBlockHash)
	if pBlock == nil {
		log.Errorf("Not found block in SliceOnFork. startBlockHash: %s len(cache): %d", startBlockHash.String(), len(cache))
		err = ErrNotFoundBlockCache
		return
	}
	pHeight := pBlock.Header.Height
	if pHeight < minHeight {
		return
	}

	// block1--block2--block3[minHeight]--block4--block5[maxHeight]--block6[startBlock]
	for pHash := startBlockHash; pHeight >= minHeight; {
		if minHeight <= pHeight && pHeight <= maxHeight {
			blockHashes = append(blockHashes, pHash)
		}

		// move to parent
		pHash = pBlock.Header.ParentHash
		pBlock = cache.Get(pHash)
		if pBlock == nil {
			// nil is possible. Sometimes the trace range in another fork maybe height 100~105, but the height 100 block is deleted by expiration on the tested fork
			break
		}
		pHeight = pBlock.Header.Height
	}

	return
}
