package network

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/network/p2p"
	"github.com/stretchr/testify/assert"
	"math/big"
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
	dm := deputynode.NewManager(5)
	dm.SaveSnapshot(0, deputynode.DeputyNodes{
		&deputynode.DeputyNode{
			MinerAddress: decodeMinerAddress("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG"),
			NodeID:       common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"),
			Rank:         0,
			Votes:        new(big.Int).SetInt64(5),
		},
	})
	pm := NewProtocolManager(1, p2p.NodeID{}, bc, dm, txPool, discover, 1, params.VersionUint())
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

func Test_mergeConfirmsFromCache(t *testing.T) {
	pm := createPm()
	testBlock := &types.Block{Header: &types.Header{Height: 1}}
	confirm1 := &BlockConfirmData{
		Height:   testBlock.Height(),
		Hash:     testBlock.Hash(),
		SignInfo: types.SignData{0x12},
	}
	confirm12 := &BlockConfirmData{
		Height:   testBlock.Height(),
		Hash:     testBlock.Hash(),
		SignInfo: types.SignData{0x23},
	}
	// other fork block
	confirm13 := &BlockConfirmData{
		Height:   testBlock.Height(),
		Hash:     common.Hash{0x01, 0x02},
		SignInfo: types.SignData{0x34},
	}
	// wrong height
	confirm14 := &BlockConfirmData{
		Height:   100,
		Hash:     testBlock.Hash(),
		SignInfo: types.SignData{0x45},
	}
	confirm2 := &BlockConfirmData{
		Height: 2,
		Hash:   common.Hash{0x02},
	}
	confirm3 := &BlockConfirmData{
		Height: 3,
		Hash:   common.Hash{0x03},
	}
	// confirm1 is exist
	testBlock.Confirms = []types.SignData{confirm1.SignInfo}

	pm.confirmsCache.Push(confirm1)
	pm.confirmsCache.Push(confirm12)
	pm.confirmsCache.Push(confirm13)
	pm.confirmsCache.Push(confirm14)
	pm.confirmsCache.Push(confirm2)
	pm.confirmsCache.Push(confirm3)
	pm.mergeConfirmsFromCache(testBlock)
	assert.Equal(t, 4, pm.confirmsCache.Size())
	assert.Equal(t, 2, len(testBlock.Confirms))
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
