package p2p

import (
	"bytes"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"time"
)

type Msg struct {
	Code       uint32
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
