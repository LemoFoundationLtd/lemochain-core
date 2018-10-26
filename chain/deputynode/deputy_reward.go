package deputynode

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"math/big"
)

var (
	minReward = new(big.Int).SetUint64(uint64(10000000000000000)) // 0.01 lemo
)

type DeputyReward struct {
	Address common.Address
	Reward  *big.Int
}

// CalcReward 计算收益
func CalcReward(height uint32) []*DeputyReward {
	rewards := make([]*DeputyReward, 0)
	nodes := Instance().getDeputiesByHeight(height)
	totalVotes := new(big.Int)
	for _, node := range nodes {
		totalVotes.Add(totalVotes, new(big.Int).SetUint64(node.Votes))
	}
	totalRewards := getTotalReward(height)
	// realRewards := new(big.Int)
	for _, node := range nodes {
		// n_v := new(big.Int).SetUint64(node.Votes)
		r := new(big.Int)
		r.Mul(new(big.Int).SetUint64(node.Votes), totalRewards)
		r.Div(r, totalVotes) // reward = vote * totalRewards / totalVotes
		r.Div(r, minReward)  // reward = reward / minReward * reward
		r.Mul(r, minReward)
		reward := &DeputyReward{
			Address: node.LemoBase,
			Reward:  r,
		}
		rewards = append(rewards, reward)
		// realRewards.Add(realRewards, r)
	}
	// remainReward := totalRewards.Sub(totalRewards, realRewards)
	return rewards
}

// getTotalReward 获取当前轮总奖励 todo
func getTotalReward(height uint32) *big.Int {
	return new(big.Int).SetUint64(1000000000000000000) // 1 lemo
}
