package deputynode

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/stretchr/testify/assert"
	"math/big"
	"math/rand"
	"testing"
	"time"
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

func TestDivideSalary(t *testing.T) {
	tests := []struct {
		Expect, TotalSalary, DeputyVotes, TotalVotes, Precision int64
	}{
		// total votes=100
		{0, 100, 0, 100, 1},
		{1, 100, 1, 100, 1},
		{2, 100, 2, 100, 1},
		{100, 100, 100, 100, 1},
		// total votes=100, precision=10
		{0, 100, 1, 100, 10},
		{10, 100, 10, 100, 10},
		{10, 100, 11, 100, 10},
		// total votes=1000
		{0, 100, 1, 1000, 1},
		{0, 100, 9, 1000, 1},
		{1, 100, 10, 1000, 1},
		{1, 100, 11, 1000, 1},
		{100, 100, 1000, 1000, 1},
		// total votes=1000, precision=10
		{10, 100, 100, 1000, 10},
		{10, 100, 120, 1000, 10},
		{20, 100, 280, 1000, 10},
		// total votes=10
		{0, 100, 0, 10, 1},
		{10, 100, 1, 10, 1},
		{100, 100, 10, 10, 1},
		// total votes=10, precision=10
		{10, 100, 1, 10, 10},
		{100, 100, 10, 10, 10},
	}
	for _, test := range tests {
		expect := big.NewInt(test.Expect)
		totalSalary := big.NewInt(test.TotalSalary)
		deputyVotes := big.NewInt(test.DeputyVotes)
		totalVotes := big.NewInt(test.TotalVotes)
		precision := big.NewInt(test.Precision)
		assert.Equalf(t, 0, divideSalary(totalSalary, deputyVotes, totalVotes, precision).Cmp(expect), "divideSalary(%v, %v, %v, %v)", totalSalary, deputyVotes, totalVotes, precision)
	}
}

func lemo2mo(lemo int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(1000000000000000000), big.NewInt(lemo))
}

func randomBigInt(r *rand.Rand) *big.Int {
	return new(big.Int).Mul(big.NewInt(r.Int63()), big.NewInt(r.Int63()))
}

func TestTermRecord_DivideSalary1(t *testing.T) {
	// empty nodes
	nodes := GenerateDeputies(0)
	term := &TermRecord{TermIndex: 0, Nodes: nodes}
	assert.Empty(t, term.DivideSalary(new(big.Int)))

	// 3 nodes
	nodes = GenerateDeputies(3)
	// total votes: 78521483187
	nodes[0].Votes = big.NewInt(16584983216)
	nodes[1].Votes = big.NewInt(28949984984)
	nodes[2].Votes = big.NewInt(32986514987)
	term = &TermRecord{TermIndex: 0, Nodes: nodes}
	salaries := term.DivideSalary(lemo2mo(12345))
	assert.Len(t, salaries, 3)
	assert.Equal(t, lemo2mo(2607), salaries[0].Salary)
	assert.Equal(t, lemo2mo(4551), salaries[1].Salary)
	assert.Equal(t, lemo2mo(5186), salaries[2].Salary)
}

// test total salary with random data
func TestTermRecord_DivideSalary2(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < 100; i++ {
		nodeCount := r.Intn(49) + 1 // [1, 50]
		nodes := GenerateDeputies(nodeCount)
		for _, node := range nodes {
			node.Votes = randomBigInt(r)
		}

		totalSalary := randomBigInt(r)
		term := &TermRecord{TermIndex: 0, Nodes: nodes}
		salaries := term.DivideSalary(totalSalary)
		assert.Len(t, salaries, nodeCount)

		actualTotal := new(big.Int)
		for _, s := range salaries {
			actualTotal.Add(actualTotal, s.Salary)
		}
		// t.Log("count", nodeCount, "totalSalary", totalSalary, "actualTotal", actualTotal)

		// errRange = nodeCount * minPrecision
		// actualTotal must be in range [totalSalary - errRange, totalSalary]
		errRange := new(big.Int).Mul(big.NewInt(int64(nodeCount)), minPrecision)
		assert.Equal(t, true, actualTotal.Cmp(new(big.Int).Sub(totalSalary, errRange)) >= 0)
		assert.Equal(t, true, actualTotal.Cmp(totalSalary) <= 0)
	}
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
	assert.Equal(t, uint32(0), GetTermIndexByHeight(0))
	assert.Equal(t, uint32(0), GetTermIndexByHeight(1))
	assert.Equal(t, uint32(0), GetTermIndexByHeight(params.TermDuration))
	assert.Equal(t, uint32(0), GetTermIndexByHeight(params.TermDuration+params.InterimDuration))
	assert.Equal(t, uint32(1), GetTermIndexByHeight(params.TermDuration+params.InterimDuration+1))
	assert.Equal(t, uint32(2), GetTermIndexByHeight(params.TermDuration*2+params.InterimDuration+1))
	assert.Equal(t, uint32(2), GetTermIndexByHeight(params.TermDuration*2+params.InterimDuration+2))
	assert.Equal(t, uint32(3), GetTermIndexByHeight(params.TermDuration*3+params.InterimDuration+1))
}
