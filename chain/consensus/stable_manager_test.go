package consensus

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
)

type testBlockStore struct {
	// These blocks must be in height ascending order. And the stable block is its first block. If set any block to stable, it will remove the previous blocks as prune
	Blocks []*types.Block
	Stable *types.Block
}

func (bs *testBlockStore) LoadLatestBlock() (*types.Block, error) {
	if bs.Stable == nil {
		return nil, store.ErrNotExist
	}
	return bs.Stable, nil
}

// SetStableBlock set a block to stable, then remove the previous blocks as prune
func (bs *testBlockStore) SetStableBlock(hash common.Hash) ([]*types.Block, error) {
	for i := 0; i < len(bs.Blocks); i++ {
		if bs.Blocks[i].Hash() == hash {
			pruned := bs.Blocks[:i]
			bs.Blocks = bs.Blocks[i:]
			bs.Stable = bs.Blocks[0]
			return pruned, nil
		}
	}
	return nil, store.ErrArgInvalid
}

func createTestBlockStore(blocks ...*types.Block) *testBlockStore {
	if len(blocks) == 0 {
		return &testBlockStore{}
	}
	return &testBlockStore{
		Blocks: blocks,
		Stable: blocks[0],
	}
}

func TestNewStableManager(t *testing.T) {
	dm := deputynode.NewManager(5, &testBlockLoader{})
	sm := NewStableManager(dm, &testBlockStore{})
	assert.Equal(t, dm, sm.dm)
}

func TestStableManager_StableBlock(t *testing.T) {
	dm := deputynode.NewManager(5, &testBlockLoader{})

	// empty
	sm := NewStableManager(dm, &testBlockStore{})
	assert.PanicsWithValue(t, store.ErrNotExist, func() {
		sm.StableBlock()
	})

	// not empty
	sm = NewStableManager(dm, createTestBlockStore(testBlocks[0]))
	block := sm.StableBlock()
	assert.Equal(t, testBlocks[0], block)
}

func TestStableManager_UpdateStable(t *testing.T) {
	dm := initDeputyManager(3)
	block0 := testBlocks[0].ShallowCopy() // stable block
	block1 := block0.ShallowCopy()        // unstable block (enough confirm)
	block1.Header = &types.Header{Height: block1.Height() + 1}
	block1.Confirms = make([]types.SignData, 2)
	sm := NewStableManager(dm, createTestBlockStore(block0, block1))

	// not higher block
	block := block0.ShallowCopy()
	block.Confirms = make([]types.SignData, 2)
	changed, prunedBlocks, err := sm.UpdateStable(block)
	assert.NoError(t, err)
	assert.Equal(t, false, changed)
	assert.Nil(t, prunedBlocks)

	// not enough confirm
	block = block1.ShallowCopy()
	block.Header = &types.Header{Height: block.Height() + 1}
	block.Confirms = make([]types.SignData, 0)
	changed, prunedBlocks, err = sm.UpdateStable(block)
	assert.NoError(t, err)
	assert.Equal(t, false, changed)
	assert.Nil(t, prunedBlocks)

	// SetStableBlock fail
	block2 := block1.ShallowCopy()
	block2.Header = &types.Header{Height: block.Height() + 1}
	block2.Confirms = make([]types.SignData, 2)
	changed, prunedBlocks, err = sm.UpdateStable(block2)
	assert.Equal(t, ErrSetStableBlockToDB, err)

	// changed successful
	changed, prunedBlocks, err = sm.UpdateStable(block1)
	assert.NoError(t, err)
	assert.Equal(t, true, changed)
	assert.Len(t, prunedBlocks, 1)
	assert.Equal(t, block0, prunedBlocks[0])
}

func TestIsConfirmEnough(t *testing.T) {
	var tests = []struct {
		ConfirmCount int
		DeputyCount  int
		Want         bool
	}{
		{0, 1, true},
		{1, 1, true},
		{2, 1, true},
		{0, 2, false},
		{1, 2, true},
		{2, 2, true},
		{0, 3, false},
		{1, 3, true},
		{2, 3, true},
		{3, 3, true},
		{0, 6, false},
		{1, 6, false},
		{2, 6, false},
		{3, 6, true},
		{4, 6, true},
		{5, 6, true},
		{6, 6, true},
	}

	for _, test := range tests {
		caseName := fmt.Sprintf("confirm=%d/%d", test.ConfirmCount, test.DeputyCount)
		t.Run(caseName, func(t *testing.T) {
			test := test // capture range variable
			t.Parallel()

			dm := initDeputyManager(test.DeputyCount)
			block := testBlocks[0].ShallowCopy()
			block.Confirms = make([]types.SignData, test.ConfirmCount)
			assert.Equal(t, test.Want, IsConfirmEnough(block, dm))
		})
	}
}
