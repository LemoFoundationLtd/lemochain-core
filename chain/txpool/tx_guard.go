package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"sync"
)

// TxGuard is used test if a transaction or block is contained in some fork
// It contains the blocks half hour before stable block, and contains unstable blocks
type TxGuard struct {
	blockBuckets *TimeBuckets // group blocks' indexes by time
	blockCache   BlockCache   // save all blocks' detail
	txTracer     TxTracer     // used to find blocks which the transaction appeared

	RW sync.RWMutex
}

func NewTxGuard(stableBlockTime uint32) *TxGuard {
	timeBase := uint32(0)
	if stableBlockTime > uint32(params.MaxTxLifeTime) {
		timeBase = stableBlockTime - uint32(params.MaxTxLifeTime)
	}
	return &TxGuard{
		// base time is 30 minutes ago before the last stable block
		blockBuckets: newTimeBucket(timeBase),
		blockCache:   make(BlockCache),
		txTracer:     make(TxTracer),
	}
}

// SaveBlock save the block and record its transactions' appearance
func (guard *TxGuard) SaveBlock(block *types.Block) {
	if block == nil {
		return
	}

	guard.RW.Lock()
	defer guard.RW.Unlock()

	blockHash := block.Hash()
	// save block index in time bucket
	if err := guard.blockBuckets.Add(block.Time(), blockHash); err != nil {
		log.Errorf("save block error for TxGuard, error: %v", err)
		return
	}

	// save block in cache
	guard.blockCache.Add(block)

	// save transactions' trace
	for _, tx := range block.Txs {
		guard.txTracer.AddTrace(tx, blockHash)
	}
}

// ExistTx 判断tx是否已经在当前分支存在，startBlockHash为指定分支的子节点的区块hash
func (guard *TxGuard) ExistTx(startBlockHash common.Hash, tx *types.Transaction) bool {
	return guard.ExistTxs(startBlockHash, types.Transactions{tx})
}

// ExistTxs 判断txs中是否有交易已经在当前分支存在，startBlockHash为指定分支的子节点的区块hash
func (guard *TxGuard) ExistTxs(startBlockHash common.Hash, txs types.Transactions) bool {
	guard.RW.Lock()
	defer guard.RW.Unlock()

	trace := guard.txTracer.LoadTraces(txs)
	return guard.blockCache.IsAppearedOnFork(trace, startBlockHash)
}

// DelOldBlocks 根据时间删去过期了的区块和交易
func (guard *TxGuard) DelOldBlocks(newStableBlockTime uint32) {
	if newStableBlockTime < uint32(params.MaxTxLifeTime) {
		log.Errorf("Invalid stable block time: %d", newStableBlockTime)
		panic(ErrInvalidBaseTime)
	}

	guard.RW.Lock()
	defer guard.RW.Unlock()

	// 1. 根据时间删除TimeBuckets，得到删除的区块hash
	blockHashes := guard.blockBuckets.Expire(newStableBlockTime - uint32(params.MaxTxLifeTime))
	if len(blockHashes) == 0 {
		return
	}
	log.Debugf("Expire %d blocks in txGuard", len(blockHashes))

	for _, hash := range blockHashes {
		// 2. 删除区块缓存
		block, ok := guard.blockCache[hash]
		if !ok {
			log.Errorf("DelOldBlocks error for TxGuard, no cache for block %s", hash.Prefix())
			continue
		}
		guard.blockCache.Del(hash)
		// 3. 删除区块中交易的记录。区块都过期了，其中的交易肯定也过期了
		for _, tx := range block.Txs {
			guard.txTracer.DelTrace(tx)
		}
	}
}

// GetTxsByBranch 根据两个区块的叶子节点，获取它们到共同父节点之间的两个分支上的区块列表，含这两个叶子节点
func (guard *TxGuard) getBlocksByBranch(block1, block2 *types.Block) (blocks1, blocks2 []*BlockNode, err error) {
	var (
		hash1    = block1.Hash()
		hash2    = block2.Hash()
		height1  = block1.Height()
		height2  = block2.Height()
		collect1 = func() {
			t := guard.blockCache.Get(hash1)
			if t == nil {
				log.Errorf("Not found block in TxGuard. block1Hash: %s", hash1.String())
				err = ErrNotFoundBlockCache
			} else {
				blocks1 = append(blocks1, t)
				height1--
				hash1 = t.Header.ParentHash
			}
		}
		collect2 = func() {
			t := guard.blockCache.Get(hash2)
			if t == nil {
				log.Errorf("Not found block in TxGuard. block2Hash: %s", hash2.String())
				err = ErrNotFoundBlockCache
			} else {
				blocks2 = append(blocks2, t)
				height2--
				hash2 = t.Header.ParentHash
			}
		}
	)

	for {
		if height1 > height2 {
			collect1()
		} else if height1 < height2 {
			collect2()
		} else {
			// height1 equals height2
			if hash1 == hash2 {
				// found the same parent
				return
			}
			if height1 == 0 {
				log.Errorf("Chain forks from genesis")
				return nil, nil, ErrDifferentGenesis
			}
			collect1()
			if err != nil {
				return nil, nil, err
			}
			collect2()
		}
		if err != nil {
			return nil, nil, err
		}
	}
}

// GetTxsByBranch 根据两个区块的叶子节点，获取它们到共同父节点之间的两个分支上的交易列表
func (guard *TxGuard) GetTxsByBranch(block1, block2 *types.Block) (txs1, txs2 types.Transactions, err error) {
	guard.RW.Lock()
	defer guard.RW.Unlock()

	blocks1, blocks2, err := guard.getBlocksByBranch(block1, block2)
	if err != nil {
		return nil, nil, err
	}

	for _, b := range blocks1 {
		txs1 = append(txs1, b.Txs...)
	}
	for _, b := range blocks2 {
		txs2 = append(txs2, b.Txs...)
	}
	return txs1, txs2, nil
}
