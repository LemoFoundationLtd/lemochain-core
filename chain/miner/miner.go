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
	"math/big"
	"sync"
	"sync/atomic"
	"time"
)

type Miner struct {
	blockInternal int64
	timeoutTime   int64
	privKey       *ecdsa.PrivateKey
	lemoBase      common.Address
	mining        int32
	engine        *chain.Dpovp
	//lemo Backend
	chain        *chain.BlockChain
	mux          sync.Mutex
	currentBlock func() *types.Block
	extra        []byte // 扩展数据 暂保留 最大256byte

	blockMineTimer *time.Timer // 出块timer

	mineNewBlockCh chan *types.Block // 挖到区块后传入通道通知外界
	recvNewBlockCh chan *types.Block // 收到新块通知
	timeToMineCh   chan struct{}     // 到出块时间了
	quitCh         chan struct{}     // 退出
}

func New(blockInternal, timeout int64, chain *chain.BlockChain, privKey *ecdsa.PrivateKey, mineNewBlockCh, recvBlockCh chan *types.Block) *Miner {
	m := &Miner{
		blockInternal:  blockInternal,
		timeoutTime:    timeout,
		privKey:        privKey,
		chain:          chain,
		currentBlock:   chain.CurrentBlock,
		mineNewBlockCh: mineNewBlockCh,
		recvNewBlockCh: make(chan *types.Block),
	}
	recvBlockCh = m.recvNewBlockCh
	return m
}

func (m *Miner) Start() {
	if !atomic.CompareAndSwapInt32(&m.mining, 0, 1) {
		log.Warn("have already start mining")
	}
	m.modifyTimer()
	go m.loop()
	log.Info("start mining...")
}

func (m *Miner) Stop() {
	select {
	case m.quitCh <- struct{}{}:
	default:
	}
	atomic.StoreInt32(&m.mining, 0)
}

func (m *Miner) IsMining() bool {
	return atomic.LoadInt32(&m.mining) == 1
}

// 获取最新区块的时间戳离当前时间的距离 单位：ms
func (m *Miner) getTimespan() int64 {
	lstSpan := m.currentBlock().Header.Time.Int64()
	if lstSpan == int64(0) {
		log.Debug("getTimespan: current block's time is 0")
		return int64(m.blockInternal)
	}
	now := time.Now().Unix()
	return (now - lstSpan) * 1000
}

// isSelfDeputyNode 本节点是否为代理节点
func (m *Miner) isSelfDeputyNode() bool {
	return deputynode.Instance().IsSelfDeputyNode(m.currentBlock().Height())
}

// 修改定时器
func (m *Miner) modifyTimer() {
	if !m.isSelfDeputyNode() {
		return
	}
	nodeCount := deputynode.Instance().GetDeputyNodesCount()
	// 只有一个主节点
	if nodeCount == 1 {
		waitTime := m.blockInternal
		m.resetMinerTimer(waitTime)
		log.Debug(fmt.Sprintf("modifyTimer: waitTime:%d", waitTime))
		return
	}
	timeDur := m.getTimespan()                                                                                          // 获取当前时间与最新块的时间差
	slot := deputynode.Instance().GetSlot(m.currentBlock().Header.Height, m.currentBlock().Header.LemoBase, m.lemoBase) // 获取新块离本节点索引的距离
	oneLoopTime := int64(nodeCount) * m.timeoutTime
	log.Debug(fmt.Sprintf("modifyTimer: timeDur:%d slot:%d oneLoopTime:%d", timeDur, slot, oneLoopTime))
	if slot == 0 { // 上一个块为自己出的块
		if timeDur > oneLoopTime { // 间隔大于一轮
			timeDur = timeDur % oneLoopTime // 求余
			waitTime := int64(nodeCount-1)*m.timeoutTime - timeDur
			if waitTime <= 0 {
				log.Debug(fmt.Sprintf("modifyTimer: slot:0 oneLoopTime:%d", oneLoopTime))
				m.timeToMineCh <- struct{}{}
			} else {
				m.resetMinerTimer(waitTime)
				log.Debug(fmt.Sprintf("modifyTimer: waitTime:%d", waitTime))
			}
			log.Debug(fmt.Sprintf("modifyTimer: slot:0 resetMinerTimer(waitTime:%d)", waitTime))
		}
	} else if slot == 1 { // 说明下一个区块就该本节点产生了
		if timeDur > oneLoopTime { // 间隔大于一轮
			timeDur = timeDur % oneLoopTime // 求余
			log.Debug(fmt.Sprintf("modifyTimer: slot:1 timeDur:%d>oneLoopTime:%d ", timeDur, oneLoopTime))
			if timeDur < m.timeoutTime { //
				log.Debug("modifyTimer: start seal")
				m.timeToMineCh <- struct{}{}
			} else {
				waitTime := oneLoopTime - timeDur
				m.resetMinerTimer(waitTime)
				log.Debug(fmt.Sprintf("ModifyTimer: slot:1 timeDur:%d>=self.timeoutTime:%d resetMinerTimer(waitTime:%d)", timeDur, m.timeoutTime, waitTime))
			}
		} else { // 间隔不到一轮
			if timeDur > m.timeoutTime { // 过了本节点该出块的时机
				waitTime := oneLoopTime - timeDur
				m.resetMinerTimer(waitTime)
				log.Debug(fmt.Sprintf("modifyTimer: slot:1 timeDur<oneLoopTime, timeDur>self.timeoutTime, resetMinerTimer(waitTime:%d)", waitTime))
			} else if timeDur >= m.blockInternal { // 如果上一个区块的时间与当前时间差大或等于3s（区块间的最小间隔为3s），则直接出块无需休眠
				log.Debug(fmt.Sprintf("modifyTimer: slot:1 timeDur<oneLoopTime, timeDur>=self.blockInternal isTurn=true"))
				m.timeToMineCh <- struct{}{}
			} else {
				waitTime := m.blockInternal - timeDur // 如果上一个块时间与当前时间非常近（小于3s），则设置休眠
				m.resetMinerTimer(waitTime)
				log.Debug(fmt.Sprintf("modifyTimer: slot:1, else, resetMinerTimer(waitTime:%d)", waitTime))
			}
		}
	} else { // 说明还不该自己出块，但是需要修改超时时间了
		timeDur = timeDur % oneLoopTime
		if timeDur >= int64(slot-1)*m.timeoutTime && timeDur < int64(slot)*m.timeoutTime {
			log.Debug(fmt.Sprintf("modifyTimer: start seal,slot>1,timeDur:%d", timeDur))
			m.timeToMineCh <- struct{}{}
		} else {
			waitTime := (int64(slot-1)*m.timeoutTime - timeDur + oneLoopTime) % oneLoopTime
			m.resetMinerTimer(waitTime)
			log.Debug(fmt.Sprintf("modifyTimer: slot:>1, timeDur:%d, resetMinerTimer(waitTime:%d)", timeDur, waitTime))
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
		log.Debug("resetMinerTimer: isTurn=true")
		if atomic.LoadInt32(&m.mining) == 1 {
			m.timeToMineCh <- struct{}{}
		}
	})
}

func (m *Miner) loop() {
	for {
		select {
		case <-m.timeToMineCh:
			m.sealBlock()
		case <-m.recvNewBlockCh:
			m.modifyTimer()
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
	header := m.sealHead()
	var txs []*types.Transaction // todo
	// txs:=m.lemo.TxPool().Pending()
	txsRes := chain.ApplyTxs(m.chain, header, txs)
	header.Bloom = txsRes.Bloom
	header.GasUsed = txsRes.GasUsed
	header.TxRoot = types.DeriveTxsSha(txsRes.Txs)
	header.EventRoot = types.DeriveEventsSha(txsRes.Events)
	m.engine.Finalize(header)
	m.chain.AccountManager().Finalise()
	verRoot := m.chain.AccountManager().GetVersionRoot()
	header.VersionRoot = verRoot
	changeLogs := m.chain.AccountManager().GetChangeLogs()
	header.LogsRoot = types.DeriveChangeLogsSha(changeLogs)
	hash := header.Hash()
	signData, err := crypto.Sign(hash[:], m.privKey)
	if err != nil {
		log.Error(fmt.Sprintf("sign for block failed! block hash:%s", hash.Hex()))
		return
	}
	header.SignData = signData
	block, err := m.engine.Seal(header, txsRes.Txs, changeLogs, txsRes.Events)
	if err != nil {
		log.Error("seal block error!!")
		return
	}
	m.chain.AccountManager().Save(header.Hash())
	m.mineNewBlockCh <- block
}

// sealHead 生成区块头
func (m *Miner) sealHead() *types.Header {
	parent := m.currentBlock()
	return &types.Header{
		ParentHash: parent.Hash(),
		LemoBase:   m.lemoBase,
		Height:     parent.Height() + 1,
		GasLimit:   calcGasLimit(parent),
		Time:       big.NewInt(time.Now().Unix()),
		Extra:      m.extra,
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
