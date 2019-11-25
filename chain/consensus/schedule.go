package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
)

func findDeputyByRank(deputies []*types.DeputyNode, rank uint32) *types.DeputyNode {
	for _, node := range deputies {
		if node.Rank == rank {
			return node
		}
	}
	return nil
}

// GetNextMineWindow get next time window to mine block. The times are timestamps in millisecond
func GetNextMineWindow(nextHeight uint32, distance uint32, parentTime int64, currentTime int64, mineTimeout int64, dm *deputynode.Manager) (int64, int64) {
	nodeCount := dm.GetDeputiesCount(nextHeight)
	// 所有节点都超时所需要消耗的时间，也可以看作是下一轮出块的开始时间
	oneLoopTime := int64(nodeCount) * mineTimeout
	// 网络传输耗时，即当前时间减去父块区块头中的时间戳
	passTime := currentTime - parentTime
	if passTime < 0 {
		passTime = 0
	}
	// 从父块开始，经过的整轮数
	passLoop := passTime / oneLoopTime
	// 可以出块的时间窗口
	windowFrom := parentTime + passLoop*oneLoopTime + int64(distance-1)*mineTimeout
	windowTo := parentTime + passLoop*oneLoopTime + int64(distance)*mineTimeout
	if windowTo <= currentTime {
		windowFrom += oneLoopTime
		windowTo += oneLoopTime
	}

	log.Debug("GetNextMineWindow", "windowFrom", windowFrom, "windowTo", windowTo, "parentTime", parentTime, "passTime", passTime, "distance", distance, "passLoop", passLoop, "nodeCount", nodeCount)
	return windowFrom, windowTo
}

// GetCorrectMiner get the correct miner to mine a block after parent block
func GetCorrectMiner(parent *types.Header, mineTime int64, mineTimeout int64, dm *deputynode.Manager) (common.Address, error) {
	if mineTime < 1e10 {
		panic("mineTime should be milliseconds")
	}
	passTime := mineTime - int64(parent.Time)*1000
	if passTime < 0 {
		return common.Address{}, ErrSmallerMineTime
	}
	nodeCount := dm.GetDeputiesCount(parent.Height + 1)
	// 所有节点都超时所需要消耗的时间，也可以看作是下一轮出块的开始时间
	oneLoopTime := int64(nodeCount) * mineTimeout
	minerDistance := (passTime%oneLoopTime)/mineTimeout + 1

	deputy, err := dm.GetDeputyByDistance(parent.Height+1, parent.MinerAddress, uint32(minerDistance))
	if err != nil {
		return common.Address{}, err
	}
	log.Debug("GetCorrectMiner", "correctMiner", deputy.MinerAddress, "parent", parent.MinerAddress, "mineTime", mineTime, "mineTimeout", mineTimeout, "passTime", passTime, "nodeCount", nodeCount)
	return deputy.MinerAddress, nil
}
