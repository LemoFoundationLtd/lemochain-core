package network

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
)

// protocol code
const (
	HeartbeatMsg    = 0x01 // 心跳
	ProHandshakeMsg = 0x02
	LstStatusMsg    = 0x03
	GetLstStatusMsg = 0x04
	BlockHashMsg    = 0x05 // block的hash集消息
	TxsMsg          = 0x06
	GetBlocksMsg    = 0x07 // 获取区块集合消息
	BlocksMsg       = 0x08 // 区块集消息
	ConfirmMsg      = 0x09 // 新区块确认消息
	GetConfirmsMsg  = 0x0a // 获取确认包信息
	ConfirmsMsg     = 0x0b // 收到确信包信息
	// for find node
	DiscoverReqMsg = 0x0c // find node request message
	DiscoverResMsg = 0x0d // find node response message
)

// GetLatestStatus get latest status
type GetLatestStatus struct {
	Revert uint32
}

// BlockHashData 新区块Hash数据
type BlockHashData struct {
	Height uint32
	Hash   common.Hash
}

// getBlocksData 获取区块集
type GetBlocksData struct {
	From uint32
	To   uint32
}

// GetSingleBlockData 单独获取一个区块
type GetSingleBlockData struct {
	Hash   common.Hash
	Height uint32
}

// blockConfirmData 区块确认信息
type BlockConfirmData struct {
	Hash     common.Hash    // 区块Hash
	Height   uint32         //区块高度
	SignInfo types.SignData // 签名信息
}

// getConfirmInfo 主动拉取确认包
type GetConfirmInfo struct {
	Height uint32
	Hash   common.Hash
}

// BlockConfirms 某个区块的所有确认信息
type BlockConfirms struct {
	Height uint32      //区块高度
	Hash   common.Hash // 区块Hash
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
