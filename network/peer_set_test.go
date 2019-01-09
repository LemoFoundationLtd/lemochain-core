package network

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func Test_RegUnReg(t *testing.T) {
	discover := new(p2p.DiscoverManager)
	ps := NewPeerSet(discover)
	p := newPeer(&testPeer{})
	ps.Register(p)

	assert.Equal(t, 1, ps.Size())

	ps.UnRegister(p)
	assert.Equal(t, 0, ps.Size())
}

func Test_BestToSync(t *testing.T) {
	discover := new(p2p.DiscoverManager)
	ps := NewPeerSet(discover)

	pSync := ps.BestToSync(90)
	assert.Nil(t, pSync)

	p := newPeer(&testPeer{})
	p.lstStatus.CurHash = common.Hash{0x03, 0x06, 0x12, 0x09}
	p.lstStatus.CurHeight = 100
	p.lstStatus.StaHeight = 80
	p.lstStatus.StaHash = common.Hash{0x02, 0x01, 0x13, 0x04}
	ps.Register(p)

	pSync = ps.BestToSync(90)
	assert.NotNil(t, pSync)
	pSync = ps.BestToSync(110)
	assert.Nil(t, pSync)
}

func Test_BestToDiscover(t *testing.T) {
	discover := new(p2p.DiscoverManager)
	ps := NewPeerSet(discover)

	isDis := ps.BestToDiscover()
	assert.Nil(t, isDis)

	p := newPeer(&testPeer{})
	p.lstStatus.CurHash = common.Hash{0x13, 0x16, 0x02, 0x19}
	p.lstStatus.CurHeight = 50
	p.lstStatus.StaHeight = 50
	p.lstStatus.StaHash = common.Hash{0x13, 0x16, 0x02, 0x19}
	ps.Register(p)

	isDis = ps.BestToDiscover()
	assert.NotNil(t, isDis)
}

func Test_BestToFetchConfirms(t *testing.T) {
	discover := new(p2p.DiscoverManager)
	ps := NewPeerSet(discover)

	isDis := ps.BestToFetchConfirms(100)
	assert.Nil(t, isDis)

	p := newPeer(&testPeer{})
	p.lstStatus.CurHash = common.Hash{0x13, 0x16, 0x02, 0x19}
	p.lstStatus.CurHeight = 90
	p.lstStatus.StaHeight = 50
	p.lstStatus.StaHash = common.Hash{0x43, 0x16, 0x02, 0x19}
	ps.Register(p)

	assert.NotNil(t, ps.BestToFetchConfirms(49))
	assert.NotNil(t, ps.BestToFetchConfirms(80))
	assert.Nil(t, ps.BestToFetchConfirms(100))
}

func decodeMinerAddress(input string) common.Address {
	if address, err := common.StringToAddress(input); err == nil {
		return address
	}
	panic(fmt.Sprintf("deputy nodes have invalid miner address: %s", input))
}

var deputyNodes = deputynode.DeputyNodes{
	&deputynode.DeputyNode{
		MinerAddress: decodeMinerAddress("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG"),
		NodeID:       common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"),
		IP:           net.ParseIP("149.28.68.93"),
		Port:         7003,
		Rank:         0,
		Votes:        50000,
	},
	&deputynode.DeputyNode{
		MinerAddress: decodeMinerAddress("Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY"),
		NodeID:       common.FromHex("0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0"),
		IP:           net.ParseIP("149.28.68.93"),
		Port:         7005,
		Rank:         1,
		Votes:        40000,
	},
	&deputynode.DeputyNode{
		MinerAddress: decodeMinerAddress("Lemo842BJZ4DKCC764C63Y6A943775JH6NQ3Z33Y"),
		NodeID:       common.FromHex("0x7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2d0347692f36ef736f342fb5ce4710f7e337f062cc2110d134b63a9575f78cb167bfae2f43"),
		IP:           net.ParseIP("149.28.25.8"),
		Port:         7003,
		Rank:         2,
		Votes:        30000,
	},
	&deputynode.DeputyNode{
		MinerAddress: decodeMinerAddress("Lemo837QGPS3YNTYNF53CD88WA5DR3ABNA95W2DG"),
		NodeID:       common.FromHex("0x34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"),
		IP:           net.ParseIP("45.77.121.107"),
		Port:         7003,
		Rank:         3,
		Votes:        20000,
	},
	&deputynode.DeputyNode{
		MinerAddress: decodeMinerAddress("Lemo83HKZK68JQZDRGS5PWT2ZBSKR5CRADCSJB9B"),
		NodeID:       common.FromHex("0x5b980ffb1b463fce4773a22ebf376c07c6207023b016b36ccfaba7be1cd1ab4a91737741cd43b7fcb10879e0fcf314d69fa953daec0f02be0f8f9cedb0cb3797"),
		IP:           net.ParseIP("63.211.111.245"),
		Port:         7003,
		Rank:         4,
		Votes:        10000,
	},
}

func init() {
	deputynode.Instance().Add(1, deputyNodes)
}

func createPeerSet() *peerSet {
	discover := new(p2p.DiscoverManager)
	ps := NewPeerSet(discover)
	p1 := newPeer(&testPeer{state: 1})
	p1.lstStatus.CurHeight = 15
	p1.lstStatus.StaHeight = 10
	p2 := newPeer(&testPeer{state: 2})
	p2.lstStatus.CurHeight = 15
	p2.lstStatus.StaHeight = 10
	p3 := newPeer(&testPeer{state: 3})
	p3.lstStatus.CurHeight = 25
	p3.lstStatus.StaHeight = 20
	p4 := newPeer(&testPeer{state: 4})
	p4.lstStatus.CurHeight = 25
	p4.lstStatus.StaHeight = 15
	p5 := newPeer(&testPeer{state: 5})
	p5.lstStatus.CurHeight = 25
	p5.lstStatus.StaHeight = 25
	ps.Register(p1)
	ps.Register(p2)
	ps.Register(p3)
	ps.Register(p4)
	ps.Register(p5)
	return ps
}

func Test_NodesType(t *testing.T) {
	ps := createPeerSet()
	peers := ps.DeputyNodes(4)
	assert.Len(t, peers, 3)

	peers = ps.DelayNodes(4)
	assert.Len(t, peers, 2)
}

func Test_LatestStableHeight(t *testing.T) {
	ps := createPeerSet()
	assert.Equal(t, uint32(25), ps.LatestStableHeight())
}
