package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewForkManager(t *testing.T) {
	dm := deputynode.NewManager(5, testBlockLoader{})

	// not set stable
	assert.PanicsWithValue(t, ErrNoHeadBlock, func() {
		NewForkManager(dm, createUnconfirmBlockLoader([]int{}), nil)
	})

	fm := NewForkManager(dm, createUnconfirmBlockLoader([]int{}), testBlocks[0])
	assert.Equal(t, testBlocks[0], fm.GetHeadBlock())
}

func TestForkManager_SetHeadBlock_GetHeadBlock(t *testing.T) {
	dm := deputynode.NewManager(5, testBlockLoader{})

	fm := NewForkManager(dm, createUnconfirmBlockLoader([]int{}), testBlocks[0])
	fm.SetHeadBlock(testBlocks[1])
	assert.Equal(t, testBlocks[1], fm.GetHeadBlock())
}

// test special cases
func TestGetMinerDistance_Error(t *testing.T) {
	dm := deputynode.NewManager(5, testBlockLoader{})

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
	dm = deputynode.NewManager(0, testBlockLoader{})
	_, err = GetMinerDistance(10, common.Address{}, common.Address{}, dm)
	assert.Equal(t, ErrNotDeputy, err)
}

// test normal cases
func TestGetMinerDistance(t *testing.T) {
	dm := deputynode.NewManager(5, testBlockLoader{})

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
	dm := deputynode.NewManager(5, testBlockLoader{})

	// no blocks
	fm := NewForkManager(dm, createUnconfirmBlockLoader([]int{}), testBlocks[0])
	newBlock := fm.ChooseNewFork()
	assert.Nil(t, newBlock)
}

// test normal cases
func TestForkManager_ChooseNewFork(t *testing.T) {
	dm := deputynode.NewManager(5, testBlockLoader{})

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
		{"[1,2,3,5,6,7,8] 6", []int{1, 2, 3, 5, 6, 7, 8}, 6},
		{"[1,8,6,5,7,2,3] 6", []int{1, 8, 6, 5, 7, 2, 3}, 6},
		{"[1,2,3,5,6,7,8,9] 9", []int{1, 2, 3, 5, 6, 7, 8, 9}, 9},
	}

	for _, test := range tests {
		t.Run(test.CaseName, func(t *testing.T) {
			test := test // capture range variable
			t.Parallel()

			fm := NewForkManager(dm, createUnconfirmBlockLoader(test.PickBlockIndexes), testBlocks[0])
			newBlock := fm.ChooseNewFork()
			assert.Equal(t, test.ExpectBlockIndex, getTestBlockIndex(newBlock))
		})
	}
}

func setTermDeputiesCount(dm *deputynode.Manager, snapshotHeight uint32, deputyCount int) {
	nodeIndexList := make([]int, deputyCount)
	for i := range nodeIndexList {
		nodeIndexList[i] = i
	}
	nodes := pickNodes(nodeIndexList...)
	dm.SaveSnapshot(snapshotHeight, nodes)
}

// test normal cases
func TestForkManager_TrySwitchFork(t *testing.T) {
	type testChooseForkData struct {
		CaseName         string
		DeputiesCount    int
		PickBlockIndexes []int
		ExpectBlockIndex int
		ExpectSwitched   bool
	}
	//       ┌─2
	// 0───1─┼─3───6
	//       ├─4─┬─7───9
	//       │   └─8
	//       └─5
	stableBlock := testBlocks[0]
	currentBlock := testBlocks[5]
	var tests = []testChooseForkData{
		{"not choose new fork", 5, []int{0, 1, 5}, 5, false},
		{"not switch if no higher block", 1, []int{0, 1, 2, 5}, 5, false},
		{"not switch for distance 3 and 2 deputies", 2, []int{0, 1, 2, 4, 5, 8}, 5, false},
		{"switch for distance 4 and 2 deputies", 2, []int{0, 1, 4, 5, 7, 8, 9}, 9, true},
		{"switch for distance 4 and 3 deputies", 3, []int{0, 1, 4, 5, 7, 8, 9}, 9, true},
	}

	for _, test := range tests {
		t.Run(test.CaseName, func(t *testing.T) {
			test := test // capture range variable
			t.Parallel()

			dm := deputynode.NewManager(test.DeputiesCount, testBlockLoader{})
			setTermDeputiesCount(dm, 0, test.DeputiesCount)
			fm := NewForkManager(dm, createUnconfirmBlockLoader(test.PickBlockIndexes), currentBlock)
			newBlock, switched := fm.TrySwitchFork(stableBlock)
			assert.Equal(t, test.ExpectSwitched, switched)
			assert.Equal(t, test.ExpectBlockIndex, getTestBlockIndex(newBlock))
		})
	}
}
