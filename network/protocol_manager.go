package network

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/subscribe"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"sync"
	"time"
)

const (
	ForceSyncInternal = 10 * time.Second
	DiscoverInternal  = 10 * time.Second
	ReqStatusTimeout  = 5 * time.Second // must less than ForceSyncInternal

	SyncTimeout = int64(20)
)

type rcvBlockObj struct {
	p      *peer
	blocks types.Blocks
}

type ProtocolManager struct {
	chainID     uint16
	nodeID      p2p.NodeID
	nodeVersion uint32

	chain    BlockChain
	discover *p2p.DiscoverManager
	txPool   TxPool

	peers         *peerSet      // connected peers
	confirmsCache *ConfirmCache // received confirm info before block, cache them
	blockSyncFlag *BlockSyncFlag

	addPeerCh    chan p2p.IPeer
	removePeerCh chan p2p.IPeer

	txsCh           chan types.Transactions
	newMinedBlockCh chan *types.Block
	stableBlockCh   chan *types.Block
	rcvBlocksCh     chan *rcvBlockObj
	lstStatusCh     chan *LatestStatus // peer's latest status channel
	confirmCh       chan *BlockConfirmData

	wg     sync.WaitGroup
	quitCh chan struct{}
}

func NewProtocolManager(chainID uint16, nodeID p2p.NodeID, chain BlockChain, txPool TxPool, discover *p2p.DiscoverManager, nodeVersion uint32) *ProtocolManager {
	pm := &ProtocolManager{
		chainID:       chainID,
		nodeID:        nodeID,
		nodeVersion:   nodeVersion,
		chain:         chain,
		txPool:        txPool,
		discover:      discover,
		peers:         NewPeerSet(),
		confirmsCache: NewConfirmCache(),
		blockSyncFlag: NewBlockSync(),

		addPeerCh:    make(chan p2p.IPeer),
		removePeerCh: make(chan p2p.IPeer),

		txsCh:           make(chan types.Transactions),
		newMinedBlockCh: make(chan *types.Block),
		stableBlockCh:   make(chan *types.Block),
		rcvBlocksCh:     make(chan *rcvBlockObj),
		lstStatusCh:     make(chan *LatestStatus),
		confirmCh:       make(chan *BlockConfirmData),

		quitCh: make(chan struct{}),
	}
	pm.sub()
	return pm
}

// sub subscribe channel
func (pm *ProtocolManager) sub() {
	subscribe.Sub(subscribe.AddNewPeer, pm.addPeerCh)
	subscribe.Sub(subscribe.DeletePeer, pm.removePeerCh)
	subscribe.Sub(subscribe.NewMinedBlock, pm.newMinedBlockCh)
	subscribe.Sub(subscribe.NewStableBlock, pm.stableBlockCh)
	subscribe.Sub(subscribe.NewTxs, pm.txsCh)
	subscribe.Sub(subscribe.NewConfirm, pm.confirmCh)
}

// unSub unsubscribe channel
func (pm *ProtocolManager) unSub() {
	subscribe.UnSub(subscribe.AddNewPeer, pm.addPeerCh)
	subscribe.UnSub(subscribe.DeletePeer, pm.removePeerCh)
	subscribe.UnSub(subscribe.NewMinedBlock, pm.newMinedBlockCh)
	subscribe.UnSub(subscribe.NewStableBlock, pm.stableBlockCh)
	subscribe.UnSub(subscribe.NewTxs, pm.txsCh)
	subscribe.UnSub(subscribe.NewConfirm, pm.confirmCh)
}

// Start
func (pm *ProtocolManager) Start() {
	go pm.txConfirmLoop()
	go pm.blockLoop()
	go pm.peerLoop()
}

// Stop
func (pm *ProtocolManager) Stop() {
	pm.unSub()
	close(pm.quitCh)
	pm.wg.Wait()
	log.Debug("ProtocolManager has stopped")
}

// txConfirmLoop receive transactions and confirm and then broadcast them
func (pm *ProtocolManager) txConfirmLoop() {
	pm.wg.Add(1)
	defer pm.wg.Done()

	for {
		select {
		case <-pm.quitCh:
			log.Debug("txLoop finished")
			return
		case txs := <-pm.txsCh:
			curHeight := pm.chain.CurrentBlock().Height()
			peers := pm.peers.DeputyNodes(curHeight)
			if len(peers) == 0 {
				peers = pm.peers.DelayNodes(curHeight)
			}
			pm.broadcastTxs(peers, txs)
		case info := <-pm.confirmCh:
			curHeight := pm.chain.CurrentBlock().Height()
			peers := pm.peers.DeputyNodes(curHeight)
			if len(peers) > 0 {
				pm.broadcastConfirm(peers, info)
			}
		}
	}
}

// blockLoop receive special type block event
func (pm *ProtocolManager) blockLoop() {
	pm.wg.Add(1)
	defer pm.wg.Done()

	proInterval := 3 * time.Second
	queueTimer := time.NewTimer(proInterval)
	container := NewBlockCache()

	for {
		select {
		case <-pm.quitCh:
			log.Debug("blockLoop finished")
			return
		case block := <-pm.newMinedBlockCh:
			peers := pm.peers.DeputyNodes(block.Height())
			if len(peers) > 0 {
				pm.broadcastBlock(peers, block, true)
			}
		case block := <-pm.stableBlockCh:
			peers := pm.peers.DelayNodes(block.Height())
			if len(peers) > 0 {
				pm.broadcastBlock(peers, block, false)
			}
			pm.confirmsCache.Clear(block.Height())
			container.Clear(block.Height())
		case rcvMsg := <-pm.rcvBlocksCh:
			// peer's latest height
			pLstHeight := rcvMsg.p.LatestStatus().CurHeight
			// first block of blocks
			fBlock := rcvMsg.blocks[0]

			// synchronising
			if pm.blockSyncFlag.running {
				if pm.blockSyncFlag.peer.NodeID() == rcvMsg.p.NodeID() {
					// broadcast
					if len(rcvMsg.blocks) == 1 && fBlock.Height() > pLstHeight {
						rcvMsg.p.UpdateStatus(fBlock.Height(), fBlock.Hash())
						container.Add(fBlock)
						break
					}
					// sync
					for _, block := range rcvMsg.blocks {
						curBlock := pm.chain.CurrentBlock()
						if curBlock.Hash() != block.ParentHash() {
							pm.blockSyncFlag.Finish()
							break
						}
						pm.chain.InsertChain(block, true)
					}
				} else if len(rcvMsg.blocks) == 1 && fBlock.Height() > pLstHeight {
					rcvMsg.p.UpdateStatus(fBlock.Height(), fBlock.Hash())
					container.Add(fBlock)
				}
			} else if len(rcvMsg.blocks) == 1 && fBlock.Height() > pLstHeight {
				rcvMsg.p.UpdateStatus(fBlock.Height(), fBlock.Hash())
				curBlock := pm.chain.CurrentBlock()
				if curBlock.Hash() == fBlock.ParentHash() {
					pm.chain.InsertChain(fBlock, true)
				} else {
					container.Add(fBlock)
				}
			}
		case <-queueTimer.C:
			processBlock := func(block *types.Block) bool {
				if pm.chain.HasBlock(block.ParentHash()) {
					pm.chain.InsertChain(block, false)
					return true
				}
				return false
			}
			container.Iterate(processBlock)
			queueTimer.Reset(proInterval)
		}
	}
}

// peerLoop something about peer event
func (pm *ProtocolManager) peerLoop() {
	pm.wg.Add(1)
	defer pm.wg.Done()

	forceSyncTimer := time.NewTimer(ForceSyncInternal)
	discoverTimer := time.NewTimer(DiscoverInternal)
	for {
		select {
		case <-pm.quitCh:
			log.Debug("peerLoop finished")
			return
		case rPeer := <-pm.addPeerCh: // new peer added
			p := newPeer(rPeer)
			go pm.handlePeer(p)
		case rPeer := <-pm.removePeerCh:
			p := newPeer(rPeer)
			pm.peers.UnRegister(p)
		case <-forceSyncTimer.C: // time to synchronise block
			now := time.Now().Unix()
			if !pm.blockSyncFlag.running || (now-pm.blockSyncFlag.lastUpdate) > SyncTimeout {
				p := pm.peers.BestToSync(pm.chain.CurrentBlock().Height())
				if p != nil {
					pm.blockSyncFlag.Init(p)
					go pm.forceSyncBlock(p)
				}
			}
			forceSyncTimer.Reset(ForceSyncInternal)
		case <-discoverTimer.C: // time to discover
			if len(pm.peers.peers) < 5 {
				p := pm.peers.BestToDiscover()
				if p != nil {
					p.SendDiscover()
				}
			}
			discoverTimer.Reset(DiscoverInternal)
		}
	}
}

// broadcastTxs broadcast transaction
func (pm *ProtocolManager) broadcastTxs(peers []*peer, txs types.Transactions) {
	for _, p := range peers {
		p.SendTxs(txs)
	}
}

// broadcastConfirm broadcast confirm info to deputy nodes
func (pm *ProtocolManager) broadcastConfirm(peers []*peer, confirmInfo *BlockConfirmData) {
	for _, p := range peers {
		p.SendConfirmInfo(confirmInfo)
	}
}

// broadcastBlock broadcast block
func (pm *ProtocolManager) broadcastBlock(peers []*peer, block *types.Block, withBody bool) {
	for _, p := range peers {
		if withBody {
			err := p.SendBlocks([]*types.Block{block})
			if err != nil {
				log.Warnf("broadcast block failed: %v", err)
			}
		} else {
			err := p.SendBlockHash(block.Height(), block.Hash())
			if err != nil {
				log.Warnf("broadcast block failed: %v", err)
			}
		}
	}

}

// handlePeer handle about peer
func (pm *ProtocolManager) handlePeer(p *peer) {
	// handshake
	rStatus, err := pm.handshake(p)
	if err != nil {
		log.Warnf("protocol handshake failed: %v", err)
		pm.discover.SetConnectResult(p.NodeID(), false)
		p.Close()
		return
	}
	// register peer to set
	pm.peers.Register(p)
	// synchronise block
	if pm.chain.CurrentBlock().Height() < rStatus.LatestStatus.CurHeight {
		from, err := pm.findSyncFrom(&rStatus.LatestStatus)
		if err != nil {
			log.Warnf("find sync from error: %v", err)
			pm.discover.SetConnectResult(p.NodeID(), false)
			p.Close()
			return
		}
		now := time.Now().Unix()
		if !pm.blockSyncFlag.running || now-pm.blockSyncFlag.lastUpdate >= SyncTimeout {
			pm.blockSyncFlag.Init(p)
			p.RequestBlocks(from, rStatus.LatestStatus.CurHeight)
		}
	}
	// set connect result
	pm.discover.SetConnectResult(p.NodeID(), true)

	for {
		// handle peer net message
		if err := pm.handleMsg(p); err != nil {
			log.Debugf("handle message failed: %v", err)
			pm.peers.UnRegister(p)
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
func (pm *ProtocolManager) forceSyncBlock(p *peer) {
	// request remote latest status
	p.SendReqLatestStatus()
	// set timeout
	timeoutTimer := time.NewTimer(ReqStatusTimeout)
	select {
	case status := <-pm.lstStatusCh:
		from, err := pm.findSyncFrom(status)
		if err != nil {
			log.Warnf("find sync from error: %v", err)
			pm.blockSyncFlag.Finish()
			pm.peers.UnRegister(p)
			return
		}
		p.RequestBlocks(from, status.CurHeight)
	case <-timeoutTimer.C:
		pm.blockSyncFlag.Error()
	case <-pm.quitCh:
		pm.blockSyncFlag.Finish()
	}
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
	switch msg.Code {
	case HeartbeatMsg:
		return nil
	case LstStatusMsg:
		log.Debugf("handleMsg: receive LstStatusMsg from: %s", common.ToHex(p.NodeID()[:8]))
		return pm.handleLstStatusMsg(msg)
	case GetLstStatusMsg:
		log.Debugf("handleMsg: receive GetLstStatusMsg", common.ToHex(p.NodeID()[:8]))
		return pm.handleGetLstStatusMsg(msg, p)
	case BlockHashMsg:
		log.Debugf("handleMsg: receive BlockHashMsg from: %s", common.ToHex(p.NodeID()[:8]))
		return pm.handleBlockHashMsg(msg, p)
	case TxsMsg:
		log.Debugf("handleMsg: receive TxsMsg from: %s", common.ToHex(p.NodeID()[:8]))
		return pm.handleTxsMsg(msg)
	case BlocksMsg:
		log.Debugf("handleMsg: receive BlocksMsg from: %s", common.ToHex(p.NodeID()[:8]))
		return pm.handleBlocksMsg(msg, p)
	case GetBlocksMsg:
		log.Debugf("handleMsg: receive GetBlocksMsg from: %s", common.ToHex(p.NodeID()[:8]))
		return pm.handleGetBlocksMsg(msg, p)
	case GetConfirmsMsg:
		log.Debugf("handleMsg: receive GetConfirmsMsg from: %s", common.ToHex(p.NodeID()[:8]))
		return pm.handleGetConfirmsMsg(msg, p)
	case ConfirmsMsg:
		log.Debugf("handleMsg: receive ConfirmsMsg from: %s", common.ToHex(p.NodeID()[:8]))
		return pm.handleConfirmsMsg(msg)
	case ConfirmMsg:
		log.Debugf("handleMsg: receive ConfirmMsg from: %s", common.ToHex(p.NodeID()[:8]))
		return pm.handleConfirmMsg(msg)
	case DiscoverReqMsg:
		log.Debugf("handleMsg: receive DiscoverReqMsg from: %s", common.ToHex(p.NodeID()[:8]))
		return pm.handleDiscoverReqMsg(msg, p)
	case DiscoverResMsg:
		log.Debugf("handleMsg: receive DiscoverResMsg from: %s", common.ToHex(p.NodeID()[:8]))
		return pm.handleDiscoverResMsg(msg)
	default:
		log.Debugf("invalid code: %d, from: %s", msg.Code, common.ToHex(p.NodeID()[:8]))
		return ErrInvalidCode
	}
	return nil
}

// handleLstStatusMsg handle latest remote status message
func (pm *ProtocolManager) handleLstStatusMsg(msg *p2p.Msg) error {
	var status LatestStatus
	if err := msg.Decode(&status); err != nil {
		return err
	}
	pm.lstStatusCh <- &status
	return nil
}

// handleGetLstStatusMsg handle request of latest status
func (pm *ProtocolManager) handleGetLstStatusMsg(msg *p2p.Msg, p *peer) error {
	var req GetLatestStatus
	if err := msg.Decode(&req); err != nil {
		return err
	}
	status := &LatestStatus{
		CurHeight: pm.chain.CurrentBlock().Height(),
		CurHash:   pm.chain.CurrentBlock().Hash(),
		StaHeight: pm.chain.StableBlock().Height(),
		StaHash:   pm.chain.StableBlock().Hash(),
	}
	return p.SendLstStatus(status)
}

// handleBlockHashMsg handle receiving block's hash message
func (pm *ProtocolManager) handleBlockHashMsg(msg *p2p.Msg, p *peer) error {
	var hashMsg BlockHashData
	if err := msg.Decode(&hashMsg); err != nil {
		return err
	}
	if pm.chain.HasBlock(hashMsg.Hash) {
		return nil
	}
	p.RequestBlocks(hashMsg.Height, hashMsg.Height)
	return nil
}

// handleTxsMsg handle transactions message
func (pm *ProtocolManager) handleTxsMsg(msg *p2p.Msg) error {
	var txs types.Transactions
	if err := msg.Decode(&txs); err != nil {
		return err
	}
	pm.txPool.AddTxs(txs)
	return nil
}

// handleBlocksMsg handle receiving blocks message
func (pm *ProtocolManager) handleBlocksMsg(msg *p2p.Msg, p *peer) error {
	var blocks types.Blocks
	if err := msg.Decode(&blocks); err != nil {
		return err
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
		return err
	}
	const eachSize = 10
	if query.From > query.To {
		return errors.New("invalid request blocks' param")
	}
	total := query.To - query.From
	var count uint32
	if total%eachSize == 0 {
		count = total / eachSize
	} else {
		count = total/eachSize + 1
	}
	height := query.From
	var err error
	for i := uint32(0); i < count; i++ {
		blocks := make(types.Blocks, 0, eachSize)
		for j := 0; j < eachSize; j++ {
			blocks = append(blocks, pm.chain.GetBlockByHeight(height))
			height++
			if height > query.To {
				break
			}
		}
		if err = p.SendBlocks(blocks); err != nil {
			return err
		}
	}
	return nil
}

// handleConfirmsMsg handle received block's confirm package message
func (pm *ProtocolManager) handleConfirmsMsg(msg *p2p.Msg) error {
	var confirms BlockConfirms
	if err := msg.Decode(&confirms); err != nil {
		return err
	}
	return pm.chain.ReceiveConfirms(confirms)
}

// handleGetConfirmsMsg handle remote request of block's confirm package message
func (pm *ProtocolManager) handleGetConfirmsMsg(msg *p2p.Msg, p *peer) error {
	var condition GetConfirmInfo
	if err := msg.Decode(&condition); err != nil {
		return err
	}
	confirmInfo := pm.chain.GetConfirms(&condition)
	resMsg := &BlockConfirms{
		Height: condition.Height,
		Hash:   condition.Hash,
		Pack:   confirmInfo,
	}
	return p.SendConfirms(resMsg)
}

// handleConfirmMsg handle confirm broadcast info
func (pm *ProtocolManager) handleConfirmMsg(msg *p2p.Msg) error {
	confirm := new(BlockConfirmData)
	if err := msg.Decode(confirm); err != nil {
		return err
	}
	if confirm.Height < pm.chain.StableBlock().Height() {
		return nil
	}

	if pm.chain.HasBlock(confirm.Hash) {
		return pm.chain.ReceiveConfirm(confirm)
	} else {
		pm.confirmsCache.Push(confirm)
	}
	return nil
}

// handleDiscoverReqMsg handle discover nodes request
func (pm *ProtocolManager) handleDiscoverReqMsg(msg *p2p.Msg, p *peer) error {
	var condition DiscoverReqData
	if err := msg.Decode(&condition); err != nil {
		return err
	}
	res := new(DiscoverResData)
	res.Sequence = condition.Sequence
	res.Nodes = pm.discover.GetNodesForDiscover(res.Sequence)
	return p.SendDiscoverResp(res)
}

// handleDiscoverResMsg handle discover nodes response
func (pm *ProtocolManager) handleDiscoverResMsg(msg *p2p.Msg) error {
	var disRes DiscoverResData
	if err := msg.Decode(&disRes); err != nil {
		return err
	}
	return pm.discover.AddNewList(disRes.Nodes)
}
