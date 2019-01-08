package network

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"sync"
)

// blockConfirms same block's confirm data set
type blockConfirms map[common.Hash][]*BlockConfirmData

// ConfirmCache record block confirm data which block doesn't exist in local
type ConfirmCache struct {
	cache map[uint32]blockConfirms

	lock sync.Mutex
}

func NewConfirmCache() *ConfirmCache {
	return &ConfirmCache{
		cache: make(map[uint32]blockConfirms),
	}
}

// Push push block confirm data to cache
func (c *ConfirmCache) Push(data *BlockConfirmData) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, ok := c.cache[data.Height]; !ok {
		c.cache[data.Height] = make(map[common.Hash][]*BlockConfirmData)
		c.cache[data.Height][data.Hash] = make([]*BlockConfirmData, 0, 2)
	}
	c.cache[data.Height][data.Hash] = append(c.cache[data.Height][data.Hash], data)

	if len(c.cache) > 10240 {
		c.Clear(^uint32(0))
	}
}

// Pop get special confirm data by height and hash and then delete from cache
func (c *ConfirmCache) Pop(height uint32, hash common.Hash) []*BlockConfirmData {
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, ok := c.cache[height]; !ok {
		return nil
	}
	if _, ok := c.cache[height][hash]; !ok {
		return nil
	}
	res := make([]*BlockConfirmData, 0, len(c.cache[height][hash]))
	for _, v := range c.cache[height][hash] {
		res = append(res, v)
	}
	delete(c.cache[height], hash)
	return res
}

// Clear clear dirty data form cache
func (c *ConfirmCache) Clear(height uint32) {
	c.lock.Lock()
	defer c.lock.Unlock()

	for h, _ := range c.cache {
		if h <= height {
			delete(c.cache, h)
		}
	}
}

// Size calculate cache's size
func (c *ConfirmCache) Size() int {
	c.lock.Lock()
	defer c.lock.Unlock()

	count := 0
	for _, blockConfirms := range c.cache {
		for _, confirms := range blockConfirms {
			count += len(confirms)
		}
	}
	return count
}

type blocksSameHeight struct {
	Height uint32
	Blocks map[common.Hash]*types.Block
}

type BlockCache struct {
	cache []*blocksSameHeight
	lock  sync.Mutex
}

func NewBlockCache() *BlockCache {
	return &BlockCache{
		cache: make([]*blocksSameHeight, 0, 100),
	}
}

func (c *BlockCache) Add(block *types.Block) {
	c.lock.Lock()
	defer c.lock.Unlock()

	blocks := make(map[common.Hash]*types.Block)
	blocks[block.Hash()] = block
	height := block.Height()
	bsh := &blocksSameHeight{
		Height: block.Height(),
		Blocks: blocks,
	}
	length := len(c.cache)
	if length == 0 || height < c.cache[0].Height { // not exist or less than min height
		c.cache = append([]*blocksSameHeight{bsh}, c.cache...)
	} else if height > c.cache[length-1].Height { // larger than max height
		c.cache = append(c.cache, bsh)
	} else {
		for i := 0; i < len(c.cache); i++ {
			if c.cache[i].Height == height { // already exist
				c.cache[i].Blocks[block.Hash()] = block
				break
			} else if c.cache[i].Height > height { // not exist
				tmp := append(c.cache[:i+1], bsh)
				c.cache = append(tmp, c.cache[i+1:]...)
				break
			}
		}
	}

	if len(c.cache) > 10240 {
		c.Clear(^uint32(0))
	}
}

func (c *BlockCache) Iterate(callback func(*types.Block) bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, blocks := range c.cache {
		for _, v := range blocks.Blocks {
			if callback(v) {
				delete(blocks.Blocks, v.Hash())
			}
		}
	}
}

func (c *BlockCache) Clear(height uint32) {
	c.lock.Lock()
	defer c.lock.Unlock()

	index := -1
	for i, item := range c.cache {
		if item.Height <= height {
			index = i
		} else if item.Height > height {
			break
		}
	}
	c.cache = c.cache[index+1:]
}

func (c *BlockCache) Size() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	count := 0
	for _, blocks := range c.cache {
		count += len(blocks.Blocks)
	}
	return count
}

func (c *BlockCache) FirstHeight() uint32 {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.cache == nil || len(c.cache) == 0 {
		return 0
	}
	return c.cache[0].Height
}
