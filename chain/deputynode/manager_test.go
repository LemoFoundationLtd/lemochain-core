package deputynode

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

var (
	testDeputies = GenerateDeputies(17)
)

type testBlockLoader map[uint32]*types.Block

func (loader testBlockLoader) GetBlockByHeight(height uint32) (*types.Block, error) {
	block, ok := loader[height]
	if !ok {
		return nil, store.ErrNotExist
	}
	return block, nil
}

// GenerateDeputies generate random deputy nodes
func GenerateDeputies(num int) types.DeputyNodes {
	var result []*types.DeputyNode
	for i := 0; i < num; i++ {
		private, _ := crypto.GenerateKey()
		result = append(result, &types.DeputyNode{
			MinerAddress: crypto.PubkeyToAddress(private.PublicKey),
			NodeID:       crypto.PrivateKeyToNodeID(private),
			Rank:         uint32(i),
			Votes:        big.NewInt(int64(10000000000 - i)),
		})
	}
	return result
}

// pickNodes picks some test deputy nodes by index
func pickNodes(nodeIndexList ...int) types.DeputyNodes {
	var result []*types.DeputyNode
	for i, nodeIndex := range nodeIndexList {
		newDeputy := testDeputies[nodeIndex].Copy()
		// reset rank
		newDeputy.Rank = uint32(i)
		result = append(result, newDeputy)
	}
	return result
}

func TestNewManager(t *testing.T) {
	// no blocks
	loader := testBlockLoader{}
	m := NewManager(5, loader)
	assert.Len(t, m.termList, 0)

	// invalid term
	loader = testBlockLoader{}
	loader[0] = &types.Block{Header: &types.Header{Height: 0, Time: 100}}
	assert.PanicsWithValue(t, ErrNoDeputyInBlock, func() {
		m = NewManager(5, loader)
	})

	// 2 terms
	loader = testBlockLoader{}
	block0 := &types.Block{Header: &types.Header{Height: 0, Time: 100}, DeputyNodes: pickNodes(0, 1)}
	loader[0] = block0
	loader[1] = &types.Block{Header: &types.Header{Height: 1, Time: 101}}
	block2 := &types.Block{Header: &types.Header{Height: params.TermDuration, Time: 200}, DeputyNodes: pickNodes(0, 1, 2)}
	loader[params.TermDuration] = block2
	m = NewManager(5, loader)
	assert.Len(t, m.termList, 2)
	assert.Equal(t, uint32(0), m.termList[0].TermIndex)
	assert.Equal(t, block0.DeputyNodes, m.termList[0].Nodes)
	assert.Equal(t, uint32(1), m.termList[1].TermIndex)
	assert.Equal(t, block2.DeputyNodes, m.termList[1].Nodes)
}

func TestManager_SaveSnapshot(t *testing.T) {
	m := NewManager(5, testBlockLoader{})

	// save genesis
	height := uint32(0)
	nodes := pickNodes(0, 1)
	m.SaveSnapshot(height, nodes)
	assert.Len(t, m.termList, 1)
	assert.Equal(t, uint32(0), m.termList[0].TermIndex)
	assert.Equal(t, nodes, m.termList[0].Nodes)

	// save snapshot
	height = uint32(params.TermDuration * 1)
	nodes = pickNodes(2)
	m.SaveSnapshot(height, nodes)
	assert.Len(t, m.termList, 2)
	assert.Equal(t, uint32(0), m.termList[0].TermIndex)
	assert.Equal(t, uint32(1), m.termList[1].TermIndex)
	assert.Equal(t, nodes, m.termList[1].Nodes)

	// save exist node
	height = uint32(params.TermDuration * 2)
	nodes = pickNodes(1, 3)
	m.SaveSnapshot(height, nodes)
	assert.Len(t, m.termList, 3)
	assert.Equal(t, uint32(2), m.termList[2].TermIndex)
	assert.Equal(t, nodes, m.termList[2].Nodes)

	// save nothing
	height = uint32(params.TermDuration * 3)
	nodes = pickNodes()
	assert.PanicsWithValue(t, ErrNoDeputyInBlock, func() {
		m.SaveSnapshot(height, nodes)
	})

	// save exist snapshot height
	height = uint32(params.TermDuration * 2)
	nodes = pickNodes(4)
	m.SaveSnapshot(height, nodes)
	assert.Len(t, m.termList, 3)
	assert.Equal(t, nodes, m.termList[2].Nodes)

	// save exist snapshot height then drop the terms after it
	height = uint32(params.TermDuration * 1)
	nodes = pickNodes(5)
	m.SaveSnapshot(height, nodes)
	assert.Len(t, m.termList, 2)

	// save skipped snapshot
	height = uint32(params.TermDuration * 3)
	nodes = pickNodes(4)
	assert.PanicsWithValue(t, ErrMissingTerm, func() {
		m.SaveSnapshot(height, nodes)
	})
}

func TestManager_GetTermByHeight(t *testing.T) {
	m := NewManager(5, testBlockLoader{})

	// no any terms
	_, err := m.GetTermByHeight(0)
	assert.Equal(t, ErrNoTerms, err)

	nodes0 := pickNodes(0, 1)
	m.SaveSnapshot(0, nodes0)
	nodes1 := pickNodes(0, 1, 2)
	m.SaveSnapshot(params.TermDuration*1, nodes1)
	nodes2 := pickNodes(1, 2, 3, 4, 5, 6)
	m.SaveSnapshot(params.TermDuration*2, nodes2)

	// genesis term
	term, err := m.GetTermByHeight(0)
	assert.NoError(t, err)
	assert.Equal(t, uint32(0), term.TermIndex)
	assert.Equal(t, nodes0, term.Nodes)
	term, err = m.GetTermByHeight(1)
	assert.NoError(t, err)
	assert.Equal(t, uint32(0), term.TermIndex)
	assert.Equal(t, nodes0, term.Nodes)
	term, err = m.GetTermByHeight(params.TermDuration + params.InterimDuration)
	assert.NoError(t, err)
	assert.Equal(t, uint32(0), term.TermIndex)
	assert.Equal(t, nodes0, term.Nodes)

	// second term
	term, err = m.GetTermByHeight(params.TermDuration + params.InterimDuration + 1)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), term.TermIndex)
	assert.Equal(t, nodes1, term.Nodes)
	term, err = m.GetTermByHeight(params.TermDuration*2 + params.InterimDuration)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), term.TermIndex)
	assert.Equal(t, nodes1, term.Nodes)

	// third term
	term, err = m.GetTermByHeight(params.TermDuration*2 + params.InterimDuration + 1)
	assert.NoError(t, err)
	assert.Equal(t, uint32(2), term.TermIndex)
	assert.Equal(t, nodes2, term.Nodes)

	// not exist term
	term, err = m.GetTermByHeight(1000000000)
	assert.Equal(t, ErrQueryFutureTerm, err)
}

func TestManager_GetDeputyByAddress(t *testing.T) {
	m := NewManager(5, testBlockLoader{})

	nodes := pickNodes(0, 1, 2)
	m.SaveSnapshot(0, nodes)

	assert.Equal(t, nodes[0], m.GetDeputyByAddress(0, testDeputies[0].MinerAddress))
	assert.Equal(t, nodes[2], m.GetDeputyByAddress(0, testDeputies[2].MinerAddress))
	// not exist
	assert.Nil(t, m.GetDeputyByAddress(0, testDeputies[5].MinerAddress))
	assert.Nil(t, m.GetDeputyByAddress(0, common.Address{}))
}

func TestManager_GetDeputyByNodeID(t *testing.T) {
	m := NewManager(5, testBlockLoader{})

	nodes := pickNodes(0, 1, 2)
	m.SaveSnapshot(0, nodes)

	assert.Equal(t, nodes[0], m.GetDeputyByNodeID(0, testDeputies[0].NodeID))
	assert.Equal(t, nodes[2], m.GetDeputyByNodeID(0, testDeputies[2].NodeID))
	// not exist
	assert.Nil(t, m.GetDeputyByNodeID(0, testDeputies[5].NodeID))
	assert.Nil(t, m.GetDeputyByNodeID(0, []byte{}))
	assert.Nil(t, m.GetDeputyByNodeID(0, nil))
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
	dm := NewManager(5, &testBlockLoader{})

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
	_, err := dm.GetMinerDistance(0, common.Address{}, common.Address{})
	assert.Equal(t, ErrMineGenesis, err)

	// not exist target miner
	_, err = dm.GetMinerDistance(term0Height, common.Address{}, common.Address{})
	assert.Equal(t, ErrNotDeputy, err)
	_, err = dm.GetMinerDistance(term0Height, common.Address{}, testDeputies[5].MinerAddress)
	assert.Equal(t, ErrNotDeputy, err)

	// not exist last miner
	_, err = dm.GetMinerDistance(term0Height, common.Address{}, testDeputies[0].MinerAddress)
	assert.Equal(t, ErrNotDeputy, err)
	_, err = dm.GetMinerDistance(term0Height, testDeputies[5].MinerAddress, testDeputies[0].MinerAddress)
	assert.Equal(t, ErrNotDeputy, err)

	// only one deputy
	dis, err := dm.GetMinerDistance(term1RewardHeight, common.Address{}, testDeputies[1].MinerAddress)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), dis)

	// first block
	dis, err = dm.GetMinerDistance(1, common.Address{}, testDeputies[0].MinerAddress)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), dis)
	dis, err = dm.GetMinerDistance(1, common.Address{}, testDeputies[2].MinerAddress)
	assert.NoError(t, err)
	assert.Equal(t, uint32(3), dis)

	// reward block
	dis, err = dm.GetMinerDistance(term2RewardHeight, common.Address{}, testDeputies[2].MinerAddress)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), dis)
	dis, err = dm.GetMinerDistance(term2RewardHeight, common.Address{}, testDeputies[5].MinerAddress)
	assert.NoError(t, err)
	assert.Equal(t, uint32(4), dis)

	// no deputies
	dm = NewManager(0, &testBlockLoader{})
	_, err = dm.GetMinerDistance(10, common.Address{}, common.Address{})
	assert.Equal(t, ErrNotDeputy, err)
}

// test normal cases
func TestGetMinerDistance(t *testing.T) {
	dm := NewManager(5, &testBlockLoader{})

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
			dis, err := dm.GetMinerDistance(test.TargetHeight, lastBlockMiner, targetMiner)
			assert.NoError(t, err)
			assert.Equal(t, test.ExpectDistance, dis)
		})
	}
}

func TestGetNextMineWindow(t *testing.T) {
	deputyCount := 3
	dm := NewManager(deputyCount, &testBlockLoader{})
	dm.SaveSnapshot(0, testDeputies[:deputyCount])
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
			windowFrom, windowTo := dm.GetNextMineWindow(1, test.distance, parentBlockTime, parentBlockTime+test.timeDistance, mineTimeout)
			assert.Equal(t, parentBlockTime+test.wantFrom, windowFrom)
			assert.Equal(t, parentBlockTime+test.wantTo, windowTo)
		})
	}
}

func TestGetCorrectMiner_Error(t *testing.T) {
	deputyCount := 3
	dm := NewManager(deputyCount, &testBlockLoader{})
	dm.SaveSnapshot(0, testDeputies[:deputyCount])
	var mineTimeout int64 = 2000

	parent := &types.Header{Time: uint32(time.Now().Unix())}
	_, err := dm.GetCorrectMiner(parent, int64(parent.Time-10)*1000, mineTimeout)
	assert.Equal(t, ErrSmallerMineTime, err)

	parent = &types.Header{Time: uint32(time.Now().Unix()), Height: 1, MinerAddress: common.HexToAddress("0x123")}
	_, err = dm.GetCorrectMiner(parent, int64(parent.Time+10)*1000, mineTimeout)
	assert.Equal(t, ErrNotDeputy, err)
}

func TestGetCorrectMiner(t *testing.T) {
	deputyCount := 3
	dm := NewManager(deputyCount, &testBlockLoader{})
	dm.SaveSnapshot(0, testDeputies[:deputyCount])
	dm.SaveSnapshot(params.TermDuration, testDeputies[:deputyCount])
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
			miner, err := dm.GetCorrectMiner(parent, parentBlockTime+test.timeDistance, mineTimeout)
			assert.NoError(t, err)
			assert.Equal(t, testDeputies[test.wantMinerIndex].MinerAddress, miner)
		})
	}
}

func TestGetCorrectMiner_CrossTerm(t *testing.T) {
	deputyCount := 3
	dm := NewManager(deputyCount, &testBlockLoader{})
	dm.SaveSnapshot(0, testDeputies[:deputyCount])
	// different deputies in the second term
	term2Deputies := GenerateDeputies(deputyCount)
	dm.SaveSnapshot(params.TermDuration, term2Deputies)
	type testInfo struct {
		parentHeight uint32
		// for the first term
		parentMinerIndex int
		timeDistance     int64
		wantDeputy       *types.DeputyNode
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
			miner, err := dm.GetCorrectMiner(parent, parentBlockTime+test.timeDistance, mineTimeout)
			assert.NoError(t, err)
			assert.Equal(t, test.wantDeputy.MinerAddress, miner)
		})
	}
}
