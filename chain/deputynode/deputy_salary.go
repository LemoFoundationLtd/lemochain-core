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
func CalcSalary(height uint32) []*DeputySalary {
	salaries := make([]*DeputySalary, 0)
	nodes := Instance().getDeputiesByHeight(height)
	totalVotes := new(big.Int)
	for _, node := range nodes {
		totalVotes.Add(totalVotes, new(big.Int).SetUint64(uint64(node.Votes)))
	}
	totalRewards := getTotalSalary(height)
	// realRewards := new(big.Int)
	for _, node := range nodes {
		// n_v := new(big.Int).SetUint64(node.Votes)
		r := new(big.Int)
		r.Mul(new(big.Int).SetUint64(uint64(node.Votes)), totalRewards)
		r.Div(r, totalVotes) // reward = vote * totalRewards / totalVotes
		r.Div(r, minSalary)  // reward = reward / minSalary * reward
		r.Mul(r, minSalary)
		reward := &DeputySalary{
			Address: node.LemoBase,
			Salary:  r,
		}
		salaries = append(salaries, reward)
		// realRewards.Add(realRewards, r)
	}
	// remainReward := totalRewards.Sub(totalRewards, realRewards)
	return salaries
}

// getTotalSalary 获取当前轮总奖励 todo
func getTotalSalary(height uint32) *big.Int {
	return new(big.Int).SetUint64(1000000000000000000) // 1 lemo
}
