package deputynode

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"math/big"
)

var (
	minSalary = new(big.Int).SetUint64(uint64(10000000000000000)) // 0.01 lemo
)

type DeputySalary struct {
	Address common.Address
	Salary  *big.Int
}

// CalcSalary 计算收益
func CalcSalary(height uint32, termRewards *big.Int) []*DeputySalary {
	nodes := Instance().GetDeputiesByHeight(height, true)
	salaries := make([]*DeputySalary, len(nodes))
	totalVotes := new(big.Int)
	for _, node := range nodes {
		totalVotes.Add(totalVotes, node.Votes)
	}
	for i, node := range nodes {
		r := new(big.Int)
		r.Mul(node.Votes, termRewards)
		r.Div(r, totalVotes) // reward = vote * termRewards / totalVotes
		r.Div(r, minSalary)  // reward = reward / minSalary * reward
		r.Mul(r, minSalary)
		reward := &DeputySalary{
			Address: node.MinerAddress,
			Salary:  r,
		}
		salaries[i] = reward
	}
	return salaries
}
