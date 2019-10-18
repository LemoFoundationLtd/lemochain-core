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
	SleepTime               int64 // 区块最小间隔时间
	Timeout                 int64 // 区块最大间隔时间
	ReservedPropagationTime int64 // 用于传播区块的最小预留时间
}

type Chain interface {
	CurrentBlock() *types.Block
	MineBlock(int64)
}

type TxPool interface {
	ExistCanPackageTx(time uint32) bool
}

type MineInfo struct {
	endOfMineWindow int64
}

// Miner 负责出块调度算法，决定什么时候该出块。本身并不负责区块封装的逻辑
type Miner struct {
	blockInterval           int64 // millisecond
	timeoutTime             int64 // millisecond
	reservedPropagationTime int64 // 打包区块之后预留给传播区块的最小时间。单位：millisecond
	mining                  int32
	chain                   Chain
	dm                      *deputynode.Manager
	txPool                  TxPool
	mineTimer               *time.Timer // 出块timer
	retryTimer              *time.Timer // 出块失败时重试出块的timer

	recvNewBlockCh chan *types.Block // 收到新块通知
	timeToMineCh   chan *MineInfo    // 到出块时间了
	stopCh         chan struct{}     // 停止挖矿
	quitCh         chan struct{}     // 退出
}

func New(cfg MineConfig, chain Chain, dm *deputynode.Manager, txPool TxPool) *Miner {
	return &Miner{
		blockInterval:           cfg.SleepTime,
		timeoutTime:             cfg.Timeout,
		reservedPropagationTime: cfg.ReservedPropagationTime,
		chain:                   chain,
		dm:                      dm,
		txPool:                  txPool,
		recvNewBlockCh:          make(chan *types.Block, 1),
		timeToMineCh:            make(chan *MineInfo),
		stopCh:                  make(chan struct{}),
		quitCh:                  make(chan struct{}),
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

	subscribe.Sub(subscribe.NewCurrentBlock, m.recvNewBlockCh)
	log.Info("Start mining success")
}

func (m *Miner) Stop() {
	if !atomic.CompareAndSwapInt32(&m.mining, 1, 0) {
		log.Warn("Stopped mining already")
		return
	}
	m.stopMineTimer()
	m.stopCh <- struct{}{}
	subscribe.UnSub(subscribe.NewCurrentBlock, m.recvNewBlockCh)
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
func (m *Miner) resetMineTimer(timeDur, endOfMineWindow int64) {
	// 停掉之前的定时器
	m.stopMineTimer()

	// 重开新的定时器
	m.mineTimer = time.AfterFunc(time.Duration(timeDur*int64(time.Millisecond)), func() {
		if atomic.LoadInt32(&m.mining) == 1 {
			// MineBlock may fail. Then the new block event won't come. So we'd better set a new timer in advance to make sure the mine loop will continue
			// If mine success, the timer will be clear
			m.retryTimer = time.AfterFunc(time.Duration(m.timeoutTime*int64(time.Millisecond)), func() {
				// mine the same height block again in next mine loop
				log.Debug("Last mine failed. Try again")
				m.schedule(m.chain.CurrentBlock())
			})
			log.Debug("Time to mine")
			m.timeToMineCh <- &MineInfo{endOfMineWindow}
		}
	})
}

// runMineLoop
func (m *Miner) runMineLoop() {
	defer log.Debug("Stop mine loop")
	for {
		select {
		case mineCh := <-m.timeToMineCh:
			m.sealBlock(mineCh.endOfMineWindow)

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

// getSleepTime get sleep time (millisecond) to seal block, return waitTime and absolute time of block timeout
func (m *Miner) getSleepTime(mineHeight uint32, distance uint64, parentTime int64, currentTime int64) (int64, int64) {
	nodeCount := m.dm.GetDeputiesCount(mineHeight)
	// 所有节点都超时所需要消耗的时间，也可以看作是下一轮出块的开始时间
	oneLoopTime := int64(nodeCount) * m.timeoutTime
	// 网络传输耗时，即当前时间减去收到的区块头中的时间戳
	totalPassTime := currentTime - parentTime
	// 机器时间不同步导致收到了未来的区块。需要额外的对齐时间
	offset := int64(0)
	if totalPassTime < 0 {
		offset = -totalPassTime
		totalPassTime = 0
	}
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
	log.Debug("getSleepTime", "waitTime", waitTime, "distance", distance, "parentTime", parentTime, "totalPassTime", totalPassTime, "passTime", passTime, "offset", offset, "nodeCount", nodeCount, "blockInterval", m.blockInterval, "timeoutTime", m.timeoutTime, "windowFrom", windowFrom, "windowTo", windowTo)
	return offset + waitTime, parentTime + windowTo
}

// schedule wait some time to mine next block
func (m *Miner) schedule(parentBlock *types.Block) bool {
	mineHeight := parentBlock.Height() + 1

	minerAddress, ok := m.dm.GetMyMinerAddress(mineHeight)
	if !ok {
		log.Warnf("Not a deputy at height %d. can't mine", mineHeight)
		return false
	}
	// 获取新块离本节点索引的距离，永远在(0,DeputyCount]区间中
	distance, err := consensus.GetMinerDistance(mineHeight, parentBlock.MinerAddress(), minerAddress, m.dm)
	if err != nil {
		log.Errorf("GetMinerDistance error: %v", err)
		return false
	}

	// wait if the time from last miner is bigger with mine
	parentTime := int64(parentBlock.Time()) * 1000
	now := time.Now().UnixNano() / 1e6
	timeDur, endOfMineWindow := m.getSleepTime(mineHeight, distance, parentTime, now)
	m.resetMineTimer(timeDur, endOfMineWindow)
	return true
}

// sealBlock 出块
func (m *Miner) sealBlock(endOfMineWindow int64) {
	if !m.isSelfDeputyNode() {
		return
	}
	log.Debug("Start seal block")
	endOfWaitWindow := endOfMineWindow - m.reservedPropagationTime // 允许矿工等待的超时时间
	m.waitCanPackageTx(endOfWaitWindow)
	// mine asynchronously
	// The time limit for mining is (m.timeoutTime - m.blockInterval). The rest 1/3 is used to transfer to other nodes
	nowTimestamp := time.Now().UnixNano() / 1e6        // 当前时间戳 单位为毫秒
	txProcessTimeout := endOfWaitWindow - nowTimestamp // 允许矿工使用执行交易的最大时间
	if txProcessTimeout < 0 {
		txProcessTimeout = 0
	}
	m.chain.MineBlock(txProcessTimeout)
}

// waitCanPackageTx 等待交易池中存在可以打包的交易
func (m *Miner) waitCanPackageTx(endOfWaitWindow int64) {
	// 当交易池中没有交易的时候，每隔一秒钟轮询一次，直到get到交易或者即将超过规定的出块时间之后退出
	for {
		now := time.Now().UnixNano() / 1e6 // 当前时间戳单位为毫秒
		// 如果当前时间已经超过了允许挖矿到的最大时间戳则退出
		if now >= endOfWaitWindow {
			break
		}
		if m.txPool.ExistCanPackageTx(uint32(now / 1e3)) {
			break
		} else {
			// 休眠500毫秒
			time.Sleep(500 * time.Millisecond)
		}
	}
}
