package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_buildBlockNode(t *testing.T) {
	node := buildBlockNode(nil)
	assert.Empty(t, node)

	b := makeBlock(1, 100, makeTx(0, 0))
	node = buildBlockNode(b)
	assert.Equal(t, b.Header, node.Header)
	assert.Equal(t, b.Txs, node.Txs)
}

func TestBlockNodes_getHeightRange(t *testing.T) {
	// empty
	nodes := BlockNodes{}
	min, max := nodes.getHeightRange()
	assert.Equal(t, uint32(0), min)
	assert.Equal(t, uint32(0), max)

	// one node
	nodes = BlockNodes{
		buildBlockNode(makeBlock(1, 100)),
	}
	min, max = nodes.getHeightRange()
	assert.Equal(t, uint32(1), min)
	assert.Equal(t, uint32(1), max)

	// many nodes
	nodes = BlockNodes{
		buildBlockNode(makeBlock(10, 100)),
		buildBlockNode(makeBlock(1, 100)),
		buildBlockNode(makeBlock(3, 100)),
	}
	min, max = nodes.getHeightRange()
	assert.Equal(t, uint32(1), min)
	assert.Equal(t, uint32(10), max)
}

func TestBlockCache_Add_Del(t *testing.T) {
	// empty
	trie := make(BlockCache)
	assert.Equal(t, 0, len(trie))

	// delete not exist
	trie.Del(common.HexToHash("1"))

	// add one block and delete it
	b := makeBlock(10, 100)
	trie.Add(b)
	assert.Equal(t, 1, len(trie))
	trie.Del(b.Hash())
	assert.Equal(t, 0, len(trie))

	// add exist block
	trie.Add(b)
	trie.Add(b)
	assert.Equal(t, 1, len(trie))

	// add more blocks and delete one
	b = makeBlock(11, 100)
	trie.Add(b)
	assert.Equal(t, 2, len(trie))
	b = makeBlock(12, 100)
	trie.Add(b)
	assert.Equal(t, 3, len(trie))
}

func TestBlockCache_Get(t *testing.T) {
	// empty
	trie := make(BlockCache)
	assert.Nil(t, trie.Get(common.HexToHash("1")))

	// get exist
	b := makeBlock(10, 100)
	trie.Add(b)
	trie.Add(makeBlock(11, 100))
	assert.Equal(t, b.Header, trie.Get(b.Hash()).Header)
	assert.Equal(t, b.Txs, trie.Get(b.Hash()).Txs)

	// get not exist
	assert.Nil(t, trie.Get(common.HexToHash("2")))
}

func TestBlockCache_CollectBlocks(t *testing.T) {
	// empty cache
	trie := make(BlockCache)
	// empty target
	hashes := make(HashSet)
	nodes, err := trie.CollectBlocks(hashes)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(nodes))
	// not exist target
	hashes.Add(common.HexToHash("1"))
	nodes, err = trie.CollectBlocks(hashes)
	assert.Equal(t, ErrNotFoundBlockCache, err)

	// one block cache
	trie = make(BlockCache)
	b := makeBlock(10, 100)
	trie.Add(b)
	// empty target
	hashes = make(HashSet)
	nodes, err = trie.CollectBlocks(hashes)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(nodes))
	// not exist target
	hashes.Add(common.HexToHash("1"))
	nodes, err = trie.CollectBlocks(hashes)
	assert.Equal(t, ErrNotFoundBlockCache, err)
	// exist target
	hashes = make(HashSet)
	hashes.Add(b.Hash())
	nodes, err = trie.CollectBlocks(hashes)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, b.Header, nodes[0].Header)
	assert.Equal(t, b.Txs, nodes[0].Txs)
	// one exist and one not exist target
	hashes.Add(common.HexToHash("1"))
	nodes, err = trie.CollectBlocks(hashes)
	assert.Equal(t, ErrNotFoundBlockCache, err)
}

func TestBlockCache_SliceOnFork(t *testing.T) {
	type testConfig struct {
		startBlockHash     common.Hash
		minHeight          uint32
		maxHeight          uint32
		expectBlockIndexes []int
		expectErr          error
	}

	//       ┌─2
	// 0───1─┼─3───6
	//       ├─4─┬─7───9
	//       │   └─8
	//       └─5
	// the height of 0 block is 100
	blocks := generateBlocks()
	trie := make(BlockCache)
	for _, b := range blocks {
		trie.Add(b)
	}

	tests := []testConfig{
		// not exist start block
		{common.HexToHash("1"), 100, 104, []int{}, ErrNotFoundBlockCache},
		{blocks[7].Hash(), 99, 104, []int{7, 4, 1, 0}, nil},
		// success
		{blocks[7].Hash(), 100, 104, []int{7, 4, 1, 0}, nil},
		{blocks[7].Hash(), 100, 103, []int{7, 4, 1, 0}, nil},
		{blocks[7].Hash(), 101, 103, []int{7, 4, 1}, nil},
		{blocks[7].Hash(), 103, 103, []int{7}, nil},
		{blocks[7].Hash(), 103, 104, []int{7}, nil},
		{blocks[7].Hash(), 104, 104, []int{}, nil},
		{blocks[7].Hash(), 109, 110, []int{}, nil},
		{blocks[5].Hash(), 100, 101, []int{1, 0}, nil},
		{blocks[5].Hash(), 103, 104, []int{}, nil},
	}

	for i, test := range tests {
		hashes, err := trie.SliceOnFork(test.startBlockHash, test.minHeight, test.maxHeight)
		if test.expectErr != nil {
			assert.Equal(t, test.expectErr, err, "index=%d", i)
			continue
		}
		assert.NoError(t, err)
		assert.Equal(t, len(test.expectBlockIndexes), len(hashes), "index=%d", i)
		for j, hash := range hashes {
			bIndex := test.expectBlockIndexes[j]
			assert.Equal(t, blocks[bIndex].Hash(), hash, "index=%d, bIndex=%d", i, bIndex)
		}
	}
}

func TestBlockCache_IsAppearedOnFork(t *testing.T) {
	type testConfig struct {
		targetBlockIndexes []int
		startBlockIndex    int
		expect             bool
	}

	//       ┌─2
	// 0───1─┼─3───6
	//       ├─4─┬─7───9
	//       │   └─8
	//       └─5
	// the height of 0 block is 100
	blocks := generateBlocks()
	trie := make(BlockCache)
	for _, b := range blocks {
		trie.Add(b)
	}

	tests := []testConfig{
		// not exist hash set
		{[]int{}, 7, false},
		// success
		{[]int{7}, 7, true},
		{[]int{7, 8}, 7, true},
		{[]int{7, 8}, 9, true},
		{[]int{7, 8}, 6, false},
		{[]int{7, 8}, 4, false},
		{[]int{6}, 3, false},
		{[]int{6}, 6, true},
		{[]int{4, 9}, 7, true},
		{[]int{6, 9}, 7, false},
		{[]int{1, 9}, 7, true},
		{[]int{6, 8, 9}, 7, false},
	}

	for i, test := range tests {
		set := make(HashSet)
		for _, index := range test.targetBlockIndexes {
			set.Add(blocks[index].Hash())
		}
		exist := trie.IsAppearedOnFork(set, blocks[test.startBlockIndex].Hash())
		assert.Equal(t, test.expect, exist, "index=%d", i)
	}
}

func TestBlockCache_IsAppearedOnFork_Error(t *testing.T) {
	//       ┌─2
	// 0───1─┼─3───6
	//       ├─4─┬─7───9
	//       │   └─8
	//       └─5
	// the height of 0 block is 100
	blocks := generateBlocks()
	trie := make(BlockCache)
	for _, b := range blocks {
		trie.Add(b)
	}

	// not exist hash set
	assert.PanicsWithValue(t, ErrNotFoundBlockCache, func() {
		set := make(HashSet)
		set.Add(common.HexToHash("1"))
		trie.IsAppearedOnFork(set, common.HexToHash("1"))
	})
	// not exist start block
	assert.PanicsWithValue(t, ErrNotFoundBlockCache, func() {
		set := make(HashSet)
		set.Add(blocks[7].Hash())
		trie.IsAppearedOnFork(set, common.HexToHash("1"))
	})
}
