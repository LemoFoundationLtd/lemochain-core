package consensus

import (
	"fmt"
	"testing"

	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
)

func TestNewForkManager(t *testing.T) {
	dm := deputynode.NewManager(5, &testBlockLoader{})

	// not set stable
	assert.PanicsWithValue(t, ErrNoHeadBlock, func() {
		NewForkManager(dm, createBlockLoader([]int{}, 0), nil)
	})

	fm := NewForkManager(dm, createBlockLoader([]int{}, -1), testBlocks[0])
	assert.Equal(t, testBlocks[0], fm.GetHeadBlock())
}

func TestForkManager_SetHeadBlock_GetHeadBlock(t *testing.T) {
	dm := deputynode.NewManager(5, &testBlockLoader{})

	fm := NewForkManager(dm, createBlockLoader([]int{}, -1), testBlocks[0])
	fm.SetHeadBlock(testBlocks[1])
	assert.Equal(t, testBlocks[1], fm.GetHeadBlock())
}

func Test_findDeputyByAddress(t *testing.T) {
	// no deputies
	node := findDeputyByAddress([]*types.DeputyNode{}, testDeputies[0].MinerAddress)
	assert.Nil(t, node)

	// not match any one
	node = findDeputyByAddress(pickNodes(0), testDeputies[1].MinerAddress)
	assert.Nil(t, node)

	// match one
	node = findDeputyByAddress(pickNodes(0, 1, 2), testDeputies[1].MinerAddress)
	assert.Equal(t, testDeputies[1], node)
}

// test special cases
func TestGetMinerDistance_Error(t *testing.T) {
	dm := deputynode.NewManager(5, &testBlockLoader{})

	nodes0 := pickNodes(0, 1, 2)
	dm.SaveSnapshot(0, nodes0)
	nodes1 := pickNodes(1)
	dm.SaveSnapshot(params.TermDuration, nodes1)
	nodes2 := pickNodes(2, 3, 4, 5)
	dm.SaveSnapshot(params.TermDuration*2, nodes2)
	term0Height := uint32(10)
	term1RewardHeight := params.TermDuration + params.InterimDuration + 1
	term2RewardHeight := params.TermDuration*2 + params.InterimDuration + 1

	// height is 0
	_, err := GetMinerDistance(0, common.Address{}, common.Address{}, dm)
	assert.Equal(t, ErrMineGenesis, err)

	// not exist target miner
	_, err = GetMinerDistance(term0Height, common.Address{}, common.Address{}, dm)
	assert.Equal(t, ErrNotDeputy, err)
	_, err = GetMinerDistance(term0Height, common.Address{}, testDeputies[5].MinerAddress, dm)
	assert.Equal(t, ErrNotDeputy, err)

	// not exist last miner
	_, err = GetMinerDistance(term0Height, common.Address{}, testDeputies[0].MinerAddress, dm)
	assert.Equal(t, ErrNotDeputy, err)
	_, err = GetMinerDistance(term0Height, testDeputies[5].MinerAddress, testDeputies[0].MinerAddress, dm)
	assert.Equal(t, ErrNotDeputy, err)

	// only one deputy
	dis, err := GetMinerDistance(term1RewardHeight, common.Address{}, testDeputies[1].MinerAddress, dm)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), dis)

	// first block
	dis, err = GetMinerDistance(1, common.Address{}, testDeputies[0].MinerAddress, dm)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), dis)
	dis, err = GetMinerDistance(1, common.Address{}, testDeputies[2].MinerAddress, dm)
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), dis)

	// reward block
	dis, err = GetMinerDistance(term2RewardHeight, common.Address{}, testDeputies[2].MinerAddress, dm)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), dis)
	dis, err = GetMinerDistance(term2RewardHeight, common.Address{}, testDeputies[5].MinerAddress, dm)
	assert.NoError(t, err)
	assert.Equal(t, uint64(4), dis)

	// no deputies
	dm = deputynode.NewManager(0, &testBlockLoader{})
	_, err = GetMinerDistance(10, common.Address{}, common.Address{}, dm)
	assert.Equal(t, ErrNotDeputy, err)
}

// test normal cases
func TestGetMinerDistance(t *testing.T) {
	dm := deputynode.NewManager(5, &testBlockLoader{})

	nodes0 := pickNodes(0, 1, 2)
	dm.SaveSnapshot(0, nodes0)
	nodes1 := pickNodes(1)
	dm.SaveSnapshot(params.TermDuration, nodes1)
	nodes2 := pickNodes(2, 3, 4, 5)
	dm.SaveSnapshot(params.TermDuration*2, nodes2)
	term0Height := uint32(10)
	term2RewardHeight := params.TermDuration*2 + params.InterimDuration + 1
	term2Height := term2RewardHeight + 10

	type testDistanceData struct {
		CaseName          string
		TargetHeight      uint32
		LastDeputyIndex   int
		TargetDeputyIndex int
		ExpectDistance    uint64
	}
	var tests = []testDistanceData{
		{"[0,1,2] 2-0=2", term0Height, 0, 2, 2},
		{"[0,1,2] 0-2=1", term0Height, 2, 0, 1},
		{"[0,1,2] 2-2=3", term0Height, 2, 2, 3},
		{"[2,3,4,5] 3-2=1", term2Height, 2, 3, 1},
		{"[2,3,4,5] 4-2=2", term2Height, 2, 4, 2},
		{"[2,3,4,5] 4-4=4", term2Height, 4, 4, 4},
		{"[2,3,4,5] 2-5=1", term2Height, 5, 2, 1},
		{"[2,3,4,5] 2-3=3", term2Height, 3, 2, 3},
		{"[2,3,4,5] 2-2=4", term2Height, 2, 2, 4},
	}

	for _, test := range tests {
		t.Run(test.CaseName, func(t *testing.T) {
			test := test // capture range variable
			t.Parallel()

			lastBlockMiner := testDeputies[test.LastDeputyIndex].MinerAddress
			targetMiner := testDeputies[test.TargetDeputyIndex].MinerAddress
			dis, err := GetMinerDistance(test.TargetHeight, lastBlockMiner, targetMiner, dm)
			assert.NoError(t, err)
			assert.Equal(t, test.ExpectDistance, dis)
		})
	}
}

// test normal cases
func TestForkManager_ChooseNewFork_Error(t *testing.T) {
	dm := deputynode.NewManager(5, &testBlockLoader{})

	// no blocks
	fm := NewForkManager(dm, createBlockLoader([]int{}, -1), testBlocks[0])
	newBlock := fm.ChooseNewFork(nil)
	assert.Nil(t, newBlock)
}

// test normal cases
func TestForkManager_ChooseNewFork(t *testing.T) {
	dm := deputynode.NewManager(5, &testBlockLoader{})

	type testChooseForkData struct {
		CaseName         string
		PickBlockIndexes []int
		ExpectBlockIndex int
	}
	//       ┌─2
	// 0───1─┼─3───6
	//       ├─4─┬─7───9
	//       │   └─8
	//       └─5
	var tests = []testChooseForkData{
		{"[0] 0", []int{0}, 0},
		{"[0,1] 1", []int{0, 1}, 1},
		{"[0,1,2] 2", []int{0, 1, 2}, 2},
		{"[0,1,2,3] 2", []int{0, 1, 2, 3}, 2},
		{"[0,1,3,2,4] 2", []int{0, 1, 3, 2, 4}, 2},
		{"[0,1,2,3,6] 6", []int{0, 1, 2, 3, 6}, 6},
		{"[1,2,3,4,5,6,7,8] 6", []int{1, 2, 3, 4, 5, 6, 7, 8}, 6},
		{"[1,8,6,5,4,7,2,3] 6", []int{1, 8, 6, 5, 4, 7, 2, 3}, 6},
		{"[1,2,3,4,5,6,7,8,9] 9", []int{1, 2, 3, 4, 5, 6, 7, 8, 9}, 9},
	}

	for _, test := range tests {
		t.Run(test.CaseName, func(t *testing.T) {
			test := test // capture range variable
			t.Parallel()

			fm := NewForkManager(dm, createBlockLoader(test.PickBlockIndexes, test.PickBlockIndexes[0]), testBlocks[0])
			newBlock := fm.ChooseNewFork(testBlocks[0])
			assert.Equal(t, test.ExpectBlockIndex, getTestBlockIndex(newBlock))
		})
	}
}

// test normal cases
func TestForkManager_UpdateFork(t *testing.T) {
	type testChooseForkData struct {
		PickBlockIndexes []int
		CurrentHeadIndex int
		NewHeadIndex     int
		StableBlockIndex int
		ExpectHeadIndex  int
	}
	//       ┌─2
	// 0───1─┼─3───6
	//       ├─4─┬─7───9
	//       │   └─8
	//       └─5
	var tests = []testChooseForkData{
		// current=2 new=3
		{[]int{0, 1, 2, 3}, 2, 3, 0, -1},
		{[]int{0, 1, 2, 3}, 2, 3, 1, -1},
		{[]int{0, 1, 2, 3}, 2, 3, 3, 3},
		// current=3 new=2
		{[]int{0, 1, 2, 3}, 3, 2, 0, -1},
		{[]int{0, 1, 2, 3}, 3, 2, 1, -1},
		{[]int{0, 1, 2, 3}, 3, 2, 2, 2},
		// current=2 new=6
		{[]int{0, 1, 2, 3, 6}, 2, 6, 0, -1},
		{[]int{0, 1, 2, 3, 6}, 2, 6, 1, 6},
		{[]int{0, 1, 2, 3, 6}, 2, 6, 6, 6},
		// current=3 new=6
		{[]int{0, 1, 2, 3, 6}, 3, 6, 0, 6},
		{[]int{0, 1, 2, 3, 6}, 3, 6, 1, 6},
		{[]int{0, 1, 2, 3, 6}, 3, 6, 6, 6},
		// current=6 new=2
		{[]int{0, 1, 2, 3, 6}, 6, 2, 0, -1},
		{[]int{0, 1, 2, 3, 6}, 6, 2, 1, -1},
		{[]int{0, 1, 2, 3, 6}, 6, 2, 2, 2},
		// current=2 new=6
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 2, 6, 0, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 2, 6, 2, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 2, 6, 6, 6},
		// current=2 new=9
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 2, 9, 0, 9},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 2, 9, 2, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 2, 9, 9, 9},
		// current=6 new=9
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 6, 9, 0, 9},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 6, 9, 1, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 6, 9, 3, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 6, 9, 6, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 6, 9, 9, 9},
		// current=6 new=8
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 6, 8, 0, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 6, 8, 1, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 6, 8, 3, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 6, 8, 6, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 6, 8, 8, 8},
		// current=8 new=6
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 6, 0, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 6, 1, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 6, 4, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 6, 6, 6},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 6, 8, -1},
	}

	for _, test := range tests {
		caseName := fmt.Sprintf("current=%d,new=%d,stable=%d,blockCount=%d,newHead=%d", test.CurrentHeadIndex, test.NewHeadIndex, test.StableBlockIndex, len(test.PickBlockIndexes), test.ExpectHeadIndex)
		t.Run(caseName, func(t *testing.T) {
			test := test // capture range variable
			t.Parallel()

			dm := initDeputyManager(3)

			fm := NewForkManager(dm, createBlockLoader(test.PickBlockIndexes, test.StableBlockIndex), testBlocks[test.CurrentHeadIndex])
			newBlock := fm.UpdateFork(testBlocks[test.NewHeadIndex], testBlocks[test.StableBlockIndex])
			assert.Equal(t, test.ExpectHeadIndex, getTestBlockIndex(newBlock))
		})
	}
}

func TestForkManager_UpdateForkForConfirm(t *testing.T) {
	type testChooseForkData struct {
		PickBlockIndexes []int
		CurrentHeadIndex int
		StableBlockIndex int
		ExpectHeadIndex  int
	}
	//       ┌─2
	// 0───1─┼─3───6
	//       ├─4─┬─7───9
	//       │   └─8
	//       └─5
	var tests = []testChooseForkData{
		// current=2
		{[]int{0, 1, 2, 3}, 2, 0, -1},
		{[]int{0, 1, 2, 3}, 2, 1, -1},
		{[]int{0, 1, 2, 3}, 2, 2, -1},
		{[]int{0, 1, 2, 3}, 2, 3, 3},
		// current=2
		{[]int{0, 1, 2, 3, 6}, 2, 0, -1},
		{[]int{0, 1, 2, 3, 6}, 2, 1, -1},
		{[]int{0, 1, 2, 3, 6}, 2, 2, -1},
		{[]int{0, 1, 2, 3, 6}, 2, 3, 6},
		{[]int{0, 1, 2, 3, 6}, 2, 6, 6},
		// current=6
		{[]int{0, 1, 2, 3, 6}, 6, 0, -1},
		{[]int{0, 1, 2, 3, 6}, 6, 1, -1},
		{[]int{0, 1, 2, 3, 6}, 6, 2, 2},
		{[]int{0, 1, 2, 3, 6}, 6, 3, -1},
		{[]int{0, 1, 2, 3, 6}, 6, 6, -1},
		// current=2
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 2, 0, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 2, 1, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 2, 2, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 2, 3, 6},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 2, 4, 7},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 2, 5, 5},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 2, 6, 6},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 2, 7, 7},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 2, 8, 8},
		// current=6
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 6, 0, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 6, 1, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 6, 2, 2},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 6, 3, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 6, 4, 7},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 6, 5, 5},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 6, 6, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 6, 7, 7},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 6, 8, 8},
		// current=8
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 0, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 1, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 2, 2},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 3, 6},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 4, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 5, 5},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 6, 6},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 7, 7},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 8, -1},
		// current=6
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 6, 0, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 6, 1, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 6, 2, 2},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 6, 3, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 6, 6, -1},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 6, 7, 9},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 6, 8, 8},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 6, 9, 9},
	}

	for _, test := range tests {
		caseName := fmt.Sprintf("current=%d,stable=%d,blockCount=%d,newHead=%d", test.CurrentHeadIndex, test.StableBlockIndex, len(test.PickBlockIndexes), test.ExpectHeadIndex)
		t.Run(caseName, func(t *testing.T) {
			test := test // capture range variable
			t.Parallel()

			dm := initDeputyManager(3)

			fm := NewForkManager(dm, createBlockLoader(test.PickBlockIndexes, test.StableBlockIndex), testBlocks[test.CurrentHeadIndex])
			newBlock := fm.UpdateForkForConfirm(testBlocks[test.StableBlockIndex])
			assert.Equal(t, test.ExpectHeadIndex, getTestBlockIndex(newBlock))
		})
	}
}

func TestForkManager_needSwitchFork(t *testing.T) {
	type testChooseForkData struct {
		CaseName         string
		DeputiesCount    int
		PickBlockIndexes []int
		CurrentHeadIndex int
		NewHeadIndex     int
		ExpectSwitched   bool
	}
	//       ┌─2
	// 0───1─┼─3───6
	//       ├─4─┬─7───9
	//       │   └─8
	//       └─5
	stableBlock := testBlocks[0]
	var tests = []testChooseForkData{
		{"not switch for same head", 1, []int{0, 1, 2}, 2, 2, false},
		{"not switch for same height head", 1, []int{0, 1, 2, 3}, 2, 3, false},
		{"not switch for lower height head", 1, []int{0, 1, 2, 3, 6}, 6, 2, false},
		{"switch for higher head", 1, []int{0, 1, 2, 3, 6}, 2, 6, true},
		{"not switch for distance 3 and 2 deputies", 2, []int{0, 1, 2, 3, 6}, 2, 6, false},
		{"switch for distance 4 and 2 deputies", 2, []int{0, 1, 2, 4, 7, 8, 9}, 2, 9, true},
		{"switch for distance 4 and 3 deputies", 3, []int{0, 1, 2, 4, 7, 8, 9}, 2, 9, true},
		{"not switch for distance 3 and 3 deputies", 3, []int{0, 1, 2, 3, 6}, 2, 6, false},
	}

	for _, test := range tests {
		t.Run(test.CaseName, func(t *testing.T) {
			test := test // capture range variable
			t.Parallel()

			dm := initDeputyManager(test.DeputiesCount)
			currentBlock := testBlocks[test.CurrentHeadIndex]
			fm := NewForkManager(dm, createBlockLoader(test.PickBlockIndexes, 0), currentBlock)
			needSwitch := fm.needSwitchFork(currentBlock, testBlocks[test.NewHeadIndex], stableBlock)
			assert.Equal(t, test.ExpectSwitched, needSwitch)
		})
	}
}

func TestForkManager_isCurrentForkCut(t *testing.T) {
	dm := deputynode.NewManager(3, &testBlockLoader{})

	//       ┌─2
	// 0───1─┼─3───6
	//       ├─4─┬─7───9
	//       │   └─8
	//       └─5
	fm := NewForkManager(dm, createBlockLoader([]int{0, 1, 2, 3, 6}, 3), testBlocks[2])
	cut := fm.isCurrentForkCut()
	assert.Equal(t, true, cut)

	fm.SetHeadBlock(testBlocks[3])
	cut = fm.isCurrentForkCut()
	assert.Equal(t, true, cut)

	fm.SetHeadBlock(testBlocks[6])
	cut = fm.isCurrentForkCut()
	assert.Equal(t, false, cut)
}
