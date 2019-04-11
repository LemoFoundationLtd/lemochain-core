package deputynode

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"net"
	"testing"
)

// NewDeputyNode create a correct deputy node
func NewDeputyNode() *DeputyNode {
	return &DeputyNode{
		MinerAddress: common.HexToAddress("0x01"),
		NodeID:       common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"),
		IP:           net.IPv4(127, 0, 0, 1),
		Port:         7002,
		Rank:         0,
		Votes:        big.NewInt(0),
	}
}

func TestDeputyNode_Hash(t *testing.T) {
	node := NewDeputyNode()
	// don't crash
	hash1 := node.Hash()

	// hash changes when data is changed
	node.Port = 7003
	hash2 := node.Hash()
	assert.NotEqual(t, hash1, hash2)
}

func TestDeputyNode_Check(t *testing.T) {
	// correct
	node := NewDeputyNode()
	assert.Nil(t, node.Check())

	// MinerAddress
	node = NewDeputyNode()
	node.MinerAddress = common.HexToAddress("0x00")
	assert.Equal(t, ErrMinerAddressInvalid, node.Check())

	// NodeID
	node = NewDeputyNode()
	node.NodeID = common.FromHex("0x01")
	assert.Equal(t, ErrNodeIDInvalid, node.Check())

	// Port
	node = NewDeputyNode()
	node.Port = 666666
	assert.Equal(t, ErrPortInvalid, node.Check())

	// Rank
	node = NewDeputyNode()
	node.Rank = 666666
	assert.Equal(t, ErrRankInvalid, node.Check())

	// Votes
	node = NewDeputyNode()
	node.Votes = big.NewInt(-1)
	assert.Equal(t, ErrVotesInvalid, node.Check())
}

func TestDeputyNode_NodeAddrString(t *testing.T) {
	node := NewDeputyNode()
	assert.Equal(t, "5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0@127.0.0.1:7002", node.NodeAddrString())
}

func TestDeputyNodes_String(t *testing.T) {
	node := NewDeputyNode()
	nodes := (DeputyNodes)([]*DeputyNode{node})
	assert.Equal(t, `[{"minerAddress":"Lemo8888888888888888888888888888888888BW","nodeID":"0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0","ip":"127.0.0.1","port":"7002","rank":"0","votes":"0"}]`, nodes.String())
}
