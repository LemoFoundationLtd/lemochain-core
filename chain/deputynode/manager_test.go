package deputynode

import (
	"crypto/ecdsa"
	"crypto/rand"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto/secp256k1"
	"github.com/stretchr/testify/assert"
	"math/big"
	"net"
	"testing"
)

var (
	testDeputies = GenerateDeputies(17)
)

// GenerateDeputies generate random deputy nodes
func GenerateDeputies(num int) DeputyNodes {
	var result []*DeputyNode
	for i := 0; i < num; i++ {
		private, _ := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
		result = append(result, &DeputyNode{
			MinerAddress: crypto.PubkeyToAddress(private.PublicKey),
			NodeID:       (crypto.FromECDSAPub(&private.PublicKey))[1:],
			IP:           net.IPv4(127, 0, 0, byte(i%256)),
			Port:         uint32(i % 9999),
			Rank:         uint32(i),
			Votes:        big.NewInt(int64(10000000000 - i)),
		})
	}
	return result
}

// pickNodes picks some test deputy nodes by index
func pickNodes(nodeIndexList ...int) DeputyNodes {
	var result []*DeputyNode
	for i, nodeIndex := range nodeIndexList {
		newDeputy := testDeputies[nodeIndex].Copy()
		// reset rank
		newDeputy.Rank = uint32(i)
		result = append(result, newDeputy)
	}
	return result
}

func TestManager_SaveSnapshot(t *testing.T) {
	m := NewManager(5)

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
	m := NewManager(5)

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
	m := NewManager(5)

	nodes := pickNodes(0, 1, 2)
	m.SaveSnapshot(0, nodes)

	assert.Equal(t, nodes[0], m.GetDeputyByAddress(0, testDeputies[0].MinerAddress))
	assert.Equal(t, nodes[2], m.GetDeputyByAddress(0, testDeputies[2].MinerAddress))
	// not exist
	assert.Nil(t, m.GetDeputyByAddress(0, testDeputies[5].MinerAddress))
	assert.Nil(t, m.GetDeputyByAddress(0, common.Address{}))
}

func TestManager_GetDeputyByNodeID(t *testing.T) {
	m := NewManager(5)

	nodes := pickNodes(0, 1, 2)
	m.SaveSnapshot(0, nodes)

	assert.Equal(t, nodes[0], m.GetDeputyByNodeID(0, testDeputies[0].NodeID))
	assert.Equal(t, nodes[2], m.GetDeputyByNodeID(0, testDeputies[2].NodeID))
	// not exist
	assert.Nil(t, m.GetDeputyByNodeID(0, testDeputies[5].NodeID))
	assert.Nil(t, m.GetDeputyByNodeID(0, []byte{}))
	assert.Nil(t, m.GetDeputyByNodeID(0, nil))
}

// test special cases
func TestManager_GetSlot(t *testing.T) {
	m := NewManager(5)

	nodes0 := pickNodes(0, 1, 2)
	m.SaveSnapshot(0, nodes0)
	nodes1 := pickNodes(1)
	m.SaveSnapshot(params.TermDuration, nodes1)
	nodes2 := pickNodes(2, 3, 4, 5)
	m.SaveSnapshot(params.TermDuration*2, nodes2)
	term0Height := uint32(10)
	term1RewardHeight := params.TermDuration + params.InterimDuration + 1
	term2RewardHeight := params.TermDuration*2 + params.InterimDuration + 1

	// height is 0
	_, err := m.GetMinerDistance(0, common.Address{}, common.Address{})
	assert.Equal(t, ErrMineGenesis, err)

	// not exist target miner
	_, err = m.GetMinerDistance(term0Height, common.Address{}, common.Address{})
	assert.Equal(t, ErrNotDeputy, err)
	_, err = m.GetMinerDistance(term0Height, common.Address{}, testDeputies[5].MinerAddress)
	assert.Equal(t, ErrNotDeputy, err)

	// not exist last miner
	_, err = m.GetMinerDistance(term0Height, common.Address{}, testDeputies[0].MinerAddress)
	assert.Equal(t, ErrNotDeputy, err)
	_, err = m.GetMinerDistance(term0Height, testDeputies[5].MinerAddress, testDeputies[0].MinerAddress)
	assert.Equal(t, ErrNotDeputy, err)

	// only one deputy
	dis, err := m.GetMinerDistance(term1RewardHeight, common.Address{}, testDeputies[1].MinerAddress)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), dis)

	// first block
	dis, err = m.GetMinerDistance(1, common.Address{}, testDeputies[0].MinerAddress)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), dis)
	dis, err = m.GetMinerDistance(1, common.Address{}, testDeputies[2].MinerAddress)
	assert.NoError(t, err)
	assert.Equal(t, uint32(3), dis)

	// reward block
	dis, err = m.GetMinerDistance(term2RewardHeight, common.Address{}, testDeputies[2].MinerAddress)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), dis)
	dis, err = m.GetMinerDistance(term2RewardHeight, common.Address{}, testDeputies[5].MinerAddress)
	assert.NoError(t, err)
	assert.Equal(t, uint32(4), dis)
}

// test normal cases
func TestManager_GetSlot2(t *testing.T) {
	m := NewManager(5)

	nodes0 := pickNodes(0, 1, 2)
	m.SaveSnapshot(0, nodes0)
	nodes1 := pickNodes(1)
	m.SaveSnapshot(params.TermDuration, nodes1)
	nodes2 := pickNodes(2, 3, 4, 5)
	m.SaveSnapshot(params.TermDuration*2, nodes2)
	term0Height := uint32(10)
	term2RewardHeight := params.TermDuration*2 + params.InterimDuration + 1
	term2Height := term2RewardHeight + 10

	type testDistanceData struct {
		CaseName          string
		TargetHeight      uint32
		LastDeputyIndex   int
		TargetDeputyIndex int
		ExpectDis         uint32
	}
	var tests = []testDistanceData{
		{"[0,1,2] 2-0=2", term0Height, 0, 2, 2},
		{"[0,1,2] 0-2=1", term0Height, 2, 0, 1},
		{"[0,1,2] 2-2=0", term0Height, 2, 2, 0},
		{"[2,3,4,5] 3-2=1", term2Height, 2, 3, 1},
		{"[2,3,4,5] 4-2=2", term2Height, 2, 4, 2},
		{"[2,3,4,5] 4-4=0", term2Height, 4, 4, 0},
		{"[2,3,4,5] 2-5=1", term2Height, 5, 2, 1},
		{"[2,3,4,5] 2-3=3", term2Height, 3, 2, 3},
		{"[2,3,4,5] 2-2=0", term2Height, 2, 2, 0},
	}

	for _, test := range tests {
		t.Run(test.CaseName, func(t *testing.T) {
			test := test // capture range variable
			t.Parallel()

			lastBlockMiner := testDeputies[test.LastDeputyIndex].MinerAddress
			targetMiner := testDeputies[test.TargetDeputyIndex].MinerAddress
			dis, err := m.GetMinerDistance(test.TargetHeight, lastBlockMiner, targetMiner)
			assert.NoError(t, err)
			assert.Equal(t, test.ExpectDis, dis)
		})
	}
}
