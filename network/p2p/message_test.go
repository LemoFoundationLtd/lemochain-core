package p2p

import (
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Empty(t *testing.T) {
	msg := &Msg{}
	assert.Equal(t, true, msg.Empty())
}

func Test_Decode(t *testing.T) {
	origin := &struct {
		Name string
		ID   uint32
	}{
		Name: "sman",
		ID:   uint32(110),
	}

	buf, err := rlp.EncodeToBytes(origin)
	assert.NoError(t, err)
	msg := &Msg{
		Content: buf,
	}
	target := &struct {
		Name string
		ID   uint32
	}{}
	err = msg.Decode(target)
	assert.NoError(t, err)
	assert.Equal(t, origin.Name, target.Name)
	assert.Equal(t, origin.ID, target.ID)

	msg.Content = []byte{0x01, 0x02, 0x03}
	err = msg.Decode(target)
	assert.Equal(t, ErrRlpDecode, err)
}

func Test_CheckCode(t *testing.T) {
	msg := &Msg{
		Code: MsgCode(2),
	}
	assert.Equal(t, true, msg.CheckCode())

	msg.Code = MsgCode(100000)
	assert.Equal(t, false, msg.CheckCode())
}
