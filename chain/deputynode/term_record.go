package deputynode

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
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
	TermIndex uint32      `json:"termIndex"` // start from 0
	Nodes     DeputyNodes `json:"nodes"`     // include deputy nodes and candidate nodes
}

type termRecordMarshaling struct {
	TermIndex hexutil.Uint32
}

func NewTermRecord(snapshotHeight uint32, nodes DeputyNodes) *TermRecord {
	// check snapshot block height
	if snapshotHeight%params.TermDuration != 0 {
		log.Error("invalid snapshot block height", "height", snapshotHeight)
		panic(ErrInvalidSnapshotHeight)
	}
	// check nodes to make sure it is not empty
	if nodes == nil || len(nodes) == 0 {
		log.Error("can't save empty deputy nodes", "height", snapshotHeight)
		panic(ErrEmptyDeputies)
	}
	for i, node := range nodes {
		// check nodes' rank
		if uint32(i) != node.Rank {
			log.Error("invalid deputy rank", "index", i, "rank", node.Rank, "expect", i)
			panic(ErrInvalidDeputyRank)
		}
		// check nodes' votes
		if i > 0 {
			lastNode := nodes[i-1]
			if node.Votes.Cmp(lastNode.Votes) > 0 {
				log.Error("deputy should sort by votes", "index", i, "votes", node.Votes, "last node votes", lastNode.Votes)
				panic(ErrInvalidDeputyVotes)
			}
		}
	}

	return &TermRecord{TermIndex: snapshotHeight / params.TermDuration, Nodes: nodes}
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

// IsRewardBlock 是否该发出块奖励了
func IsRewardBlock(height uint32) bool {
	if height < params.TermDuration+params.InterimDuration+1 {
		// in genesis term
		return false
	} else if height%params.TermDuration == params.InterimDuration+1 {
		// term start block
		return true
	} else {
		// other normal block
		return false
	}
}

// GetTermIndexByHeight return the index of the term which in charge of consensus the specific block
//
//   0 term start at height 0
//   1 term start at 100W+1K+1
//   2 term start at 200W+1K+1
//   ...
//
func GetTermIndexByHeight(height uint32) uint32 {
	if height < params.TermDuration+params.InterimDuration+1 {
		// in genesis term
		return 0
	}

	return (height - params.InterimDuration - 1) / params.TermDuration
}