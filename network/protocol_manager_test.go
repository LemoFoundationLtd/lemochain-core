package network

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"github.com/stretchr/testify/assert"
	"testing"
)

type testChain struct {
	hasBlock bool
}

func (bc *testChain) Genesis() *types.Block {
	h := &types.Header{}
	return &types.Block{
		Header: h,
	}
}
func (bc *testChain) HasBlock(hash common.Hash) bool { return bc.hasBlock }
func (bc *testChain) GetBlockByHeight(height uint32) *types.Block {
	h := &types.Header{
		Height: 9,
	}
	return &types.Block{
		Header: h,
	}
}
func (bc *testChain) GetBlockByHash(hash common.Hash) *types.Block { return nil }
func (bc *testChain) CurrentBlock() *types.Block {
	h := &types.Header{
		Height: 9,
	}
	return &types.Block{
		Header: h,
	}
}
func (bc *testChain) StableBlock() *types.Block {
	h := &types.Header{
		Height: 8,
	}
	return &types.Block{
		Header: h,
	}
}
func (bc *testChain) InsertChain(block *types.Block, isSyncing bool) error { return nil }
func (bc *testChain) SetStableBlock(hash common.Hash, height uint32) error { return nil }

func (bc *testChain) ReceiveConfirm(info *BlockConfirmData) (err error) { return nil }

func (bc *testChain) GetConfirms(query *GetConfirmInfo) []types.SignData { return nil }

func (bc *testChain) ReceiveConfirms(pack BlockConfirms) {}

type testTxPool struct {
}

// AddTxs add transaction
func (p *testTxPool) AddTxs(txs []*types.Transaction) error { return nil }

// Remove remove transaction
func (p *testTxPool) Remove(keys []common.Hash) {}

func createPm() *ProtocolManager {
	bc := new(testChain)
	txPool := new(testTxPool)
	discover := new(p2p.DiscoverManager)
	pm := NewProtocolManager(1, p2p.NodeID{}, bc, txPool, discover, 1)
	pm.setTest()
	return pm
}

func Test_StartStop(t *testing.T) {
	pm := createPm()
	pm.Start()
	pm.Stop()
}

func Test_txConfirmLoop(t *testing.T) {
	pm := createPm()

	go pm.txConfirmLoop()

	// test tx
	tx := &types.Transaction{}
	pm.txCh <- tx
	res := <-pm.testOutput
	assert.Equal(t, testBroadcastTxs, res)

	// test confirm
	confirm := new(BlockConfirmData)
	pm.confirmCh <- confirm
	res = <-pm.testOutput
	assert.Equal(t, testBroadcastConfirm, res)
	close(pm.quitCh)
}

func Test_rcvBlockLoop(t *testing.T) {
	pm := createPm()
	go pm.rcvBlockLoop()

	h1 := &types.Header{
		Height: 10,
	}
	b1 := &types.Block{
		Header: h1,
	}
	pm.newMinedBlockCh <- b1

	res := <-pm.testOutput
	assert.Equal(t, testBroadcastBlock, res)

	testP := &testPeer{}
	p := newPeer(testP)
	p.lstStatus.CurHeight = 7
	p.lstStatus.CurHash = common.Hash{0x01, 0x02, 0x03}
	p.lstStatus.StaHeight = 6
	p.lstStatus.StaHash = common.Hash{0x02, 0x02, 0x02}
	pm.peers.Register(p)

	bs := make(types.Blocks, 0)
	bs = append(bs, b1)
	bo := &rcvBlockObj{
		p:      p,
		blocks: bs,
	}
	pm.rcvBlocksCh <- bo
	res = <-pm.testOutput
	assert.Equal(t, testRcvBlocks, res)

	h2 := &types.Header{
		Height: 10,
	}
	b2 := &types.Block{
		Header: h2,
	}
	bs = make(types.Blocks, 0)
	bs = append(bs, b2)
	pm.chain.(*testChain).hasBlock = true
	bo = &rcvBlockObj{
		p:      p,
		blocks: bs,
	}
	pm.rcvBlocksCh <- bo
	res = <-pm.testOutput
	assert.Equal(t, testRcvBlocks, res)

	res = <-pm.testOutput
	assert.Equal(t, testQueueTimer, res)
	close(pm.quitCh)
}

func Test_stableBlockLoop(t *testing.T) {
	pm := createPm()
	go pm.stableBlockLoop()

	testP := &testPeer{}
	p := newPeer(testP)
	p.lstStatus.CurHeight = 9
	p.lstStatus.CurHash = common.Hash{0x01, 0x02, 0x03}
	p.lstStatus.StaHeight = 8
	p.lstStatus.StaHash = common.Hash{0x02, 0x02, 0x02}
	pm.peers.Register(p)

	h1 := &types.Header{
		Height: 10,
	}
	b1 := &types.Block{
		Header: h1,
	}

	h2 := &types.Header{
		Height: 6,
	}
	b2 := &types.Block{
		Header: h2,
	}

	pm.oldStableBlock.Store(b2)

	pm.stableBlockCh <- b1
	res := <-pm.testOutput
	assert.Equal(t, testStableBlock, res)

	close(pm.quitCh)
}

func Test_setConfirmsFromCache(t *testing.T) {
	pm := createPm()
	confirm1 := &BlockConfirmData{
		Height: 1,
		Hash:   common.Hash{0x01},
	}
	confirm12 := &BlockConfirmData{
		Height: 1,
		Hash:   common.Hash{0x01, 0x02},
	}
	confirm2 := &BlockConfirmData{
		Height: 2,
		Hash:   common.Hash{0x02},
	}
	confirm3 := &BlockConfirmData{
		Height: 3,
		Hash:   common.Hash{0x03},
	}
	pm.confirmsCache.Push(confirm1)
	pm.confirmsCache.Push(confirm12)
	pm.confirmsCache.Push(confirm2)
	pm.confirmsCache.Push(confirm3)
	go pm.setConfirmsFromCache(1, common.Hash{0x01})
	res := <-pm.testOutput
	assert.Equal(t, testSetConfirmFromCache, res)
	assert.Equal(t, 3, pm.confirmsCache.Size())
	close(pm.quitCh)
}

func Test_peerLoop(t *testing.T) {
	pm := createPm()
	go pm.peerLoop()
	p := &testPeer{}
	pm.addPeerCh <- p
	res := <-pm.testOutput
	assert.Equal(t, testAddPeer, res)

	pm.removePeerCh <- p
	res = <-pm.testOutput
	assert.Equal(t, testRemovePeer, res)

	rp := newPeer(p)
	rp.lstStatus.CurHeight = 100
	pm.peers.Register(rp)
	res = <-pm.testOutput
	if res != testForceSync && res != testDiscover {
		t.Error("result not match")
	}
	res = <-pm.testOutput
	if res != testForceSync && res != testDiscover {
		t.Error("result not match")
	}
	close(pm.quitCh)
}

func Test_handleMsg(t *testing.T) {

}
