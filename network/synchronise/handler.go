package synchronise

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/subscribe"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"github.com/LemoFoundationLtd/lemochain-go/network/synchronise/protocol"
	"strings"
	"sync"
	"time"
)

type ProtocolManager struct {
	chainID uint64
	nodeID  []byte

	blockchain *chain.BlockChain

	downloader *Downloader
	fetcher    *Fetcher
	discover   *p2p.DiscoverManager

	peers *peerSet

	txPool *chain.TxPool

	newPeerCh       chan *peer
	txsCh           chan types.Transactions
	newMinedBlockCh chan *types.Block
	stableBlockCh   chan *types.Block
	txsSub          subscribe.Subscription
	minedBlockSub   subscribe.Subscription
	stableBlockSub  subscribe.Subscription

	quitSync chan struct{}

	wg sync.WaitGroup

	addPeerCh    chan *p2p.Peer
	removePeerCh chan *p2p.Peer
}

func NewProtocolManager(chainID uint64, nodeID []byte, blockchain *chain.BlockChain, txpool *chain.TxPool, discover *p2p.DiscoverManager) *ProtocolManager {
	manager := &ProtocolManager{
		chainID:         chainID,
		nodeID:          nodeID,
		blockchain:      blockchain,
		peers:           newPeerSet(),
		txPool:          txpool,
		newPeerCh:       make(chan *peer),
		addPeerCh:       make(chan *p2p.Peer),
		removePeerCh:    make(chan *p2p.Peer),
		txsCh:           make(chan types.Transactions, 10),
		newMinedBlockCh: make(chan *types.Block, 1),
		stableBlockCh:   make(chan *types.Block, 1),
		quitSync:        make(chan struct{}),
		discover:        discover,
	}
	// 获取本地链高度
	getLocalHeight := func() uint32 {
		return blockchain.CurrentBlock().Height()
	}
	// 获取本地链已共识的高度
	getConsensusHeight := func() uint32 {
		return blockchain.StableBlock().Height()
	}
	// 将区块集合插入链
	insertToChain := func(block *types.Block) error {
		return blockchain.InsertChain(block, false)
	}
	manager.fetcher = NewFetcher(blockchain.HasBlock, manager.broadcastCurrentBlock, getLocalHeight, getConsensusHeight, insertToChain, manager.dropPeer)
	manager.downloader = NewDownloader(manager.peers, blockchain, manager.dropPeer)

	blockchain.BroadcastConfirmInfo = manager.broadcastConfirmInfo // 广播区块的确认信息
	// blockchain.BroadcastStableBlock = manager.broadcastStableBlock // 广播稳定区块

	return manager
}

// broadcastCurrentBlock 广播区块(只有hash|height:别人挖到的块，完整块：自己挖到的块)
func (pm *ProtocolManager) broadcastCurrentBlock(block *types.Block, hasBody bool) {
	if block == nil {
		log.Warn("can't broadcast nil block")
		return
	}
	var (
		blocks      types.Blocks
		blockHashes protocol.BlockHashesData
	)
	if hasBody {
		blocks = types.Blocks{block}
	} else {
		blockHashes = protocol.BlockHashesData{{block.Hash(), block.Height()}}
	}
	// 分辨共识节点与普通节点 后期可以提到外部，每轮间隔只执行一次
	witnessPeers := make(map[string]*peerConnection, len(pm.peers.peers))
	delayPeers := make(map[string]*peerConnection, len(pm.peers.peers))
	for id, p := range pm.peers.peers {
		if pm.isSelfDeputyNode() { // 本节点为共识节点，
			witnessPeers[id] = p
		} else {
			delayPeers[id] = p
		}
	}
	// 判断本节点是共识节点否
	if pm.isSelfDeputyNode() {
		// 若是则广播给所有节点
		for _, p := range pm.peers.peers {
			if hasBody {
				p.peer.send(protocol.BlockHashesMsg, &blocks)
			} else {
				p.peer.send(protocol.BlockHashesMsg, &blockHashes)
			}
		}
	} else {
		// 若否则不能向共识节点广播
		for _, p := range delayPeers {
			if hasBody {
				p.peer.send(protocol.BlockHashesMsg, &blocks)
			} else {
				p.peer.send(protocol.BlockHashesMsg, &blockHashes)
			}
		}
	}
}

// isPeerDeputyNode 判断节点是否为共识节点
func (pm *ProtocolManager) isPeerDeputyNode(height uint32, id string) bool {
	nodeID := common.FromHex(id)
	node := deputynode.Instance().GetDeputyByNodeID(height, nodeID)
	if node == nil {
		return false
	}
	return true
}

// isSelfDeputyNode 本节点是否为共识节点
func (pm *ProtocolManager) isSelfDeputyNode() bool {
	node := deputynode.Instance().GetDeputyByNodeID(pm.blockchain.CurrentBlock().Height(), pm.nodeID)
	if node == nil {
		return false
	}
	return true
}

// broadcastConfirmInfo 广播自己对收到的区块确认签名信息
func (pm *ProtocolManager) broadcastConfirmInfo(hash common.Hash, height uint32) {
	data := protocol.BlockConfirmData{
		Hash:   hash,
		Height: height,
	}
	privateKey := deputynode.GetSelfNodeKey()
	signInfo, err := crypto.Sign(hash[:], privateKey)
	if err != nil {
		log.Error("sign for confirm data error")
		return
	}
	copy(data.SignInfo[:], signInfo)
	// record to local db
	if err := pm.blockchain.Db().SetConfirmInfo(hash, data.SignInfo); err != nil {
		log.Warnf("record confirm info to local failed.error: %v", err)
	}
	for id, p := range pm.peers.peers {
		if pm.isPeerDeputyNode(height, id) {
			p.peer.send(protocol.NewConfirmMsg, &data)
		}
	}
}

// broadcastStableBlock 广播稳定区块给普通全节点
func (pm *ProtocolManager) broadcastStableBlock(block *types.Block) {
	if block == nil {
		log.Warn("can't broadcast nil stable block ")
		return
	}
	for id, p := range pm.peers.peers {
		if !pm.isPeerDeputyNode(block.Height(), id) {
			p.peer.send(protocol.NewConfirmMsg, &block)
		}
	}
}

// dropPeer 断开连接
func (pm *ProtocolManager) dropPeer(id string) {
	pm.peers.Unregister(id)
}

// handle 处理新的节点连接
func (pm *ProtocolManager) handle(p *peer) error {
	block := pm.blockchain.CurrentBlock()
	if err := p.Handshake(pm.chainID, block.Height(), block.Hash(), pm.blockchain.Genesis().Hash()); err != nil {
		p.DisableReConnect()
		p.Close()
		log.Infof("lemochain handshake failed: %v", err)

		// for discover
		pm.discover.SetConnectResult(p.NodeID(), false)

		return err
	}
	log.Infof("A new peer has connected. peer: %s", p.id[:16])
	pm.syncTransactions(p.id)

	pConn := &peerConnection{
		id:   p.id,
		peer: p,
	}
	pm.peers.Register(pConn)
	// pm.downloader.RegisterPeer(p.id, p)

	// for discover
	pm.discover.SetConnectResult(p.NodeID(), true)

	pm.newPeerCh <- p

	// 死循环 处理收到的网络消息
	for {
		if err := pm.handleMsg(pConn); err != nil {
			log.Debug("lemo chain message handled failed")
			return err
		}
	}
}

// handleMsg 处理节点发送的消息
func (pm *ProtocolManager) handleMsg(p *peerConnection) error {
	msg := p.peer.ReadMsg()
	if msg.Empty() {
		return errors.New("read message error")
	}
	switch msg.Code {
	case protocol.BlockHashesMsg: // 只有block的hash
		if pm.isSelfDeputyNode() && !pm.isPeerDeputyNode(pm.blockchain.CurrentBlock().Height(), p.id) {
			return errors.New("recv block hashes message broadcast by delay node")
		}
		var announces protocol.BlockHashesData
		if err := msg.Decode(&announces); err != nil {
			return errResp(protocol.ErrDecode, "%v: %v", msg, err)
		}
		for _, block := range announces {
			p.peer.MarkBlock(block.Hash)
		}
		unknown := make(protocol.BlockHashesData, 0, len(announces))
		for _, block := range announces {
			if !pm.blockchain.HasBlock(block.Hash) {
				unknown = append(unknown, block)
			}
		}
		for _, block := range unknown {
			pm.fetcher.Notify(p.id, block.Hash, block.Height, p.peer.RequestOneBlock)
		}
	case protocol.TxMsg:
		var txs types.Transactions
		if err := msg.Decode(&txs); err != nil {
			return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
		}
		for i, tx := range txs {
			if tx == nil {
				return errResp(protocol.ErrDecode, "transaction %d is nil", i)
			}
			p.peer.MarkTransaction(tx.Hash())
		}
		pm.txPool.AddTxs(txs)
	case protocol.GetBlocksMsg:
		const eachSize = 50
		var query protocol.GetBlocksData
		if err := msg.Decode(&query); err != nil {
			return errResp(protocol.ErrDecode, "%v: %v", msg, err)
		}
		if query.From > query.To {
			return errResp(protocol.ErrInvalidMsg, "%v: %s", msg, "from > to")
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
				blocks = append(blocks, pm.blockchain.GetBlockByHeight(height))
				height++
				if height > query.To {
					break
				}
			}
			if err = p.peer.send(protocol.BlocksMsg, blocks); err != nil { // 一次发送10个区块
				return errResp(protocol.ErrSendBlocks, "%v: %v", msg, err)
			}
		}
	case protocol.BlocksMsg: // 远程节点应答的区块集消息
		var blocks types.Blocks
		if err := msg.Decode(&blocks); err != nil {
			return errResp(protocol.ErrDecode, "%v: %v", msg, err)
		}
		if len(blocks) == 0 {
			return errResp(protocol.ErrNoBlocks, "%v: %s", msg, "no blocks data")
		}
		log.Infof("Receive blocks from: %s. from: %d -- to: %d", p.id[:16], blocks[0].Height(), blocks[len(blocks)-1].Height())
		filter := len(blocks) == 1 // len(blocks) == 1 不一定为fetcher，但len(blocks)>1肯定是downloader
		if filter {
			blocks = pm.fetcher.FilterBlocks(p.id, blocks, p.peer.RequestOneBlock)
		}
		if !filter || len(blocks) > 0 {
			if err := pm.downloader.DeliverBlocks(p.id, blocks); err != nil {
				log.Debug("failed to deliver blocks", "err", err)
			}
		}
	case protocol.GetSingleBlockMsg: // 收到一个获取区块的消息
		var query protocol.GetSingleBlockData
		if err := msg.Decode(&query); err != nil {
			return errResp(protocol.ErrDecode, "%v: %v", msg, err)
		}
		block := pm.blockchain.GetBlockByHash(query.Hash)
		p.peer.send(protocol.SingleBlockMsg, block)
	case protocol.SingleBlockMsg: // 收到一个获取区块返回
		var block types.Block
		if err := msg.Decode(&block); err != nil {
			return errResp(protocol.ErrDecode, "%v: %v", msg, err)
		}
		log.Infof("Receive single block. height: %d", block.Height())
		pm.fetcher.Enqueue(p.id, &block, p.peer.RequestOneBlock)
	case protocol.NewBlockMsg: // 远程节点主动推送的挖到的最新区块消息
		if !pm.isSelfDeputyNode() {
			return errors.New("self node isn't a deputy node")
		}
		if !pm.isPeerDeputyNode(pm.blockchain.CurrentBlock().Height(), p.id) {
			return errors.New("recv new block message broadcast by delay node")
		}
		var block *types.Block
		if err := msg.Decode(&block); err != nil {
			return errResp(protocol.ErrDecode, "%v: %v", msg, err)
		}
		p.peer.MarkBlock(block.Hash()) // 标记区块
		pm.fetcher.Enqueue(p.id, block, p.peer.RequestOneBlock)
		log.Infof("Receive new block. height: %d. hash: %s", block.Height(), block.Hash().Hex())
		if block.Height() > p.peer.height {
			p.peer.SetHead(block.Hash(), block.Height())
			// log.Debugf("setHead to peer: %s. height: %d", p.peer.id[:16], block.Height())
			// remove tx from pool
			txs := make([]common.Hash, len(block.Txs))
			for i, tx := range block.Txs {
				txs[i] = tx.Hash()
			}
			pm.txPool.Remove(txs)
			currentHeight := pm.blockchain.CurrentBlock().Height()
			if currentHeight+1 < block.Height() {
				go pm.synchronise(p.id)
			}
		}
	case protocol.NewConfirmMsg: // 被动收到远程节点推送的新区块确认信息
		var confirmMsg protocol.BlockConfirmData
		if err := msg.Decode(&confirmMsg); err != nil {
			return errResp(protocol.ErrDecode, "%v: %v", msg, err)
		}
		// 是否有对应的区块 后续优化
		if block := pm.blockchain.GetBlockByHash(confirmMsg.Hash); block == nil {
			go p.peer.RequestOneBlock(confirmMsg.Hash, confirmMsg.Height)
			log.Debugf("Receive confirm package, but block doesn't exist in local chain. hash:%s height:%d", confirmMsg.Hash.Hex(), confirmMsg.Height)
		} else {
			pm.blockchain.ReceiveConfirm(&confirmMsg)
		}
	case protocol.GetConfirmInfoMsg: // 收到远程节点发来的请求
		var query protocol.GetConfirmInfo
		if err := msg.Decode(&query); err != nil {
			return errResp(protocol.ErrDecode, "%v: %v", msg, err)
		}
		confirmInfo := pm.blockchain.GetConfirms(&query)
		if confirmInfo == nil {
			log.Warn(fmt.Sprintf("can't get confirm package of block: height(%d) hash(%s)", query.Height, query.Hash.Hex()))
			return nil
		}
		if err := p.peer.send(protocol.ConfirmInfoMsg, confirmInfo); err != nil {
			log.Debug("send confirm info message failed.")
		}
	case protocol.ConfirmInfoMsg: // 收到远程节点的获取确信包答复
		var pack protocol.BlockConfirms
		if err := msg.Decode(&pack); err != nil {
			return errResp(protocol.ErrDecode, "%v: %v", msg, err)
		}
		pm.blockchain.ReceiveConfirms(pack)
	case protocol.FindNodeReqMsg: // for discover
		log.Debug("recv node discovery request....")
		var req protocol.FindNodeReqData
		if err := msg.Decode(&req); err != nil {
			return errResp(protocol.ErrDecode, "%v: %v", msg, err)
		}
		res := new(protocol.FindNodeResData)
		res.Sequence = req.Sequence
		res.Nodes = pm.discover.GetNodesForDiscover(req.Sequence)
		if err := p.peer.send(protocol.FindNodeResMsg, &res); err != nil {
			log.Debug("send confirm info message failed.")
		}
	case protocol.FindNodeResMsg: // for discover
		var data protocol.FindNodeResData
		if err := msg.Decode(&data); err != nil {
			return errResp(protocol.ErrDecode, "%v: %v", msg, err)
		}
		pm.discover.AddNewList(data.Nodes)
	default:
		return errors.New("can not math message type")
	}
	return nil
}

// errResp 根据code生成错误
func errResp(code protocol.ErrCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

// Start 启动pm
func (pm *ProtocolManager) Start() {
	pm.txsSub = pm.txPool.NewTxsFeed.Subscribe(pm.txsCh)
	pm.minedBlockSub = pm.blockchain.MinedBlockFeed.Subscribe(pm.newMinedBlockCh)
	pm.stableBlockSub = pm.blockchain.StableBlockFeed.Subscribe(pm.stableBlockCh)

	subscribe.Sub(subscribe.AddNewPeer, pm.addPeerCh)
	subscribe.Sub(subscribe.DeletePeer, pm.removePeerCh)

	go pm.txBroadcastLoop()
	go pm.blockBroadcastLoop()
	go pm.syncAndDiscover()
	go pm.peerEventLoop()
}

// Stop 停止pm
func (pm *ProtocolManager) Stop() {
	pm.txsSub.Unsubscribe()
	pm.minedBlockSub.Unsubscribe()
	pm.stableBlockSub.Unsubscribe()

	subscribe.UnSub(subscribe.AddNewPeer, pm.addPeerCh)
	subscribe.UnSub(subscribe.DeletePeer, pm.removePeerCh)

	close(pm.quitSync)
	pm.wg.Wait()
	log.Info("ProtocolManager stop")
}

func (pm *ProtocolManager) peerEventLoop() {
	pm.wg.Add(1)
	defer pm.wg.Done()

	for {
		select {
		case p := <-pm.addPeerCh:
			peer := newPeer(p)
			go pm.handle(peer)
		case p := <-pm.removePeerCh:
			go pm.dropPeer(p.NodeID().String())
		case <-pm.quitSync:
			return
		}
	}
}

// txBroadcastLoop 交易广播
func (pm *ProtocolManager) txBroadcastLoop() {
	pm.wg.Add(1)
	defer pm.wg.Done()
	for {
		select {
		case txs := <-pm.txsCh:
			pm.BroadcastTxs(txs)
		case <-pm.quitSync:
			return
		}
	}
}

// blockBroadcastLoop 出块广播
func (pm *ProtocolManager) blockBroadcastLoop() {
	pm.wg.Add(1)
	defer pm.wg.Done()
	for {
		select {
		case block := <-pm.newMinedBlockCh:
			if block == nil {
				log.Warn("Can't broadcast nil block ")
				return
			}
			for id, p := range pm.peers.peers {
				if pm.isPeerDeputyNode(block.Height(), id) {
					go p.peer.send(protocol.NewBlockMsg, &block)
				}
			}
			time.AfterFunc(2*time.Second, func() {
				go pm.broadcastConfirmInfo(block.Hash(), block.Height())
			})
		case block := <-pm.stableBlockCh:
			pm.broadcastStableBlock(block)
		case <-pm.quitSync:
			return
		}
	}
}

// syncAndDiscover 同步区块/发现节点
func (pm *ProtocolManager) syncAndDiscover() {
	pm.wg.Add(1)
	defer pm.wg.Done()

	pm.fetcher.Start()
	defer pm.fetcher.Stop()
	defer pm.downloader.Terminate()

	duration := 10 * time.Second
	disTimer := time.NewTimer(duration)
	for {
		select {
		case p := <-pm.newPeerCh:
			if p.height > pm.blockchain.CurrentBlock().Height() {
				go pm.synchronise(p.id)
			}
		case <-disTimer.C: // for discover
			if len(pm.peers.peers) > 5 {
				disTimer.Reset(duration)
				break
			}
			p := pm.peers.ToDiscover()
			if p != nil {
				req := &protocol.FindNodeReqData{
					Sequence: p.sequence,
				}
				if err := p.peer.send(protocol.FindNodeReqMsg, req); err == nil {
					log.Debugf("send discovery request to: %s", p.peer.RemoteAddr())
				} else {
					log.Debugf("send discovery request to: %s failed!!!!", p.peer.RemoteAddr())
				}
			}
			disTimer.Reset(duration)
		case <-pm.quitSync:
			disTimer.Stop()
			return
		}
	}
}

// synchronise 同步区块
func (pm *ProtocolManager) synchronise(p string) {
	if strings.Compare(p, "") == 0 {
		return
	}
	log.Infof("start synchronise from: %s", p[:16])
	if err := pm.downloader.Synchronise(p); err != nil {
		return
	}
	// 通知其他节点本节点当前高度
	if block := pm.blockchain.CurrentBlock(); block.Height() > 0 {
		go pm.broadcastCurrentBlock(block, false)
	}
}

// syncTransactions 同步交易 发送本地所有交易到该节点
func (pm *ProtocolManager) syncTransactions(id string) {
	pending := pm.txPool.Pending(1000000)
	if len(pending) == 0 {
		return
	}
	p := pm.peers.Peer(id)
	if p != nil {
		go p.peer.SendTransactions(pending)
	}
}

// BroadcastTx 广播交易
func (pm *ProtocolManager) BroadcastTxs(txs types.Transactions) {
	peers := pm.peers.PeersWithoutTx(txs[0].Hash())
	for _, peer := range peers {
		peer.SendTransactions(txs)
	}
}
