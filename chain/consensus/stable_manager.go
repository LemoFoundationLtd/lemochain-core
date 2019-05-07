package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"math"
	"sync"
)

// StableManager process the fork logic
type StableManager struct {
	db protocol.ChainDB
	dm *deputynode.Manager

	lock sync.Mutex
}

func NewStableManager(dm *deputynode.Manager, db protocol.ChainDB) *StableManager {
	dpovp := &StableManager{
		db: db,
		dm: dm,
	}
	return dpovp
}

// StableBlock get latest stable block
func (sm *StableManager) StableBlock() *types.Block {
	block, err := sm.db.LoadLatestBlock()
	if err != nil {
		log.Warn("load stable block fail")
		// We would make sure genesis is available at least. So err is not tolerable
		panic(err)
	}
	return block
}

// UpdateStable check if the block can be stable. Return true if the stable block changed, and return the pruned uncle blocks
func (sm *StableManager) UpdateStable(block *types.Block) (bool, []*types.Block, error) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	hash := block.Hash()
	oldStable := sm.StableBlock()

	if block.Height() <= oldStable.Height() {
		return false, nil, nil
	}
	if !IsConfirmEnough(block, sm.dm) {
		return false, nil, nil
	}

	// update stable block
	prunedBlocks, err := sm.db.SetStableBlock(hash)
	if err != nil {
		log.Errorf("SetStableBlock error. height:%d hash:%s, err:%s", block.Height(), common.ToHex(hash[:]), err.Error())
		return false, nil, ErrSetStableBlockToDB
	}
	log.Infof("Stable block changes from %s to %s", oldStable.ShortString(), block.ShortString())

	// This may not the latest state, but it's fine. Because deputy nodes snapshot will be used after the interim duration, it's about 1000 blocks
	sm.updateDeputyNodes(block)

	return true, prunedBlocks, nil
}

// IsConfirmEnough test if the confirms in block is enough
func IsConfirmEnough(block *types.Block, dm *deputynode.Manager) bool {
	// nodeCount < 3 means two deputy nodes scene: One node mined a block and broadcasted it. Then it means two confirms after the receiver one's verification
	// if dm.GetDeputiesCount(height) < 3 {
	// 	return true
	// }

	// +1 for the miner
	singerCount := len(block.Confirms) + 1

	// fast test
	if uint32(singerCount) >= uint32(math.Ceil(float64(dm.DeputyCount)*2.0/3.0)) {
		return true
	}

	return uint32(singerCount) >= dm.TwoThirdDeputyCount(block.Height())
}

// updateDeputyNodes update deputy nodes map
func (sm *StableManager) updateDeputyNodes(block *types.Block) {
	if deputynode.IsSnapshotBlock(block.Height()) {
		sm.dm.SaveSnapshot(block.Height(), block.DeputyNodes)
		log.Debug("save new term", "deputies", log.Lazy{Fn: func() string {
			return block.DeputyNodes.String()
		}})
	}
}
