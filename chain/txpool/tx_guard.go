package txpool

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/math"
)

var (
	ErrNotFoundBlock = errors.New("not found block in TxGuard'HeightBuckets")
	ErrBlockTime     = errors.New("block timestamp error")
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

// newBlocksInTime
func newBlocksInTime(lastMaxTime uint32) *BlocksInTime {
	return &BlocksInTime{
		LastMaxTime: lastMaxTime,
		BlockSet:    make([]BlocksByHash, 0),
	}
}

// SaveBlock
func (timer *BlocksInTime) SaveBlock(block *types.Block) error {
	// 此区块时间小于LastMaxTime
	if block.Time() < timer.LastMaxTime {
		// 不保存
		return ErrBlockTime
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
	return nil
}

// DelBlock
func (timer *BlocksInTime) DelBlock(block *types.Block) error {
	// block时间是否小于LastMaxTime
	if block.Time() < timer.LastMaxTime {
		return ErrBlockTime
	}
	differTime := block.Time() - timer.LastMaxTime
	index := differTime / 60

	// block时间超过保存的区块的最大时间
	if len(timer.BlockSet) < int(index+1) {
		return ErrBlockTime
	}
	// 存在则删除
	delete(timer.BlockSet[index], block.Hash())
	return nil
}

// DelOldBlocks
func (timer *BlocksInTime) DelOldBlocks(maxTime uint32) []*Block {
	// 加60s是因为lastMaxTime变化的基本单位的1分钟
	if timer.LastMaxTime+60 > maxTime {
		return nil
	}
	differTime := maxTime - timer.LastMaxTime
	// 计算需要删除的index组数
	cnt := int(differTime / 60)
	// 如果需要删除的组数量大于总的数量，说明需要删除所有缓存中的block
	if cnt > len(timer.BlockSet) {
		cnt = len(timer.BlockSet)
	}
	blocksByHashes := make([]BlocksByHash, 0, cnt)
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
	// 修改LastMaxTime
	if cnt == len(timer.BlockSet) {
		// 这里表示要删除所有的缓存block，所以可以从新的时间开始每隔60s进行分割时间片段
		timer.LastMaxTime = maxTime
	} else {
		timer.LastMaxTime = timer.LastMaxTime + uint32(cnt*60)
	}
	// 移动数组BlockSet
	timer.BlockSet = timer.BlockSet[cnt:]

	return blocks
}

/** 根据区块Hash对应区块高度， */
type Trace map[common.Hash]uint32

// getMaxAndMinHeight
func (t Trace) getMaxAndMinHeight() (minHeight, maxHeight uint32) {
	if len(t) == 0 {
		return
	}
	minHeight = uint32(math.MaxUint32)
	maxHeight = uint32(0)
	for _, height := range t {
		if height < minHeight {
			minHeight = height
		}
		if height > maxHeight {
			maxHeight = height
		}
	}
	return
}

type TxGuard struct {
	BlocksInTime *BlocksInTime

	/** 链存在分支，所以一个高度可能存在多个区块 */
	HeightBuckets map[uint32]BlocksByHash

	/** 所有交易，由于链存在分支，所以一个交易可能存在于多个区块中。 */
	Traces map[common.Hash]Trace
}

func NewTxGuard(lastMaxTime uint32) *TxGuard {
	return &TxGuard{
		BlocksInTime:  newBlocksInTime(lastMaxTime),
		HeightBuckets: make(map[uint32]BlocksByHash),
		Traces:        make(map[common.Hash]Trace),
	}
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
			// 交易hash对应的区块删除完了则删除交易hash的索引
			if len(guard.Traces[txHash]) == 0 {
				delete(guard.Traces, txHash)
			}
		}
		// 3. 删除HeightBuckets中的block
		delete(guard.HeightBuckets[b.Header.Height], b.Header.Hash())
		// 高度对应的区块都被删除完了之后则可以删除高度索引
		if len(guard.HeightBuckets[b.Header.Height]) == 0 {
			delete(guard.HeightBuckets, b.Header.Height)
		}
	}
}

func (guard *TxGuard) SaveBlock(block *types.Block) {
	// 1. 存入区块到BlocksInTime中
	if err := guard.BlocksInTime.SaveBlock(block); err != nil {
		log.Errorf("save block error for TxGuard, error: %v", err)
		return
	}
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

func (guard *TxGuard) DelBlock(block *types.Block) error {
	return guard.BlocksInTime.DelBlock(block)
}

// IsTxExist 判断tx是否已经在当前分支存在，startBlockHash和startBlockHeight为指定分支的子节点的区块hash和height
func (guard *TxGuard) IsTxExist(startBlockHash common.Hash, startBlockHeight uint32, tx *types.Transaction) (bool, error) {
	// 1. 查找交易是否在Traces中存在
	if _, ok := guard.Traces[tx.Hash()]; !ok {
		return false, nil
	}
	// 2. 存在则取出那些区块打包了此交易
	trace := guard.Traces[tx.Hash()]
	// 3. 找出trace中的mineHeight和maxHeight
	minHeight, maxHeight := trace.getMaxAndMinHeight()
	// 4. 从startHeight开始逐个往前遍历当前分支的区块是否在trace中
	return guard.existBlock(minHeight, maxHeight, trace, startBlockHash, startBlockHeight)
}

// IsTxsExist 判断传入block中的交易是否已经被block所在分支的其他区块打包了
func (guard *TxGuard) IsTxsExist(block *types.Block) (bool, error) {
	// 1. 获取block中的交易存在的所有区块
	trace := make(Trace)
	for _, tx := range block.Txs {
		if t, ok := guard.Traces[tx.Hash()]; ok {
			for bHash, height := range t {
				trace[bHash] = height
			}
		}
	}
	// 不存在
	if len(trace) == 0 {
		return false, nil
	}
	// 2. 找出trace中的最大高度和最小高度
	minHeight, maxHeight := trace.getMaxAndMinHeight()
	// 3. 从block的父块开始，往前查找block所在的分支上的区块，直到找到高度为minHeight为止，并判断本分支高度在minHeight--maxHeight的区块是否在trace中
	return guard.existBlock(minHeight, maxHeight, trace, block.ParentHash(), block.Height()-1)

}

// existBlock 判断给定高度区间内指定分支上的区块是否在区块集合trace中
func (guard *TxGuard) existBlock(minHeight, maxHeight uint32, trace Trace, startBlockHash common.Hash, startBlockHeight uint32) (bool, error) {
	// 判断传入的startBlock是否在HeightBuckets中
	if blocks, ok := guard.HeightBuckets[startBlockHeight]; !ok {
		log.Errorf("Not found block in TxGuard by startBlockHeight. startBlockHeight: %d", startBlockHeight)
		return false, ErrNotFoundBlock
	} else if _, exist := blocks[startBlockHash]; !exist {
		log.Errorf("Not found block in TxGuard by startBlockHash. startBlockHash: %s", startBlockHash.String())
		return false, ErrNotFoundBlock
	}
	pHash := startBlockHash
	pHeight := startBlockHeight
	var pBlock *Block
	for pHeight >= minHeight {
		if pHeight <= maxHeight {
			// 判断是否在trace中
			if _, ok := trace[pHash]; ok {
				return true, nil
			}
		}
		// 前移动一个block
		pBlock = guard.HeightBuckets[pHeight][pHash]
		pHash = pBlock.Header.ParentHash
		pHeight = pHeight - 1
	}
	return false, nil
}

// GetTxsByBranch 根据两个区块的叶子节点，获取它们到共同父节点之间的两个分支上的交易列表
func (guard *TxGuard) GetTxsByBranch(block01, block02 *types.Block) (txHashes1, txHashes2 []common.Hash, err error) {
	// 0. 判断block01和 block02是否存在
	if blocks, ok := guard.HeightBuckets[block01.Height()]; !ok {
		log.Errorf("Not found block in TxGuard by blockHeight. blockHeight: %d", block01.Height())
		return nil, nil, ErrNotFoundBlock
	} else if _, exist := blocks[block01.Hash()]; !exist {
		log.Errorf("Not found block in TxGuard by blockHash. blockHash: %s", block01.Hash().String())
		return nil, nil, ErrNotFoundBlock
	}

	if blocks, ok := guard.HeightBuckets[block02.Height()]; !ok {
		log.Errorf("Not found block in TxGuard by blockHeight. blockHeight: %d", block02.Height())
		return nil, nil, ErrNotFoundBlock
	} else if _, exist := blocks[block02.Hash()]; !exist {
		log.Errorf("Not found block in TxGuard by blockHash. blockHash: %s", block02.Hash().String())
		return nil, nil, ErrNotFoundBlock
	}

	// 标记block01和block02是否交换了
	swap := false
	// 1. 设置block01的高度要小于block02的高度
	if block01.Height() > block02.Height() {
		temp := block01
		block01 = block02
		block02 = temp
		swap = true // 标记交换为true
	}
	// 2. 先收集block02分支高度差多出的区块的交易
	bHash := block02.Hash()
	for i := block02.Height(); i > block01.Height(); i-- {
		// 找到对应的区块
		block := guard.HeightBuckets[i][bHash]
		for tHash := range block.TxHashes {
			txHashes2 = append(txHashes2, tHash)
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
			txHashes1 = append(txHashes1, tHash)
		}
		for tHash := range pBlock02.TxHashes {
			txHashes2 = append(txHashes2, tHash)
		}
		// 修改pBlock01，pBlock01为前一个高度的区块
		pBlock01 = guard.HeightBuckets[pBlock01.Header.Height-1][pBlock01.Header.ParentHash]
		pBlock02 = guard.HeightBuckets[pBlock02.Header.Height-1][pBlock02.Header.ParentHash]
	}

	// 如果block01和block02交换了，最后输出的值要交换回来，实现输入值和输出值一一对应的效果
	if swap {
		return txHashes2, txHashes1, nil
	} else {
		return txHashes1, txHashes2, nil
	}
}
