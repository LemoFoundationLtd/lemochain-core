package network

import (
	"crypto/rand"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"math/big"
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

	if peer, ok := ps.peers[*p.NodeID()]; ok {
		peer.conn.Close()
		delete(ps.peers, *p.NodeID())
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
