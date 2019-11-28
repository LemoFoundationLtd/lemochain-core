package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
)

/** 一个区块包含的交易Hash */
type Block struct {
	/** 区块头 */
	Header *types.Header

	/** 该区块包含的交易Hash，采用map的目的是：可以快速查找该交易是否存在于该区块 */
	TxHashes map[common.Hash]struct{}
}

/** 根据区块Hash索引区块 */
type BlocksByHash map[common.Hash]*Block

/** 以时间维度组织区块，用来决定缓存中留多少数据。原理：以UTC时间，按每分钟产生的区块组织数据 */
type BlocksInTime struct {
	/** 上次删除缓存的时间 */
	LastMaxTime uint32 // 此时间戳是随着链上的stable变化而变化的

	/** 从上次删除时间开始，截止到目前的区块。
	 * (1) 任何一个区块所在的数组下标为： 区块时间(取分钟)- LastMaxTime(取分钟)
	 * (2) 删除指定maxTime的区块流程为：把Blocks向前移动maxTime(取分钟) - LastMaxTime(取分钟)即可
	 * (3) 重新赋值LastMaxTime为maxTime
	 */
	BlockSet []BlocksByHash
}

// SaveBlock
func (timer *BlocksInTime) SaveBlock(block *types.Block) {
	// 此区块时间小于LastMaxTime或者此区块中没有交易则不保存此block
	if block.Time() < timer.LastMaxTime || len(block.Txs) == 0 {
		// 不保存
		return
	}
	// 保存区块
	differTime := block.Time() - timer.LastMaxTime
	// 获取对应的index
	index := differTime / 60

	// 判断BlockSet切片是否有此index，不然会报错index out of range
	if len(timer.BlockSet) < int(index+1) {
		extendSlice := make([]BlocksByHash, int(index+1)-len(timer.BlockSet))
		timer.BlockSet = append(timer.BlockSet, extendSlice...)
	}
	// 判断此index下的map是否存在，不存在要make一个新的map
	if timer.BlockSet[index] == nil {
		timer.BlockSet[index] = make(BlocksByHash)
	}
	// 获取block中所有的交易hash
	txHashes := make(map[common.Hash]struct{})
	for _, tx := range block.Txs {
		txHashes[tx.Hash()] = struct{}{}
	}
	timer.BlockSet[index][block.Hash()] = &Block{
		Header:   block.Header,
		TxHashes: txHashes,
	}
}

// DelBlock
func (timer *BlocksInTime) DelBlock(block *types.Block) {
	// block时间是否小于LastMaxTime或者block中的交易数量为0则不处理
	if block.Time() < timer.LastMaxTime || len(block.Txs) == 0 {
		return
	}
	differTime := block.Time() - timer.LastMaxTime
	index := differTime / 60

	// 不能删除一个未保存的block
	if len(timer.BlockSet) < int(index+1) {
		return
	}
	// 存在则删除
	delete(timer.BlockSet[index], block.Hash())
}

// DelOldBlocks
func (timer *BlocksInTime) DelOldBlocks(maxTime uint32) []*Block {

	if timer.LastMaxTime > maxTime-60 {
		return nil
	}
	differTime := maxTime - timer.LastMaxTime
	// 计算需要删除的index组数
	cnt := int(differTime / 60)
	// 如果需要删除的组数量大于总的数量，说明maxTime为未来的时间了，不合法
	if cnt > len(timer.BlockSet) {
		log.Errorf("maxTime incorrect. maxTime: %d", maxTime)
		return nil
	}
	blocksByHashes := make([]BlocksByHash, 0, cnt-1)
	// 获取需要删除的block的index
	for i := 0; i < cnt; i++ {
		blocksByHashes = append(blocksByHashes, timer.BlockSet[i])
	}
	if len(blocksByHashes) == 0 {
		return nil
	}
	// 取出blocksByHashes中所有的block
	blocks := make([]*Block, 0)
	for _, blocksMap := range blocksByHashes {
		for _, block := range blocksMap {
			blocks = append(blocks, block)
		}
	}
	// 修改LastMaxTime并移动数组BlockSet
	timer.LastMaxTime = timer.LastMaxTime + uint32(cnt*60)
	timer.BlockSet = timer.BlockSet[cnt:]

	return blocks
}

/** 根据区块Hash对应区块高度， */
type Trace map[common.Hash]uint32

type TxGuard struct {
	BlocksInTime *BlocksInTime

	/** 链存在分支，所以一个高度可能存在多个区块 */
	HeightBuckets map[uint32]BlocksByHash

	/** 所有交易，由于链存在分支，所以一个交易可能存在于多个区块中。 */
	Traces map[common.Hash]Trace
}

func (guard *TxGuard) DelOldBlocks(maxTime uint32) {
	/**
	 * (1) 调用BlocksInTime.DelOldBlocks删除区块，并得到已删除的区块
	 * (2) 遍历已删除区块中的所有交易：根据交易Hash删除Traces中的交易
	 * (3) 遍历已删除区块：根据区块高度和区块Hash删除HeightBuckets中的区块
	 */
	// 1. 删除BlocksInTime区块
	blocks := guard.BlocksInTime.DelOldBlocks(maxTime)
	if blocks == nil {
		return
	}

	for _, b := range blocks {
		// 2. 删除区块中的交易在Traces中的记录
		for txHash := range b.TxHashes {
			delete(guard.Traces[txHash], b.Header.Hash())
		}
		// 3. 删除HeightBuckets中的block
		delete(guard.HeightBuckets[b.Header.Height], b.Header.Hash())
	}
}

func (guard *TxGuard) SaveBlock(block *types.Block) {
	// 1. 存入区块到BlocksInTime中
	guard.BlocksInTime.SaveBlock(block)
	// 2. 保存Traces
	for _, tx := range block.Txs {
		if trace, ok := guard.Traces[tx.Hash()]; ok {
			// 已经存在过此交易的记录,则新增记录
			trace[block.Hash()] = block.Height()
		} else {
			// 首次出现此交易,则初始化map并赋值
			trace := make(Trace)
			trace[block.Hash()] = block.Height()
			guard.Traces[tx.Hash()] = trace
		}
	}
	// 3. 保存进HeightBuckets
	// 获取block中所有的交易hash
	txHashes := make(map[common.Hash]struct{})
	for _, tx := range block.Txs {
		txHashes[tx.Hash()] = struct{}{}
	}
	newBlock := &Block{
		Header:   block.Header,
		TxHashes: txHashes,
	}
	if blocksByHash, ok := guard.HeightBuckets[block.Height()]; ok {
		// 已经存在此高度的记录
		blocksByHash[block.Hash()] = newBlock
	} else {
		// 第一次出现此记录
		newBlocksByHash := make(BlocksByHash)
		newBlocksByHash[block.Hash()] = newBlock
		guard.HeightBuckets[block.Height()] = newBlocksByHash
	}
}

func (guard *TxGuard) IsTxExist(startBlockHash common.Hash, tx *types.Transaction) {
	// TODO
}

func (guard *TxGuard) IsTxsExist(block *types.Block) {
	/**
	 * (1) 根据block中的交易，从Traces获取这些交易所在的区块
	 * (2) 根据这些区块，计算这些区块的最大高度和最小高度
	 * (3) 从HeightBuckets中，从最小高度和最小高度的区块开始，截止最大高度，找出所有的路径区块
	 * (4) 判断所有的路径区块，是否在这些交易的区块中，如果存在，则存在；否则不存在
	 */
}
