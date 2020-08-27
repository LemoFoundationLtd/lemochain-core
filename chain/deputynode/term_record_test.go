package deputynode

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestNewTermRecord(t *testing.T) {
	// invalid snapshot height
	nodes := pickNodes(0)
	assert.PanicsWithValue(t, ErrInvalidSnapshotHeight, func() {
		NewTermRecord(1, nodes)
	})
	assert.PanicsWithValue(t, ErrInvalidSnapshotHeight, func() {
		NewTermRecord(params.InterimDuration, nodes)
	})

	// no deputy
	nodes = pickNodes()
	assert.PanicsWithValue(t, ErrNoDeputyInBlock, func() {
		NewTermRecord(params.TermDuration, nodes)
	})

	// invalid rank
	nodes = pickNodes(4, 5)
	nodes[0].Rank = 5
	assert.PanicsWithValue(t, ErrInvalidDeputyRank, func() {
		NewTermRecord(params.TermDuration, nodes)
	})

	// invalid votes
	nodes = pickNodes(4, 5)
	nodes[0].Votes = big.NewInt(1)
	nodes[1].Votes = big.NewInt(2)
	assert.PanicsWithValue(t, ErrInvalidDeputyVotes, func() {
		NewTermRecord(params.TermDuration, nodes)
	})

	// success
	nodes = pickNodes(4, 5)
	record := NewTermRecord(0, nodes)
	assert.Equal(t, uint32(0), record.TermIndex)
	assert.Equal(t, nodes, record.Nodes)
	nodes = pickNodes(5)
	record = NewTermRecord(params.TermDuration*2, nodes)
	assert.Equal(t, uint32(2), record.TermIndex)
	assert.Equal(t, nodes, record.Nodes)
}

func TestTermRecord_GetDeputies(t *testing.T) {
	// empty nodes
	nodes := GenerateDeputies(0)
	term := &TermRecord{TermIndex: 0, Nodes: nodes}
	assert.Equal(t, nodes, term.GetDeputies(5))

	// less than deputy nodes
	nodes = GenerateDeputies(3)
	term = &TermRecord{TermIndex: 0, Nodes: nodes}
	assert.Equal(t, nodes, term.GetDeputies(5))

	// more than deputy nodes
	nodes = GenerateDeputies(25)
	term = &TermRecord{TermIndex: 0, Nodes: nodes}
	assert.Equal(t, nodes[:5], term.GetDeputies(5))
}

func TestTermRecord_GetTotalVotes(t *testing.T) {
	// empty nodes
	nodes := GenerateDeputies(0)
	term := &TermRecord{TermIndex: 0, Nodes: nodes}
	assert.Equal(t, new(big.Int), term.GetTotalVotes())

	// 3 nodes
	nodes = GenerateDeputies(3)
	nodes[0].Votes = big.NewInt(100)
	nodes[1].Votes = big.NewInt(100)
	nodes[2].Votes = big.NewInt(100)
	term = &TermRecord{TermIndex: 0, Nodes: nodes}
	assert.Equal(t, big.NewInt(300), term.GetTotalVotes())
}

func TestGetLastSnapshotHeight(t *testing.T) {
	assert.Equal(t, uint32(0), GetLastSnapshotHeight(0))
	assert.Equal(t, uint32(0), GetLastSnapshotHeight(1))
	assert.Equal(t, params.TermDuration, GetLastSnapshotHeight(params.TermDuration))
	assert.Equal(t, params.TermDuration, GetLastSnapshotHeight(params.TermDuration+params.InterimDuration+1))
	assert.Equal(t, params.TermDuration, GetLastSnapshotHeight(params.TermDuration*2-1))
	assert.Equal(t, params.TermDuration*2, GetLastSnapshotHeight(params.TermDuration*2))
	assert.Equal(t, params.TermDuration*3, GetLastSnapshotHeight(params.TermDuration*3+1))
}

func TestIsSnapshotBlock(t *testing.T) {
	assert.Equal(t, true, IsSnapshotBlock(0))
	assert.Equal(t, false, IsSnapshotBlock(1))
	assert.Equal(t, true, IsSnapshotBlock(params.TermDuration))
	assert.Equal(t, false, IsSnapshotBlock(params.TermDuration+params.InterimDuration+1))
	assert.Equal(t, true, IsSnapshotBlock(params.TermDuration*2))
	assert.Equal(t, false, IsSnapshotBlock(params.TermDuration*2+2))
	assert.Equal(t, false, IsSnapshotBlock(params.TermDuration*3+params.InterimDuration+1))
}

func TestIsRewardBlock(t *testing.T) {
	assert.Equal(t, false, IsRewardBlock(0))
	assert.Equal(t, false, IsRewardBlock(1))
	assert.Equal(t, false, IsRewardBlock(params.TermDuration))
	assert.Equal(t, true, IsRewardBlock(params.TermDuration+params.InterimDuration+1))
	assert.Equal(t, true, IsRewardBlock(params.TermDuration*2+params.InterimDuration+1))
	assert.Equal(t, false, IsRewardBlock(params.TermDuration*2+params.InterimDuration+2))
	assert.Equal(t, true, IsRewardBlock(params.TermDuration*3+params.InterimDuration+1))
}

func TestGetTermIndexByHeight(t *testing.T) {
	assert.Equal(t, uint32(0), GetSignerTermIndexByHeight(0))
	assert.Equal(t, uint32(0), GetSignerTermIndexByHeight(1))
	assert.Equal(t, uint32(0), GetSignerTermIndexByHeight(params.TermDuration))
	assert.Equal(t, uint32(0), GetSignerTermIndexByHeight(params.TermDuration+params.InterimDuration))
	assert.Equal(t, uint32(1), GetSignerTermIndexByHeight(params.TermDuration+params.InterimDuration+1))
	assert.Equal(t, uint32(2), GetSignerTermIndexByHeight(params.TermDuration*2+params.InterimDuration+1))
	assert.Equal(t, uint32(2), GetSignerTermIndexByHeight(params.TermDuration*2+params.InterimDuration+2))
	assert.Equal(t, uint32(3), GetSignerTermIndexByHeight(params.TermDuration*3+params.InterimDuration+1))
}

func TestGetDeputyTermIndexByHeight(t *testing.T) {
	assert.Equal(t, uint32(0), GetDeputyTermIndexByHeight(0))
	assert.Equal(t, uint32(0), GetDeputyTermIndexByHeight(1))
	assert.Equal(t, uint32(0), GetDeputyTermIndexByHeight(params.TermDuration-1))
	assert.Equal(t, uint32(1), GetDeputyTermIndexByHeight(params.TermDuration))
	assert.Equal(t, uint32(1), GetDeputyTermIndexByHeight(params.TermDuration+params.InterimDuration))
	assert.Equal(t, uint32(1), GetDeputyTermIndexByHeight(params.TermDuration+params.InterimDuration+1))
	assert.Equal(t, uint32(1), GetDeputyTermIndexByHeight(params.TermDuration*2-1))
	assert.Equal(t, uint32(2), GetDeputyTermIndexByHeight(params.TermDuration*2))
	assert.Equal(t, uint32(2), GetDeputyTermIndexByHeight(params.TermDuration*2+1))
	assert.Equal(t, uint32(3), GetDeputyTermIndexByHeight(params.TermDuration*3))
}
