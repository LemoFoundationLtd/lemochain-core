package miner

import (
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
	mining        int32
	chain         Chain
	dm            *deputynode.Manager
	extra         []byte // 扩展数据 暂保留 最大256byte

	mineTimer *time.Timer // 出块timer

	recvNewBlockCh chan *types.Block // 收到新块通知
	recvBlockSub   subscribe.Subscription
	timeToMineCh   chan struct{} // 到出块时间了
	stopCh         chan struct{} // 停止挖矿
	quitCh         chan struct{} // 退出
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
		stopCh:         make(chan struct{}),
		quitCh:         make(chan struct{}),
	}
}

func (m *Miner) Start() {
	if !atomic.CompareAndSwapInt32(&m.mining, 0, 1) {
		log.Warn("Started mining already")
		return
	}
	select {
	case <-m.timeToMineCh:
	default:
	}

	// Start loop even if we are not deputy node, so that we can start mine immediately when we become a deputy node
	go m.runMineLoop()

	// Active the mine timer. To make sure the first miner can start work
	if m.isSelfDeputyNode() {
		currentBlock := m.chain.CurrentBlock()
		m.schedule(currentBlock.Height()+1, currentBlock.MinerAddress())
		// if waitTime < 0 {
		// 	log.Error("interval error. start mining failed")
		// 	m.Stop()
		// 	return
		// }
	}

	m.recvBlockSub = m.chain.SubscribeNewBlock(m.recvNewBlockCh)
	log.Info("Start mining success")
}

func (m *Miner) Stop() {
	if !atomic.CompareAndSwapInt32(&m.mining, 1, 0) {
		log.Warn("Stopped mining already")
		return
	}
	m.stopMineTimer()
	m.stopCh <- struct{}{}
	if m.recvBlockSub != nil {
		m.recvBlockSub.Unsubscribe()
	}
	log.Info("Stop mining success")
}

func (m *Miner) Close() {
	close(m.quitCh)
}

func (m *Miner) IsMining() bool {
	if !m.isSelfDeputyNode() {
		return false
	}
	return atomic.LoadInt32(&m.mining) == 1
}

func (m *Miner) GetMinerAddress() common.Address {
	// Get self deputy info in the term which next block in
	minerAddress, _ := m.dm.GetMyMinerAddress(m.chain.CurrentBlock().Height() + 1)
	return minerAddress
}

// 获取最新区块的时间戳离当前时间的距离 单位：ms
func (m *Miner) getTimespan() int64 {
	lastTime := m.chain.CurrentBlock().Header.Time
	now := time.Now().UnixNano() / 1e6
	return now - int64(lastTime)*1000
}

// isSelfDeputyNode 本节点是否为代理节点
func (m *Miner) isSelfDeputyNode() bool {
	return m.dm.IsSelfDeputyNode(m.chain.CurrentBlock().Height() + 1)
}

// getSleepTime get sleep time to seal block
func (m *Miner) getSleepTime() int {
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
		// 	m.resetMineTimer(waitTime)
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
					log.Debugf("ModifyTimer: slot:1 timeDur:%d>=self.timeoutTime:%d resetMineTimer(waitTime:%d)", timeDur, m.timeoutTime, waitTime)
				}
				return int(waitTime)
			}
		} else { // 间隔不到一轮
			if timeDur >= m.timeoutTime { // 过了本节点该出块的时机
				waitTime := oneLoopTime - timeDur
				// for test
				if curHeight%params.TermDuration >= params.InterimDuration && curHeight%params.TermDuration < params.InterimDuration+20 {
					log.Debugf("getSleepTime: slot:1 timeDur<oneLoopTime, timeDur>self.timeoutTime, resetMineTimer(waitTime:%d)", waitTime)
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
					log.Debugf("getSleepTime: slot:1, else, resetMineTimer(waitTime:%d)", waitTime)
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
				log.Debugf("getSleepTime: slot:>1, timeDur:%d, resetMineTimer(waitTime:%d)", timeDur, waitTime)
			}
			return int(waitTime)
		}
	}
}

// stopMineTimer stop timer
func (m *Miner) stopMineTimer() {
	if m.mineTimer != nil {
		m.mineTimer.Stop()
	}
}

// resetMineTimer reset timer
func (m *Miner) resetMineTimer(timeDur int64) {
	// 停掉之前的定时器
	m.stopMineTimer()
	if timeDur <= 0 {
		return
	}
	// 重开新的定时器
	m.mineTimer = time.AfterFunc(time.Duration(timeDur*int64(time.Millisecond)), func() {
		if atomic.LoadInt32(&m.mining) == 1 {
			log.Debugf("wait %dms to mine", timeDur)
			m.timeToMineCh <- struct{}{}
		}
	})
}

// runMineLoop
func (m *Miner) runMineLoop() {
	defer log.Debug("Stop mine loop")
	for {
		select {
		case <-m.timeToMineCh:
			m.sealBlock()

		case block := <-m.recvNewBlockCh:
			// include mine block by self and receive other's block
			log.Debug("Got a new block. Reset timer.", "block", block.ShortString())
			m.schedule(block.Height()+1, block.MinerAddress())

		case <-m.stopCh:
			return

		case <-m.quitCh:
			return
		}
	}
}

// schedule reset timer or mine block when a new block come
func (m *Miner) schedule(nextHeight uint32, currentMiner common.Address) {
	myDeputyNode := m.dm.GetMyDeputyInfo(nextHeight)
	if myDeputyNode == nil {
		m.stopMineTimer()
		return
	}

	var timeDur int64
	// block 是换届的最后一个区块
	if deputynode.IsRewardBlock(nextHeight) {
		if myDeputyNode.Rank == 0 {
			log.Debug("Start a new term, I'm the first miner")
			timeDur = m.blockInterval
		} else {
			timeDur = int64(myDeputyNode.Rank) * m.timeoutTime
		}
		m.resetMineTimer(timeDur)
		return
	}

	// latest block is self mined
	if currentMiner == myDeputyNode.MinerAddress {
		log.Debug("Reset timer by the block mined by self")
		nodeCount := m.dm.GetDeputiesCount(nextHeight)
		if nodeCount == 1 {
			timeDur = m.blockInterval
		} else {
			timeDur = int64(nodeCount-1) * m.timeoutTime
		}
		m.resetMineTimer(timeDur)
		return
	}

	// sleep
	timeDur = int64(m.getSleepTime())
	if timeDur == 0 {
		log.Debug("time to mine immediately")
		m.sealBlock()
	} else if timeDur > 0 {
		m.resetMineTimer(timeDur)
	} else {
		log.Error("getSleepTime interval error.")
	}
}

// sealBlock 出块
func (m *Miner) sealBlock() {
	if !m.isSelfDeputyNode() {
		return
	}
	parent := m.chain.CurrentBlock()
	log.Debugf("Start seal block %d", parent.Height()+1)

	// mine asynchronously
	m.chain.MineBlock(&consensus.BlockMaterial{
		Extra:         m.extra,
		MineTimeLimit: m.timeoutTime * 2 / 3,
	})

	// MineBlock may fail. Then the new block event won't come. So we'd better set timer in advance
	m.schedule(parent.Height()+1, m.GetMinerAddress())
}
