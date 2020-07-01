package network

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

// GetLatestStatus get latest status
type GetLatestStatus struct {
	Revert uint32
}

// BlockHashData
type BlockHashData struct {
	Height uint32
	Hash   common.Hash
}

// getBlocksData
type GetBlocksData struct {
	From uint32
	To   uint32
}

// GetSingleBlockData
type GetSingleBlockData struct {
	Hash   common.Hash
	Height uint32
}

// blockConfirmData
type BlockConfirmData struct {
	Hash     common.Hash    // block Hash
	Height   uint32         // block height
	SignInfo types.SignData // block sign info
}

// getConfirmInfo
type GetConfirmInfo struct {
	Height uint32
	Hash   common.Hash
}

// BlockConfirms confirms of a block
type BlockConfirms struct {
	Height uint32      // block height
	Hash   common.Hash // block hash
	Pack   []types.SignData
}

// for find node
type DiscoverResData struct {
	Sequence uint
	Nodes    []string
}

// for find node
type DiscoverReqData struct {
	Sequence uint
}
