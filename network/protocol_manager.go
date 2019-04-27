package network

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/subscribe"
	"github.com/LemoFoundationLtd/lemochain-core/network/p2p"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ForceSyncInternal = 10 * time.Second
	DiscoverInternal  = 10 * time.Second
	DefaultLimit      = 50 // default connection limit
)

// just for test
const (
	testBroadcastTxs int = 1 + iota
	testBroadcastBlock
	testBroadcastConfirm
	testRcvBlocks
	testQueueTimer
	testStableBlock
	testAddPeer
	testRemovePeer
	testForceSync
	testDiscover
)

// var testRcvFlag = false   // for test

type rcvBlockObj struct {
	p      *peer
	blocks types.Blocks
}

type ProtocolManager struct {
	chainID     uint16
	nodeID      p2p.NodeID
	nodeVersion uint32

	chain         BlockChain
	dm            *deputynode.Manager
	discover      *p2p.DiscoverManager
	txPool        TxPool
	limit         int
	peers         *peerSet      // connected peers
	confirmsCache *ConfirmCache // received confirm info before block, cache them
	blockCache    *BlockCache

	oldStableBlock atomic.Value

	addPeerCh    chan p2p.IPeer
	removePeerCh chan p2p.IPeer

	txCh            chan *types.Transaction
	newMinedBlockCh chan *types.Block
	stableBlockCh   chan *types.Block
	rcvBlocksCh     chan *rcvBlockObj
	confirmCh       chan *BlockConfirmData

	wg     sync.WaitGroup
	quitCh chan struct{}

	test       bool
	testOutput chan int
}

func NewProtocolManager(chainID uint16, nodeID p2p.NodeID, chain BlockChain, dm *deputynode.Manager, txPool TxPool, discover *p2p.DiscoverManager, limit int, nodeVersion uint32) *ProtocolManager {
	if limit == 0 {
		limit = DefaultLimit
	}
	pm := &ProtocolManager{
		chainID:       chainID,
		nodeID:        nodeID,
		nodeVersion:   nodeVersion,
		chain:         chain,
		dm:            dm,
		txPool:        txPool,
		discover:      discover,
		limit:         limit,
		peers:         NewPeerSet(discover, dm),
		confirmsCache: NewConfirmCache(),
		blockCache:    NewBlockCache(),

		addPeerCh:    make(chan p2p.IPeer),
		removePeerCh: make(chan p2p.IPeer),

		txCh:            make(chan *types.Transaction, 10),
		newMinedBlockCh: make(chan *types.Block),
		stableBlockCh:   make(chan *types.Block, 10),
		rcvBlocksCh:     make(chan *rcvBlockObj, 10),
		confirmCh:       make(chan *BlockConfirmData, 10),

		quitCh: make(chan struct{}),
	}
	pm.sub()
	return pm
}

func (pm *ProtocolManager) setTest() {
	pm.test = true
	pm.testOutput = make(chan int)
}

// sub subscribe channel
func (pm *ProtocolManager) sub() {
	subscribe.Sub(subscribe.AddNewPeer, pm.addPeerCh)
	subscribe.Sub(subscribe.DeletePeer, pm.removePeerCh)
	subscribe.Sub(subscribe.NewMinedBlock, pm.newMinedBlockCh)
	subscribe.Sub(subscribe.NewStableBlock, pm.stableBlockCh)
	subscribe.Sub(subscribe.NewTx, pm.txCh)
	subscribe.Sub(subscribe.NewConfirm, pm.confirmCh)
}

// unSub unsubscribe channel
func (pm *ProtocolManager) unSub() {
	subscribe.UnSub(subscribe.AddNewPeer, pm.addPeerCh)
	subscribe.UnSub(subscribe.DeletePeer, pm.removePeerCh)
	subscribe.UnSub(subscribe.NewMinedBlock, pm.newMinedBlockCh)
	subscribe.UnSub(subscribe.NewStableBlock, pm.stableBlockCh)
	subscribe.UnSub(subscribe.NewTx, pm.txCh)
	subscribe.UnSub(subscribe.NewConfirm, pm.confirmCh)
}

// Start
func (pm *ProtocolManager) Start() {
	go pm.txConfirmLoop()
	go pm.rcvBlockLoop()
	go pm.stableBlockLoop()
	go pm.peerLoop()
}

// Stop
func (pm *ProtocolManager) Stop() {
	pm.unSub()
	close(pm.quitCh)
	pm.wg.Wait()
	log.Info("ProtocolManager has stopped")
}

// txConfirmLoop receive transactions and confirm and then broadcast them
func (pm *ProtocolManager) txConfirmLoop() {
	pm.wg.Add(1)
	defer func() {
		pm.wg.Done()
		log.Debugf("txConfirmLoop finished")
	}()

	for {
		select {
		case <-pm.quitCh:
			log.Info("txConfirmLoop finished")
			return
		case tx := <-pm.txCh:
			nextHeight := pm.chain.CurrentBlock().Height() + 1
			peers := pm.peers.DeputyNodes(nextHeight)
			if !pm.dm.IsSelfDeputyNode(nextHeight) && len(peers) == 0 {
				peers = pm.peers.DelayNodes(nextHeight)
			}
			go pm.broadcastTxs(peers, types.Transactions{tx})
		case info := <-pm.confirmCh:
			if pm.peers.LatestStableHeight() > info.Height {
				continue
			}
			curHeight := pm.chain.CurrentBlock().Height()
			peers := pm.peers.DeputyNodes(curHeight)
			go pm.broadcastConfirm(peers, info)
			log.Debugf("broadcast confirm, len(peers)=%d, height: %d", len(peers), info.Height)
		}
	}
}

// blockLoop receive special type block event
func (pm *ProtocolManager) rcvBlockLoop() {
	pm.wg.Add(1)
	defer func() {
		pm.wg.Done()
		log.Debugf("RcvBlockLoop finished")
	}()

	proInterval := 500 * time.Millisecond
	queueTimer := time.NewTimer(proInterval)

	// just for test
	// testRcvTimer := time.NewTimer(8 * time.Second)

	for {
		select {
		case <-pm.quitCh:
			log.Info("BlockLoop finished")
			return
		case block := <-pm.newMinedBlockCh:
			log.Debugf("Current peers count: %d", len(pm.peers.peers))
			peers := pm.peers.DeputyNodes(block.Height())
			go pm.broadcastBlock(peers, block, true)
		case rcvMsg := <-pm.rcvBlocksCh:
			// for test
			// testRcvFlag = false
			// testRcvTimer.Reset(8 * time.Second)

			// peer's latest height
			pLstHeight := rcvMsg.p.LatestStatus().CurHeight

			for _, b := range rcvMsg.blocks {
				// update latest status
				if b.Height() > pLstHeight && rcvMsg.p != nil {
					rcvMsg.p.UpdateStatus(b.Height(), b.Hash())
				}
				// block is stale
				if b.Height() <= pm.chain.StableBlock().Height() || pm.chain.HasBlock(b.Hash()) {
					continue
				}
				// local chain has this block
				if pm.chain.HasBlock(b.ParentHash()) {
					log.Infof("Got a block %s from peer: %#x", b.ShortString(), rcvMsg.p.NodeID()[:8])
					go pm.insertBlock(b)
				} else {
					pm.blockCache.Add(b)
					if rcvMsg.p != nil {
						// request parent block
						go rcvMsg.p.RequestBlocks(b.Height()-1, b.Height()-1)
					}
				}
			}
			// for test
			if pm.test {
				pm.testOutput <- testRcvBlocks
			}
		case <-queueTimer.C:
			processBlock := func(block *types.Block) bool {
				if pm.chain.HasBlock(block.ParentHash()) {
					go pm.insertBlock(block)
					return true
				}
				return false
			}
			pm.blockCache.Iterate(processBlock)
			queueTimer.Reset(proInterval)
			// output cache size
			cacheSize := pm.blockCache.Size()
			if cacheSize > 0 {
				p := pm.peers.BestToSync(pm.blockCache.FirstHeight())
				if p != nil {
					go p.RequestBlocks(pm.blockCache.FirstHeight()-1, pm.blockCache.FirstHeight()-1)
					log.Debugf("BlockCache's size: %d", cacheSize)
				}
			}
			// for test
			if pm.test {
				pm.testOutput <- testQueueTimer
			}
			// case <-testRcvTimer.C: // just for test
			// 	testRcvFlag = true
		}
	}
}

// insertBlock insert block
func (pm *ProtocolManager) insertBlock(b *types.Block) {
	// pop the confirms which arrived before block
	pm.mergeConfirmsFromCache(b)
	pm.chain.InsertBlock(b)
}

// stableBlockLoop block has been stable
func (pm *ProtocolManager) stableBlockLoop() {
	pm.wg.Add(1)
	defer func() {
		pm.wg.Done()
		log.Debugf("StableBlockLoop finished")
	}()

	for {
		select {
		case <-pm.quitCh:
			return
		case block := <-pm.stableBlockCh:
			var oldStableBlock *types.Block
			if pm.oldStableBlock.Load() != nil {
				oldStableBlock = pm.oldStableBlock.Load().(*types.Block)
				if oldStableBlock != nil && oldStableBlock.Height()+1 < block.Height() {
					go pm.fetchConfirmsFromRemote(oldStableBlock.Height()+1, block.Height()-1)
				}
			}
			pm.oldStableBlock.Store(block)
			peers := pm.peers.DelayNodes(block.Height())
			if len(peers) > 0 {
				// for debug
				log.Debug("Broadcast stable block to delay node")
				go pm.broadcastBlock(peers, block, false)
			}
			go func() {
				pm.confirmsCache.Clear(block.Height())
				pm.blockCache.Clear(block.Height())
			}()
			// for test
			if pm.test {
				pm.testOutput <- testStableBlock
			}
		}
	}
}

// fetchConfirmsFromRemote fetch confirms from remote
func (pm *ProtocolManager) fetchConfirmsFromRemote(start, end uint32) {
	p := pm.peers.BestToFetchConfirms(end)
	if p == nil {
		return
	}
	for h := start; h <= end; h++ {
		b := pm.chain.GetBlockByHeight(h)
		if b == nil {
			continue
		}
		p.SendGetConfirms(b.Height(), b.Hash())
	}
}

// mergeConfirmsFromCache merge confirms into block from cache
func (pm *ProtocolManager) mergeConfirmsFromCache(block *types.Block) {
	confirms := pm.confirmsCache.Pop(block.Height(), block.Hash())
	for _, confirm := range confirms {
		// The duplicates will be remove by consensus validator
		block.Confirms = append(block.Confirms, confirm.SignInfo)
	}
	log.Debugf("Pop %d confirms from cache, now we have %d confirms", len(confirms), len(block.Confirms))
}

// peerLoop something about peer event
func (pm *ProtocolManager) peerLoop() {
	pm.wg.Add(1)
	defer func() {
		pm.wg.Done()
		log.Debugf("PeerLoop finished")
	}()

	forceSyncTimer := time.NewTimer(ForceSyncInternal)
	discoverTimer := time.NewTimer(DiscoverInternal)
	for {
		select {
		case <-pm.quitCh:
			log.Info("PeerLoop finished")
			return
		case rPeer := <-pm.addPeerCh: // new peer added
			if pm.checkConnectionLimit(rPeer) {
				p := newPeer(rPeer)
				go pm.handlePeer(p)
				// for test
				if pm.test {
					pm.testOutput <- testAddPeer
				}
			}
		case rPeer := <-pm.removePeerCh:
			p := newPeer(rPeer)
			pm.peers.UnRegister(p)
			log.Infof("Connection has dropped, nodeID: %s", p.NodeID().String()[:16])
			// for test
			if pm.test {
				pm.testOutput <- testRemovePeer
			}
		case <-forceSyncTimer.C: // time to synchronise block
			p := pm.peers.BestToSync(pm.chain.CurrentBlock().Height())
			if p != nil {
				go p.SendReqLatestStatus()
			}
			forceSyncTimer.Reset(ForceSyncInternal)
			// for test
			if pm.test {
				pm.testOutput <- testForceSync
			}
		case <-discoverTimer.C: // time to discover
			if len(pm.peers.peers) < 5 {
				p := pm.peers.BestToDiscover()
				if p != nil {
					go p.SendDiscover()
				}
			}
			discoverTimer.Reset(DiscoverInternal)
			// for test
			if pm.test {
				pm.testOutput <- testDiscover
			}
		}
	}
}

func (pm *ProtocolManager) checkConnectionLimit(p p2p.IPeer) bool {
	height := pm.chain.CurrentBlock().Height() + 1
	rNodeID := p.RNodeID()
	// deputy node
	if n := pm.dm.GetDeputyByNodeID(height, rNodeID[:]); n != nil {
		return true
	}
	// if node in white list
	if pm.discover.InWhiteList(rNodeID) {
		return true
	}
	// limit
	connected := pm.peers.DelayNodes(height)
	if len(connected) <= pm.limit {
		return true
	}
	return false
}

// broadcastTxs broadcast transaction
func (pm *ProtocolManager) broadcastTxs(peers []*peer, txs types.Transactions) {
	for _, p := range peers {
		p.SendTxs(txs)
	}
	if pm.test {
		pm.testOutput <- testBroadcastTxs
	}
}

// broadcastConfirm broadcast confirm info to deputy nodes
func (pm *ProtocolManager) broadcastConfirm(peers []*peer, confirmInfo *BlockConfirmData) {
	for _, p := range peers {
		p.SendConfirmInfo(confirmInfo)
	}
	if pm.test {
		pm.testOutput <- testBroadcastConfirm
	}
}

// broadcastBlock broadcast block
func (pm *ProtocolManager) broadcastBlock(peers []*peer, block *types.Block, withBody bool) {
	for _, p := range peers {
		if withBody {
			p.SendBlocks([]*types.Block{block})
		} else {
			p.SendBlockHash(block.Height(), block.Hash())
		}
	}
	if pm.test {
		pm.testOutput <- testBroadcastBlock
	}
}

// handlePeer handle about peer
func (pm *ProtocolManager) handlePeer(p *peer) {
	// handshake
	rStatus, err := pm.handshake(p)
	if err != nil {
		log.Warnf("protocol handshake failed: %v", err)
		if err = pm.discover.SetConnectResult(p.NodeID(), false); err != nil {
			log.Debugf("handlePeer: %v", err)
		}
		p.FailedHandshakeClose()
		return
	}
	// register peer to set
	pm.peers.Register(p)
	// synchronise block
	if pm.chain.CurrentBlock().Height() < rStatus.LatestStatus.CurHeight {
		from, err := pm.findSyncFrom(&rStatus.LatestStatus)
		if err != nil {
			log.Warnf("Find sync from error: %v", err)
			if err = pm.discover.SetConnectResult(p.NodeID(), false); err != nil {
				log.Debugf("HandlePeer: %v", err)
			}
			p.HardForkClose()
			return
		}
		p.RequestBlocks(from, rStatus.LatestStatus.CurHeight)
	}
	// set connect result
	if err = pm.discover.SetConnectResult(p.NodeID(), true); err != nil {
		log.Debugf("HandlePeer set connect result: %v", err)
	}

	for {
		// handle peer net message
		if err := pm.handleMsg(p); err != nil {
			log.Debugf("Handle message failed: %v", err)
			if err != io.EOF {
				p.conn.Close()
			}
			return
		}
	}
}

// handshake protocol handshake
func (pm *ProtocolManager) handshake(p *peer) (*ProtocolHandshake, error) {
	phs := &ProtocolHandshake{
		ChainID:     pm.chainID,
		GenesisHash: pm.chain.Genesis().Hash(),
		NodeVersion: pm.nodeVersion,
		LatestStatus: LatestStatus{
			CurHash:   pm.chain.CurrentBlock().Hash(),
			CurHeight: pm.chain.CurrentBlock().Height(),
			StaHash:   pm.chain.StableBlock().Hash(),
			StaHeight: pm.chain.StableBlock().Height(),
		},
	}
	content := phs.Bytes()
	if content == nil {
		return nil, errors.New("rlp encode error")
	}
	remoteStatus, err := p.Handshake(content)
	if err != nil {
		return nil, err
	}
	return remoteStatus, nil
}

// forceSyncBlock force to sync block
func (pm *ProtocolManager) forceSyncBlock(status *LatestStatus, p *peer) {
	if status.CurHeight <= pm.chain.CurrentBlock().Height() {
		return
	}

	from, err := pm.findSyncFrom(status)
	if err != nil {
		log.Warnf("Find sync from error: %v", err)
		p.HardForkClose()
		pm.peers.UnRegister(p)
		return
	}
	p.RequestBlocks(from, status.CurHeight)
}

// findSyncFrom find height of which sync from
func (pm *ProtocolManager) findSyncFrom(rStatus *LatestStatus) (uint32, error) {
	var from uint32
	curBlock := pm.chain.CurrentBlock()
	staBlock := pm.chain.StableBlock()

	if staBlock.Height() < rStatus.StaHeight {
		if curBlock.Height() < rStatus.StaHeight {
			from = staBlock.Height() + 1
		} else {
			if pm.chain.HasBlock(rStatus.StaHash) {
				from = rStatus.StaHeight + 1
			} else {
				from = staBlock.Height() + 1
			}
		}
	} else {
		if pm.chain.HasBlock(rStatus.StaHash) {
			from = staBlock.Height() + 1
		} else {
			return 0, errors.New("error: CHAIN FORK")
		}
	}
	return from, nil
}

// handleMsg handle net received message
func (pm *ProtocolManager) handleMsg(p *peer) error {
	msg, err := p.ReadMsg()
	if err != nil {
		return err
	}

	// if testRcvFlag {
	// 	if msg.Code == BlocksMsg {
	// 		log.Debug("handleMsg receive blocks, but not process.")
	// 	} else {
	// 		log.Debug("not receive block, but receive other types of message.")
	// 	}
	// }

	switch msg.Code {
	case LstStatusMsg:
		return pm.handleLstStatusMsg(msg, p)
	case GetLstStatusMsg:
		return pm.handleGetLstStatusMsg(msg, p)
	case BlockHashMsg:
		return pm.handleBlockHashMsg(msg, p)
	case TxsMsg:
		return pm.handleTxsMsg(msg)
	case BlocksMsg:
		return pm.handleBlocksMsg(msg, p)
	case GetBlocksMsg:
		return pm.handleGetBlocksMsg(msg, p)
	case GetConfirmsMsg:
		return pm.handleGetConfirmsMsg(msg, p)
	case ConfirmsMsg:
		return pm.handleConfirmsMsg(msg)
	case ConfirmMsg:
		return pm.handleConfirmMsg(msg)
	case DiscoverReqMsg:
		return pm.handleDiscoverReqMsg(msg, p)
	case DiscoverResMsg:
		return pm.handleDiscoverResMsg(msg)
	case GetBlocksWithChangeLogMsg:
		return pm.handleGetBlocksWithChangeLogMsg(msg, p)
	default:
		log.Debugf("invalid code: %d, from: %s", msg.Code, common.ToHex(p.NodeID()[:8]))
		return ErrInvalidCode
	}
}

// handleLstStatusMsg handle latest remote status message
func (pm *ProtocolManager) handleLstStatusMsg(msg *p2p.Msg, p *peer) error {
	var status LatestStatus
	if err := msg.Decode(&status); err != nil {
		return fmt.Errorf("handleLstStatusMsg error: %v", err)
	}
	go pm.forceSyncBlock(&status, p)
	return nil
}

// handleGetLstStatusMsg handle request of latest status
func (pm *ProtocolManager) handleGetLstStatusMsg(msg *p2p.Msg, p *peer) error {
	var req GetLatestStatus
	if err := msg.Decode(&req); err != nil {
		return fmt.Errorf("handleGetLstStatusMsg error: %v", err)
	}
	status := &LatestStatus{
		CurHeight: pm.chain.CurrentBlock().Height(),
		CurHash:   pm.chain.CurrentBlock().Hash(),
		StaHeight: pm.chain.StableBlock().Height(),
		StaHash:   pm.chain.StableBlock().Hash(),
	}
	go p.SendLstStatus(status)
	return nil
}

// handleBlockHashMsg handle receiving block's hash message
func (pm *ProtocolManager) handleBlockHashMsg(msg *p2p.Msg, p *peer) error {
	var hashMsg BlockHashData
	if err := msg.Decode(&hashMsg); err != nil {
		return fmt.Errorf("handleBlockHashMsg error: %v", err)
	}
	if pm.chain.HasBlock(hashMsg.Hash) {
		return nil
	}
	// update status
	p.UpdateStatus(hashMsg.Height, hashMsg.Hash)
	go p.RequestBlocks(hashMsg.Height, hashMsg.Height)
	return nil
}

// handleTxsMsg handle transactions message
func (pm *ProtocolManager) handleTxsMsg(msg *p2p.Msg) error {
	var txs types.Transactions
	if err := msg.Decode(&txs); err != nil {
		return fmt.Errorf("handleTxsMsg error: %v", err)
	}
	go pm.txPool.RecvTxs(txs)
	return nil
}

// handleBlocksMsg handle receiving blocks message
func (pm *ProtocolManager) handleBlocksMsg(msg *p2p.Msg, p *peer) error {
	var blocks types.Blocks
	if err := msg.Decode(&blocks); err != nil {
		return fmt.Errorf("handleBlocksMsg error: %v", err)
	}
	rcvMsg := &rcvBlockObj{
		p:      p,
		blocks: blocks,
	}
	pm.rcvBlocksCh <- rcvMsg
	return nil
}

// handleGetBlocksMsg handle get blocks message
func (pm *ProtocolManager) handleGetBlocksMsg(msg *p2p.Msg, p *peer) error {
	var query GetBlocksData
	if err := msg.Decode(&query); err != nil {
		return fmt.Errorf("handleGetBlocksMsg error: %v", err)
	}
	if query.From > query.To {
		return errors.New("invalid request blocks' param")
	}
	go pm.respBlocks(query.From, query.To, p, false)
	return nil
}

// respBlocks response blocks to remote peer
func (pm *ProtocolManager) respBlocks(from, to uint32, p *peer, hasChangeLog bool) {
	if from == to {
		b := pm.chain.GetBlockByHeight(from)
		if b == nil {
			log.Warnf("Can't get a block of height %d", from)
			return
		}
		if !hasChangeLog {
			b = b.Copy()
		}
		if b != nil && p != nil {
			p.SendBlocks([]*types.Block{b})
		}
		return
	}

	const eachSize = 10
	total := to - from + 1
	var count uint32
	if total%eachSize == 0 {
		count = total / eachSize
	} else {
		count = total/eachSize + 1
	}
	height := from
	for i := uint32(0); i < count; i++ {
		blocks := make(types.Blocks, 0, eachSize)
		for j := 0; j < eachSize; j++ {
			b := pm.chain.GetBlockByHeight(height)
			if b == nil {
				log.Warnf("Can't get a block of height %d", height)
				break
			}
			if !hasChangeLog {
				b = b.Copy()
			}
			blocks = append(blocks, b)
			height++
			if height > to {
				break
			}
		}
		if p != nil {
			p.SendBlocks(blocks)
		}
	}
}

// handleConfirmsMsg handle received block's confirm package message
func (pm *ProtocolManager) handleConfirmsMsg(msg *p2p.Msg) error {
	var confirms BlockConfirms
	if err := msg.Decode(&confirms); err != nil {
		return fmt.Errorf("handleConfirmsMsg error: %v", err)
	}
	go pm.chain.ReceiveStableConfirms(confirms)
	return nil
}

// handleGetConfirmsMsg handle remote request of block's confirm package message
func (pm *ProtocolManager) handleGetConfirmsMsg(msg *p2p.Msg, p *peer) error {
	var condition GetConfirmInfo
	if err := msg.Decode(&condition); err != nil {
		return fmt.Errorf("handleGetConfirmsMsg error: %v", err)
	}
	confirmInfo := pm.chain.GetConfirms(&condition)
	resMsg := &BlockConfirms{
		Height: condition.Height,
		Hash:   condition.Hash,
		Pack:   confirmInfo,
	}
	go p.SendConfirms(resMsg)
	return nil
}

// handleConfirmMsg handle confirm broadcast info
func (pm *ProtocolManager) handleConfirmMsg(msg *p2p.Msg) error {
	confirm := new(BlockConfirmData)
	if err := msg.Decode(confirm); err != nil {
		return fmt.Errorf("handleConfirmMsg error: %v", err)
	}
	block := pm.chain.GetBlockByHash(confirm.Hash)
	if block != nil {
		if !pm.chain.IsConfirmEnough(block) {
			go pm.chain.InsertConfirm(confirm)
		}
	} else {
		pm.confirmsCache.Push(confirm)
		if pm.confirmsCache.Size() > 100 {
			log.Debugf("confirmsCache's size: %d", pm.confirmsCache.Size())
		}
	}
	return nil
}

// handleDiscoverReqMsg handle discover nodes request
func (pm *ProtocolManager) handleDiscoverReqMsg(msg *p2p.Msg, p *peer) error {
	var condition DiscoverReqData
	if err := msg.Decode(&condition); err != nil {
		return fmt.Errorf("handleDiscoverReqMsg error: %v", err)
	}
	res := new(DiscoverResData)
	res.Sequence = condition.Sequence
	res.Nodes = pm.discover.GetNodesForDiscover(res.Sequence)
	go p.SendDiscoverResp(res)
	return nil
}

// handleDiscoverResMsg handle discover nodes response
func (pm *ProtocolManager) handleDiscoverResMsg(msg *p2p.Msg) error {
	var disRes DiscoverResData
	if err := msg.Decode(&disRes); err != nil {
		return fmt.Errorf("handleDiscoverResMsg error: %v", err)
	}
	pm.discover.AddNewList(disRes.Nodes)
	return nil
}

// handleGetBlocksWithChangeLogMsg for
func (pm *ProtocolManager) handleGetBlocksWithChangeLogMsg(msg *p2p.Msg, p *peer) error {
	var query GetBlocksData
	if err := msg.Decode(&query); err != nil {
		return fmt.Errorf("handleGetBlocksMsg error: %v", err)
	}
	if query.From > query.To {
		return errors.New("invalid request blocks' param")
	}
	go pm.respBlocks(query.From, query.To, p, true)
	return nil
}
