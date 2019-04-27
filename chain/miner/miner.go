package miner

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/consensus"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/subscribe"
	"sync/atomic"
	"time"
)

type MineConfig struct {
	SleepTime int64
	Timeout   int64
}

type Chain interface {
	CurrentBlock() *types.Block
	SubscribeNewBlock(ch chan *types.Block) subscribe.Subscription
	MineBlock(*consensus.BlockMaterial)
}

type Miner struct {
	blockInterval int64
	timeoutTime   int64
	minerAddress  common.Address
	mining        int32
	chain         Chain
	dm            *deputynode.Manager
	extra         []byte // 扩展数据 暂保留 最大256byte

	blockMineTimer *time.Timer // 出块timer

	recvNewBlockCh chan *types.Block // 收到新块通知
	recvBlockSub   subscribe.Subscription
	timeToMineCh   chan struct{} // 到出块时间了
	// startCh        chan struct{}
	stopCh chan struct{} // 停止挖矿
	quitCh chan struct{} // 退出
}

func New(cfg MineConfig, chain Chain, dm *deputynode.Manager, extra []byte) *Miner {
	return &Miner{
		blockInterval:  cfg.SleepTime,
		timeoutTime:    cfg.Timeout,
		chain:          chain,
		dm:             dm,
		extra:          extra,
		recvNewBlockCh: make(chan *types.Block, 1),
		timeToMineCh:   make(chan struct{}),
		// startCh:        make(chan struct{}),
		stopCh: make(chan struct{}),
		quitCh: make(chan struct{}),
	}
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
			log.Error("interval error. start mining failed")
			atomic.CompareAndSwapInt32(&m.mining, 1, 0)
			m.stopCh <- struct{}{}
			return
		}
	} else {
		m.resetMinerTimer(-1)
	}
	m.recvBlockSub = m.chain.SubscribeNewBlock(m.recvNewBlockCh)
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

func (m *Miner) GetMinerAddress() common.Address {
	m.updateMiner(m.chain.CurrentBlock())
	return m.minerAddress
}

// 获取最新区块的时间戳离当前时间的距离 单位：ms
func (m *Miner) getTimespan() int64 {
	lstSpan := m.chain.CurrentBlock().Header.Time
	if lstSpan == 0 {
		log.Debug("getTimespan: current block's time is 0")
		return int64(m.blockInterval)
	}
	now := time.Now().UnixNano() / 1e6
	return now - int64(lstSpan)*1000
}

// isSelfDeputyNode 本节点是否为代理节点
func (m *Miner) isSelfDeputyNode() bool {
	return m.dm.IsSelfDeputyNode(m.chain.CurrentBlock().Height() + 1)
}

// getSleepTime get sleep time to seal block
func (m *Miner) getSleepTime() int {
	if !m.isSelfDeputyNode() {
		log.Debugf("self not deputy node. mining forbidden")
		return -1
	}
	curBlock := m.chain.CurrentBlock()
	curHeight := curBlock.Height()
	nodeCount := m.dm.GetDeputiesCount(curHeight + 1)
	if nodeCount == 1 { // 只有一个主节点
		waitTime := m.blockInterval
		log.Debugf("getSleepTime: waitTime:%d", waitTime)
		return int(waitTime)
	}
	if (curHeight > params.InterimDuration) && (curHeight-params.InterimDuration)%params.TermDuration == 0 {
		if deputyNode := m.dm.GetDeputyByNodeID(curHeight, deputynode.GetSelfNodeID()); deputyNode != nil {
			waitTime := int(deputyNode.Rank) * int(m.timeoutTime)
			log.Debugf("getSleepTime: waitTime:%d", waitTime)
			return waitTime
		}
		log.Error("not deputy node")
		return -1
	}
	timeDur := m.getTimespan() // 获取当前时间与最新块的时间差
	myself := m.dm.GetDeputyByNodeID(curHeight+1, deputynode.GetSelfNodeID())
	slot, err := consensus.GetMinerDistance(curHeight+1, curBlock.Header.MinerAddress, myself.MinerAddress, m.dm) // 获取新块离本节点索引的距离
	if err != nil {
		log.Debugf("GetMinerDistance error: %v", err)
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
					deputyNode := m.dm.GetDeputyByAddress(block.Height()+1, m.minerAddress)
					// TODO if deputyNode == nil
					if deputyNode.Rank == 0 {
						timeDur = m.blockInterval
					} else {
						timeDur = int64(deputyNode.Rank) * m.timeoutTime
					}
				} else {
					nodeCount := m.dm.GetDeputiesCount(block.Height() + 1)
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
					deputyNode := m.dm.GetDeputyByAddress(block.Height()+1, m.minerAddress)
					if deputyNode == nil {
						log.Error("self not deputy node in this term")
					} else if deputyNode.Rank == 0 {
						timeDur = m.blockInterval
					} else {
						timeDur = int64(deputyNode.Rank) * m.timeoutTime
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
						log.Error("getSleepTime interval error.")
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
	// Get self deputy info in the term which next block in
	deputyNode := m.dm.GetMyDeputyInfo(block.Height() + 1)
	// A node can not set minerAddress until it becomes deputy node. And deputy node's minerAddress may changes between terms.
	if deputyNode != nil {
		oldMiner := m.minerAddress
		m.minerAddress = deputyNode.MinerAddress
		if oldMiner != deputyNode.MinerAddress {
			log.Info("update miner", "from", oldMiner.String(), "addr", deputyNode.MinerAddress.String())
		}
	}
}

// sealBlock 出块
func (m *Miner) sealBlock() {
	if !m.isSelfDeputyNode() {
		return
	}
	log.Debug("Start seal")

	parent := m.chain.CurrentBlock()
	// mine asynchronously
	m.chain.MineBlock(&consensus.BlockMaterial{
		MinerAddr:     m.minerAddress,
		Extra:         m.extra,
		MineTimeLimit: m.timeoutTime * 2 / 3,
	})

	m.resetTimerAfterMine(parent.Height() + 1)
}

// sealBlock 出块
func (m *Miner) resetTimerAfterMine(minedHeight uint32) {
	var timeDur int64
	// snapshot block
	if minedHeight > params.InterimDuration && minedHeight%params.TermDuration == params.InterimDuration {
		deputyNode := m.dm.GetDeputyByAddress(minedHeight+1, m.minerAddress)
		// TODO if deputyNode == nil
		if deputyNode.Rank == 0 {
			timeDur = m.blockInterval
		}
	} else {
		nodeCount := m.dm.GetDeputiesCount(minedHeight + 1)
		if nodeCount == 1 {
			timeDur = m.blockInterval
		} else {
			timeDur = int64(nodeCount-1) * m.timeoutTime
		}
	}
	m.resetMinerTimer(timeDur)
}
