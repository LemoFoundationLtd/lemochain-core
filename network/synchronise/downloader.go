package synchronise

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/network/synchronise/blockchain"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	requestTimeout = 15 * time.Second
)

var (
	errBusy          = errors.New("is synchronising")
	errUnknownPeer   = errors.New("peer is unknown")
	errBadPeer       = errors.New("bad peer ignored")
	errForceQuit     = errors.New("force quit")
	errUnknownParent = errors.New("Unknown Parent")
)

// peerConnection 一个网络连接对象
type peerConnection struct {
	id    string
	peer  *peer
	rwMux sync.RWMutex
}

// peerSet 网络连接节点集
type peerSet struct {
	peers map[string]*peerConnection
	mux   sync.Mutex
}

func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peerConnection, 0),
	}
}

// blockPack 区块包
type blockPack struct {
	peerID string
	blocks types.Blocks
}

// BestPeer get peer with highest block
func (ps *peerSet) BestPeer() *peerConnection {
	var p *peerConnection
	height := uint32(0)
	for _, item := range ps.peers {
		if item.peer.height >= height {
			p = item
			height = item.peer.height
		}
	}
	return p
}

// Peer get peer with id
func (ps *peerSet) Peer(id string) *peerConnection {
	ps.mux.Lock()
	defer ps.mux.Unlock()
	return ps.peers[id]
}

// Register register peer to peers
func (ps *peerSet) Register(p *peerConnection) {
	ps.mux.Lock()
	defer ps.mux.Unlock()
	ps.peers[p.id] = p
}

// Unregister unregister peer
func (ps *peerSet) Unregister(id string) {
	ps.mux.Lock()
	defer ps.mux.Unlock()
	if _, ok := ps.peers[id]; ok {
		delete(ps.peers, id)
	}
}

// PeersWithoutTx fetch peers which doesn't have special tx
func (ps *peerSet) PeersWithoutTx(hash common.Hash) []*peer {
	ps.mux.Lock()
	defer ps.mux.Unlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.peer.knownTxs.Has(hash) {
			list = append(list, p.peer)
		}
	}
	return list
}

func (ps *peerSet) Close() {
	for _, p := range ps.peers {
		p.peer.Close()
	}
}

// Downloader 区块同步工人
type Downloader struct {
	peers         *peerSet   // 节点集
	lock          sync.Mutex // 读写锁
	synchronising int32      // 标识是否正在同步
	blockChain    blockchain.BlockChain
	dropPeer      peerDropFn // 断开连接

	newBlocksCh chan *blockPack   // 通过网络收到区块包
	blockDoneCh chan *types.Block // 区块处理完毕
	insertErrCh chan error        // 区块入链出错
	quitCh      chan struct{}     // 退出

	queue *prque.Prque // 存储下载的区块队列
}

// New crete Downloader object
func NewDownloader(peers *peerSet, chain blockchain.BlockChain, dropPeer peerDropFn) *Downloader {
	d := &Downloader{
		peers:       peers,
		blockChain:  chain,
		dropPeer:    dropPeer,
		newBlocksCh: make(chan *blockPack),
		blockDoneCh: make(chan *types.Block),
		insertErrCh: make(chan error),
		quitCh:      make(chan struct{}),
		queue:       prque.New(),
	}
	return d
}

// IsSynchronising 是否已经在同步了
func (d *Downloader) IsSynchronising() bool {
	return atomic.LoadInt32(&d.synchronising) > 0
}

// Synchronise 同步启动函数，供外部调用
func (d *Downloader) Synchronise(id string) error {
	if !atomic.CompareAndSwapInt32(&d.synchronising, 0, 1) {
		return errBusy
	}
	defer atomic.StoreInt32(&d.synchronising, 1)
	p := d.peers.Peer(id)
	return d.syncWithPeer(p)
}

// syncWithPeer 从某peer同步，同步时阻塞，直至同步完成
func (d *Downloader) syncWithPeer(p *peerConnection) error {
	stableBlock := d.blockChain.StableBlock()
	remoteHeight := p.peer.height
	if stableBlock.Height() >= remoteHeight {
		return nil
	}
	// 发送获取区块请求
	go p.peer.RequestBlockFromAndTo(stableBlock.Height()+1, remoteHeight)
	// 请求超时定时器
	timeout := time.NewTimer(requestTimeout)
	defer timeout.Stop()
	for {
		// 处理已收到的区块
		for !d.queue.Empty() {
			block := d.queue.PopItem().(*types.Block)
			localHeight := d.blockChain.CurrentBlock().Height()
			if block.Height() > localHeight+1 {
				d.queue.Push(block, -float32(block.Height()))
				break
			}
			if err := d.blockChain.Verify(block); err != nil {
				if d.dropPeer != nil {
					d.dropPeer(p.id)
				}
				return err
			}
			d.insert(block, p.id)
		}
		select {
		case blockPack := <-d.newBlocksCh: // 接收到网络新块
			if strings.Compare(blockPack.peerID, p.id) == 0 {
				d.enqueue(blockPack.blocks)
				timeout.Reset(requestTimeout)
			}
		case block := <-d.blockDoneCh: // 插入本地链成功
			if block.Height() >= remoteHeight {
				return nil
			}
		case <-timeout.C: // 接收超时
			log.Infof("Sync with peer timeout, drop peer.")
			if d.dropPeer != nil {
				d.dropPeer(p.id)
			}
			return errBadPeer
		case err := <-d.insertErrCh: // 插入链出错
			log.Infof("Insert chain err. drop peer")
			if d.dropPeer != nil {
				d.dropPeer(p.id)
			}
			return err
		case <-d.quitCh:
			return errForceQuit
		}
	}
	return nil
}

// enqueue 经区块集合压入带处理队列中
func (d *Downloader) enqueue(blocks types.Blocks) {
	for _, block := range blocks {
		d.queue.Push(block, -float32(block.Height()))
	}
}

// insert 将块插入本地连
func (d *Downloader) insert(block *types.Block, peer string) {
	hash := block.Hash()
	go func() {
		if parent := d.blockChain.GetBlock(block.ParentHash(), block.Height()-1); parent == nil {
			d.insertErrCh <- errUnknownParent
			log.Debugf("Unknown parent block. peer id: %s. height: %d. hash: %s. parent: %s", peer, block.Height(), hash.Hex(), block.ParentHash())
			return
		}
		if err := d.blockChain.InsertChain(block); err == nil {
			d.blockDoneCh <- block
		} else {
			d.insertErrCh <- err
			log.Debugf("block import failed. peer: %s. height: %d. hash: %s. err: %v", peer, block.Height(), hash.Hex(), err)
		}
	}()
}

// DeliverBlocks 将收到的区块分发给loop
func (d *Downloader) DeliverBlocks(id string, blocks types.Blocks) error {
	if blocks == nil {
		return errors.New("deliver-blocks receive nil blocks")
	}
	blockPack := &blockPack{
		peerID: id,
		blocks: blocks,
	}
	d.newBlocksCh <- blockPack
	return nil
}

// Terminate 强行终止同步
func (d *Downloader) Terminate() {
	select {
	case <-d.quitCh:
	default:
		close(d.quitCh)
	}
}
