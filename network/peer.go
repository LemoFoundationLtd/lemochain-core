package network

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"sync"
	"time"
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

// Close
func (p *peer) Close() {
	p.conn.Close()
}

// RequestBlocks request blocks from remote
func (p *peer) RequestBlocks(from, to uint32) error {
	msg := &GetBlocksData{From: from, To: to}
	buf, err := rlp.EncodeToBytes(&msg)
	if err != nil {
		return err
	}
	return p.conn.WriteMsg(GetBlocksMsg, buf)
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
	timeout := time.NewTimer(5 * time.Second)
	select {
	case <-timeout.C:
		return nil, errors.New("protocol handshake timeout")
	case msg := <-msgCh:
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

// SendTxs send txs to remote
func (p *peer) SendTxs(txs types.Transactions) error {
	buf, err := rlp.EncodeToBytes(&txs)
	if err != nil {
		return err
	}
	return p.conn.WriteMsg(TxsMsg, buf)
}

// SendBlockHash send block hash to remote
func (p *peer) SendBlockHash(height uint32, hash common.Hash) error {
	msg := &BlockHashData{Height: height, Hash: hash}
	buf, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	return p.conn.WriteMsg(BlockHashMsg, buf)
}

// SendBlocks send blocks to remote
func (p *peer) SendBlocks(blocks types.Blocks) error {
	buf, err := rlp.EncodeToBytes(&blocks)
	if err != nil {
		return err
	}
	return p.conn.WriteMsg(BlocksMsg, buf)
}

// SendConfirms send confirms to remote peer
func (p *peer) SendConfirms(confirms *BlockConfirms) error {
	buf, err := rlp.EncodeToBytes(confirms)
	if err != nil {
		return err
	}
	return p.conn.WriteMsg(ConfirmsMsg, buf)
}

// SendDiscover send discover request
func (p *peer) SendDiscover() error {
	msg := &DiscoverReqData{Sequence: 1}
	buf, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	p.discoverCounter++
	return p.conn.WriteMsg(DiscoverReqMsg, buf)
}

// SendDiscoverResp send response for discover
func (p *peer) SendDiscoverResp(resp *DiscoverResData) error {
	buf, err := rlp.EncodeToBytes(resp)
	if err != nil {
		return err
	}
	return p.conn.WriteMsg(DiscoverResMsg, buf)
}

// SendReqLatestStatus send request of latest status
func (p *peer) SendReqLatestStatus() error {
	msg := &GetLatestStatus{Revert: uint32(0)}
	buf, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	return p.conn.WriteMsg(GetLstStatusMsg, buf)
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
