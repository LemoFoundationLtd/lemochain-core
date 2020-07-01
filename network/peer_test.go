package network

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-core/network/p2p"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type testPeer struct {
	writeStatus int
	readStatus  int
	closeStatus int32

	state int
}

func (p *testPeer) ReadMsg() (msg *p2p.Msg, err error) {
	if p.readStatus == 1 {
		msg = new(p2p.Msg)
		msg.Content = []byte{1, 2, 3}
		msg.Code = p2p.ProHandshakeMsg
		msg.ReceivedAt = time.Now()
	} else if p.readStatus == 2 {
		time.Sleep(9 * time.Second)
	} else if p.readStatus == 3 {
		h := ProtocolHandshake{ChainID: 100, NodeVersion: 12}
		msg = new(p2p.Msg)
		msg.Code = p2p.ProHandshakeMsg
		msg.ReceivedAt = time.Now()
		buf, _ := rlp.EncodeToBytes(&h)
		msg.Content = buf
	} else {
		err = errors.New("EOF")
	}
	return
}
func (p *testPeer) WriteMsg(code p2p.MsgCode, msg []byte) (err error) {
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
	if p.state == 1 {
		buf := common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0")
		id := new(p2p.NodeID)
		copy(id[:], buf)
		return id
	} else if p.state == 2 {
		buf := common.FromHex("0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0")
		id := new(p2p.NodeID)
		copy(id[:], buf)
		return id
	} else if p.state == 3 {
		buf := common.FromHex("0x7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2d0347692f36ef736f342fb5ce4710f7e337f062cc2110d134b63a9575f78cb167bfae2f43")
		id := new(p2p.NodeID)
		copy(id[:], buf)
		return id
	} else if p.state == 4 {
		buf := common.FromHex("0x33333f789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f")
		id := new(p2p.NodeID)
		copy(id[:], buf)
		return id
	} else if p.state == 5 {
		return &p2p.NodeID{0x01, 0x02, 0x03, 0x04, 0x05}
	}
	return &p2p.NodeID{0x01, 0x01, 0x01}
}
func (p *testPeer) RAddress() string                                            { return "" }
func (p *testPeer) LAddress() string                                            { return "nil" }
func (p *testPeer) DoHandshake(prv *ecdsa.PrivateKey, nodeID *p2p.NodeID) error { return nil }
func (p *testPeer) Run() (err error)                                            { return nil }
func (p *testPeer) NeedReConnect() bool                                         { return true }
func (p *testPeer) SetStatus(status int32)                                      { p.closeStatus = status }
func (p *testPeer) Close()                                                      {}

func Test_Bytes(t *testing.T) {
	target := common.FromHex("0xf86901a0010203000000000000000000000000000000000000000000000000000000000002f8440aa0010100000000000000000000000000000000000000000000000000000000000009a00202000000000000000000000000000000000000000000000000000000000000")
	shake := &ProtocolHandshake{
		ChainID:     1,
		GenesisHash: common.Hash{0x01, 0x02, 0x03},
		NodeVersion: 2,
		LatestStatus: LatestStatus{
			CurHeight: 10,
			CurHash:   common.Hash{0x01, 0x01},
			StaHeight: 9,
			StaHash:   common.Hash{0x02, 0x02},
		},
	}
	tmp := shake.Bytes()
	if bytes.Compare(target, tmp) != 0 {
		t.Error("ProtocolHandshake.Bytes not match")
	}
}

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
	assert.NoError(t, res)

	status := new(LatestStatus)
	rawP.writeStatus = 1
	res = p.SendLstStatus(status)
	assert.Error(t, res)

	rawP.writeStatus = 0
	res = p.SendLstStatus(status)
	assert.NoError(t, res)
}

func Test_SendTxs(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendTxs(nil)
	assert.NoError(t, res)

	rawP.writeStatus = 1
	res = p.SendTxs(nil)
	assert.Error(t, res)

	txs := make([]*types.Transaction, 0)
	rawP.writeStatus = 0
	res = p.SendTxs(txs)
	assert.NoError(t, res)
}

func Test_SendConfirmInfo(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendConfirmInfo(nil)
	assert.NoError(t, res)

	rawP.writeStatus = 1
	res = p.SendConfirmInfo(nil)
	assert.Error(t, res)

	confirm := new(BlockConfirmData)
	rawP.writeStatus = 0
	res = p.SendConfirmInfo(confirm)
	assert.NoError(t, res)
}

func Test_SendBlockHash(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendBlockHash(222, common.Hash{})
	assert.NoError(t, res)

	rawP.writeStatus = 1
	res = p.SendBlockHash(222, common.Hash{})
	assert.Error(t, res)
}

func Test_SendBlocks(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendBlocks(nil)
	assert.NoError(t, res)

	rawP.writeStatus = 1
	res = p.SendBlocks(nil)
	assert.Error(t, res)

	blocks := make([]*types.Block, 0)
	rawP.writeStatus = 0
	res = p.SendBlocks(blocks)
	assert.NoError(t, res)
}

func Test_SendConfirms(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendConfirms(nil)
	assert.NoError(t, res)

	rawP.writeStatus = 1
	res = p.SendConfirms(nil)
	assert.Error(t, res)

	confirms := new(BlockConfirms)
	rawP.writeStatus = 0
	res = p.SendConfirms(confirms)
	assert.NoError(t, res)
}

func Test_SendGetConfirms(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendGetConfirms(10, common.Hash{})
	assert.NoError(t, res)

	rawP.writeStatus = 1
	res = p.SendGetConfirms(10, common.Hash{})
	assert.Error(t, res)

	rawP.writeStatus = 0
	res = p.SendGetConfirms(10, common.Hash{})
	assert.NoError(t, res)
}

func Test_SendDiscover(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendDiscover()
	assert.NoError(t, res)

	rawP.writeStatus = 1
	res = p.SendDiscover()
	assert.Error(t, res)

	rawP.writeStatus = 0
	res = p.SendDiscover()
	assert.NoError(t, res)
}

func Test_SendDiscoverResp(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendDiscoverResp(nil)
	assert.NoError(t, res)

	rawP.writeStatus = 1
	res = p.SendDiscoverResp(nil)
	assert.Error(t, res)

	resp := new(DiscoverResData)
	rawP.writeStatus = 0
	res = p.SendDiscoverResp(resp)
	assert.NoError(t, res)
}

func Test_SendReqLatestStatus(t *testing.T) {
	rawP := &testPeer{}
	p := newPeer(rawP)
	rawP.writeStatus = 0
	res := p.SendReqLatestStatus()
	assert.NoError(t, res)

	rawP.writeStatus = 1
	res = p.SendReqLatestStatus()
	assert.Error(t, res)

	rawP.writeStatus = 0
	res = p.SendReqLatestStatus()
	assert.NoError(t, res)
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
