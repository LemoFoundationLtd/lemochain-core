package network

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
)

// protocol code
const (
	HeartbeatMsg    = 0x01 // heartbeat message
	ProHandshakeMsg = 0x02 // protocol handshake message
	LstStatusMsg    = 0x03 // latest status message
	GetLstStatusMsg = 0x04 // get latest status message
	BlockHashMsg    = 0x05 // block's hash message
	TxsMsg          = 0x06 // transactions message
	GetBlocksMsg    = 0x07 // get blocks message
	BlocksMsg       = 0x08 // blocks message
	ConfirmMsg      = 0x09 // a confirm of one block message
	GetConfirmsMsg  = 0x0a // get confirms of one block message
	ConfirmsMsg     = 0x0b // confirms of one block message
	// for find node
	DiscoverReqMsg = 0x0c // find node request message
	DiscoverResMsg = 0x0d // find node response message

	// for lemochain-server and light node
	GetBlocksWithChangeLogMsg = 0x0e
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
