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
	tx1 := makeTxRandom(common.HexToAddress("0x01"))
	tx2 := makeTxRandom(common.HexToAddress("0x02"))
	tx3 := makeTxRandom(common.HexToAddress("0x03"))
	tx4 := makeTxRandom(common.HexToAddress("0x04"))
	tx5 := makeTxRandom(common.HexToAddress("0x05"))
	tx6 := makeTxRandom(common.HexToAddress("0x06"))
	tx7 := makeTxRandom(common.HexToAddress("0x07"))
	tx8 := makeTxRandom(common.HexToAddress("0x08"))
	tx9 := makeTxRandom(common.HexToAddress("0x09"))

	block1 := store.GetBlock1()
	block1.Header.Time = uint32(curTime)
	block1.Txs = append(block1.Txs, tx1)
	block1.Txs = append(block1.Txs, tx2)
	block1.Txs = append(block1.Txs, tx3)
	trie.PushBlock(block1)
	assert.Equal(t, 3, len(trie.BlocksByHash[1].BlocksByHash[block1.Hash()].TxsIndex))

	slot := curTime % int64(TransactionExpiration)
	assert.Equal(t, 1, len(trie.BlocksByTime[slot].BlocksByHeight))

	block2 := store.GetBlock2()
	block2.Header.ParentHash = block1.Hash()
	block2.Header.Time = uint32(curTime)
	block2.Txs = append(block2.Txs, tx4)
	block2.Txs = append(block2.Txs, tx5)
	block2.Txs = append(block2.Txs, tx6)
	block2.Txs = append(block2.Txs, tx7)
	trie.PushBlock(block2)
	assert.Equal(t, 4, len(trie.BlocksByHash[2].BlocksByHash[block2.Hash()].TxsIndex))

	slot = curTime % int64(TransactionExpiration)
	assert.Equal(t, 2, len(trie.BlocksByTime[slot].BlocksByHeight))

	block3 := store.GetBlock3()
	block3.Header.ParentHash = block2.Hash()
	block3.Txs = append(block3.Txs, tx8)
	block3.Txs = append(block3.Txs, tx9)
	block3.Header.Time = uint32(curTime + 3600)

	trie.PushBlock(block3)
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

	assert.Equal(t, 4, len(trie.BlocksByHash[2].BlocksByHash[block2.Hash()].TxsIndex))

	slot := curTime % int64(TransactionExpiration)
	assert.Equal(t, 2, len(trie.BlocksByTime[slot].BlocksByHeight))

	item := trie.BlocksByTime[slot].BlocksByHeight[block1.Height()]
	assert.Equal(t, 0, len(item))
}

func TestBlocksTrie_Path(t *testing.T) {
	trie := NewBlocksTrie()
	block0 := store.GetBlock0()
	trie.PushBlock(block0)

	block1 := store.GetBlock1()
	trie.PushBlock(block1)

	block2 := store.GetBlock2()
	trie.PushBlock(block2)

	block3 := store.GetBlock3()
	trie.PushBlock(block3)

	block4 := store.GetBlock4()
	trie.PushBlock(block4)

	nodes := trie.Path(block4.Hash(), block4.Height(), 2, 3)
	assert.Equal(t, 2, len(nodes))
}
