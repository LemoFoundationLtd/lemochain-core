package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

func TestBlocksTrie_PushBlock(t *testing.T) {
	curTime := time.Now().Unix()

	var trie BlocksTrie
	tx1 := makeTx(testPrivate, common.HexToAddress("0x01"), params.OrdinaryTx, new(big.Int).SetInt64(100))
	tx2 := makeTx(testPrivate, common.HexToAddress("0x02"), params.OrdinaryTx, new(big.Int).SetInt64(200))
	tx3 := makeTx(testPrivate, common.HexToAddress("0x03"), params.OrdinaryTx, new(big.Int).SetInt64(300))
	tx4 := makeTx(testPrivate, common.HexToAddress("0x04"), params.OrdinaryTx, new(big.Int).SetInt64(400))
	tx5 := makeTx(testPrivate, common.HexToAddress("0x05"), params.OrdinaryTx, new(big.Int).SetInt64(500))
	tx6 := makeTx(testPrivate, common.HexToAddress("0x06"), params.OrdinaryTx, new(big.Int).SetInt64(600))
	tx7 := makeTx(testPrivate, common.HexToAddress("0x07"), params.OrdinaryTx, new(big.Int).SetInt64(700))
	tx8 := makeTx(testPrivate, common.HexToAddress("0x08"), params.OrdinaryTx, new(big.Int).SetInt64(800))
	tx9 := makeTx(testPrivate, common.HexToAddress("0x09"), params.OrdinaryTx, new(big.Int).SetInt64(900))

	block1 := store.GetBlock1()
	block1.Header.Time = uint32(curTime)
	block1.Txs = append(block1.Txs, tx1)
	block1.Txs = append(block1.Txs, tx2)
	block1.Txs = append(block1.Txs, tx3)
	trie.PushBlock(block1)
	assert.Equal(t, 3, len(trie.BlocksByHash[1].BlocksByHash[block1.Hash()].TxsIndex))

	slot := curTime % int64(2*TransactionExpiration)
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

	slot = curTime % int64(2*TransactionExpiration)
	assert.Equal(t, 2, len(trie.BlocksByTime[slot].BlocksByHeight))

	block3 := store.GetBlock3()
	block3.Header.ParentHash = block2.Hash()
	block3.Txs = append(block3.Txs, tx8)
	block3.Txs = append(block3.Txs, tx9)
	block3.Header.Time = uint32(curTime + 3600)

	hashes := trie.PushBlock(block3)
	assert.Equal(t, 7, len(hashes))
}

func TestBlocksTrie_DelBlock(t *testing.T) {
	curTime := time.Now().Unix()

	var trie BlocksTrie
	tx1 := makeTx(testPrivate, common.HexToAddress("0x01"), params.OrdinaryTx, new(big.Int).SetInt64(100))
	tx2 := makeTx(testPrivate, common.HexToAddress("0x02"), params.OrdinaryTx, new(big.Int).SetInt64(200))
	tx3 := makeTx(testPrivate, common.HexToAddress("0x03"), params.OrdinaryTx, new(big.Int).SetInt64(300))
	tx4 := makeTx(testPrivate, common.HexToAddress("0x04"), params.OrdinaryTx, new(big.Int).SetInt64(400))
	tx5 := makeTx(testPrivate, common.HexToAddress("0x05"), params.OrdinaryTx, new(big.Int).SetInt64(500))
	tx6 := makeTx(testPrivate, common.HexToAddress("0x06"), params.OrdinaryTx, new(big.Int).SetInt64(600))
	tx7 := makeTx(testPrivate, common.HexToAddress("0x07"), params.OrdinaryTx, new(big.Int).SetInt64(700))

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

	slot := curTime % int64(2*TransactionExpiration)
	assert.Equal(t, 2, len(trie.BlocksByTime[slot].BlocksByHeight))

	item := trie.BlocksByTime[slot].BlocksByHeight[block1.Height()]
	assert.Equal(t, 0, len(item))
}

func TestBlocksTrie_Path(t *testing.T) {
	var trie BlocksTrie
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
