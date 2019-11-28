package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

/** 一个区块包含的交易Hash */
type Block struct {
	/** 区块头 */
	Header *types.Header

	/** 该区块包含的交易Hash，采用map的目的是：可以快速查找该交易是否存在于该区块 */
	TxHashes map[common.Hash]struct{}
}

/** 根据区块Hash索引区块 */
type Blocks map[common.Hash]*Block

/** 以时间维度组织区块，用来决定缓存中留多少数据。原理：以UTC时间，按每分钟产生的区块组织数据 */
type BlocksInTime struct {
	/** 上次删除缓存的时间 */
	LastMaxTime int

	/** 从上次删除时间开始，截止到目前的区块。
	 * (1) 任何一个区块所在的数组下标为： 区块时间(取分钟)- LastMaxTime(取分钟)
	 * (2) 删除指定maxTime的区块流程为：把Blocks向前移动maxTime(取分钟) - LastMaxTime(取分钟)即可
	 * (3) 重新赋值LastMaxTime为maxTime
	 */
	Blocks []*Blocks
}

func (timer *BlocksInTime) SaveBlock(block *types.Block) {
	/**
	 * (1) 取区块块时间，并把该时间换算成分钟: CurTime = time / 60 ?
	 * (2) 如果CurTime < LastMaxTime，则丢弃？
	 * (3) 得到数组下标：index = CurTime - LastMaxTime
	 * (4) 把该区块放入到index指定下标，并设置LastMaxTime = CurTime
	 */
}

func (timer *BlocksInTime) DelBlock(block *types.Block) {
	/**
	 * (1) 取区块块时间，并把该时间换算成分钟: CurTime = time / 60 ?
	 * (2) 如果CurTime < LastMaxTime，则丢弃？
	 * (3) 得到数组下标：index = CurTime - LastMaxTime
	 * (4) 获取index的区块集合，根据区块hash删除该区块
	 */
}

func (timer *BlocksInTime) DelOldBlocks(maxTime int) *types.Block {
	/**
	 * (1) 取maxTime，并换算成分钟：CurTime = time / 60?
	 * (2) 如果CurTime < LastMaxTime，则返回
	 * (3) 得到删除多少个数组下标：cnt = CurTime - LastMaxTime
	 * (4) 采用 Blocks = append(Blocks[0:], Blocks[cnt:])删除
	 * (5) 返回删除的区块
	 */
	return nil
}

/** 根据区块Hash对应区块高度， */
type Trace map[common.Hash]int64

type TxGuard struct {
	BlocksInTime *BlocksInTime

	/** 链存在分支，所以一个高度可能存在多个区块 */
	HeightBuckets map[uint32]*Blocks

	/** 所有交易，由于链存在分支，所以一个交易可能存在于多个区块中。 */
	Traces map[common.Hash]*Trace
}

func (searcher *TxGuard) DelOldBlocks(maxTime int) {
	/**
	 * (1) 调用BlocksInTime.DelOldBlocks删除区块，并得到已删除的区块
	 * (2) 遍历已删除区块中的所有交易：根据交易Hash删除Traces中的交易
	 * (3) 遍历已删除区块：根据区块高度和区块Hash删除HeightBuckets中的区块
	 */
}

func (searcher *TxGuard) SaveBlock(block *types.Block) {
	/** 见删除函数DelOldBlocks的顺序，添加类似 */
}

func (searcher *TxGuard) IsTxExist(startBlockHash common.Hash, tx *types.Transaction) {
	// TODO
}

func (searcher *TxGuard) IsTxsExist(block *types.Block) {
	/**
	 * (1) 根据block中的交易，从Traces获取这些交易所在的区块
	 * (2) 根据这些区块，计算这些区块的最大高度和最小高度
	 * (3) 从HeightBuckets中，从最小高度和最小高度的区块开始，截止最大高度，找出所有的路径区块
	 * (4) 判断所有的路径区块，是否在这些交易的区块中，如果存在，则存在；否则不存在
	 */
}
