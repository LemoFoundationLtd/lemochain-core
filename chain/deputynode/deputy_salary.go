package deputynode

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
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
		totalVotes.Add(totalVotes, node.Votes)
	}
	totalRewards := getTotalSalary(height)
	for _, node := range nodes {
		r := new(big.Int)
		r.Mul(node.Votes, totalRewards)
		r.Div(r, totalVotes) // reward = vote * totalRewards / totalVotes
		r.Div(r, minSalary)  // reward = reward / minSalary * reward
		r.Mul(r, minSalary)
		reward := &DeputySalary{
			Address: node.MinerAddress,
			Salary:  r,
		}
		salaries = append(salaries, reward)
	}
	return salaries
}

// getTotalSalary 获取当前轮总奖励
func getTotalSalary(height uint32) *big.Int {
	res, ok := new(big.Int).SetString("1800000000000000000000000", 10) // 180W lemo each epoch
	if !ok {
		log.Crit("getTotalSalary failed")
	}
	return res
}
