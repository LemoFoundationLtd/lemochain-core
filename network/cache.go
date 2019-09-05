package network

import (
	"bufio"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/network/p2p"
	"os"
	"path/filepath"
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

// Push set block confirm data to cache
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

// Clear clear blocks of block'Height <= height
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

// Remove remove a block
func (c *BlockCache) Remove(block *types.Block) {
	c.lock.Lock()
	defer c.lock.Unlock()
	height := block.Height()
	hash := block.Hash()
	length := len(c.cache)
	// cache has not this block'height block
	if length == 0 || c.cache[0].Height > height || c.cache[length-1].Height < height {
		return
	} else {
		// 	find this block and remove it
		for i := 0; i < len(c.cache); i++ {
			if c.cache[i].Height == height {
				if _, ok := c.cache[i].Blocks[hash]; ok {
					delete(c.cache[i].Blocks, hash)
				}
				// if blocks is nil, then delete this height
				if len(c.cache[i].Blocks) == 0 {
					c.cache = append(c.cache[:i], c.cache[i+1:]...)
				}
			}
		}
	}
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

// IsExit
func (c *BlockCache) IsExit(hash common.Hash, height uint32) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	length := len(c.cache)
	if length == 0 || height < c.cache[0].Height || height > c.cache[length-1].Height {
		return false
	}
	for _, blocks := range c.cache {
		if _, ok := blocks.Blocks[hash]; ok {
			return true
		} else {
			return false
		}
	}
	return false
}

type MsgCache struct {
	cache chan *p2p.Msg
}

const cacheCap = 1024

func NewMsgCache() *MsgCache {
	return &MsgCache{
		cache: make(chan *p2p.Msg, cacheCap),
	}
}

// Pop
func (m *MsgCache) Pop() *p2p.Msg {
	msg := <-m.cache
	return msg
}

// Push
func (m *MsgCache) Push(msg *p2p.Msg) {
	m.cache <- msg
}

func (m *MsgCache) Size() int {
	return len(m.cache)
}

type HashSet struct {
	cache map[common.Hash]struct{}
	sync.Mutex
}

func (s *HashSet) set(hash common.Hash) {
	s.cache[hash] = struct{}{}
}

func (s *HashSet) size() int {
	return len(s.cache)
}

func (s *HashSet) isExist(hash common.Hash) bool {
	_, ok := s.cache[hash]
	return ok
}

// 区块黑名单cache
const BlackBlockFile = "blackblocklist"

type invalidBlockCache struct {
	HashSet
}

func (bbc *invalidBlockCache) IsBlackBlock(blockHash, parentHash common.Hash) bool {
	bbc.Lock()
	defer bbc.Unlock()
	if bbc.size() == 0 {
		return false
	}
	if bbc.isExist(blockHash) {
		return true
	}
	// 查找父块是否为黑名单
	if bbc.isExist(parentHash) {
		// 查找到父块为黑名单，则保存此块为黑名单块
		bbc.set(blockHash)
		return true
	}
	return false
}

func InitBlockBlackCache(dataDir string) *invalidBlockCache {
	cache := readBlockBlacklistFile(dataDir)
	return &invalidBlockCache{
		HashSet: HashSet{
			cache: cache,
			Mutex: sync.Mutex{},
		},
	}
}

// readBlockBlacklistFile 读取文件中的区块黑名单到缓存里面
func readBlockBlacklistFile(dataDir string) map[common.Hash]struct{} {
	cache := make(map[common.Hash]struct{}, 0)
	filePath := filepath.Join(dataDir, BlackBlockFile)
	f, err := os.OpenFile(filePath, os.O_RDONLY, 666)
	if err != nil {
		return cache
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Errorf("Close file failed: %v", err)
		}
	}()

	buf := bufio.NewReader(f)
	line, _, err := buf.ReadLine()
	for err == nil {
		hash := common.HexToHash(string(line))
		cache[hash] = struct{}{}
		line, _, err = buf.ReadLine()
	}
	// 打印日志

	list := make([]string, 0, len(cache))
	for k := range cache {
		list = append(list, k.String())
	}
	log.Infof("Black block list: %v", list)

	return cache
}
