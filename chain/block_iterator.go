package chain

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
)

type blockLoader interface {
	GetBlockByHash(hash common.Hash) (*types.Block, error)
}

// FindParentByHeight find the specific height block parent by parent
func FindParentByHeight(height uint32, knownSonBlock *types.Block, db blockLoader) (*types.Block, error) {

	block := knownSonBlock
	for block.Height() > height {
		parent, err := db.GetBlockByHash(block.ParentHash())
		if err != nil {
			log.Error("load parent block fail", "height", block.Height()-1, "hash", block.ParentHash(), "err", err)
			return nil, err
		}
		block = parent
	}
	return block, nil
}

// FindFirstForkBlocks iterate from two blocks to their first fork parent blocks
func FindFirstForkBlocks(block1, block2 *types.Block, db blockLoader) (*types.Block, *types.Block, error) {
	var err error
	// make block1 and block2 at same height
	if block1.Height() > block2.Height() {
		block1, err = FindParentByHeight(block2.Height(), block1, db)
		if err != nil {
			log.Error("load parent block fail", "targetHeight", block2.Height(), "sonBlockHash", block1.Hash().Prefix())
			return nil, nil, err
		}
	} else if block2.Height() > block1.Height() {
		block2, err = FindParentByHeight(block1.Height(), block2, db)
		if err != nil {
			log.Error("load parent block fail", "targetHeight", block1.Height(), "sonBlockHash", block2.Hash().Prefix())
			return nil, nil, err
		}
	}

	// find same ancestor
	for block1.ParentHash() != block2.ParentHash() {
		newBlock, err := db.GetBlockByHash(block1.ParentHash())
		if err != nil {
			log.Error("load parent block fail", "targetHeight", block1.Height()-1, "targetHash", block1.ParentHash().Prefix())
			return nil, nil, err
		}
		block1 = newBlock

		newBlock, err = db.GetBlockByHash(block2.ParentHash())
		if err != nil {
			log.Error("load parent block fail", "targetHeight", block2.Height()-1, "targetHash", block2.ParentHash().Prefix())
			return nil, nil, err
		}
		block2 = newBlock
	}

	return block1, block2, nil
}
