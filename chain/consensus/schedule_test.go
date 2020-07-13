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

	parent = &types.Header{Time: uint32(time.Now().Unix()), Height: 1, MinerAddress: common.HexToAddress("0x123")}
	_, err = GetCorrectMiner(parent, int64(parent.Time+10)*1000, mineTimeout, dm)
	assert.Equal(t, deputynode.ErrNotDeputy, err)
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

func TestGetCorrectMiner_CrossTerm(t *testing.T) {
	deputyCount := 3
	dm := deputynode.NewManager(deputyCount, &testBlockLoader{})
	dm.SaveSnapshot(0, testDeputies.ToDeputyNodes()[:deputyCount])
	// different deputies in the second term
	term2Deputies := generateDeputies(deputyCount)
	dm.SaveSnapshot(params.TermDuration, term2Deputies.ToDeputyNodes())
	type testInfo struct {
		parentHeight uint32
		// for the first term
		parentMinerIndex int
		timeDistance     int64
		wantDeputy       deputyTestData
	}

	var blockInterval int64 = 1000
	var mineTimeout int64 = 2000
	oneLoopTime := mineTimeout * int64(deputyCount)
	parentBlockTime := int64(time.Now().Unix()) * 1000
	tests := []testInfo{
		// mine normal block
		{10, 0, 0, testDeputies[1]},
		{10, 0, 10, testDeputies[1]},
		{10, 0, blockInterval, testDeputies[1]},
		{10, 0, mineTimeout * 1, testDeputies[2]},
		{10, 0, mineTimeout * 2, testDeputies[0]},
		{10, 0, oneLoopTime, testDeputies[1]},
		{10, 1, 0, testDeputies[2]},
		{10, 1, 10, testDeputies[2]},
		{10, 1, blockInterval, testDeputies[2]},
		{10, 1, mineTimeout * 1, testDeputies[0]},
		{10, 1, mineTimeout * 2, testDeputies[1]},
		{10, 1, oneLoopTime, testDeputies[2]},
		{10, 2, 0, testDeputies[0]},
		{10, 2, 10, testDeputies[0]},
		{10, 2, blockInterval, testDeputies[0]},
		{10, 2, mineTimeout * 1, testDeputies[1]},
		{10, 2, mineTimeout * 2, testDeputies[2]},
		{10, 2, oneLoopTime, testDeputies[0]},
		// mine first block
		{0, 0, 0, testDeputies[0]},
		{0, 0, 10, testDeputies[0]},
		{0, 0, blockInterval, testDeputies[0]},
		{0, 0, mineTimeout * 1, testDeputies[1]},
		{0, 0, mineTimeout * 2, testDeputies[2]},
		{0, 0, mineTimeout * 3, testDeputies[0]},
		{0, 1, 0, testDeputies[0]},
		{0, 2, 0, testDeputies[0]},
		// mine reward block
		{params.TermDuration + params.InterimDuration, 0, 0, term2Deputies[0]},
		{params.TermDuration + params.InterimDuration, 0, 10, term2Deputies[0]},
		{params.TermDuration + params.InterimDuration, 0, blockInterval, term2Deputies[0]},
		{params.TermDuration + params.InterimDuration, 0, mineTimeout * 1, term2Deputies[1]},
		{params.TermDuration + params.InterimDuration, 0, mineTimeout * 2, term2Deputies[2]},
		{params.TermDuration + params.InterimDuration, 0, mineTimeout * 3, term2Deputies[0]},
		{params.TermDuration + params.InterimDuration, 1, 0, term2Deputies[0]},
		{params.TermDuration + params.InterimDuration, 2, 0, term2Deputies[0]},
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
			assert.Equal(t, test.wantDeputy.MinerAddress, miner)
		})
	}
}
