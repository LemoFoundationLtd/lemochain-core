package deputynode

import (
	"crypto/ecdsa"
	"crypto/rand"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto/secp256k1"
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
		private, _ := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
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
