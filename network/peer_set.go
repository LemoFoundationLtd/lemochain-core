package network

import (
	"bytes"
	"crypto/rand"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/network/p2p"
	"math/big"
	"sync"
)

type peerSet struct {
	peers    map[p2p.NodeID]*peer
	discover *p2p.DiscoverManager
	dm       *deputynode.Manager
	lock     sync.RWMutex
}

// NewPeerSet
func NewPeerSet(discover *p2p.DiscoverManager, dm *deputynode.Manager) *peerSet {
	return &peerSet{
		peers:    make(map[p2p.NodeID]*peer),
		discover: discover,
		dm:       dm,
	}
}

// Size set's length
func (ps *peerSet) Size() int {
	return len(ps.peers)
}

// Register register peer to set
func (ps *peerSet) Register(p *peer) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	ps.peers[*p.NodeID()] = p
}

// UnRegister remove peer from set
func (ps *peerSet) UnRegister(p *peer) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if peer, ok := ps.peers[*p.NodeID()]; ok {
		peer.conn.Close()
		delete(ps.peers, *p.NodeID())
	} else {
		return
	}
	if p.conn.NeedReConnect() {
		if err := ps.discover.SetReconnect(p.NodeID()); err != nil {
			log.Infof("SetReconnect failed: %v", err)
		}
	}
}

// BestToSync best peer to synchronise
func (ps *peerSet) BestToSync(height uint32) (p *peer) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if len(ps.peers) == 0 {
		return nil
	}
	peers := make([]*peer, 0)
	for _, peer := range ps.peers {
		if peer.lstStatus.CurHeight > height {
			peers = append(peers, peer)
		}
	}
	if len(peers) == 0 {
		return nil
	}
	v, err := rand.Int(rand.Reader, new(big.Int).SetInt64(int64(len(peers))))
	if err != nil {
		log.Error("Rand a int value failed, use default value: 0")
		v = new(big.Int)
	}
	index := int(v.Int64())
	p = peers[index]
	return p
}

// BestToDiscover best peer to discovery
func (ps *peerSet) BestToDiscover() *peer {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if len(ps.peers) == 0 {
		return nil
	}

	var res *peer
	loopCount := uint32(10)
	discoverCounter := uint32(0)
	for res == nil {
		for _, p := range ps.peers {
			if p.discoverCounter%loopCount == discoverCounter {
				res = p
				break
			}
		}
		if discoverCounter == loopCount {
			discoverCounter = 0
		} else {
			discoverCounter++
		}
	}
	return res
}

// BestToFetchConfirms best peer to fetch confirms package
func (ps *peerSet) BestToFetchConfirms(height uint32) (p *peer) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if len(ps.peers) == 0 {
		return nil
	}
	for _, peer := range ps.peers {
		if peer.lstStatus.StaHeight >= height {
			p = peer
			height = peer.lstStatus.StaHeight
			break
		}
	}
	if p == nil {
		for _, peer := range ps.peers {
			if peer.lstStatus.CurHeight >= height {
				p = peer
				height = peer.lstStatus.CurHeight
				break
			}
		}
	}
	return p
}

// DeputyNodes filter deputy node
func (ps *peerSet) DeputyNodes(height uint32) []*peer {
	peers := make([]*peer, 0)
	for _, p := range ps.peers {
		if ps.dm.IsNodeDeputy(height, p.NodeID()[:]) {
			peers = append(peers, p)
		}
	}
	return peers
}

// NeedBroadcastTxsNodes 获取需要广播交易的peer列表。首选广播给正在出高度为currentHeight + 1的节点和即将出高度为currentHeight + 2的节点
func (ps *peerSet) NeedBroadcastTxsNodes(currentHeight uint32, currentMiner common.Address) []*peer {
	// 1. 获取本节点相连的所有出块节点
	deputyNodePeers := ps.DeputyNodes(currentHeight + 1)
	// 如果获取到的出块节点为空则返回未出块的peer
	if len(deputyNodePeers) == 0 {
		return ps.DelayNodes(currentHeight + 1)
	}
	// 2. 获取下一个区块的出块者的deputy信息
	nextMineDeputy := ps.dm.GetNextBlockMineDeputy(currentHeight, currentMiner)
	// 获取下一个出块deputy失败则返回所有的连接的peers
	if nextMineDeputy == nil {
		return deputyNodePeers
	}

	// 3. 获取下下个区块的出块者的deputy信息
	thirdMineDeputy := ps.dm.GetNextBlockMineDeputy(currentHeight+1, nextMineDeputy.MinerAddress)
	// 3.1 获取下下个出块deputy失败则返回所有的连接的peers
	if thirdMineDeputy == nil {
		return deputyNodePeers
	}
	// 3.2 如果下下个即将出块的deputy为自己，则不用再广播出去了,防止nextMineDeputy 和thirdMineDeputy相互转,即使是通过api传过来的交易，交易执行等待时间最多为30s。
	if bytes.Compare(deputynode.GetSelfNodeID(), thirdMineDeputy.NodeID) == 0 {
		return nil
	}
	// 4. 通过nodeId判断nextMineDeputy和 thirdMineDeputy是否在deputyNodePeers中
	peers := make([]*peer, 0, 2)
	for _, p := range deputyNodePeers {
		if (bytes.Compare(p.NodeID()[:], nextMineDeputy.NodeID) == 0) || (bytes.Compare(p.NodeID()[:], thirdMineDeputy.NodeID) == 0) {
			peers = append(peers, p)
		}
	}
	// 5. 如果本节点没有与nextMineDeputy和thirdMineDeputy节点相连则返回本节点连接的所有的peers
	if len(peers) == 0 {
		return deputyNodePeers
	} else {
		return peers
	}
}

// DelayNodes filter delay node
func (ps *peerSet) DelayNodes(height uint32) []*peer {
	peers := make([]*peer, 0)
	for _, p := range ps.peers {
		if ps.dm.IsNodeDeputy(height, p.NodeID()[:]) == false {
			peers = append(peers, p)
		}
	}
	return peers
}

// LatestStableHeight get peer's latest stable block's height
func (ps *peerSet) LatestStableHeight() uint32 {
	height := uint32(0)
	for _, p := range ps.peers {
		if p.lstStatus.StaHeight > height {
			height = p.lstStatus.StaHeight
		}
	}
	return height
}
