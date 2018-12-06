package p2p

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"time"
)

type Msg struct {
	Code       uint32
	Content    []byte
	ReceivedAt time.Time
}

// Empty 判断Msg是否为空消息
func (msg Msg) Empty() bool {
	emptyMsg := Msg{}
	return msg.Code == emptyMsg.Code && msg.Content == nil && msg.ReceivedAt == emptyMsg.ReceivedAt
}

// Decode 将msg实际有效内容解码
func (msg Msg) Decode(data interface{}) error {
	reader := bytes.NewReader(msg.Content)
	length := len(msg.Content)
	s := rlp.NewStream(reader, uint64(length))
	if err := s.Decode(data); err != nil {
		return errors.New(fmt.Sprintf("rlp decode error, code:%d size:%d err:%v", msg.Code, length, err))
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
