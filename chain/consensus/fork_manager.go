package consensus

import (
	"bytes"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"sync/atomic"
)

var ErrNoHeadBlock = errors.New("head block is required")

// ForkManager process the fork logic
type ForkManager struct {
	blockLoader BlockLoader
	dm          *deputynode.Manager
	head        atomic.Value // the last block on current fork
}

func NewForkManager(dm *deputynode.Manager, db BlockLoader, stable *types.Block) *ForkManager {
	fm := &ForkManager{
		blockLoader: db,
		dm:          dm,
	}
	fm.SetHeadBlock(stable)
	return fm
}

// CurrentBlock get latest block on current fork
func (fm *ForkManager) GetHeadBlock() *types.Block {
	return fm.head.Load().(*types.Block)
}

// CurrentBlock get latest block on current fork
func (fm *ForkManager) SetHeadBlock(block *types.Block) {
	if block == nil {
		panic(ErrNoHeadBlock)
	}
	fm.head.Store(block)
}

func findDeputyByAddress(deputies []*types.DeputyNode, addr common.Address) *types.DeputyNode {
	for _, node := range deputies {
		if node.MinerAddress == addr {
			return node
		}
	}
	return nil
}

// GetMinerDistance get miner index distance. It is always greater than 0 and not greater than deputy count
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

// UpdateFork check if the current fork can be update, or switch to a better fork. Return the new current block or nil
func (fm *ForkManager) UpdateFork(newBlock, stableBlock *types.Block) *types.Block {
	var (
		oldHead = fm.GetHeadBlock()
		newHead *types.Block
	)

	// Maybe a block on other fork is stable now. So we need to check if the current fork is still there
	if fm.isCurrentForkCut() {
		//   ┌─2 [oldHead]
		// 1─┴─3 [newBlock] [became stable]
		// or
		//   ┌─2 [oldHead]───4 [newBlock] [became stable]
		// 1─┴─3
		// Choose the longest fork to be new current block
		newHead = fm.ChooseNewFork(stableBlock)
	} else if newBlock.ParentHash() == oldHead.Hash() {
		//            ┌─2 [oldHead]───4 [newBlock]
		// 1 [stable]─┴─3
		// A block after last head (best fork), it must make a new best fork
		newHead = newBlock
	} else {
		//            ┌─2 [oldHead]
		// 1 [stable]─┴─3───4 [newBlock]
		// or
		//            ┌─2───3 [oldHead]
		// 1 [stable]─┴─4 [newBlock]
		// or
		//            ┌─2 [oldHead]
		// 1 [stable]─┼─3
		//            └─4 [newBlock]
		// The new block is inserted to other fork. So maybe we need to update fork
		// candidateHead must not be the stableBlock, or it means the current fork is cut
		candidateHead := fm.ChooseNewFork(stableBlock)
		if fm.needSwitchFork(oldHead, candidateHead, stableBlock) {
			newHead = candidateHead
		}
	}

	if newHead != nil && newHead.Hash() != oldHead.Hash() {
		fm.SetHeadBlock(newHead)
		return newHead
	}
	return nil
}

// UpdateForkForConfirm switch to a better fork if the current fork is not exist. Return the new current block or nil
func (fm *ForkManager) UpdateForkForConfirm(stableBlock *types.Block) *types.Block {
	var (
		oldHead = fm.GetHeadBlock()
		newHead *types.Block
	)

	// Maybe a block on other fork is stable now. So we need to check if the current fork is still there
	if fm.isCurrentForkCut() {
		// Choose the longest fork to be new current block
		newHead = fm.ChooseNewFork(stableBlock)
	}

	if newHead != nil && newHead.Hash() != oldHead.Hash() {
		fm.SetHeadBlock(newHead)
		return newHead
	}
	return nil
}

// needSwitchFork test if the new fork's head distance reached to a multiple of "deputy nodes count * 2/3"
func (fm *ForkManager) needSwitchFork(oldHead, newHead, stable *types.Block) bool {
	// make sure the fork is the first one reaching the height
	if newHead.Height() > oldHead.Height() {
		signDistance := fm.dm.TwoThirdDeputyCount(newHead.Height())
		if (newHead.Height()-stable.Height())%signDistance == 0 {
			return true
		}
	}
	return false
}

// ChooseNewFork choose a fork and return the last block on the fork. It would return the current stable block if there is no unstable block
func (fm *ForkManager) ChooseNewFork(stableBlock *types.Block) *types.Block {
	var max = stableBlock
	fm.blockLoader.IterateUnConfirms(func(node *types.Block) {
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

// isCurrentForkCut check whether or not the current fork is cut
func (fm *ForkManager) isCurrentForkCut() bool {
	oldHead := fm.GetHeadBlock()

	// Test if currentBlock is still in unconfirmed blocks. It must has be pruned by stable block updating
	_, err := fm.blockLoader.GetUnConfirmByHeight(oldHead.Height(), oldHead.Hash())
	return err == store.ErrNotExist
}
