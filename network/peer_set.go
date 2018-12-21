package network

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"sync"
)

type peerSet struct {
	peers    map[p2p.NodeID]*peer
	discover *p2p.DiscoverManager
	lock     sync.RWMutex
}

// NewPeerSet
func NewPeerSet(discover *p2p.DiscoverManager) *peerSet {
	return &peerSet{
		peers:    make(map[p2p.NodeID]*peer),
		discover: discover,
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

	if _, ok := ps.peers[*p.NodeID()]; ok {
		delete(ps.peers, *p.NodeID())
	}
	if p.conn.NeedReConnect() {
		ps.discover.SetReconnect(p.NodeID())
	}
}

// BestToSync best peer to synchronise
func (ps *peerSet) BestToSync(height uint32) (p *peer) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if len(ps.peers) == 0 {
		return nil
	}

	maxBadSync := uint32(100)
	for _, peer := range ps.peers {
		if peer.lstStatus.CurHeight > height && peer.badSyncCounter < maxBadSync {
			height = peer.lstStatus.CurHeight
			p = peer
		}
	}
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

// DeputyNodes filter deputy node
func (ps *peerSet) DeputyNodes(height uint32) []*peer {
	peers := make([]*peer, 0)
	for _, p := range ps.peers {
		if deputynode.Instance().IsNodeDeputy(height, p.NodeID()[:]) {
			peers = append(peers, p)
		}
	}
	return peers
}

// DelayNodes filter delay node
func (ps *peerSet) DelayNodes(height uint32) []*peer {
	peers := make([]*peer, 0)
	for _, p := range ps.peers {
		if !deputynode.Instance().IsNodeDeputy(height, p.NodeID()[:]) {
			peers = append(peers, p)
		}
	}
	return peers
}
