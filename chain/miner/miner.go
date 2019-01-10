package miner

import (
	"crypto/ecdsa"
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
	startCh        chan struct{}
	stopCh         chan struct{} // 停止挖矿
	quitCh         chan struct{} // 退出
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
		startCh:        make(chan struct{}),
		stopCh:         make(chan struct{}),
		quitCh:         make(chan struct{}),
	}
	go m.loopRecvBlock()
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
	m.startCh <- struct{}{}
	go m.loopMiner()
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
	nodeCount := deputynode.Instance().GetDeputiesCount()
	if nodeCount == 1 { // 只有一个主节点
		waitTime := m.blockInterval
		log.Debugf("getSleepTime: waitTime:%d", waitTime)
		return int(waitTime)
	}
	timeDur := m.getTimespan() // 获取当前时间与最新块的时间差
	myself := deputynode.Instance().GetDeputyByNodeID(m.currentBlock().Height(), deputynode.GetSelfNodeID())
	curHeader := m.currentBlock().Header
	slot := deputynode.Instance().GetSlot(curHeader.Height+1, curHeader.MinerAddress, myself.MinerAddress) // 获取新块离本节点索引的距离
	if slot == -1 {
		log.Debugf("slot = -1")
		return -1
	}
	oneLoopTime := int64(nodeCount) * m.timeoutTime
	// log.Debugf("getSleepTime: timeDur:%d slot:%d oneLoopTime:%d", timeDur, slot, oneLoopTime)
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
			// log.Debugf("getSleepTime: slot:1 timeDur:%d>oneLoopTime:%d ", timeDur, oneLoopTime)
			if timeDur < m.timeoutTime { //
				log.Debugf("getSleepTime: timeDur: %d. isTurn=true --3", timeDur)
				return 0
			} else {
				waitTime := oneLoopTime - timeDur
				// log.Debugf("ModifyTimer: slot:1 timeDur:%d>=self.timeoutTime:%d resetMinerTimer(waitTime:%d)", timeDur, m.timeoutTime, waitTime)
				return int(waitTime)
			}
		} else { // 间隔不到一轮
			if timeDur >= m.timeoutTime { // 过了本节点该出块的时机
				waitTime := oneLoopTime - timeDur
				// log.Debugf("getSleepTime: slot:1 timeDur<oneLoopTime, timeDur>self.timeoutTime, resetMinerTimer(waitTime:%d)", waitTime)
				return int(waitTime)
			} else if timeDur >= m.blockInterval { // 如果上一个区块的时间与当前时间差大或等于3s（区块间的最小间隔为3s），则直接出块无需休眠
				// log.Debugf("getSleepTime: timeDur: %d. isTurn=true. --4", timeDur)
				return 0
			} else {
				waitTime := m.blockInterval - timeDur // 如果上一个块时间与当前时间非常近（小于3s），则设置休眠
				if waitTime <= 0 {
					log.Warnf("getSleepTime: waitTime: %d", waitTime)
					return -1
				}
				// log.Debugf("getSleepTime: slot:1, else, resetMinerTimer(waitTime:%d)", waitTime)
				return int(waitTime)
			}
		}
	} else { // 说明还不该自己出块，但是需要修改超时时间了
		timeDur = timeDur % oneLoopTime
		if timeDur >= int64(slot-1)*m.timeoutTime && timeDur < int64(slot)*m.timeoutTime {
			// log.Debugf("getSleepTime: timeDur:%d. isTurn=true. --5", timeDur)
			return 0
		} else {
			waitTime := (int64(slot-1)*m.timeoutTime - timeDur + oneLoopTime) % oneLoopTime
			if waitTime <= 0 {
				log.Warnf("getSleepTime: waitTime: %d", waitTime)
				return -1
			}
			// log.Debug(fmt.Sprintf("getSleepTime: slot:>1, timeDur:%d, resetMinerTimer(waitTime:%d)", timeDur, waitTime))
			return int(waitTime)
		}
	}
}

// 重置出块定时器
func (m *Miner) resetMinerTimer(timeDur int64) {
	// 停掉之前的定时器
	if m.blockMineTimer != nil {
		m.blockMineTimer.Stop()
	}
	// 重开新的定时器
	m.blockMineTimer = time.AfterFunc(time.Duration(timeDur*int64(time.Millisecond)), func() {
		if atomic.LoadInt32(&m.mining) == 1 {
			log.Debug("resetMinerTimer: isTurn=true")
			m.timeToMineCh <- struct{}{}
		}
	})
}

// loopRecvBlock drop no use block
func (m *Miner) loopRecvBlock() {
	for {
		if atomic.LoadInt32(&m.mining) == 0 {
			select {
			case <-m.recvNewBlockCh:
				log.Debugf("Receive new block. but not start mining")
			case <-m.quitCh:
				return
			case <-m.startCh:
			}
		} else {
			time.Sleep(200 * time.Millisecond)
		}
	}
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
			waitTime := m.getSleepTime()
			if waitTime == 0 {
				log.Debug("time to mine direct")
				m.resetMinerTimer(int64(10 * time.Second))
				m.sealBlock()
			} else if waitTime > 0 {
				m.resetMinerTimer(int64(waitTime))
			} else {
				log.Error("getSleepTime internal error.")
			}
		case <-m.stopCh:
			return
		case <-m.quitCh:
			return
		}
	}
}

// sealBlock 出块
func (m *Miner) sealBlock() {
	if !m.isSelfDeputyNode() {
		return
	}
	log.Debug("start seal")
	header := m.sealHead()
	txs := m.txPool.Pending(10000000)
	newHeader, packagedTxs, invalidTxs, err := m.txProcessor.ApplyTxs(header, txs)
	if err != nil {
		log.Errorf("apply transactions for block failed! %v", err)
		return
	}
	log.Debug("ApplyTxs ok")
	hash := newHeader.Hash()
	signData, err := crypto.Sign(hash[:], m.privKey)
	if err != nil {
		log.Errorf("sign for block failed! block hash:%s", hash.Hex())
		return
	}
	newHeader.SignData = signData
	block, err := m.engine.Seal(newHeader, packagedTxs, m.chain.AccountManager().GetChangeLogs(), m.chain.AccountManager().GetEvents())
	if err != nil {
		log.Error("seal block error!!")
		return
	}
	m.chain.SetMinedBlock(block)
	// remove txs from pool
	txsKeys := make([]common.Hash, len(packagedTxs)+len(invalidTxs))
	for i, tx := range packagedTxs {
		txsKeys[i] = tx.Hash()
	}
	for i, tx := range invalidTxs {
		txsKeys[i+len(packagedTxs)] = tx.Hash()
	}
	m.txPool.Remove(txsKeys)
	nodeCount := deputynode.Instance().GetDeputiesCount()
	var timeDur int64
	if nodeCount == 1 {
		timeDur = m.blockInterval
	} else {
		timeDur = int64(nodeCount-1) * m.timeoutTime
	}
	m.resetMinerTimer(timeDur)
	log.Infof("Mine a new block. height: %d hash: %s", block.Height(), block.Hash().String())
}

// sealHead 生成区块头
func (m *Miner) sealHead() *types.Header {
	// check is need to change minerAddress
	parent := m.currentBlock()
	if (parent.Height()+1)%1001000 == 1 {
		n := deputynode.Instance().GetDeputyByNodeID(parent.Height()+1, deputynode.GetSelfNodeID())
		m.SetMinerAddress(n.MinerAddress)
	}

	// allowable 1 second time error
	// but next block's time can't be small than parent block
	parTime := parent.Time()
	blockTime := uint32(time.Now().Unix())
	if parTime > blockTime {
		blockTime = parTime
	}

	return &types.Header{
		ParentHash:   parent.Hash(),
		MinerAddress: m.minerAddress,
		Height:       parent.Height() + 1,
		GasLimit:     calcGasLimit(parent),
		Time:         blockTime,
		Extra:        m.extra,
	}
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
