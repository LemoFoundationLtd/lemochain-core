package protocol

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
)

var ProtocolName = "lemo"

// lemo 协议 codes
const (
	HeartbeatMsg      = 0x01 // 心跳
	StatusMsg         = 0x02 // 用于握手时发送/接收当前节点状态包括版本号，genesis的hash，current的hash等
	BlockHashesMsg    = 0x03 // block的hash集消息
	TxMsg             = 0x04
	GetBlocksMsg      = 0x05 // 获取区块集合消息
	GetSingleBlockMsg = 0x06 // 获取一个区块
	SingleBlockMsg    = 0x07 // 返回一个区块
	BlocksMsg         = 0x08 // 区块集消息
	NewBlockMsg       = 0x09 // 新的完整的block消息
	NewConfirmMsg     = 0x0a // 新区块确认消息
	GetConfirmInfoMsg = 0x0b // 获取确认包信息
	ConfirmInfoMsg    = 0x0c // 收到确信包信息
	// for find node
	FindNodeReqMsg = 0x0d // find node request message
	FindNodeResMsg = 0x0e // find node response message
)

type ErrCode int

const (
	ErrMsgTooLarge    = iota
	ErrDecode         // 解码错误
	ErrInvalidMsgCode // 不可用的消息code
	ErrInvalidMsg     // 消息内部出错
	ErrSendBlocks     // 发送区块到远程出错
	ErrNoBlocks       // 消息内没有区块数据
	ErrProtocolVersionMismatch
	// ErrNetworkIdMismatch
	ErrGenesisBlockMismatch
	ErrNoStatusMsg
	ErrExtraStatusMsg
	ErrSuspendedPeer
)

func (e ErrCode) String() string {
	return ErrorToString[int(e)]
}

// XXX change once legacy code is out
var ErrorToString = map[int]string{
	ErrMsgTooLarge:             "Message too long",
	ErrDecode:                  "Invalid message",
	ErrInvalidMsgCode:          "Invalid message code",
	ErrProtocolVersionMismatch: "Protocol version mismatch",
	// ErrNetworkIdMismatch:       "NetworkId mismatch",
	ErrGenesisBlockMismatch: "Genesis block mismatch",
	ErrNoStatusMsg:          "No status message",
	ErrExtraStatusMsg:       "Extra status message",
	ErrSuspendedPeer:        "Suspended peer",
}

// 节点当前状态信息
type NodeStatusData struct {
	ChainID       uint64
	CurrentHeight uint32
	CurrentBlock  common.Hash
	GenesisBlock  common.Hash
}

// BlockHashesData 新区块Hash数据
type BlockHashesData []struct {
	Hash   common.Hash
	Height uint32
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
	Hash   common.Hash
	Height uint32
}

// BlockConfirms 某个区块的所有确认信息
type BlockConfirms struct {
	Hash   common.Hash // 区块Hash
	Height uint32      //区块高度
	Pack   []types.SignData
}

// for find node
type FindNodeResData struct {
	Sequence int32
	Nodes    []string
}

// for find node
type FindNodeReqData struct {
	Sequence int32
}
