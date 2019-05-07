package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestBlocksTrie_PushBlock(t *testing.T) {
	curTime := time.Now().Unix()

	trie := NewBlocksTrie()
	trie.PushBlock(nil)
	assert.Equal(t, 0, len(trie.HeightBuckets))

	tx1 := makeTxRandom(common.HexToAddress("0x01"))
	tx2 := makeTxRandom(common.HexToAddress("0x02"))
	tx3 := makeTxRandom(common.HexToAddress("0x03"))
	tx4 := makeTxRandom(common.HexToAddress("0x04"))
	tx5 := makeTxRandom(common.HexToAddress("0x05"))
	tx6 := makeTxRandom(common.HexToAddress("0x06"))
	tx7 := makeTxRandom(common.HexToAddress("0x07"))
	tx8 := makeTxRandom(common.HexToAddress("0x08"))
	tx9 := makeTxRandom(common.HexToAddress("0x09"))

	// 正常添加块
	block1 := store.GetBlock1()
	block1.Header.Time = uint32(curTime)
	block1.Txs = append(block1.Txs, tx1)
	block1.Txs = append(block1.Txs, tx2)
	block1.Txs = append(block1.Txs, tx3)
	trie.PushBlock(block1)
	assert.Equal(t, 1, len(trie.HeightBuckets))
	assert.Equal(t, 3, len(trie.HeightBuckets[1][block1.Hash()].TxHashSet))

	slot := curTime % int64(TransactionExpiration)
	assert.Equal(t, 1, len(trie.TimeBuckets[slot].BlocksByHeight))

	block2 := store.GetBlock2()
	block2.Header.ParentHash = block1.Hash()
	block2.Header.Time = uint32(curTime)
	block2.Txs = append(block2.Txs, tx4)
	block2.Txs = append(block2.Txs, tx5)
	block2.Txs = append(block2.Txs, tx6)
	block2.Txs = append(block2.Txs, tx7)
	trie.PushBlock(block2)
	slot = curTime % int64(TransactionExpiration)
	assert.Equal(t, 2, len(trie.TimeBuckets[slot].BlocksByHeight))

	assert.Equal(t, 2, len(trie.HeightBuckets))
	assert.Equal(t, 4, len(trie.HeightBuckets[2][block2.Hash()].TxHashSet))

	// 淘汰过期数据
	block3 := store.GetBlock3()
	block3.Header.ParentHash = block2.Hash()
	block3.Txs = append(block3.Txs, tx8)
	block3.Txs = append(block3.Txs, tx9)
	block3.Header.Time = uint32(curTime + 3600)
	trie.PushBlock(block3)

	assert.Equal(t, 1, len(trie.HeightBuckets))
	assert.Equal(t, 2, len(trie.HeightBuckets[3][block3.Hash()].TxHashSet))

	slot = (curTime + 3600) % int64(TransactionExpiration)
	assert.Equal(t, 1, len(trie.TimeBuckets[slot].BlocksByHeight))

	// 添加过期块
	tx10 := makeTxRandom(common.HexToAddress("0x10"))
	block4 := store.GetBlock4()
	block4.Header.ParentHash = block3.Hash()
	block4.Txs = append(block4.Txs, tx10)
	block4.Header.Time = uint32(curTime)
	err := trie.PushBlock(block4)
	assert.Equal(t, ErrTxPoolBlockExpired, err)

	assert.Equal(t, 1, len(trie.HeightBuckets))
	assert.Equal(t, 2, len(trie.HeightBuckets[3][block3.Hash()].TxHashSet))
	slot = (curTime + 3600) % int64(TransactionExpiration)
	assert.Equal(t, 1, len(trie.TimeBuckets[slot].BlocksByHeight))
}

func TestBlocksTrie_DelBlock(t *testing.T) {
	curTime := time.Now().Unix()

	trie := NewBlocksTrie()
	tx1 := makeTxRandom(common.HexToAddress("0x01"))
	tx2 := makeTxRandom(common.HexToAddress("0x02"))
	tx3 := makeTxRandom(common.HexToAddress("0x03"))
	tx4 := makeTxRandom(common.HexToAddress("0x04"))
	tx5 := makeTxRandom(common.HexToAddress("0x05"))
	tx6 := makeTxRandom(common.HexToAddress("0x06"))
	tx7 := makeTxRandom(common.HexToAddress("0x07"))

	block1 := store.GetBlock1()
	block1.Header.Time = uint32(curTime)
	block1.Txs = append(block1.Txs, tx1)
	block1.Txs = append(block1.Txs, tx2)
	block1.Txs = append(block1.Txs, tx3)
	trie.PushBlock(block1)

	block2 := store.GetBlock2()
	block2.Header.ParentHash = block1.Hash()
	block2.Header.Time = uint32(curTime)
	block2.Txs = append(block2.Txs, tx4)
	block2.Txs = append(block2.Txs, tx5)
	block2.Txs = append(block2.Txs, tx6)
	block2.Txs = append(block2.Txs, tx7)
	trie.PushBlock(block2)

	trie.DelBlock(block1)

	assert.Equal(t, 1, len(trie.HeightBuckets))
	slot := curTime % int64(TransactionExpiration)
	assert.Equal(t, 1, len(trie.TimeBuckets[slot].BlocksByHeight))
	item := trie.TimeBuckets[slot].BlocksByHeight[block1.Height()]
	assert.Nil(t, item)

	trie.DelBlock(block2)

	assert.Equal(t, 0, len(trie.HeightBuckets))
	slot = curTime % int64(TransactionExpiration)
	assert.Equal(t, 0, len(trie.TimeBuckets[slot].BlocksByHeight))
	item = trie.TimeBuckets[slot].BlocksByHeight[block2.Height()]
	assert.Nil(t, item)
}

func TestBlocksTrie_Path1(t *testing.T) {
	cur := time.Now().Unix()

	// height 0
	trie := NewBlocksTrie()
	block0 := store.GetBlock0()
	trie.PushBlock(block0)

	// height 1
	block11 := store.GetBlock1()
	block11.Header.ParentHash = block0.Hash()
	block11.Header.Time = uint32(cur + 11)
	trie.PushBlock(block11)

	block12 := store.GetBlock1()
	block12.Header.ParentHash = block0.Hash()
	block12.Header.Time = uint32(cur + 12)
	trie.PushBlock(block12)

	block13 := store.GetBlock1()
	block13.Header.ParentHash = block0.Hash()
	block13.Header.Time = uint32(cur + 13)
	trie.PushBlock(block13)

	// height 2
	block1121 := store.GetBlock2()
	block1121.Header.ParentHash = block11.Hash()
	block1121.Header.Time = uint32(cur + 1121)
	trie.PushBlock(block1121)

	block1122 := store.GetBlock2()
	block1122.Header.ParentHash = block11.Hash()
	block1122.Header.Time = uint32(cur + 1122)
	trie.PushBlock(block1122)

	block1221 := store.GetBlock2()
	block1221.Header.ParentHash = block12.Hash()
	block1221.Header.Time = uint32(cur + 1221)
	trie.PushBlock(block1221)

	block1222 := store.GetBlock2()
	block1222.Header.ParentHash = block12.Hash()
	block1222.Header.Time = uint32(cur + 1222)
	trie.PushBlock(block1222)

	// height 3
	block112131 := store.GetBlock3()
	block112131.Header.ParentHash = block1121.Hash()
	block112131.Header.Time = uint32(cur + 112131)
	trie.PushBlock(block112131)

	block112132 := store.GetBlock3()
	block112132.Header.ParentHash = block1121.Hash()
	block112132.Header.Time = uint32(cur + 112132)
	trie.PushBlock(block112132)

	block122131 := store.GetBlock3()
	block122131.Header.ParentHash = block1221.Hash()
	block122131.Header.Time = uint32(cur + 122131)
	trie.PushBlock(block122131)

	// height 4
	block11213141 := store.GetBlock4()
	block11213141.Header.ParentHash = block112131.Hash()
	block11213141.Header.Time = uint32(cur + 11213141)
	trie.PushBlock(block11213141)

	nodes := trie.Path(block13.Hash(), block13.Height(), 0, 1)
	assert.Equal(t, 2, len(nodes))
	assert.Equal(t, block13.Header, nodes[0].Header)
	assert.Equal(t, block0.Header, nodes[1].Header)

	nodes = trie.Path(block13.Hash(), block13.Height(), 0, 0)
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, block0.Header, nodes[0].Header)

	nodes = trie.Path(block13.Hash(), block13.Height(), 1, 1)
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, block13.Header, nodes[0].Header)

	nodes = trie.Path(block11213141.Hash(), block11213141.Height(), 0, 15)
	assert.Equal(t, 5, len(nodes))
	assert.Equal(t, block11213141.Header, nodes[0].Header)
	assert.Equal(t, block112131.Header, nodes[1].Header)
	assert.Equal(t, block1121.Header, nodes[2].Header)
	assert.Equal(t, block11.Header, nodes[3].Header)
	assert.Equal(t, block0.Header, nodes[4].Header)

	nodes = trie.Path(block11213141.Hash(), block11213141.Height(), 0, 2)
	assert.Equal(t, 3, len(nodes))
	assert.Equal(t, block1121.Header, nodes[0].Header)
	assert.Equal(t, block11.Header, nodes[1].Header)
	assert.Equal(t, block0.Header, nodes[2].Header)

	nodes = trie.Path(block11213141.Hash(), block11213141.Height(), 2, 2)
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, block1121.Header, nodes[0].Header)

	// 开始块不在块缓存，获取的路径节点为空
	block11213142 := store.GetBlock4()
	block11213142.Header.ParentHash = block112131.Hash()
	block11213142.Header.Time = uint32(cur + 11213142)

	nodes = trie.Path(block11213142.Hash(), block11213142.Height(), 2, 2)
	assert.Equal(t, 0, len(nodes))
}

func TestBlocksTrie_Path2(t *testing.T) {
	cur := time.Now().Unix()

	// height 0
	trie := NewBlocksTrie()

	// height 1
	block11 := store.GetBlock1()
	block11.Header.ParentHash = common.HexToHash("0xaabbccddee")
	block11.Header.Time = uint32(cur + 11)
	trie.PushBlock(block11)

	// height 2
	block1121 := store.GetBlock2()
	block1121.Header.ParentHash = block11.Hash()
	block1121.Header.Time = uint32(cur + 1121)
	trie.PushBlock(block1121)

	block1122 := store.GetBlock2()
	block1122.Header.ParentHash = block11.Hash()
	block1122.Header.Time = uint32(cur + 1122)
	trie.PushBlock(block1122)

	// height 3
	block112131 := store.GetBlock3()
	block112131.Header.ParentHash = block1121.Hash()
	block112131.Header.Time = uint32(cur + 112131)
	trie.PushBlock(block112131)

	block112132 := store.GetBlock3()
	block112132.Header.ParentHash = block1121.Hash()
	block112132.Header.Time = uint32(cur + 112132)
	trie.PushBlock(block112132)

	// height 4
	block11213141 := store.GetBlock4()
	block11213141.Header.ParentHash = block112131.Hash()
	block11213141.Header.Time = uint32(cur + 11213141)
	trie.PushBlock(block11213141)

	nodes := trie.Path(block11213141.Hash(), block11213141.Height(), 0, 15)
	assert.Equal(t, 4, len(nodes))
	assert.Equal(t, block11213141.Header, nodes[0].Header)
	assert.Equal(t, block112131.Header, nodes[1].Header)
	assert.Equal(t, block1121.Header, nodes[2].Header)
	assert.Equal(t, block11.Header, nodes[3].Header)
}
