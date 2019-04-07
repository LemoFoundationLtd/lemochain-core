package deputynode

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"math/big"
)

var (
	minPrecision = new(big.Int).SetUint64(uint64(1000000000000000000)) // 1 LEMO
)

type DeputySalary struct {
	Address common.Address
	Salary  *big.Int
}

//go:generate gencodec -type TermRecord --field-override termRecordMarshaling -out gen_term_record_json.go
type TermRecord struct {
	StartHeight uint32      `json:"height"` // 0, 100W+1K+1, 200W+1K+1, 300W+1K+1...
	Nodes       DeputyNodes `json:"nodes"`  // include deputy nodes and candidate nodes
}

type termRecordMarshaling struct {
	StartHeight hexutil.Uint32
}

// GetDeputies return deputy nodes. They are first TotalCount items in t.Nodes
func (t *TermRecord) GetDeputies() DeputyNodes {
	if len(t.Nodes) > TotalCount {
		return t.Nodes[:TotalCount]
	} else {
		return t.Nodes[:]
	}
}

func (t *TermRecord) GetTotalVotes() *big.Int {
	totalVotes := new(big.Int)
	for _, node := range t.Nodes {
		totalVotes.Add(totalVotes, node.Votes)
	}
	return totalVotes
}

// DivideSalary divide term salary to every node including of candidate nodes
func (t *TermRecord) DivideSalary(totalSalary *big.Int) []*DeputySalary {
	salaries := make([]*DeputySalary, len(t.Nodes))
	totalVotes := t.GetTotalVotes()
	for i, node := range t.Nodes {
		salaries[i] = &DeputySalary{
			Address: node.MinerAddress,
			Salary:  divideSalary(totalSalary, node.Votes, totalVotes, minPrecision),
		}
	}
	return salaries
}

func divideSalary(totalSalary, deputyVotes, totalVotes, precision *big.Int) *big.Int {
	r := new(big.Int)
	// totalSalary * deputyVotes / totalVotes
	r.Mul(totalSalary, deputyVotes)
	r.Div(r, totalVotes)
	// r - ( r % precision )
	mod := new(big.Int).Mod(r, precision)
	r.Sub(r, mod)
	return r
}
