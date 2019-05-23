package transaction

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

// BlockLoader supports retrieving headers and consensus parameters from the
// current blockchain to be used during transaction processing.
type BlockLoader interface {
	// GetBlockByHash returns the hash corresponding to their hash.
	GetBlockByHash(hash common.Hash) *types.Block
	// GetBlockByHash returns the hash corresponding to their hash.
	GetParentByHeight(height uint32, sonBlockHash common.Hash) *types.Block
}
