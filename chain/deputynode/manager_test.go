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

func TestManager_PutEvilDeputyNode(t *testing.T) {
	m := NewManager(5, testBlockLoader{})

	a1 := common.HexToAddress("1")
	a2 := common.HexToAddress("2")
	m.PutEvilDeputyNode(a1, 0)
	m.PutEvilDeputyNode(a1, 10)
	m.PutEvilDeputyNode(a2, 100)

	assert.Equal(t, params.ReleaseEvilNodeDuration+10, m.evilDeputies[a1])
	assert.Equal(t, params.ReleaseEvilNodeDuration+100, m.evilDeputies[a2])
	_, ok := m.evilDeputies[common.HexToAddress("3")]
	assert.Equal(t, false, ok)
}

func TestManager_IsEvilDeputyNode(t *testing.T) {
	m := NewManager(5, testBlockLoader{})

	a1 := common.HexToAddress("1")
	a2 := common.HexToAddress("2")
	m.PutEvilDeputyNode(a1, 0)
	m.PutEvilDeputyNode(a1, 10)
	m.PutEvilDeputyNode(a2, 100)

	assert.Equal(t, true, m.IsEvilDeputyNode(a1, 0))
	assert.Equal(t, true, m.IsEvilDeputyNode(a1, params.ReleaseEvilNodeDuration+9))
	assert.Equal(t, false, m.IsEvilDeputyNode(a1, params.ReleaseEvilNodeDuration+10))
	_, ok := m.evilDeputies[common.HexToAddress("1")]
	assert.Equal(t, false, ok)

	assert.Equal(t, true, m.IsEvilDeputyNode(a2, params.ReleaseEvilNodeDuration+10))
	assert.Equal(t, false, m.IsEvilDeputyNode(a2, params.ReleaseEvilNodeDuration+101))
	_, ok = m.evilDeputies[a2]
	assert.Equal(t, false, ok)
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
	assert.Equal(t, ErrNoStableTerm, err)

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
	assert.Equal(t, ErrNoStableTerm, err)
}

func TestManager_GetDeputiesByHeight(t *testing.T) {
	m := NewManager(5, testBlockLoader{})

	nodes0 := pickNodes(0, 1)
	m.SaveSnapshot(0, nodes0)
	nodes1 := pickNodes(1, 2, 3)
	m.SaveSnapshot(params.TermDuration, nodes1)

	nodes := m.GetDeputiesByHeight(0)
	assert.Equal(t, nodes0, nodes)
	nodes = m.GetDeputiesByHeight(params.TermDuration + params.InterimDuration)
	assert.Equal(t, nodes0, nodes)
	nodes = m.GetDeputiesByHeight(params.TermDuration + params.InterimDuration + 1)
	assert.Equal(t, nodes1, nodes)
	nodes = m.GetDeputiesByHeight(params.TermDuration*2 + params.InterimDuration + 1)
	assert.Empty(t, nodes)
}

func TestManager_GetDeputiesCount(t *testing.T) {
	m := NewManager(5, testBlockLoader{})

	nodes0 := pickNodes(0, 1)
	m.SaveSnapshot(0, nodes0)
	nodes1 := pickNodes(1, 2, 3)
	m.SaveSnapshot(params.TermDuration, nodes1)

	length := m.GetDeputiesCount(0)
	assert.Equal(t, 2, length)
	length = m.GetDeputiesCount(params.TermDuration + params.InterimDuration)
	assert.Equal(t, 2, length)
	length = m.GetDeputiesCount(params.TermDuration + params.InterimDuration + 1)
	assert.Equal(t, 3, length)
	length = m.GetDeputiesCount(params.TermDuration*2 + params.InterimDuration + 1)
	assert.Equal(t, 0, length)
}

func TestManager_TwoThirdDeputyCount(t *testing.T) {
	m := NewManager(5, testBlockLoader{})

	nodes0 := pickNodes(1)
	m.SaveSnapshot(0, nodes0)
	nodes1 := pickNodes(0, 1)
	m.SaveSnapshot(params.TermDuration*1, nodes1)
	nodes2 := pickNodes(2, 3, 4)
	m.SaveSnapshot(params.TermDuration*2, nodes2)

	count := m.TwoThirdDeputyCount(0)
	assert.Equal(t, uint32(1), count)
	count = m.TwoThirdDeputyCount(params.TermDuration + params.InterimDuration)
	assert.Equal(t, uint32(1), count)
	count = m.TwoThirdDeputyCount(params.TermDuration + params.InterimDuration + 1)
	assert.Equal(t, uint32(2), count)
	count = m.TwoThirdDeputyCount(params.TermDuration*2 + params.InterimDuration + 1)
	assert.Equal(t, uint32(2), count)
	count = m.TwoThirdDeputyCount(params.TermDuration*3 + params.InterimDuration + 1)
	assert.Equal(t, uint32(0), count)
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

func TestManager_GetMyMinerAddress(t *testing.T) {
	m := NewManager(5, testBlockLoader{})

	private, err := crypto.GenerateKey()
	assert.NoError(t, err)
	SetSelfNodeKey(private)

	nodes1 := pickNodes(0, 1, 2)
	myNode := nodes1[2].Copy()
	myNode.MinerAddress = crypto.PubkeyToAddress(private.PublicKey)
	myNode.NodeID = crypto.PrivateKeyToNodeID(private)
	nodes1[2] = myNode
	m.SaveSnapshot(0, nodes1)
	nodes2 := pickNodes(0, 1, 3)
	m.SaveSnapshot(params.TermDuration, nodes2)

	addr, success := m.GetMyMinerAddress(0)
	assert.Equal(t, myNode.MinerAddress, addr)
	assert.Equal(t, true, success)

	addr, success = m.GetMyMinerAddress(params.TermDuration + params.InterimDuration)
	assert.Equal(t, myNode.MinerAddress, addr)
	assert.Equal(t, true, success)

	addr, success = m.GetMyMinerAddress(params.TermDuration + params.InterimDuration + 1)
	assert.Equal(t, common.Address{}, addr)
	assert.Equal(t, false, success)

	addr, success = m.GetMyMinerAddress(params.TermDuration*2 + params.InterimDuration + 1)
	assert.Equal(t, common.Address{}, addr)
	assert.Equal(t, false, success)
}

func TestManager_GetDeputyByDistance_Error(t *testing.T) {
	m := NewManager(5, testBlockLoader{})

	nodes := pickNodes(0, 1, 2)
	m.SaveSnapshot(params.TermDuration, nodes)

	// invalid targetHeight
	assert.PanicsWithValue(t, ErrMineGenesis, func() {
		_, _ = m.GetDeputyByDistance(0, testDeputies[0].MinerAddress, 0)
	})

	// invalid distance
	assert.PanicsWithValue(t, ErrInvalidDistance, func() {
		_, _ = m.GetDeputyByDistance(params.TermDuration+params.InterimDuration+1, testDeputies[0].MinerAddress, 0)
	})

	// future term
	_, err := m.GetDeputyByDistance(params.TermDuration*2+params.InterimDuration+1, testDeputies[0].MinerAddress, 1)
	assert.Equal(t, ErrNotDeputy, err)

	// unknown deputy
	_, err = m.GetDeputyByDistance(params.TermDuration+params.InterimDuration+2, testDeputies[5].MinerAddress, 1)
	assert.Equal(t, ErrNotDeputy, err)
}

func TestManager_GetDeputyByDistance(t *testing.T) {
	type testInfo struct {
		targetHeight uint32
		parentDeputy *types.DeputyNode
		distance     uint32
		expectDeputy *types.DeputyNode
	}

	m := NewManager(5, testBlockLoader{})
	nodes1 := pickNodes(0, 1, 2)
	m.SaveSnapshot(0, nodes1)
	nodes2 := pickNodes(3, 4)
	m.SaveSnapshot(params.TermDuration, nodes2)

	tests := []testInfo{
		// parent and target are in different terms
		{1, nodes1[0], 1, nodes1[0]},
		{1, nodes1[0], 2, nodes1[1]},
		{1, nodes1[1], 3, nodes1[2]},
		{1, nodes1[2], 4, nodes1[0]},
		{params.TermDuration + params.InterimDuration + 1, nodes2[0], 1, nodes2[0]},
		{params.TermDuration + params.InterimDuration + 1, nodes2[1], 2, nodes2[1]},
		{params.TermDuration + params.InterimDuration + 1, nodes2[0], 1, nodes2[0]},
		{params.TermDuration + params.InterimDuration + 1, nodes2[1], 11, nodes2[0]},
		// parent and target are in same term
		{2, nodes1[0], 1, nodes1[1]},
		{2, nodes1[0], 2, nodes1[2]},
		{3, nodes1[0], 3, nodes1[0]},
		{2, nodes1[1], 1, nodes1[2]},
		{2, nodes1[1], 2, nodes1[0]},
		{2, nodes1[1], 3, nodes1[1]},
		{params.TermDuration + params.InterimDuration + 2, nodes2[0], 1, nodes2[1]},
		{params.TermDuration + params.InterimDuration + 2, nodes2[0], 2, nodes2[0]},
		{params.TermDuration + params.InterimDuration + 2, nodes2[1], 2, nodes2[1]},
		{params.TermDuration + params.InterimDuration + 2, nodes2[1], 20, nodes2[1]},
	}

	for i, test := range tests {
		caseName := fmt.Sprintf("case %d. height=%d. distance=%d", i, test.targetHeight, test.distance)
		t.Run(caseName, func(t *testing.T) {
			test := test // capture range variable
			t.Parallel()

			node, err := m.GetDeputyByDistance(test.targetHeight, test.parentDeputy.MinerAddress, test.distance)
			assert.NoError(t, err)
			assert.Equal(t, test.expectDeputy, node)
		})
	}
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
	assert.PanicsWithValue(t, ErrMineGenesis, func() {
		_, _ = dm.GetMinerDistance(0, common.Address{}, common.Address{})
	})

	// not exist target miner
	_, err := dm.GetMinerDistance(term0Height, common.Address{}, common.Address{})
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
