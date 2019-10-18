package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"math"
)

// StableManager process the fork logic
type StableManager struct {
	store StableBlockStore
	dm    *deputynode.Manager
}

func NewStableManager(dm *deputynode.Manager, store StableBlockStore) *StableManager {
	dpovp := &StableManager{
		store: store,
		dm:    dm,
	}
	return dpovp
}

// StableBlock get latest stable block
func (sm *StableManager) StableBlock() *types.Block {
	block, err := sm.store.LoadLatestBlock()
	if err != nil {
		log.Warn("load stable block fail")
		// We would make sure genesis is available at least. So err is not tolerable
		panic(err)
	}
	return block
}

// UpdateStable check if the block can be stable. Return true if the stable block changed, and return the pruned uncle blocks
func (sm *StableManager) UpdateStable(block *types.Block) (bool, []*types.Block, error) {
	hash := block.Hash()
	oldStable := sm.StableBlock()

	if block.Height() <= oldStable.Height() {
		return false, nil, nil
	}
	if !IsConfirmEnough(block, sm.dm) {
		return false, nil, nil
	}

	// update stable block
	prunedBlocks, err := sm.store.SetStableBlock(hash)
	if err != nil {
		log.Errorf("SetStableBlock error. height:%d hash:%s, err:%s", block.Height(), common.ToHex(hash[:]), err.Error())
		return false, nil, ErrSetStableBlockToDB
	}
	log.Infof("🎉 Stable block changes from %s to %s", oldStable.ShortString(), block.ShortString())

	return true, prunedBlocks, nil
}

// IsConfirmEnough test if the confirms in block is enough
func IsConfirmEnough(block *types.Block, dm *deputynode.Manager) bool {
	// +1 for the miner
	singerCount := len(block.Confirms) + 1

	// fast test
	if uint32(singerCount) >= uint32(math.Ceil(float64(dm.DeputyCount)*2.0/3.0)) {
		return true
	}

	return uint32(singerCount) >= dm.TwoThirdDeputyCount(block.Height())
}
