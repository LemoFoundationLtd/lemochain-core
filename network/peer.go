package network

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"sync"
	"time"
)

const (
	DurShort  = 3 * time.Second
	DurMiddle = 5 * time.Second
	DurLong   = 10 * time.Second
)

// LatestStatus latest peer's status
type LatestStatus struct {
	CurHeight uint32
	CurHash   common.Hash

	StaHeight uint32
	StaHash   common.Hash
}

type peer struct {
	conn p2p.IPeer

	lstStatus       LatestStatus
	badSyncCounter  uint32
	discoverCounter uint32

	lock sync.RWMutex
}

// newPeer new peer instance
func newPeer(p p2p.IPeer) *peer {
	return &peer{
		conn:            p,
		badSyncCounter:  uint32(0),
		discoverCounter: uint32(0),
	}
}

// NormalClose
func (p *peer) NormalClose() {
	p.conn.Close()
	p.conn.SetStatus(p2p.StatusNormal)
}

// ManualClose
func (p *peer) ManualClose() {
	p.conn.Close()
	p.conn.SetStatus(p2p.StatusManualDisconnect)
}

// FailedHandshakeClose
func (p *peer) FailedHandshakeClose() {
	p.conn.Close()
	p.conn.SetStatus(p2p.StatusFailedHandshake)
}

// RcvBadDataClose
func (p *peer) RcvBadDataClose() {
	p.conn.Close()
	p.conn.SetStatus(p2p.StatusBadData)
}

// HardForkClose
func (p *peer) HardForkClose() {
	p.conn.Close()
	p.conn.SetStatus(p2p.StatusHardFork)
}

// RequestBlocks request blocks from remote
func (p *peer) RequestBlocks(from, to uint32) {
	if from > to {
		log.Warnf("RequestBlocks: from: %d can't be larger than to:%d", from, to)
		return
	}
	msg := &GetBlocksData{From: from, To: to}
	buf, err := rlp.EncodeToBytes(&msg)
	if err != nil {
		log.Warnf("RequestBlocks: rlp encode failed: %v", err)
		return
	}
	p.conn.SetWriteDeadline(DurShort)
	if err = p.conn.WriteMsg(GetBlocksMsg, buf); err != nil {
		log.Warnf("RequestBlocks: write message failed: %v", err)
	}
}

// Handshake protocol handshake
func (p *peer) Handshake(content []byte) (*ProtocolHandshake, error) {
	// write to remote
	if err := p.conn.WriteMsg(ProHandshakeMsg, content); err != nil {
		return nil, err
	}
	// read from remote
	msgCh := make(chan *p2p.Msg)
	go func() {
		if msg, err := p.conn.ReadMsg(); err == nil {
			msgCh <- msg
		} else {
			msgCh <- nil
		}
	}()
	timeout := time.NewTimer(8 * time.Second)
	select {
	case <-timeout.C:
		return nil, errors.New("protocol handshake timeout")
	case msg := <-msgCh:
		if msg == nil {
			return nil, errors.New("protocol handshake failed: read remote message failed")
		}
		var phs ProtocolHandshake
		if err := msg.Decode(&phs); err != nil {
			return nil, err
		}
		return &phs, nil
	}
}

// NodeID
func (p *peer) NodeID() *p2p.NodeID {
	return p.conn.RNodeID()
}

// ReadMsg read message from net stream
func (p *peer) ReadMsg() (*p2p.Msg, error) {
	return p.conn.ReadMsg()
}

// SendLstStatus send SyncFailednode's status to remote
func (p *peer) SendLstStatus(status *LatestStatus) {
	buf, err := rlp.EncodeToBytes(status)
	if err != nil {
		log.Warnf("SendLstStatus: rlp encode failed: %v", err)
		return
	}
	p.conn.SetWriteDeadline(DurShort)
	if err = p.conn.WriteMsg(LstStatusMsg, buf); err != nil {
		log.Warnf("send latest status failed: %v", err)
	}
}

// SendTxs send txs to remote
func (p *peer) SendTxs(txs types.Transactions) {
	buf, err := rlp.EncodeToBytes(&txs)
	if err != nil {
		log.Warnf("SendTxs: rlp failed: %v", err)
		return
	}
	if err := p.conn.WriteMsg(TxsMsg, buf); err != nil {
		log.Warnf("SendTxs to peer: %s failed. disconnect.", p.NodeID().String()[:16])
		p.conn.Close()
	}
}

// SendConfirmInfo send confirm message to deputy nodes
func (p *peer) SendConfirmInfo(confirmInfo *BlockConfirmData) {
	buf, err := rlp.EncodeToBytes(confirmInfo)
	if err != nil {
		log.Warnf("SendConfirmInfo: rlp failed: %v", err)
		return
	}
	p.conn.SetWriteDeadline(DurShort)
	if err := p.conn.WriteMsg(ConfirmMsg, buf); err != nil {
		log.Warnf("SendConfirmInfo to peer: %s failed. disconnect.", p.NodeID().String()[:16])
		p.conn.Close()
	}
}

// SendBlockHash send block hash to remote
func (p *peer) SendBlockHash(height uint32, hash common.Hash) {
	msg := &BlockHashData{Height: height, Hash: hash}
	buf, err := rlp.EncodeToBytes(msg)
	if err != nil {
		log.Warnf("SendBlockHash: rlp failed: %v", err)
		return
	}
	p.conn.SetWriteDeadline(DurShort)
	if err := p.conn.WriteMsg(BlockHashMsg, buf); err != nil {
		log.Warnf("SendBlockHash to peer: %s failed. disconnect.", p.NodeID().String()[:16])
		p.conn.Close()
	}
}

// SendBlocks send blocks to remote
func (p *peer) SendBlocks(blocks types.Blocks) {
	buf, err := rlp.EncodeToBytes(&blocks)
	if err != nil {
		log.Warnf("SendBlocks: rlp failed: %v", err)
		return
	}
	if err := p.conn.WriteMsg(BlocksMsg, buf); err != nil {
		log.Warnf("SendBlocks to peer: %s failed. disconnect.", p.NodeID().String()[:16])
		p.conn.Close()
	}
}

// SendConfirms send confirms to remote peer
func (p *peer) SendConfirms(confirms *BlockConfirms) {
	buf, err := rlp.EncodeToBytes(confirms)
	if err != nil {
		log.Warnf("SendConfirms: rlp failed: %v", err)
		return
	}
	p.conn.SetWriteDeadline(DurLong)
	if err := p.conn.WriteMsg(ConfirmsMsg, buf); err != nil {
		log.Warnf("SendConfirms to peer: %s failed. disconnect.", p.NodeID().String()[:16])
		p.conn.Close()
	}
}

// SendGetConfirms send request of getting confirms
func (p *peer) SendGetConfirms(height uint32, hash common.Hash) {
	msg := &GetConfirmInfo{Height: height, Hash: hash}
	buf, err := rlp.EncodeToBytes(msg)
	if err != nil {
		log.Warnf("SendGetConfirms: rlp failed: %v", err)
		return
	}
	p.conn.SetWriteDeadline(DurShort)
	if err := p.conn.WriteMsg(GetConfirmsMsg, buf); err != nil {
		log.Warnf("SendGetConfirms to peer: %s failed. disconnect.", p.NodeID().String()[:16])
		p.conn.Close()
	}
}

// SendDiscover send discover request
func (p *peer) SendDiscover() {
	msg := &DiscoverReqData{Sequence: 1}
	buf, err := rlp.EncodeToBytes(msg)
	if err != nil {
		log.Warnf("SendDiscover: rlp failed: %v", err)
		return
	}
	p.discoverCounter++
	p.conn.SetWriteDeadline(DurShort)
	if err := p.conn.WriteMsg(DiscoverReqMsg, buf); err != nil {
		log.Warnf("SendDiscover to peer: %s failed. disconnect.", p.NodeID().String()[:16])
		p.conn.Close()
	}
}

// SendDiscoverResp send response for discover
func (p *peer) SendDiscoverResp(resp *DiscoverResData) {
	buf, err := rlp.EncodeToBytes(resp)
	if err != nil {
		log.Warnf("SendDiscoverResp: rlp failed: %v", err)
		return
	}
	if err := p.conn.WriteMsg(DiscoverResMsg, buf); err != nil {
		log.Warnf("SendDiscoverResp to peer: %s failed. disconnect.", p.NodeID().String()[:16])
		p.conn.Close()
	}
}

// SendReqLatestStatus send request of latest status
func (p *peer) SendReqLatestStatus() {
	msg := &GetLatestStatus{Revert: uint32(0)}
	buf, err := rlp.EncodeToBytes(msg)
	if err != nil {
		log.Warnf("SendReqLatestStatus: rlp failed: %v", err)
		return
	}
	p.conn.SetWriteDeadline(DurShort)
	if err := p.conn.WriteMsg(GetLstStatusMsg, buf); err != nil {
		log.Warnf("SendReqLatestStatus to peer: %s failed. disconnect.", p.NodeID().String()[:16])
		p.conn.Close()
	}
}

// LatestStatus return record of latest status
func (p *peer) LatestStatus() LatestStatus {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.lstStatus
}

// UpdateStatus update peer's latest status
func (p *peer) UpdateStatus(height uint32, hash common.Hash) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.lstStatus.CurHeight < height {
		p.lstStatus.CurHeight = height
		p.lstStatus.CurHash = hash
	}
}

// SyncFailed
func (p *peer) SyncFailed() {
	p.badSyncCounter++
}

// BadSyncCounter
func (p *peer) BadSyncCounter() uint32 {
	return p.badSyncCounter
}

// DiscoverCounter get discover counter
func (p *peer) DiscoverCounter() uint32 {
	return p.discoverCounter
}
