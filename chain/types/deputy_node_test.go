package types

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

// newTestDeputyNode create a correct deputy node
func newTestDeputyNode() *DeputyNode {
	return &DeputyNode{
		MinerAddress: common.HexToAddress("0x01"),
		NodeID:       common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"),
		Rank:         0,
		Votes:        big.NewInt(0),
	}
}

func TestDeputyNode_Hash(t *testing.T) {
	node := newTestDeputyNode()
	// don't crash
	hash1 := node.Hash()

	node.Rank = 111
	// hash changes when data is changed
	hash2 := node.Hash()
	assert.NotEqual(t, hash1, hash2)
}

func TestDeputyNode_Check(t *testing.T) {
	// correct
	node := newTestDeputyNode()
	assert.Nil(t, node.Check())

	// MinerAddress
	node = newTestDeputyNode()
	node.MinerAddress = common.HexToAddress("0x00")
	assert.Equal(t, ErrMinerAddressInvalid, node.Check())

	// NodeID
	node = newTestDeputyNode()
	node.NodeID = common.FromHex("0x01")
	assert.Equal(t, ErrNodeIDInvalid, node.Check())

	// Rank
	node = newTestDeputyNode()
	node.Rank = 666666
	assert.Equal(t, ErrRankInvalid, node.Check())

	// Votes
	node = newTestDeputyNode()
	node.Votes = big.NewInt(-1)
	assert.Equal(t, ErrVotesInvalid, node.Check())
}

func TestDeputyNodes_String(t *testing.T) {
	node := newTestDeputyNode()
	nodes := (DeputyNodes)([]*DeputyNode{node})
	assert.Equal(t, `[{"minerAddress":"Lemo8888888888888888888888888888888888BW","nodeID":"0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0","rank":"0","votes":"0"}]`, nodes.String())
}
