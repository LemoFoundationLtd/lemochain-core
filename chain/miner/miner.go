package miner

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/consensus"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
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

	mineTimer  *time.Timer // 出块timer
	retryTimer *time.Timer // 出块失败时重试出块的timer

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
		if !m.schedule(m.chain.CurrentBlock()) {
			log.Error("Start mining fail")
			m.Stop()
			return
		}
	} else {
		log.Info("Not deputy now. waiting...")
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

// stopMineTimer stop timer
func (m *Miner) stopMineTimer() {
	if m.mineTimer != nil {
		m.mineTimer.Stop()
	}
	if m.retryTimer != nil {
		m.retryTimer.Stop()
	}
}

// resetMineTimer reset timer
func (m *Miner) resetMineTimer(timeDur int64) {
	// 停掉之前的定时器
	m.stopMineTimer()

	// 重开新的定时器
	log.Debugf("Wait %dms to mine", timeDur)
	m.mineTimer = time.AfterFunc(time.Duration(timeDur*int64(time.Millisecond)), func() {
		if atomic.LoadInt32(&m.mining) == 1 {
			// MineBlock may fail. Then the new block event won't come. So we'd better set a new timer in advance to make sure the mine loop will continue
			// If mine success, the timer will be clear
			m.retryTimer = time.AfterFunc(time.Duration(m.timeoutTime*int64(time.Millisecond)), func() {
				// mine the same height block again in next mine loop
				log.Debug("Last mine failed. Try again")
				m.schedule(m.chain.CurrentBlock())
			})

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
			m.stopMineTimer()
			m.schedule(block)

		case <-m.stopCh:
			return

		case <-m.quitCh:
			return
		}
	}
}

// getSleepTime get sleep time to seal block
func (m *Miner) getSleepTime(mineHeight uint32, distance uint64, parentBlockTime uint32) int64 {
	nodeCount := m.dm.GetDeputiesCount(mineHeight)
	// 所有节点都超时所需要消耗的时间，也可以看作是下一轮出块的开始时间
	oneLoopTime := int64(nodeCount) * m.timeoutTime
	// 网络传输耗时，即当前时间减去收到的区块头中的时间戳
	totalPassTime := (time.Now().UnixNano() / 1e6) - int64(parentBlockTime)*1000
	// 本轮出块时间表已经经过的时长
	passTime := totalPassTime % oneLoopTime
	// 可以出块的时间窗口
	windowFrom := int64(distance-1) * m.timeoutTime
	windowTo := int64(distance) * m.timeoutTime

	var waitTime int64
	if distance == 1 && totalPassTime < m.timeoutTime {
		// distance == 1表示下一个区块该本节点产生了。时间也合适，没有超时。这时需要确保延迟足够的时间，避免早期交易少时链上全是空块
		waitTime = m.blockInterval - passTime
		if waitTime < 0 {
			waitTime = 0
		}
	} else if passTime >= windowFrom && passTime < windowTo {
		// 到达当前节点的时间窗口内了，可以立即出块
		waitTime = 0
	} else {
		// 需要等待下个时间窗口
		waitTime = (windowFrom - passTime + oneLoopTime) % oneLoopTime
	}
	log.Debug("getSleepTime", "waitTime", waitTime, "distance", distance, "parentTime", parentBlockTime, "totalPassTime", totalPassTime, "passTime", passTime)
	log.Debug("getSleepTime", "nodeCount", nodeCount, "blockInterval", m.blockInterval, "timeoutTime", m.timeoutTime, "windowFrom", windowFrom, "windowTo", windowTo)
	return waitTime
}

// schedule wait some time to mine next block
func (m *Miner) schedule(parentBlock *types.Block) bool {
	mineHeight := parentBlock.Height() + 1

	minerAddress, ok := m.dm.GetMyMinerAddress(mineHeight)
	if !ok {
		log.Warnf("Not a deputy at height %d. can't mine", mineHeight)
		return false
	}
	// 获取新块离本节点索引的距离
	distance, err := consensus.GetMinerDistance(mineHeight, parentBlock.MinerAddress(), minerAddress, m.dm)
	if err != nil {
		log.Errorf("GetMinerDistance error: %v", err)
		return false
	}

	timeDur := m.getSleepTime(mineHeight, distance, parentBlock.Time())
	m.resetMineTimer(timeDur)
	return true
}

// sealBlock 出块
func (m *Miner) sealBlock() {
	if !m.isSelfDeputyNode() {
		return
	}
	log.Debug("Start seal block")

	// mine asynchronously
	m.chain.MineBlock(&consensus.BlockMaterial{
		Extra: m.extra,
		// The time for mining is (m.timeoutTime - m.blockInterval). The rest 1/3 is used to transfer to other nodes
		MineTimeLimit: (m.timeoutTime - m.blockInterval) * 2 / 3,
	})
}
