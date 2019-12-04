package network

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-core/network/p2p"
	"sync"
	"time"
)

const (
	DurShort = 3 * time.Second
	DurLong  = 10 * time.Second
)

// ProtocolHandshake protocol handshake
type ProtocolHandshake struct {
	ChainID      uint16
	GenesisHash  common.Hash
	NodeVersion  uint32
	LatestStatus LatestStatus
}

// Bytes object to bytes
func (phs *ProtocolHandshake) Bytes() []byte {
	buf, err := rlp.EncodeToBytes(phs)
	if err != nil {
		return nil
	}
	return buf
}

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
func (p *peer) RequestBlocks(from, to uint32) int {
	if from > to {
		log.Warnf("RequestBlocks: from: %d can't be larger than to:%d", from, to)
		return -1
	}
	msg := &GetBlocksData{From: from, To: to}
	buf, err := rlp.EncodeToBytes(&msg)
	if err != nil {
		log.Warnf("RequestBlocks: rlp encode failed: %v", err)
		return -2
	}
	p.conn.SetWriteDeadline(DurShort)
	if err = p.conn.WriteMsg(GetBlocksMsg, buf); err != nil {
		log.Warnf("RequestBlocks: write message failed: %v", err)
		return -3
	}
	return 0
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
			log.Errorf("Handshake ReadMsg error: %v", err)
			msgCh <- nil
		}
	}()
	timeout := time.NewTimer(8 * time.Second)
	select {
	case <-timeout.C:
		return nil, ErrReadTimeout
	case msg := <-msgCh:
		if msg == nil {
			return nil, ErrReadMsg
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
func (p *peer) SendLstStatus(status *LatestStatus) error {
	buf, err := rlp.EncodeToBytes(status)
	if err != nil {
		log.Warnf("SendLstStatus: rlp encode failed: %v", err)
		return err
	}
	p.conn.SetWriteDeadline(DurShort)
	if err = p.conn.WriteMsg(LstStatusMsg, buf); err != nil {
		log.Warnf("SendLstStatus to peer: %s failed: %v", p.NodeID().String()[:16], err)
		p.conn.Close()
		return err
	}
	return nil
}

// SendTxs send txs to remote
func (p *peer) SendTxs(txs types.Transactions) error {
	buf, err := rlp.EncodeToBytes(&txs)
	if err != nil {
		log.Warnf("SendTxs: rlp failed: %v", err)
		return err
	}
	if err := p.conn.WriteMsg(TxsMsg, buf); err != nil {
		log.Warnf("SendTxs to peer: %s failed. disconnect. %v", p.NodeID().String()[:16], err)
		p.conn.Close()
		return err
	}
	return nil
}

// SendConfirmInfo send confirm message to deputy nodes
func (p *peer) SendConfirmInfo(confirmInfo *BlockConfirmData) error {
	buf, err := rlp.EncodeToBytes(confirmInfo)
	if err != nil {
		log.Warnf("SendConfirmInfo: rlp failed: %v", err)
		return err
	}
	p.conn.SetWriteDeadline(DurShort)
	if err := p.conn.WriteMsg(ConfirmMsg, buf); err != nil {
		log.Warnf("SendConfirmInfo to peer: %s failed. disconnect. %v", p.NodeID().String()[:16], err)
		p.conn.Close()
		return err
	}
	return nil
}

// SendBlockHash send block hash to remote
func (p *peer) SendBlockHash(height uint32, hash common.Hash) error {
	msg := &BlockHashData{Height: height, Hash: hash}
	buf, err := rlp.EncodeToBytes(msg)
	if err != nil {
		log.Warnf("SendBlockHash: rlp failed: %v", err)
		return err
	}
	p.conn.SetWriteDeadline(DurShort)
	if err := p.conn.WriteMsg(BlockHashMsg, buf); err != nil {
		log.Warnf("SendBlockHash: %s failed. disconnect. %v", p.NodeID().String()[:16], err)
		p.conn.Close()
		return err
	}
	return nil
}

// SendBlocks send blocks to remote
func (p *peer) SendBlocks(blocks types.Blocks) error {
	buf, err := rlp.EncodeToBytes(&blocks)
	if err != nil {
		log.Warnf("SendBlocks: rlp failed: %v", err)
		return err
	}
	if err := p.conn.WriteMsg(BlocksMsg, buf); err != nil {
		// log.Warnf("SendBlocks to peer: %s failed. disconnect. %v", p.NodeID().String()[:16], err)
		p.conn.Close()
		return err
	}
	// for test
	// block := blocks[0]
	// if len(block.TxSet) > 0 {
	// 	for _, tx := range block.TxSet {
	// 		if tx.Type() == uint8(2) { // 0: common tx; 1: vote tx; 2: register for cand*
	// 			log.Debugf("block: %s", block.Json())
	// 			// log.Debugf("block rlp: %s", common.ToHex(buf))
	// 		} else {
	// 			log.Debugf("block has other type txs")
	// 		}
	// 	}
	// }
	return nil
}

// SendConfirms send confirms to remote peer
func (p *peer) SendConfirms(confirms *BlockConfirms) error {
	buf, err := rlp.EncodeToBytes(confirms)
	if err != nil {
		log.Warnf("SendConfirms: rlp failed: %v", err)
		return err
	}
	p.conn.SetWriteDeadline(DurLong)
	if err := p.conn.WriteMsg(ConfirmsMsg, buf); err != nil {
		log.Warnf("SendConfirms to peer: %s failed. disconnect. %v", p.NodeID().String()[:16], err)
		p.conn.Close()
		return err
	}
	return nil
}

// SendGetConfirms send request of getting confirms
func (p *peer) SendGetConfirms(height uint32, hash common.Hash) error {
	msg := &GetConfirmInfo{Height: height, Hash: hash}
	buf, err := rlp.EncodeToBytes(msg)
	if err != nil {
		log.Warnf("SendGetConfirms: rlp failed: %v", err)
		return err
	}
	p.conn.SetWriteDeadline(DurShort)
	if err := p.conn.WriteMsg(GetConfirmsMsg, buf); err != nil {
		log.Warnf("SendGetConfirms to peer: %s failed. disconnect. %v", p.NodeID().String()[:16], err)
		p.conn.Close()
		return err
	}
	return nil
}

// SendDiscover send discover request
func (p *peer) SendDiscover() error {
	msg := &DiscoverReqData{Sequence: 1}
	buf, err := rlp.EncodeToBytes(msg)
	if err != nil {
		log.Warnf("SendDiscover: rlp failed: %v", err)
		return err
	}
	p.discoverCounter++
	p.conn.SetWriteDeadline(DurShort)
	if err := p.conn.WriteMsg(DiscoverReqMsg, buf); err != nil {
		log.Warnf("SendDiscover to peer: %s failed. disconnect. %v", p.NodeID().String()[:16], err)
		p.conn.Close()
		return err
	}
	return nil
}

// SendDiscoverResp send response for discover
func (p *peer) SendDiscoverResp(resp *DiscoverResData) error {
	buf, err := rlp.EncodeToBytes(resp)
	if err != nil {
		log.Warnf("SendDiscoverResp: rlp failed: %v", err)
		return err
	}
	if err := p.conn.WriteMsg(DiscoverResMsg, buf); err != nil {
		log.Warnf("SendDiscoverResp to peer: %s failed. disconnect. %v", p.NodeID().String()[:16], err)
		p.conn.Close()
		return err
	}
	return nil
}

// SendReqLatestStatus send request of latest status
func (p *peer) SendReqLatestStatus() error {
	msg := &GetLatestStatus{Revert: uint32(0)}
	buf, err := rlp.EncodeToBytes(msg)
	if err != nil {
		log.Warnf("SendReqLatestStatus: rlp failed: %v", err)
		return err
	}
	p.conn.SetWriteDeadline(DurShort)
	if err := p.conn.WriteMsg(GetLstStatusMsg, buf); err != nil {
		log.Warnf("SendReqLatestStatus to peer: %s failed. disconnect. %v", p.NodeID().String()[:16], err)
		p.conn.Close()
		return err
	}
	return nil
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
