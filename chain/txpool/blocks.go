package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/math"
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
	// 此区块时间小于LastMaxTime
	if block.Time() < timer.LastMaxTime {
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
	// block时间是否小于LastMaxTime
	if block.Time() < timer.LastMaxTime {
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

func (guard *TxGuard) IsTxExist(startBlockHash common.Hash, startBlockHeight uint32, tx *types.Transaction) bool {
	// 1. 查找交易是否在Traces中存在
	if _, ok := guard.Traces[tx.Hash()]; !ok {
		return false
	}
	// 2. 存在则取出那些区块打包了此交易
	trace := guard.Traces[tx.Hash()]
	// 3. 找出trace中的mineHeight和maxHeight
	minHeight := uint32(math.MaxUint32)
	maxHeight := uint32(0)
	for _, height := range trace {
		if height < minHeight {
			minHeight = height
		}
		if height > maxHeight {
			maxHeight = height
		}
	}
	// 4. 从startHeight开始逐个往前遍历当前分支的区块是否在trace中
	startBlock := guard.HeightBuckets[startBlockHeight][startBlockHash]
	pHash := startBlock.Header.Hash()
	pHeight := startBlock.Header.Height
	pBlock := startBlock
	for pHeight >= minHeight {
		if pHeight <= maxHeight {
			// 判断是否在trace中
			if _, ok := trace[pHash]; ok {
				return true
			}
		}
		// 前移动一个block
		pHash = pBlock.Header.ParentHash
		pHeight = pHeight - 1
		pBlock = guard.HeightBuckets[pHeight][pHash]
	}
	return false
}

func (guard *TxGuard) IsTxsExist(block *types.Block) bool {
	// 1. 获取block中的交易存在的所有区块
	traces := make(Trace)
	for _, tx := range block.Txs {
		if trace, ok := guard.Traces[tx.Hash()]; ok {
			for bHash, height := range trace {
				traces[bHash] = height
			}
		}
	}
	// 不存在
	if len(traces) == 0 {
		return false
	}
	// 2. 找出traces中的最大高度和最小高度
	minHeight := uint32(math.MaxUint32)
	maxHeight := uint32(0)
	for _, height := range traces {
		if height < minHeight {
			minHeight = height
		}
		if height > maxHeight {
			maxHeight = height
		}
	}
	// 3. 从block的父块开始，往前查找block所在的分支上的区块，直到找到高度为minHeight为止，并判断本分支高度在minHeight--maxHeight的区块是否在traces中
	pHash := block.ParentHash()
	pHeight := block.Height() - 1
	pBlock := guard.HeightBuckets[pHeight][pHash]
	for pHeight >= minHeight {
		if pHeight <= maxHeight {
			// 判断是否在traces中
			if _, ok := traces[pHash]; ok {
				return true
			}
		}
		// 前移动一个block
		pHash = pBlock.Header.ParentHash
		pHeight = pHeight - 1
		pBlock = guard.HeightBuckets[pHeight][pHash]
	}
	return false
}

// GetTxsByBranch 根据两个区块的叶子节点，获取它们到共同父节点之间的两个分支上的交易列表
func (guard *TxGuard) GetTxsByBranch(block01, block02 *types.Block) (txs1, txs2 []common.Hash) {
	// 1. 设置block01的高度要小于block02的高度
	if block01.Height() > block02.Height() {
		temp := block01
		block01 = block02
		block02 = temp
	}
	// 2. 先收集block02分支高度差多出的区块的交易
	bHash := block02.Hash()
	for i := block02.Height(); i > block01.Height(); i-- {
		// 找到对应的区块
		block := guard.HeightBuckets[i][bHash]
		for tHash := range block.TxHashes {
			txs2 = append(txs2, tHash)
		}
		// 设置下一个区块的hash
		bHash = block.Header.ParentHash
	}
	// 3. 此时block02分支高于block01分支的区块的交易已经收集完成，并此时bHash代表的block02分支的区块高度正好等于block01区块的高度
	pBlock01 := guard.HeightBuckets[block01.Height()][block01.Hash()]
	pBlock02 := guard.HeightBuckets[block01.Height()][bHash]
	for j := block01.Height(); j > 0; j-- {
		// 表示找到了共同的父块
		if pBlock01 == pBlock02 {
			break
		}
		// 收集交易
		for tHash := range pBlock01.TxHashes {
			txs1 = append(txs1, tHash)
		}
		for tHash := range pBlock02.TxHashes {
			txs2 = append(txs2, tHash)
		}
		// 修改pBlock01，pBlock01为前一个高度的区块
		pBlock01 = guard.HeightBuckets[pBlock01.Header.Height-1][pBlock01.Header.ParentHash]
		pBlock02 = guard.HeightBuckets[pBlock02.Header.Height-1][pBlock02.Header.ParentHash]
	}

	return txs1, txs2
}
