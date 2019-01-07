package network

import (
	"crypto/ecdsa"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type testPeer struct {
	writeStatus int
	readStatus  int
	closeStatus int32
}

func (p *testPeer) ReadMsg() (msg *p2p.Msg, err error) {
	if p.readStatus == 1 {
		msg = new(p2p.Msg)
		msg.Content = []byte{1, 2, 3}
		msg.Code = 2
		msg.ReceivedAt = time.Now()
	} else if p.readStatus == 2 {
		time.Sleep(9 * time.Second)
	} else if p.readStatus == 3 {
		h := ProtocolHandshake{ChainID: 100, NodeVersion: 12}
		msg = new(p2p.Msg)
		msg.Code = 2
		msg.ReceivedAt = time.Now()
		buf, _ := rlp.EncodeToBytes(&h)
		msg.Content = buf
	} else {
		err = errors.New("EOF")
	}
	return
}
func (p *testPeer) WriteMsg(code uint32, msg []byte) (err error) {
	if p.writeStatus == 0 {
		return nil
	} else if p.writeStatus == 1 {
		return errors.New("EOF")
	} else {
		return errors.New("others")
	}
}
func (p *testPeer) SetWriteDeadline(duration time.Duration) {}
func (p *testPeer) RNodeID() *p2p.NodeID {
	return &p2p.NodeID{0x01, 0x02, 0x03, 0x04, 0x05}
}
func (p *testPeer) RAddress() string                                            { return "" }
func (p *testPeer) LAddress() string                                            { return "nil" }
func (p *testPeer) DoHandshake(prv *ecdsa.PrivateKey, nodeID *p2p.NodeID) error { return nil }
func (p *testPeer) Run() (err error)                                            { return nil }
func (p *testPeer) NeedReConnect() bool                                         { return true }
func (p *testPeer) SetStatus(status int32)                                      { p.closeStatus = status }
func (p *testPeer) Close()                                                      {}

func Test_Close(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	p.HardForkClose()
	assert.Equal(t, p2p.StatusHardFork, rawP.closeStatus)

	p.RcvBadDataClose()
	assert.Equal(t, p2p.StatusBadData, rawP.closeStatus)

	p.FailedHandshakeClose()
	assert.Equal(t, p2p.StatusFailedHandshake, rawP.closeStatus)

	p.ManualClose()
	assert.Equal(t, p2p.StatusManualDisconnect, rawP.closeStatus)

	p.NormalClose()
	assert.Equal(t, p2p.StatusNormal, rawP.closeStatus)
}

func Test_RequestBlocks(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	res := p.RequestBlocks(10, 9)
	assert.Equal(t, -1, res)

	rawP.writeStatus = 1
	res = p.RequestBlocks(1, 9)
	assert.Equal(t, -3, res)

	rawP.writeStatus = 0
	res = p.RequestBlocks(1, 9)
	assert.Equal(t, 0, res)
}

func Test_Handshake(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)

	rawP.writeStatus = 1
	res, err := p.Handshake(nil)
	assert.Nil(t, res)
	assert.Equal(t, "EOF", err.Error())

	rawP.writeStatus = 0
	rawP.readStatus = 0
	res, err = p.Handshake(nil)
	assert.Nil(t, res)
	assert.Equal(t, ErrReadMsg, err)

	rawP.writeStatus = 0
	rawP.readStatus = 2
	res, err = p.Handshake(nil)
	assert.Nil(t, res)
	assert.Equal(t, ErrReadTimeout, err)

	rawP.writeStatus = 0
	rawP.readStatus = 1
	res, err = p.Handshake(nil)
	assert.Equal(t, p2p.ErrRlpDecode, err)

	rawP.writeStatus = 0
	rawP.readStatus = 3
	res, err = p.Handshake(nil)
	assert.Nil(t, err)
}

func Test_ReadMsg(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.readStatus = 3
	msg, err := p.ReadMsg()
	assert.Nil(t, err)
	assert.NotNil(t, msg)
}

func Test_SendLstStatus(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.readStatus = 3

	res := p.SendLstStatus(nil)
	assert.Equal(t, 0, res)

	status := new(LatestStatus)
	rawP.writeStatus = 1
	res = p.SendLstStatus(status)
	assert.Equal(t, -2, res)

	rawP.writeStatus = 0
	res = p.SendLstStatus(status)
	assert.Equal(t, 0, res)
}

func Test_SendTxs(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendTxs(nil)
	assert.Equal(t, 0, res)

	rawP.writeStatus = 1
	res = p.SendTxs(nil)
	assert.Equal(t, -2, res)

	txs := make([]*types.Transaction, 0)
	rawP.writeStatus = 0
	res = p.SendTxs(txs)
	assert.Equal(t, 0, res)
}

func Test_SendConfirmInfo(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendConfirmInfo(nil)
	assert.Equal(t, 0, res)

	rawP.writeStatus = 1
	res = p.SendConfirmInfo(nil)
	assert.Equal(t, -2, res)

	confirm := new(BlockConfirmData)
	rawP.writeStatus = 0
	res = p.SendConfirmInfo(confirm)
	assert.Equal(t, 0, res)
}

func Test_SendBlockHash(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendBlockHash(222, common.Hash{})
	assert.Equal(t, 0, res)

	rawP.writeStatus = 1
	res = p.SendBlockHash(222, common.Hash{})
	assert.Equal(t, -2, res)
}

func Test_SendBlocks(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendBlocks(nil)
	assert.Equal(t, 0, res)

	rawP.writeStatus = 1
	res = p.SendBlocks(nil)
	assert.Equal(t, -2, res)

	blocks := make([]*types.Block, 0)
	rawP.writeStatus = 0
	res = p.SendBlocks(blocks)
	assert.Equal(t, 0, res)
}

func Test_SendConfirms(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendConfirms(nil)
	assert.Equal(t, 0, res)

	rawP.writeStatus = 1
	res = p.SendConfirms(nil)
	assert.Equal(t, -2, res)

	confirms := new(BlockConfirms)
	rawP.writeStatus = 0
	res = p.SendConfirms(confirms)
	assert.Equal(t, 0, res)
}

func Test_SendGetConfirms(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendGetConfirms(10, common.Hash{})
	assert.Equal(t, 0, res)

	rawP.writeStatus = 1
	res = p.SendGetConfirms(10, common.Hash{})
	assert.Equal(t, -2, res)

	rawP.writeStatus = 0
	res = p.SendGetConfirms(10, common.Hash{})
	assert.Equal(t, 0, res)
}

func Test_SendDiscover(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendDiscover()
	assert.Equal(t, 0, res)

	rawP.writeStatus = 1
	res = p.SendDiscover()
	assert.Equal(t, -2, res)

	rawP.writeStatus = 0
	res = p.SendDiscover()
	assert.Equal(t, 0, res)
}

func Test_SendDiscoverResp(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendDiscoverResp(nil)
	assert.Equal(t, 0, res)

	rawP.writeStatus = 1
	res = p.SendDiscoverResp(nil)
	assert.Equal(t, -2, res)

	resp := new(DiscoverResData)
	rawP.writeStatus = 0
	res = p.SendDiscoverResp(resp)
	assert.Equal(t, 0, res)
}

func Test_SendReqLatestStatus(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendReqLatestStatus()
	assert.Equal(t, 0, res)

	rawP.writeStatus = 1
	res = p.SendReqLatestStatus()
	assert.Equal(t, -2, res)

	rawP.writeStatus = 0
	res = p.SendReqLatestStatus()
	assert.Equal(t, 0, res)
}

func Test_UpdateStatus(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	hash := common.Hash{}
	copy(hash[:], []byte{0x01, 0x02, 0x03, 0x04, 0x05})
	p.UpdateStatus(5, hash)

	status := p.LatestStatus()

	assert.Equal(t, uint32(5), status.CurHeight)
	assert.Equal(t, hash, status.CurHash)
}
