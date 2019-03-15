package miner

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/subscribe"
	"sync"
	"sync/atomic"
	"time"
)

type MineConfig struct {
	SleepTime int64
	Timeout   int64
}

type Miner struct {
	blockInterval int64
	timeoutTime   int64
	privKey       *ecdsa.PrivateKey
	minerAddress  common.Address
	txPool        *chain.TxPool
	mining        int32
	engine        chain.Engine
	chain         *chain.BlockChain
	txProcessor   *chain.TxProcessor
	mux           sync.Mutex
	currentBlock  func() *types.Block
	extra         []byte // 扩展数据 暂保留 最大256byte

	blockMineTimer *time.Timer // 出块timer

	recvNewBlockCh chan *types.Block // 收到新块通知
	recvBlockSub   subscribe.Subscription
	timeToMineCh   chan struct{} // 到出块时间了
	// startCh        chan struct{}
	stopCh chan struct{} // 停止挖矿
	quitCh chan struct{} // 退出
}

func New(cfg *MineConfig, chain *chain.BlockChain, txPool *chain.TxPool, engine chain.Engine) *Miner {
	m := &Miner{
		blockInterval:  cfg.SleepTime,
		timeoutTime:    cfg.Timeout,
		privKey:        deputynode.GetSelfNodeKey(),
		chain:          chain,
		txPool:         txPool,
		engine:         engine,
		currentBlock:   chain.CurrentBlock,
		txProcessor:    chain.TxProcessor(),
		recvNewBlockCh: make(chan *types.Block, 1),
		timeToMineCh:   make(chan struct{}),
		// startCh:        make(chan struct{}),
		stopCh: make(chan struct{}),
		quitCh: make(chan struct{}),
	}
	// go m.loopRecvBlock()
	return m
}

func (m *Miner) Start() {
	if !atomic.CompareAndSwapInt32(&m.mining, 0, 1) {
		log.Warn("have already start mining")
		return
	}
	select {
	case <-m.timeToMineCh:
	default:
	}
	// update miner address to miner object
	curBlock := m.chain.CurrentBlock()
	m.updateMiner(curBlock)

	// m.startCh <- struct{}{}
	go m.loopMiner()
	if m.isSelfDeputyNode() {
		waitTime := m.getSleepTime()
		if waitTime == 0 {
			m.sealBlock()
		} else if waitTime > 0 {
			m.resetMinerTimer(int64(waitTime))
		} else {
			log.Error("internal error. start mining failed")
			atomic.CompareAndSwapInt32(&m.mining, 1, 0)
			m.stopCh <- struct{}{}
			return
		}
	} else {
		m.resetMinerTimer(-1)
	}
	m.recvBlockSub = m.chain.RecvBlockFeed.Subscribe(m.recvNewBlockCh)
	log.Info("start mining...")
}

func (m *Miner) Stop() {
	if !atomic.CompareAndSwapInt32(&m.mining, 1, 0) {
		log.Warn("doesn't start mining")
		return
	}
	if m.blockMineTimer != nil {
		m.blockMineTimer.Stop()
	}
	m.stopCh <- struct{}{}
	m.recvBlockSub.Unsubscribe()
	log.Info("stop mining success")
}

func (m *Miner) Close() {
	close(m.quitCh)
}

func (m *Miner) IsMining() bool {
	if m.isSelfDeputyNode() == false {
		return false
	}
	return atomic.LoadInt32(&m.mining) == 1
}

func (m *Miner) SetMinerAddress(address common.Address) {
	m.minerAddress = address
}

func (m *Miner) GetMinerAddress() common.Address {
	return m.minerAddress
}

// 获取最新区块的时间戳离当前时间的距离 单位：ms
func (m *Miner) getTimespan() int64 {
	lstSpan := m.currentBlock().Header.Time
	if lstSpan == 0 {
		log.Debug("getTimespan: current block's time is 0")
		return int64(m.blockInterval)
	}
	now := time.Now().UnixNano() / 1e6
	return now - int64(lstSpan)*1000
}

// isSelfDeputyNode 本节点是否为代理节点
func (m *Miner) isSelfDeputyNode() bool {
	return deputynode.Instance().IsSelfDeputyNode(m.currentBlock().Height() + 1)
}

// getSleepTime get sleep time to seal block
func (m *Miner) getSleepTime() int {
	if !m.isSelfDeputyNode() {
		log.Debugf("self not deputy node. mining forbidden")
		return -1
	}
	curHeight := m.currentBlock().Height()
	nodeCount := deputynode.Instance().GetDeputiesCount(curHeight + 1)
	if nodeCount == 1 { // 只有一个主节点
		waitTime := m.blockInterval
		log.Debugf("getSleepTime: waitTime:%d", waitTime)
		return int(waitTime)
	}
	if (curHeight > params.InterimDuration) && (curHeight-params.InterimDuration)%params.TermDuration == 0 {
		if rank := deputynode.Instance().GetNodeRankByNodeID(curHeight, deputynode.GetSelfNodeID()); rank > -1 {
			waitTime := rank * int(m.timeoutTime)
			log.Debugf("getSleepTime: waitTime:%d", waitTime)
			return waitTime
		}
		log.Error("not deputy node")
		return -1
	}
	timeDur := m.getTimespan() // 获取当前时间与最新块的时间差
	myself := deputynode.Instance().GetDeputyByNodeID(curHeight+1, deputynode.GetSelfNodeID())
	curHeader := m.currentBlock().Header
	slot := deputynode.Instance().GetSlot(curHeader.Height+1, curHeader.MinerAddress, myself.MinerAddress) // 获取新块离本节点索引的距离
	if slot == -1 {
		log.Debugf("slot = -1")
		return -1
	}
	oneLoopTime := int64(nodeCount) * m.timeoutTime
	// for test
	if curHeight%params.TermDuration >= params.InterimDuration && curHeight%params.TermDuration < params.InterimDuration+20 {
		log.Debugf("getSleepTime: timeDur:%d slot:%d oneLoopTime:%d", timeDur, slot, oneLoopTime)
	}
	if slot == 0 { // 上一个块为自己出的块
		minInterval := int64(nodeCount-1) * m.timeoutTime
		// timeDur = timeDur % oneLoopTime // 求余
		// if timeDur >=minInterval && timeDur< oneLoopTime{
		// 	log.Debugf("getSleepTime: timeDur: %d. isTurn=true --1", timeDur)
		// 	m.timeToMineCh <- struct{}{}
		// }else{
		// 	waitTime := minInterval - timeDur
		// 	m.resetMinerTimer(waitTime)
		// 	log.Debugf("getSleepTime: slot=0. waitTime:%d", waitTime)
		// }
		//

		if timeDur < minInterval {
			waitTime := minInterval - timeDur
			log.Debugf("getSleepTime: slot=0. waitTime:%d", waitTime)
			return int(waitTime)
		} else if timeDur < oneLoopTime {
			log.Debugf("getSleepTime: timeDur: %d. isTurn=true --1", timeDur)
			return 0
		} else { // 间隔大于一轮
			timeDur = timeDur % oneLoopTime // 求余
			waitTime := int64(nodeCount-1)*m.timeoutTime - timeDur
			if waitTime <= 0 {
				log.Debugf("getSleepTime: waitTime: %d. isTurn=true --2", waitTime)
				return 0
			} else {
				log.Debugf("getSleepTime: slot=0. waitTime:%d", waitTime)
				return int(waitTime)
			}
		}
	} else if slot == 1 { // 说明下一个区块就该本节点产生了
		if timeDur > oneLoopTime { // 间隔大于一轮
			timeDur = timeDur % oneLoopTime // 求余
			// for test
			if curHeight%params.TermDuration >= params.InterimDuration && curHeight%params.TermDuration < params.InterimDuration+20 {
				log.Debugf("getSleepTime: slot:1 timeDur:%d>oneLoopTime:%d ", timeDur, oneLoopTime)
			}
			if timeDur < m.timeoutTime { //
				log.Debugf("getSleepTime: timeDur: %d. isTurn=true --3", timeDur)
				return 0
			} else {
				waitTime := oneLoopTime - timeDur
				// for test
				if curHeight%params.TermDuration >= params.InterimDuration && curHeight%params.TermDuration < params.InterimDuration+20 {
					log.Debugf("ModifyTimer: slot:1 timeDur:%d>=self.timeoutTime:%d resetMinerTimer(waitTime:%d)", timeDur, m.timeoutTime, waitTime)
				}
				return int(waitTime)
			}
		} else { // 间隔不到一轮
			if timeDur >= m.timeoutTime { // 过了本节点该出块的时机
				waitTime := oneLoopTime - timeDur
				// for test
				if curHeight%params.TermDuration >= params.InterimDuration && curHeight%params.TermDuration < params.InterimDuration+20 {
					log.Debugf("getSleepTime: slot:1 timeDur<oneLoopTime, timeDur>self.timeoutTime, resetMinerTimer(waitTime:%d)", waitTime)
				}
				return int(waitTime)
			} else if timeDur >= m.blockInterval { // 如果上一个区块的时间与当前时间差大或等于3s（区块间的最小间隔为3s），则直接出块无需休眠
				// for test
				if curHeight%params.TermDuration >= params.InterimDuration && curHeight%params.TermDuration < params.InterimDuration+20 {
					log.Debugf("getSleepTime: timeDur: %d. isTurn=true. --4", timeDur)
				}
				return 0
			} else {
				waitTime := m.blockInterval - timeDur // 如果上一个块时间与当前时间非常近（小于3s），则设置休眠
				if waitTime <= 0 {
					log.Warnf("getSleepTime: waitTime: %d", waitTime)
					return -1
				}
				// for test
				if curHeight%params.TermDuration >= params.InterimDuration && curHeight%params.TermDuration < params.InterimDuration+20 {
					log.Debugf("getSleepTime: slot:1, else, resetMinerTimer(waitTime:%d)", waitTime)
				}
				return int(waitTime)
			}
		}
	} else { // 说明还不该自己出块，但是需要修改超时时间了
		timeDur = timeDur % oneLoopTime
		if timeDur >= int64(slot-1)*m.timeoutTime && timeDur < int64(slot)*m.timeoutTime {
			// for test
			if curHeight%params.TermDuration >= params.InterimDuration && curHeight%params.TermDuration < params.InterimDuration+20 {
				log.Debugf("getSleepTime: timeDur:%d. isTurn=true. --5", timeDur)
			}
			return 0
		} else {
			waitTime := (int64(slot-1)*m.timeoutTime - timeDur + oneLoopTime) % oneLoopTime
			if waitTime <= 0 {
				log.Warnf("getSleepTime: waitTime: %d", waitTime)
				return -1
			}
			// for test
			if curHeight%params.TermDuration >= params.InterimDuration && curHeight%params.TermDuration < params.InterimDuration+20 {
				log.Debug(fmt.Sprintf("getSleepTime: slot:>1, timeDur:%d, resetMinerTimer(waitTime:%d)", timeDur, waitTime))
			}
			return int(waitTime)
		}
	}
}

// resetMinerTimer reset timer
func (m *Miner) resetMinerTimer(timeDur int64) {
	// for debug
	if m.currentBlock().Height()%params.TermDuration >= params.InterimDuration && m.currentBlock().Height()%params.TermDuration < params.InterimDuration+20 {
		log.Debugf("resetMinerTimer: %d", timeDur)
	}
	// 停掉之前的定时器
	if m.blockMineTimer != nil {
		m.blockMineTimer.Stop()
	}
	if timeDur <= 0 {
		return
	}
	// 重开新的定时器
	m.blockMineTimer = time.AfterFunc(time.Duration(timeDur*int64(time.Millisecond)), func() {
		if atomic.LoadInt32(&m.mining) == 1 {
			log.Debug("resetMinerTimer: isTurn=true")
			m.timeToMineCh <- struct{}{}
		}
	})
}

// loopMiner
func (m *Miner) loopMiner() {
	defer log.Debug("stop miner's loop")
	for {
		select {
		case <-m.timeToMineCh:
			m.sealBlock()
		case block := <-m.recvNewBlockCh:
			log.Infof("Receive new block. height: %d. hash: %s. Reset timer.", block.Height(), block.Hash().Hex())
			if !m.isSelfDeputyNode() {
				m.resetMinerTimer(-1)
				break
			}
			// update miner address
			m.updateMiner(block)
			// latest block is self mined
			if block.MinerAddress() == m.minerAddress {
				var timeDur int64
				// snapshot block + InterimDuration 换届最后一个区块
				if block.Height() > params.InterimDuration && block.Height()%params.TermDuration == params.InterimDuration {
					rank := deputynode.Instance().GetNodeRankByAddress(block.Height()+1, m.minerAddress)
					if rank == 0 {
						timeDur = m.blockInterval
					} else {
						timeDur = int64(rank) * m.timeoutTime
					}
				} else {
					nodeCount := deputynode.Instance().GetDeputiesCount(block.Height() + 1)
					if nodeCount == 1 {
						timeDur = m.blockInterval
					} else {
						timeDur = int64(nodeCount-1) * m.timeoutTime
					}
				}
				m.resetMinerTimer(timeDur)
			} else {
				var timeDur int64
				// snapshot block + InterimDuration
				if block.Height() > params.InterimDuration && block.Height()%params.TermDuration == params.InterimDuration {
					rank := deputynode.Instance().GetNodeRankByAddress(block.Height()+1, m.minerAddress)
					if rank < 0 {
						log.Error("self not deputy node in this term")
					} else if rank == 0 {
						timeDur = m.blockInterval
					} else {
						timeDur = int64(rank) * m.timeoutTime
					}
					m.resetMinerTimer(timeDur)
				} else {
					timeDur = int64(m.getSleepTime())
					if timeDur == 0 {
						log.Debug("time to mine direct")
						m.sealBlock()
					} else if timeDur > 0 {
						m.resetMinerTimer(timeDur)
					} else {
						log.Error("getSleepTime internal error.")
					}
				}
			}
		case <-m.stopCh:
			return
		case <-m.quitCh:
			return
		}
	}
}

// updateMiner update next term's miner address
func (m *Miner) updateMiner(block *types.Block) {
	n := deputynode.Instance().GetDeputyByNodeID(block.Height()+1, deputynode.GetSelfNodeID())
	if n != nil {
		m.SetMinerAddress(n.MinerAddress)
		log.Debugf("set next term miner address: %s", n.MinerAddress.String())
	}
}

// sealBlock 出块
func (m *Miner) sealBlock() {
	if !m.isSelfDeputyNode() {
		return
	}
	log.Debug("Start seal")
	header, dNodes := m.sealHead()
	txs := m.txPool.Pending(1000000)

	defer func() {
		var timeDur int64
		// snapshot block
		if header.Height > params.InterimDuration && header.Height%params.TermDuration == params.InterimDuration {
			rank := deputynode.Instance().GetNodeRankByAddress(header.Height+1, m.minerAddress)
			if rank == 0 {
				timeDur = m.blockInterval
			}
		} else {
			nodeCount := deputynode.Instance().GetDeputiesCount(header.Height + 1)
			if nodeCount == 1 {
				timeDur = m.blockInterval
			} else {
				timeDur = int64(nodeCount-1) * m.timeoutTime
			}
		}
		m.resetMinerTimer(timeDur)
	}()

	m.chain.Lock().Lock()
	defer m.chain.Lock().Unlock()
	// apply transactions
	packagedTxs, invalidTxs, gasUsed := m.txProcessor.ApplyTxs(header, txs)
	log.Debug("ApplyTxs ok")
	// seal block
	block, err := m.engine.Seal(header, packagedTxs, gasUsed, m.chain.AccountManager(), dNodes)
	if err != nil {
		log.Errorf("Seal block error! %v", err)
		return
	}
	if err = m.signBlock(block); err != nil {
		log.Errorf("Sign for block failed! block hash:%s", block.Hash().Hex())
		return
	}

	if err = m.chain.SetMinedBlock(block); err != nil {
		log.Error("Set mined block failed!")
		return
	}
	// remove txs from pool
	txsKeys := make([]common.Hash, len(packagedTxs)+len(invalidTxs))
	for i, tx := range packagedTxs {
		txsKeys[i] = tx.Hash()
	}
	for i, tx := range invalidTxs {
		txsKeys[i+len(packagedTxs)] = tx.Hash()
	}
	m.txPool.Remove(txsKeys)
	log.Infof("Mine a new block. height: %d hash: %s, len(txs): %d", block.Height(), block.Hash().String(), len(block.Txs))
}

// signBlock signed the block and fill in header
func (m *Miner) signBlock(block *types.Block) (err error) {
	hash := block.Hash()
	signData, err := crypto.Sign(hash[:], m.privKey)
	if err == nil {
		block.Header.SignData = signData
	}
	return
}

// sealHead 生成区块头
func (m *Miner) sealHead() (*types.Header, deputynode.DeputyNodes) {
	// check is need to change minerAddress
	parent := m.currentBlock()
	height := parent.Height() + 1
	h := &types.Header{
		ParentHash:   parent.Hash(),
		MinerAddress: m.minerAddress,
		Height:       height,
		GasLimit:     calcGasLimit(parent),
		Extra:        m.extra,
	}
	var nodes deputynode.DeputyNodes = nil
	if height%params.TermDuration == 0 {
		nodes = m.chain.GetNewDeputyNodes()
		root := types.DeriveDeputyRootSha(nodes)
		h.DeputyRoot = root[:]
		deputynode.Instance().Add(height+params.InterimDuration+1, nodes)
		log.Debugf("add new term deputy nodes: %s", nodes.String())
	} else if height%params.TermDuration == params.InterimDuration {
		n := deputynode.Instance().GetDeputyByNodeID(height, deputynode.GetSelfNodeID())
		m.SetMinerAddress(n.MinerAddress)
		log.Debugf("set next term's miner address: %s", n.MinerAddress.String())
	}

	// allowable 1 second time error
	// but next block's time can't be small than parent block
	parTime := parent.Time()
	blockTime := uint32(time.Now().Unix())
	if parTime > blockTime {
		blockTime = parTime
	}
	h.Time = blockTime
	return h, nodes
}

// calcGasLimit computes the gas limit of the next block after parent.
// This is miner strategy, not consensus protocol.
func calcGasLimit(parent *types.Block) uint64 {
	// contrib = (parentGasUsed * 3 / 2) / 1024
	contrib := (parent.GasUsed() + parent.GasUsed()/2) / params.GasLimitBoundDivisor

	// decay = parentGasLimit / 1024 -1
	decay := parent.GasLimit()/params.GasLimitBoundDivisor - 1

	/*
		strategy: gasLimit of block-to-mine is set based on parent's
		gasUsed value.  if parentGasUsed > parentGasLimit * (2/3) then we
		increase it, otherwise lower it (or leave it unchanged if it's right
		at that usage) the amount increased/decreased depends on how far away
		from parentGasLimit * (2/3) parentGasUsed is.
	*/
	limit := parent.GasLimit() - decay + contrib
	if limit < params.MinGasLimit {
		limit = params.MinGasLimit
	}
	// however, if we're now below the target (TargetGasLimit) we increase the
	// limit as much as we can (parentGasLimit / 1024 -1)
	if limit < params.TargetGasLimit {
		limit = parent.GasLimit() + decay
		if limit > params.TargetGasLimit {
			limit = params.TargetGasLimit
		}
	}
	return limit
}
