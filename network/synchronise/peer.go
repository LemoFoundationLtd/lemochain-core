package synchronise

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"github.com/LemoFoundationLtd/lemochain-go/network/synchronise/protocol"
	"gopkg.in/fatih/set.v0"
	"io/ioutil"
	"sync"
	"time"
)

const (
	maxKnownTxs    = 65535
	maxKnownBlocks = 1024
)

type peer struct {
	id string
	p2p.IPeer

	head   common.Hash
	height uint32
	lock   sync.RWMutex

	knownTxs    set.Interface
	knownBlocks set.Interface
}

func newPeer(p p2p.IPeer) *peer {
	id := p.RNodeID()
	return &peer{
		id:          fmt.Sprintf("%x", id[:]),
		IPeer:       p,
		knownTxs:    set.New(set.ThreadSafe),
		knownBlocks: set.New(set.ThreadSafe),
	}
}

// Head 获取节点的最新头
func (p *peer) Head() (hash common.Hash, height uint32) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.head[:])
	height = p.height
	return
}

// SetHead 设置节点最新头
func (p *peer) SetHead(hash common.Hash, height uint32) {
	p.lock.Lock()
	defer p.lock.Unlock()

	copy(p.head[:], hash[:])
	p.height = height
}

// MarkBlock 标记节点拥有该块
func (p *peer) MarkBlock(hash common.Hash) {
	if p.knownBlocks.Size() == maxKnownBlocks {
		p.knownBlocks.Pop()
	}
	p.knownBlocks.Add(hash)
}

// MarkTransaction 标记节点拥有该交易
func (p *peer) MarkTransaction(hash common.Hash) {
	if p.knownTxs.Size() == maxKnownTxs {
		p.knownTxs.Pop()
	}
	p.knownTxs.Add(hash)
}

// Handshake 当前状态握手
func (p *peer) Handshake(chainID uint64, height uint32, head, genesis common.Hash) error {
	errs := make(chan error, 2)
	var status protocol.NodeStatusData
	// 发送自己的节点状态
	go func() {
		log.Debugf("start send node status data. nodeid: %s", p.id[:16])
		errs <- p.send(protocol.StatusMsg, &protocol.NodeStatusData{
			ChainID:       chainID,
			CurrentHeight: height,
			CurrentBlock:  head,
			GenesisBlock:  genesis,
		})
	}()
	// 读取对方的远程节点状态
	go func() {
		log.Debugf("start read remote node status data. nodeid: %s", p.id[:16])
		errs <- p.readRemoteStatus(chainID, &status, genesis)
	}()
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			return err
		}
	}
	p.head = status.CurrentBlock
	p.height = status.CurrentHeight
	return nil
}

// send 发送指定类型的数据
func (p *peer) send(msgCode uint32, data interface{}) error {
	_, r, err := rlp.EncodeToReader(data)
	if err != nil {
		return err
	}
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return p.IPeer.WriteMsg(msgCode, buf)
}

// readRemoteStatus 读取远程节点最新状态
func (p *peer) readRemoteStatus(network uint64, status *protocol.NodeStatusData, genesis common.Hash) error {
	newMsgCh := make(chan p2p.Msg)
	msgCh := make(chan p2p.Msg)
	// 读取
	go func() {
		msg, err := p.IPeer.ReadMsg()
		if err != nil {
			log.Debugf("readRemoteStatus: failed. %v", err)
			msgCh <- p2p.Msg{}
		}
		msgCh <- msg
	}()
	go func() {
		select {
		case <-time.After(5 * time.Second):
			newMsgCh <- p2p.Msg{}
		case msg := <-msgCh:
			newMsgCh <- msg
		}
	}()

	msg := <-newMsgCh
	if msg.Empty() {
		return errors.New("read message timeout")
	}
	if msg.Code != protocol.StatusMsg {
		return errors.New("code not match")
	}
	if err := msg.Decode(status); err != nil {
		return err
	}
	if status.ChainID != network {
		return errors.New("networkid not match")
	}
	if bytes.Compare(status.GenesisBlock[:], genesis[:]) != 0 {
		return p2p.ErrGenesisNotMatch
	}
	return nil
}

// RequestBlockFromAndTo 从高度为from同步区块到高度to
func (p *peer) RequestBlockFromAndTo(from, to uint32) error {
	data := &protocol.GetBlocksData{From: from, To: to}
	return p.send(protocol.GetBlocksMsg, data)
}

// RequestBlock 请求一个区块
func (p *peer) RequestOneBlock(hash common.Hash, height uint32) error {
	data := &protocol.GetSingleBlockData{Hash: hash, Height: height}
	return p.send(protocol.GetSingleBlockMsg, data)
}

// SendTransactions 发送交易
func (p *peer) SendTransactions(txs types.Transactions) error {
	for _, tx := range txs {
		p.knownTxs.Add(tx.Hash())
	}
	return p.send(protocol.TxMsg, txs)
}

func (p *peer) RemoteNetInfo() string {
	// todo
	return p.IPeer.RAddress()
}
