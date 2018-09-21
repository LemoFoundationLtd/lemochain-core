package p2p

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"io"
	"time"
)

type Msg struct {
	Code       uint32
	Size       uint32 // size of the paylod
	Payload    io.Reader
	ReceivedAt time.Time
}

// Empty 判断Msg是否为空消息
func (msg Msg) Empty() bool {
	emptyMsg := Msg{}
	return msg.Size == emptyMsg.Size && msg.Code == emptyMsg.Code && msg.ReceivedAt == emptyMsg.ReceivedAt
}

// Decode 将msg实际有效内容解码
func (msg Msg) Decode(data interface{}) error {
	s := rlp.NewStream(msg.Payload, uint64(msg.Size))
	if err := s.Decode(data); err != nil {
		return errors.New(fmt.Sprintf("rlp decode error, code:%d size:%d err:%v", msg.Code, msg.Size, err))
	}
	return nil
}

// CheckCode 检测code是否合法
func (msg Msg) CheckCode() bool {
	if msg.Code > 0x1F { // todo
		return true
	}
	return true
}
