package p2p

import (
	"bytes"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"time"
)

//go:generate stringer -type MsgCode

type MsgCode uint32

const (
	HeartbeatMsg    MsgCode = 0x01 // heartbeat message
	ProHandshakeMsg MsgCode = 0x02 // protocol handshake message
	LstStatusMsg    MsgCode = 0x03 // latest status message
	GetLstStatusMsg MsgCode = 0x04 // get latest status message
	BlockHashMsg    MsgCode = 0x05 // block's hash message
	TxsMsg          MsgCode = 0x06 // transactions message
	GetBlocksMsg    MsgCode = 0x07 // get blocks message
	BlocksMsg       MsgCode = 0x08 // blocks message
	ConfirmMsg      MsgCode = 0x09 // a confirm of one block message
	GetConfirmsMsg  MsgCode = 0x0a // get confirms of one block message
	ConfirmsMsg     MsgCode = 0x0b // confirms of one block message
	// for find node
	DiscoverReqMsg MsgCode = 0x0c // find node request message
	DiscoverResMsg MsgCode = 0x0d // find node response message

	// for lemochain-server and light node
	GetBlocksWithChangeLogMsg MsgCode = 0x0e
)

type Msg struct {
	Code       MsgCode
	Content    []byte
	ReceivedAt time.Time
}

// Empty is msg empty
func (msg *Msg) Empty() bool {
	emptyMsg := Msg{}
	return msg.Code == emptyMsg.Code && msg.Content == nil && msg.ReceivedAt == emptyMsg.ReceivedAt
}

// Decode decode stream to object
func (msg *Msg) Decode(data interface{}) error {
	reader := bytes.NewReader(msg.Content)
	length := len(msg.Content)
	s := rlp.NewStream(reader, uint64(length))
	if err := s.Decode(data); err != nil {
		log.Debugf("Msg.Decode error: %v, data: %s", err, common.ToHex(msg.Content))
		return ErrRlpDecode
	}
	return nil
}

// CheckCode is code invalid
func (msg *Msg) CheckCode() bool {
	if msg.Code > 0x1F {
		return false
	}
	return true
}
