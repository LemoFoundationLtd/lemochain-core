package consensus

import (
	"bytes"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"sync/atomic"
)

// ForkManager process the fork logic
type ForkManager struct {
	db   protocol.ChainDB
	dm   *deputynode.Manager
	head atomic.Value // the last block on current fork
}

func NewForkManager(dm *deputynode.Manager, db protocol.ChainDB, stable *types.Block) *ForkManager {
	dpovp := &ForkManager{
		db: db,
		dm: dm,
	}
	dpovp.head.Store(stable)
	return dpovp
}

// CurrentBlock get latest block on current fork
func (fm *ForkManager) GetHeadBlock() *types.Block {
	return fm.head.Load().(*types.Block)
}

// CurrentBlock get latest block on current fork
func (fm *ForkManager) SetHeadBlock(block *types.Block) {
	fm.head.Store(block)
}

func findDeputyByAddress(deputies []*deputynode.DeputyNode, addr common.Address) *deputynode.DeputyNode {
	for _, node := range deputies {
		if node.MinerAddress == addr {
			return node
		}
	}
	return nil
}

// GetMinerDistance get miner index distance. It is always greater than 0
func GetMinerDistance(targetHeight uint32, parentBlockMiner, targetMiner common.Address, dm *deputynode.Manager) (uint64, error) {
	if targetHeight == 0 {
		return 0, ErrMineGenesis
	}
	deputies := dm.GetDeputiesByHeight(targetHeight)
	nodeCount := uint64(len(deputies))

	// find target block miner deputy
	targetDeputy := findDeputyByAddress(deputies, targetMiner)
	if targetDeputy == nil {
		return 0, ErrNotDeputy
	}

	// Genesis block is pre-set, not belong to any deputy node. So only blocks start with height 1 is mined by deputies
	// The reward block changes deputy nodes, so we need recompute the slot
	if targetHeight == 1 || deputynode.IsRewardBlock(targetHeight) {
		return uint64(targetDeputy.Rank + 1), nil
	}

	// if they are same miner, then return deputy count
	if targetMiner == parentBlockMiner {
		return nodeCount, nil
	}

	// find last block miner deputy
	lastDeputy := findDeputyByAddress(deputies, parentBlockMiner)
	if lastDeputy == nil {
		return 0, ErrNotDeputy
	}
	return (nodeCount + uint64(targetDeputy.Rank) - uint64(lastDeputy.Rank)) % nodeCount, nil
}

// TrySwitchFork switch fork if its length reached to a multiple of "deputy nodes count * 2/3"
func (fm *ForkManager) TrySwitchFork(stable, current *types.Block) (*types.Block, bool) {
	maxHeightBlock := fm.ChooseNewFork()
	// make sure the fork is the first one reaching the height
	if maxHeightBlock != nil && maxHeightBlock.Height() > current.Height() {
		signDistance := fm.dm.TwoThirdDeputyCount(maxHeightBlock.Height())
		if (maxHeightBlock.Height()-stable.Height())%signDistance == 0 {
			return maxHeightBlock, true
		}
	}
	return current, false
}

// ChooseNewFork choose a fork and return the last block on the fork. It would return nil if there is no unstable block
func (fm *ForkManager) ChooseNewFork() *types.Block {
	var max *types.Block
	fm.db.IterateUnConfirms(func(node *types.Block) {
		if max == nil || node.Height() > max.Height() {
			// 1. Choose the longest fork
			max = node
		} else if node.Height() == max.Height() {
			// 2. Choose the one which has smaller hash in dictionary order
			nodeHash := node.Hash()
			maxHash := max.Hash()
			if bytes.Compare(nodeHash[:], maxHash[:]) < 0 {
				max = node
			}
		}
	})
	return max
}
