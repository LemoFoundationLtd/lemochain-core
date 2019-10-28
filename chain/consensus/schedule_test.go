package consensus

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_findDeputyByAddress(t *testing.T) {
	// no deputies
	node := findDeputyByAddress([]*types.DeputyNode{}, testDeputies[0].MinerAddress)
	assert.Nil(t, node)

	// not match any one
	node = findDeputyByAddress(pickNodes(0), testDeputies[1].MinerAddress)
	assert.Nil(t, node)

	// match one
	node = findDeputyByAddress(pickNodes(0, 1, 2), testDeputies[1].MinerAddress)
	assert.Equal(t, testDeputies[1].DeputyNode, *node)
}

func Test_findDeputyByRank(t *testing.T) {
	// no deputies
	node := findDeputyByRank([]*types.DeputyNode{}, testDeputies[0].Rank)
	assert.Nil(t, node)

	// not match any one
	node = findDeputyByRank(pickNodes(0), testDeputies[1].Rank)
	assert.Nil(t, node)

	// match one
	node = findDeputyByRank(pickNodes(0, 1, 2), testDeputies[1].Rank)
	assert.Equal(t, testDeputies[1].DeputyNode, *node)
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
	assert.Equal(t, uint32(1), dis)

	// first block
	dis, err = GetMinerDistance(1, common.Address{}, testDeputies[0].MinerAddress, dm)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), dis)
	dis, err = GetMinerDistance(1, common.Address{}, testDeputies[2].MinerAddress, dm)
	assert.NoError(t, err)
	assert.Equal(t, uint32(3), dis)

	// reward block
	dis, err = GetMinerDistance(term2RewardHeight, common.Address{}, testDeputies[2].MinerAddress, dm)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), dis)
	dis, err = GetMinerDistance(term2RewardHeight, common.Address{}, testDeputies[5].MinerAddress, dm)
	assert.NoError(t, err)
	assert.Equal(t, uint32(4), dis)

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
		ExpectDistance    uint32
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

func TestGetNextMineWindow(t *testing.T) {
	deputyCount := 3
	dm := deputynode.NewManager(deputyCount, &testBlockLoader{})
	dm.SaveSnapshot(0, testDeputies.ToDeputyNodes()[:deputyCount])
	type testInfo struct {
		distance     uint32
		timeDistance int64
		wantFrom     int64
		wantTo       int64
	}

	var blockInterval int64 = 1000
	var mineTimeout int64 = 2000
	oneLoopTime := mineTimeout * int64(deputyCount)
	parentBlockTime := int64(1000)
	tests := []testInfo{
		// next miner
		{1, 0, 0, mineTimeout * 1},
		{1, 10, 0, mineTimeout * 1},
		{1, blockInterval, 0, mineTimeout * 1},
		{1, mineTimeout, oneLoopTime, mineTimeout*1 + oneLoopTime},
		{1, oneLoopTime, oneLoopTime, mineTimeout*1 + oneLoopTime},
		{1, oneLoopTime + 10, oneLoopTime, mineTimeout*1 + oneLoopTime},
		// second miner
		{2, 0, mineTimeout, mineTimeout * 2},
		{2, 10, mineTimeout, mineTimeout * 2},
		{2, mineTimeout, mineTimeout, mineTimeout * 2},
		{2, mineTimeout * 2, mineTimeout + oneLoopTime, mineTimeout*2 + oneLoopTime},
		{2, oneLoopTime, mineTimeout + oneLoopTime, mineTimeout*2 + oneLoopTime},
		{2, oneLoopTime + 10, mineTimeout + oneLoopTime, mineTimeout*2 + oneLoopTime},
		// self miner
		{3, 0, mineTimeout * 2, mineTimeout * 3},
		{3, 10, mineTimeout * 2, mineTimeout * 3},
		{3, mineTimeout, mineTimeout * 2, mineTimeout * 3},
		{3, mineTimeout * 3, mineTimeout*2 + oneLoopTime, mineTimeout*3 + oneLoopTime},
		{3, oneLoopTime, mineTimeout*2 + oneLoopTime, mineTimeout*3 + oneLoopTime},
		{3, oneLoopTime + 10, mineTimeout*2 + oneLoopTime, mineTimeout*3 + oneLoopTime},

		// parent block is future block
		{1, -10, 0, mineTimeout * 1},
		{1, -10000, 0, mineTimeout * 1},
		{2, -10, mineTimeout, mineTimeout * 2},
		{2, -10000, mineTimeout, mineTimeout * 2},
		{3, -10, mineTimeout * 2, mineTimeout * 3},
		{3, -10000, mineTimeout * 2, mineTimeout * 3},
	}
	for _, test := range tests {
		caseName := fmt.Sprintf("distance=%d,timeDistance=%d", test.distance, test.timeDistance)
		t.Run(caseName, func(t *testing.T) {
			test := test // capture range variable
			t.Parallel()
			windowFrom, windowTo := GetNextMineWindow(1, test.distance, parentBlockTime, parentBlockTime+test.timeDistance, mineTimeout, dm)
			assert.Equal(t, parentBlockTime+test.wantFrom, windowFrom)
			assert.Equal(t, parentBlockTime+test.wantTo, windowTo)
		})
	}
}

func TestGetCorrectMiner_Error(t *testing.T) {
	deputyCount := 3
	dm := deputynode.NewManager(deputyCount, &testBlockLoader{})
	dm.SaveSnapshot(0, testDeputies.ToDeputyNodes()[:deputyCount])
	var mineTimeout int64 = 2000

	parent := &types.Header{Time: uint32(time.Now().Unix())}
	_, err := GetCorrectMiner(parent, int64(parent.Time-10)*1000, mineTimeout, dm)
	assert.Equal(t, ErrSmallerMineTime, err)

	parent = &types.Header{Time: uint32(time.Now().Unix()), MinerAddress: common.HexToAddress("0x123")}
	_, err = GetCorrectMiner(parent, int64(parent.Time+10)*1000, mineTimeout, dm)
	assert.Equal(t, ErrNotDeputy, err)
}

func TestGetCorrectMiner(t *testing.T) {
	deputyCount := 3
	dm := deputynode.NewManager(deputyCount, &testBlockLoader{})
	dm.SaveSnapshot(0, testDeputies.ToDeputyNodes()[:deputyCount])
	dm.SaveSnapshot(params.TermDuration, testDeputies.ToDeputyNodes()[:deputyCount])
	type testInfo struct {
		parentHeight     uint32
		parentMinerIndex int
		timeDistance     int64
		wantMinerIndex   int
	}

	var blockInterval int64 = 1000
	var mineTimeout int64 = 2000
	oneLoopTime := mineTimeout * int64(deputyCount)
	parentBlockTime := int64(time.Now().Unix()) * 1000
	tests := []testInfo{
		// mine normal block
		{10, 0, 0, 1},
		{10, 0, 10, 1},
		{10, 0, blockInterval, 1},
		{10, 0, mineTimeout * 1, 2},
		{10, 0, mineTimeout * 2, 0},
		{10, 0, oneLoopTime, 1},
		{10, 1, 0, 2},
		{10, 1, 10, 2},
		{10, 1, blockInterval, 2},
		{10, 1, mineTimeout * 1, 0},
		{10, 1, mineTimeout * 2, 1},
		{10, 1, oneLoopTime, 2},
		{10, 2, 0, 0},
		{10, 2, 10, 0},
		{10, 2, blockInterval, 0},
		{10, 2, mineTimeout * 1, 1},
		{10, 2, mineTimeout * 2, 2},
		{10, 2, oneLoopTime, 0},
		// mine first block
		{0, 0, 0, 0},
		{0, 0, 10, 0},
		{0, 0, blockInterval, 0},
		{0, 0, mineTimeout * 1, 1},
		{0, 0, mineTimeout * 2, 2},
		{0, 0, mineTimeout * 3, 0},
		{0, 1, 0, 0},
		{0, 2, 0, 0},
		// mine reward block
		{params.TermDuration + params.InterimDuration, 0, 0, 0},
		{params.TermDuration + params.InterimDuration, 0, 10, 0},
		{params.TermDuration + params.InterimDuration, 0, blockInterval, 0},
		{params.TermDuration + params.InterimDuration, 0, mineTimeout * 1, 1},
		{params.TermDuration + params.InterimDuration, 0, mineTimeout * 2, 2},
		{params.TermDuration + params.InterimDuration, 0, mineTimeout * 3, 0},
		{params.TermDuration + params.InterimDuration, 1, 0, 0},
		{params.TermDuration + params.InterimDuration, 2, 0, 0},
	}
	for _, test := range tests {
		caseName := fmt.Sprintf("parentHeight=%d,parentMiner=%d,timeDistance=%d", test.parentHeight, test.parentMinerIndex, test.timeDistance)
		t.Run(caseName, func(t *testing.T) {
			test := test // capture range variable
			t.Parallel()
			parent := &types.Header{
				Height:       test.parentHeight,
				Time:         uint32(parentBlockTime / 1000),
				MinerAddress: testDeputies[test.parentMinerIndex].MinerAddress,
			}
			miner, err := GetCorrectMiner(parent, parentBlockTime+test.timeDistance, mineTimeout, dm)
			assert.NoError(t, err)
			assert.Equal(t, testDeputies[test.wantMinerIndex].MinerAddress, miner)
		})
	}
}
