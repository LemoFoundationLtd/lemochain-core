package synchronise

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
	"math/rand"
	"strings"
	"time"
)

var (
	errTerminated = errors.New("terminated")
)

const (
	fetchTimeout  = 5 * time.Second
	arriveTimeout = 300 * time.Millisecond // 通知从到本地到真实请求的间隔
	gatherSlack   = 50 * time.Millisecond  // 内部校验快要过期的通知用时
	hashLimit     = 512
	blockLimit    = 128
	maxQueueDist  = 32
)

type blockRetrieveFn func(hash common.Hash, height uint32) *types.Block

type blockRequesterFn func(hash common.Hash, height uint32) error

//type confirmInfoRequesterFn func(hash common.Hash) error
type blockVerifierFn func(block *types.Block) error
type blockBroadcasterFn func(block *types.Block, totalBlock bool)
type chainHeightFn func() uint32
type chainInsertFn func(block *types.Block) error
type peerDropFn func(id string)

// announce 收到通知：远程节点有一个新块
type announce struct {
	hash       common.Hash      // hash
	height     uint32           // block height
	origin     string           // remote node id
	time       time.Time        // timespan of the announcement
	fetchBlock blockRequesterFn // 根据hash获取完整区块
	//fetchConfirmInfo confirmInfoRequesterFn // 根据hash获取确认信息
}

// newBlock 从一个Peer收到完整区块
type newBlock struct {
	origin string
	block  *types.Block
}

// Fetcher 获取区块结构体
type Fetcher struct {
	notifyCh   chan *announce
	newBlockCh chan *newBlock

	announces map[string]int              // 记录每个节点收到了多少个通知 防止内存被耗尽
	announced map[common.Hash][]*announce // 记录块对应的通知，用于fetching的调度
	fetching  map[common.Hash]*announce   // 记录当前正在fetching对应的通知
	//fetched   map[common.Hash]*announce // 记录获取完成的

	// Block 缓存
	queue    *prque.Prque              // 区块队列，已排序
	queueMp  map[string]int            // 记录每个节点收到了多少个未处理的区块
	queuedMp map[common.Hash]*newBlock // 记录每个区块hash对应的区块

	// 各种回调
	getLocalBlock        blockRetrieveFn    // 从本地链上获取块
	verifyBlock          blockVerifierFn    // 校验块头
	broadcastBlock       blockBroadcasterFn // 广播区块
	currentChainHeight   chainHeightFn      // 本地当前链高度
	consensusChainHeight chainHeightFn      // 本地当前经过共识的链高度
	insertChain          chainInsertFn      // 批量插入块到链
	dropPeer             peerDropFn         // 丢掉节点连接

	done chan common.Hash // hash对应的区块获取成功
	quit chan struct{}    // 退出
}

// NewFetcher 实例化fetcher
func NewFetcher(getLocalBlock blockRetrieveFn, verifyBlock blockVerifierFn, broadcastBlock blockBroadcasterFn, currentChainHeight chainHeightFn, consensusChainHeight chainHeightFn, insertChain chainInsertFn, dropPeer peerDropFn) *Fetcher {
	f := &Fetcher{
		notifyCh:   make(chan *announce),
		newBlockCh: make(chan *newBlock),

		announces: make(map[string]int),
		announced: make(map[common.Hash][]*announce),
		fetching:  make(map[common.Hash]*announce),
		//fetched:   make(map[common.Hash]*announce),

		queue:    prque.New(),
		queueMp:  make(map[string]int),
		queuedMp: make(map[common.Hash]*newBlock),

		getLocalBlock:        getLocalBlock,
		verifyBlock:          verifyBlock,
		broadcastBlock:       broadcastBlock,
		currentChainHeight:   currentChainHeight,
		consensusChainHeight: consensusChainHeight,
		insertChain:          insertChain,
		dropPeer:             dropPeer,
	}
	return f
}

// Start start fetcher
func (f *Fetcher) Start() {
	go f.run()
}

// Stop stop fetcher
func (f *Fetcher) Stop() {
	close(f.quit)
}

// run 死循环，用来调度获取区块
func (f *Fetcher) run() {
	fetchTimer := time.NewTimer(0)
	defer fetchTimer.Stop()
	for {
		// 如果获取超时 则不获取了
		for hash, announce := range f.fetching {
			if time.Since(announce.time) > fetchTimeout {
				f.forgetHash(hash)
			}
		}
		// 将队列中的区块导入本地链
		for !f.queue.Empty() {
			height := f.currentChainHeight()
			op := f.queue.PopItem().(*newBlock)
			if op.block.Height() > height+1 {
				f.queue.Push(op, -float32(op.block.Height()))
				break
			}
			// 判断是否为分叉 且分叉的父块没有收到
			for f.getLocalBlock(op.block.ParentHash(), op.block.Height()-1) == nil {
				f.queue.Push(op, -float32(op.block.Height()))
				op = f.queue.PopItem().(*newBlock)
			}
			hash := op.block.Hash()
			if f.getLocalBlock(hash, op.block.Height()) != nil {
				f.forgetBlock(hash)
				continue
			}
			err := f.verifyBlock(op.block)
			switch err {
			case nil:
			default:
				log.Debug(fmt.Sprintf("block verification failed", "peer", op.origin, "number", op.block.Height(), "hash", hash, "err", err))
				if f.dropPeer != nil {
					f.dropPeer(op.origin)
				}
				return
			}
			f.insert(op.origin, op.block)
		}

		select {
		case <-fetchTimer.C:
			request := make(map[string][]struct {
				hash   common.Hash
				height uint32
			})
			// 获取那些通知已到本地时间超时的集合
			for hash, announces := range f.announced {
				if time.Since(announces[0].time) > arriveTimeout-gatherSlack {
					// 随机选择一个节点来获取
					announce := announces[rand.Intn(len(announces))]
					// 从所有缓存里清空有关该hash的记录，类似于初始化
					f.forgetHash(hash)
					// 判断本地是否已有该块，没有就加入获取队列
					if f.getLocalBlock(hash, announce.height) == nil {
						request[announce.origin] = append(request[announce.origin], struct {
							hash   common.Hash
							height uint32
						}{hash: hash, height: announce.height})
						f.fetching[hash] = announce
					}
				}
			}
			// 发送获取区块请求
			for _, hashes := range request {
				fetchBlock, items := f.fetching[hashes[0].hash].fetchBlock, hashes
				go func() {
					for _, item := range items {
						fetchBlock(item.hash, item.height)
					}
				}()
			}
			// 重置调度器
			f.rescheduleFetch(fetchTimer)
		case hash := <-f.done: // 导入链成功
			f.forgetHash(hash)
			f.forgetBlock(hash)
		case <-f.quit:
			return
		case notification := <-f.notifyCh:
			// 接收到的区块高度小于已共识的高度，直接丢掉，后续将此判断逻辑移植到Handler里
			if notification.height <= f.consensusChainHeight() {
				continue
			}
			// 单个peer收到的通知减处理后的还剩下过多
			if f.announces[notification.origin] >= hashLimit {
				log.Debug(fmt.Sprintf("Peer receive announces over limit. origin:%s", notification.origin))
				break
			}
			// 记录Hash对应的所有通知，场景为多个共识节点同时向该节点推送通知
			f.announced[notification.hash] = append(f.announced[notification.hash], notification)
			f.announces[notification.origin] += 1
			// 因为调度器里在len(f.announced) == 0时停止了
			if len(f.announced) == 1 {
				f.rescheduleFetch(fetchTimer)
			}
		case op := <-f.newBlockCh:
			if op.origin == "" {
				continue
			}
			if f.getLocalBlock(op.block.Hash(), op.block.Height()) != nil {
				continue
			}
			f.enqueue(op)
		}
	}

}

// rescheduleFetch 重置获取调度器
func (f *Fetcher) rescheduleFetch(fetch *time.Timer) {
	if len(f.announced) == 0 {
		return
	}
	// 标记announced内收到的最早的那个时间
	earliest := time.Now()
	for _, announces := range f.announced {
		if earliest.After(announces[0].time) { // 因为announces是个数组，按时间先后顺序排列的，只需比较第一个时间即可
			earliest = announces[0].time
		}
	}
	fetch.Reset(arriveTimeout - time.Since(earliest))
}

// Notify 供外界调用 收到新块(hash height等)通知
func (f *Fetcher) Notify(peer string, hash common.Hash, height uint32, fetchBlock blockRequesterFn) error {
	block := &announce{
		hash:       hash,
		height:     height,
		time:       time.Now(),
		origin:     peer,
		fetchBlock: fetchBlock,
	}
	// 防止notifyCh已有数据还没处理导致的长时间休眠态突然退出问题
	select {
	case f.notifyCh <- block:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// FilterBlock 过滤blocks是否为fetcher请求的，处理掉是fetcher请求的，将不是fetcher请求的返回
func (f *Fetcher) FilterBlocks(id string, blocks types.Blocks) types.Blocks {
	unknown := types.Blocks{}
	for _, block := range blocks {
		if announce := f.fetching[block.Hash()]; announce != nil && strings.Compare(id, announce.origin) == 0 {
			f.Enqueue(id, block)
		} else {
			unknown = append(unknown, block)
		}
	}
	return unknown
}

// Enqueue 收到完整块时调用
func (f *Fetcher) Enqueue(peer string, block *types.Block) error {
	op := &newBlock{
		origin: peer,
		block:  block,
	}
	// 防止newBlockCh已有数据还没处理导致的长时间休眠态突然退出问题
	select {
	case f.newBlockCh <- op:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// forgetHash 从相关容器中移除有关hash的记录
func (f *Fetcher) forgetHash(hash common.Hash) {
	for _, announce := range f.announced[hash] {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
	}
	delete(f.announced, hash)

	if announce := f.fetching[hash]; announce != nil {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
		delete(f.fetching, hash)
	}
}

// forgetBlock 从相关容器中移除有关hash的记录
func (f *Fetcher) forgetBlock(hash common.Hash) {
	if op := f.queuedMp[hash]; op != nil {
		f.queueMp[op.origin]--
		if f.queueMp[op.origin] == 0 {
			delete(f.queueMp, op.origin)
		}
		delete(f.queuedMp, hash)
	}
}

// enqueue 将收到的区块添加到队列中
func (f *Fetcher) enqueue(newBlock *newBlock) {
	peer := newBlock.origin
	hash := newBlock.block.Hash()
	if f.queueMp[peer] >= blockLimit {
		f.forgetHash(hash)
		return
	}
	// 新收到的块高度过大 丢掉
	if dist := newBlock.block.Height() - f.currentChainHeight(); dist > maxQueueDist {
		f.forgetHash(hash)
		return
	}
	// 已经存在了 直接返回
	if _, ok := f.queuedMp[hash]; ok {
		return
	}
	f.queueMp[peer]++
	f.queuedMp[hash] = newBlock
	f.queue.Push(newBlock, -float32(newBlock.block.Height()))
	log.Debug(fmt.Sprintf("receive one block, height:%d hash:%s", newBlock.block.Height(), hash[:]))
}

// insert 启动个协程插入块到链上
func (f *Fetcher) insert(peer string, block *types.Block) {
	hash := block.Hash()
	go func() {
		defer func() {
			f.done <- hash
		}()

		if parent := f.getLocalBlock(block.ParentHash(), block.Height()-1); parent == nil {
			log.Debug(fmt.Sprintf("Unknown parent of propagated block", "peer", peer, "number", block.Height(), "hash", hash, "parent", block.ParentHash()))
			return
		}

		if err := f.insertChain(block); err != nil {
			log.Debug("block import failed", "peer", peer, "number", block.Height(), "hash", hash, "err", err)
			return
		}
		// 将块hash广播出去
		go f.broadcastBlock(block, false)
	}()
}
