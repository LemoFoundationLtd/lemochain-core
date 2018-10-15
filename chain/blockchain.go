package chain

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/network/synchronise/protocol"
	db "github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type broadcastConfirmInfoFn func(hash common.Hash, height uint32)
type broadcastBlockFn func(block *types.Block)

type BlockChain struct {
	chainID              uint16
	flags                map[string]string
	dbOpe                db.ChainDB // 数据库操作
	am                   *account.Manager
	currentBlock         atomic.Value           // 当前链最新区块
	stableBlock          atomic.Value           // 当前链最新的稳定区块
	genesisBlock         *types.Block           // 创始块
	BroadcastConfirmInfo broadcastConfirmInfoFn // 广播确认信息回调
	BroadcastStableBlock broadcastBlockFn       // 广播稳定区块回调

	chainForksHead map[common.Hash]*types.Block // 各分叉链最新头
	chainForksLock sync.Mutex                   // 分叉锁

	engine    Engine       // 共识引擎
	processor *TxProcessor // 状态处理器

	running int32 // 是否在运行

	newBlockCh chan *types.Block // 收到新区块
	quitCh     chan struct{}     // 退出chan
}

func NewBlockChain(chainID uint64, engine Engine, db db.ChainDB, newBlockCh chan *types.Block, flags map[string]string) (bc *BlockChain, err error) {
	bc = &BlockChain{
		chainID:        uint16(chainID),
		dbOpe:          db,
		newBlockCh:     newBlockCh,
		flags:          flags,
		chainForksHead: make(map[common.Hash]*types.Block, 128),
		quitCh:         make(chan struct{}),
	}
	bc.genesisBlock = bc.GetBlockByHeight(0)
	if bc.genesisBlock == nil {
		return nil, ErrNoGenesis
	}
	if err := bc.loadLastState(); err != nil {
		return nil, err
	}
	bc.engine = engine
	bc.processor = NewTxProcessor(bc)
	return bc, nil
}

func (bc *BlockChain) AccountManager() *account.Manager {
	return bc.am
}

// loadLastState 程序启动后初始化加载最新状态
func (bc *BlockChain) loadLastState() error {
	block, err := bc.dbOpe.LoadLatestBlock()
	if err != nil {
		log.Errorf("Can't load last state: %v", err)
		return err
	}
	bc.currentBlock.Store(block)
	bc.stableBlock.Store(block)
	bc.am = account.NewManager(block.Hash(), bc.dbOpe)
	return nil
}

// loadConsensusEngine 加载共识引擎
// func (bc *BlockChain) loadConsensusEngine() {
// 	bc.engine = NewDpovp(bc)
// }

// Genesis 获取创始块
func (bc *BlockChain) Genesis() *types.Block {
	return bc.genesisBlock
}

// ChainID
func (bc *BlockChain) ChainID() uint16 {
	return bc.chainID
}

func (bc *BlockChain) TxProcessor() *TxProcessor {
	return bc.processor
}

func (bc *BlockChain) Flags() map[string]string {
	return bc.flags
}

// HasBlock 本地链是否有某个块
func (bc *BlockChain) HasBlock(hash common.Hash, height uint32) bool {
	if _, err := bc.dbOpe.GetBlock(hash, height); err != nil {
		return false
	}
	return true
}

// GetBlock
func (bc *BlockChain) GetBlock(hash common.Hash, height uint32) *types.Block {
	block, err := bc.dbOpe.GetBlock(hash, height)
	if err != nil {
		// log.Debugf("can't get block height:%d", height)
		return nil
	}
	return block
}

func (bc *BlockChain) GetBlockByHeight(height uint32) *types.Block {
	if height == 0 {
		block, err := bc.dbOpe.GetBlockByHeight(height)
		if err != nil {
			log.Warnf("can't get block. height:%d, err: %v", height, err)
			return nil
		}
		return block
	}
	h_c := bc.currentBlock.Load().(*types.Block).Height()
	h_s := bc.stableBlock.Load().(*types.Block).Height()
	block := bc.currentBlock.Load().(*types.Block)
	var err error
	if h_s >= height {
		block, err = bc.dbOpe.GetBlockByHeight(height)
		if err != nil {
			log.Warnf("can't get block. height:%d, err: %v", height, err)
			return nil
		}
	} else if height <= h_c {
		for i := h_c - height; i > 0; i-- {
			block, err = bc.dbOpe.GetBlockByHash(block.ParentHash())
			if err != nil {
				log.Warnf("can't get block. height:%d, err: %v", height, err)
				return nil
			}
		}
	} else {
		return nil
	}
	return block
}

func (bc *BlockChain) GetBlockByHash(hash common.Hash) *types.Block {
	block, err := bc.dbOpe.GetBlockByHash(hash)
	if err != nil {
		log.Debugf("can't get block. hash:%s", hash.Hex())
		return nil
	}
	return block
}

// CurrentBlock 获取当前最新区块
func (bc *BlockChain) CurrentBlock() *types.Block {
	return bc.currentBlock.Load().(*types.Block)
}

// StableBlock 获取当前最新被共识的区块
func (bc *BlockChain) StableBlock() *types.Block {
	return bc.stableBlock.Load().(*types.Block)
}

// MineNewBlock 挖到新块
func (bc *BlockChain) MineNewBlock(block *types.Block) error {
	if err := bc.dbOpe.SetBlock(block.Hash(), block); err != nil { // 放入缓存中
		log.Error(fmt.Sprintf("can't insert block to cache. height:%d hash:%s", block.Height(), block.Hash().Hex()))
		return err
	}
	err := bc.AccountManager().Save(block.Hash())
	if err != nil {
		log.Error("save account error!", "hash", block.Hash().Hex(), "err", err)
		return err
	}
	nodeCount := deputynode.Instance().GetDeputyNodesCount()
	if nodeCount == 1 {
		bc.SetStableBlock(block.Hash(), block.Height())
	}
	bc.currentBlock.Store(block)
	delete(bc.chainForksHead, block.ParentHash())
	bc.chainForksHead[block.Hash()] = block
	return nil
}

func (bc *BlockChain) newBlockNotify(block *types.Block) {
	go func() { bc.newBlockCh <- block }()
}

// InsertChain insert block of non-self to chain
func (bc *BlockChain) InsertChain(block *types.Block) (err error) {
	log.Debugf("start insert block to chain. height: %d", block.Height())
	hash := block.Hash()
	parHash := block.ParentHash()
	curHash := bc.currentBlock.Load().(*types.Block).Hash()
	// execute tx
	newHeader, err := bc.processor.Process(block)
	if err != nil {
		log.Warn("process block error!", "hash", hash.Hex(), "err", err)
		return err
	}
	// verify
	if newHeader.Hash() != hash {
		log.Warn(fmt.Sprintf("verify block error! hash:%s", hash.Hex()))
		return fmt.Errorf("verify block error! hash:%s", hash.Hex())
	}
	// save
	block.SetEvents(bc.AccountManager().GetEvents())
	block.SetChangeLog(bc.AccountManager().GetChangeLogs())
	if err = bc.dbOpe.SetBlock(hash, block); err != nil { // 放入缓存中
		log.Error(fmt.Sprintf("can't insert block to cache. height:%d hash:%s", block.Height(), hash.Hex()))
		return err
	}
	log.Infof("insert block to chain. height: %d", block.Height())
	err = bc.AccountManager().Save(hash)
	if err != nil {
		log.Error("save account error!", "height", block.Height(), "hash", hash.Hex(), "err", err)
		return err
	}

	nodeCount := deputynode.Instance().GetDeputyNodesCount()
	if nodeCount < 3 {
		defer bc.SetStableBlock(hash, block.Height())
	} else {
		if block.ConfirmPackage != nil {
			minCount := int(math.Ceil(float64(nodeCount) * 2.0 / 3.0))
			if len(block.ConfirmPackage)+1 >= minCount { // default, we think the miner has confirm this block
				defer bc.SetStableBlock(hash, block.Height())
			}
		}
	}

	bc.chainForksLock.Lock()
	// needFork := false
	defer func() {
		bc.chainForksLock.Unlock()
		// if needFork {
		// 	curBlock := bc.currentBlock.Load().(*types.Block)
		// 	log.Infof("chain forked! current block: height(%d), hash(%s)", curBlock.Height(), curBlock.Hash().Hex())
		// }
		log.Debugf("Insert block to db success. height:%d", block.Height())
		// only broadcast confirm info within one hour
		c_t := uint64(time.Now().Unix())
		if c_t-block.Time().Uint64() < 60*60 {
			bc.BroadcastConfirmInfo(hash, block.Height())
		}
	}()

	// normal, in same chain
	if bytes.Compare(parHash[:], curHash[:]) == 0 {
		// needFork = false
		bc.currentBlock.Store(block)
		delete(bc.chainForksHead, curHash) // remove old record from fork container
		bc.chainForksHead[hash] = block    // record new fork
		bc.newBlockNotify(block)
		return nil
	}
	// new block height higher than current block, switch fork.
	curHeight := bc.currentBlock.Load().(*types.Block).Height()
	if block.Height() > curHeight {
		// needFork = true
		bc.currentBlock.Store(block)
		delete(bc.chainForksHead, parHash)
		log.Infof("chain forked! current block: height(%d), hash(%s)", block.Height(), block.Hash().Hex())
	} else if curHeight == block.Height() { // two block with same height, priority of lower alphabet order
		if hash.Big().Cmp(curHash.Big()) < 0 {
			// needFork = true
			bc.currentBlock.Store(block)
			delete(bc.chainForksHead, parHash)
			log.Infof("chain forked! current block: height(%d), hash(%s)", block.Height(), block.Hash().Hex())
		}
	} else {
		if _, ok := bc.chainForksHead[parHash]; ok {
			delete(bc.chainForksHead, parHash)
		}
	}
	bc.chainForksHead[hash] = block
	bc.newBlockNotify(block)
	return nil
}

// SetStableBlock 设置最新的稳定区块
func (bc *BlockChain) SetStableBlock(hash common.Hash, height uint32) error {
	if err := bc.dbOpe.SetStableBlock(hash); err != nil {
		log.Error(fmt.Sprintf("SetStableBlock error. height:%d hash:%s", height, common.ToHex(hash[:])))
		return err
	}
	block := bc.GetBlock(hash, height)
	if block == nil {
		return errors.New("please sync latest block")
	}
	bc.stableBlock.Store(block)
	defer log.Infof("block has consensus. height:%d hash:%s", block.Height(), block.Hash().Hex())
	// 判断是否需要切换分叉
	parBlock := bc.currentBlock.Load().(*types.Block)
	for parBlock.Height() > height {
		parBlock = bc.GetBlockByHash(parBlock.ParentHash())
	}
	if parBlock.Hash() == hash {
		return nil
	}
	// 切换分叉
	bc.chainForksLock.Lock()
	defer bc.chainForksLock.Unlock()
	delete(bc.chainForksHead, bc.currentBlock.Load().(*types.Block).Hash())
	var curBlock *types.Block
	var highest = uint32(0)
	for fHash, fBlock := range bc.chainForksHead {
		if fBlock.Height() < height {
			delete(bc.chainForksHead, fHash)
			continue
		}
		parBlock = fBlock
		for parBlock.Height() > height {
			parBlock = bc.GetBlockByHash(parBlock.ParentHash())
		}
		if parBlock.Hash() == hash {
			if highest < fBlock.Height() { // 高度大的优先
				highest = fBlock.Height()
				curBlock = fBlock
			}
		} else {
			delete(bc.chainForksHead, fHash)
		}
	}
	// 同一高度下字典序靠前的优先
	for fHash, fBlock := range bc.chainForksHead {
		curHash := curBlock.Hash()
		if curBlock.Height() == fBlock.Height() && bytes.Compare(curHash[:], fHash[:]) > 0 {
			curBlock = fBlock
		}
	}
	bc.currentBlock.Store(curBlock)
	log.Infof("chain forked! current block: height(%d), hash(%s)", curBlock.Height(), curBlock.Hash().Hex())
	return nil
}

// Verify 验证区块是否合法
func (bc *BlockChain) Verify(block *types.Block) error {
	err := bc.engine.VerifyHeader(block)
	if err != nil {
		return err
	}
	if err = bc.verifyBody(block); err != nil {
		return err
	}
	return nil
}

// verifyBody 验证包体
func (bc *BlockChain) verifyBody(block *types.Block) error {
	header := block.Header
	if hash := types.DeriveTxsSha(block.Txs); hash == header.TxRoot {
		return nil
	}
	return fmt.Errorf("verify body failed. hash:%s height:%d", block.Hash(), block.Height())
}

// ReceiveConfirm 收到确认消息
func (bc *BlockChain) ReceiveConfirm(info *protocol.BlockConfirmData) (err error) {
	// 恢复公钥
	pubKey, err := crypto.Ecrecover(info.Hash[:], info.SignInfo[:])
	if err != nil {
		log.Warnf("Can't recover signer. hash:%s SignInfo:%s", info.Hash.Hex(), common.ToHex(info.SignInfo[:]))
		return err
	}
	// 获取确认者在主节点列表索引
	index := bc.getSignerIndex(pubKey[1:], info.Height)
	if index < 0 {
		log.Warnf("unavailable confirm info. from: %s", common.ToHex(pubKey[1:]))
		return fmt.Errorf("unavailable confirm info. from: %s", common.ToHex(pubKey[1:]))
	}

	// 查看是否已达成共识
	stableBlock := bc.stableBlock.Load().(*types.Block)
	if stableBlock.Height() >= info.Height { // stable block's confirm info
		bc.dbOpe.AppendConfirmInfo(info.Hash, info.SignInfo)
		return nil
	}

	// 将确认信息缓存起来
	if err = bc.dbOpe.SetConfirmInfo(info.Hash, info.SignInfo); err != nil {
		log.Errorf("can't SetConfirmInfo. hash:%s", info.Hash.Hex())
		return err
	}
	// log.Debugf("Receive confirm info. height: %d. hash: %s", info.Height, info.Hash.String())

	confirmCount, err := bc.getConfirmCount(info.Hash)
	if err != nil {
		log.Warnf("Can't GetConfirmInfo. hash:%s. error: %v", info.Hash.Hex(), err)
		return err
	}
	nodeCount := deputynode.Instance().GetDeputyNodesCount()
	minCount := int(math.Ceil(float64(nodeCount) * 2.0 / 3.0))
	if confirmCount >= minCount {
		return bc.SetStableBlock(info.Hash, info.Height)
	}
	return nil
}

// getConfirmCount get confirm count by hash
func (bc *BlockChain) getConfirmCount(hash common.Hash) (int, error) {
	pack, err := bc.dbOpe.GetConfirmPackage(hash)
	if err != nil {
		return -1, err
	}
	return len(pack) + 1, nil // 出块者默认有确认包
}

// 获取签名者在代理节点列表中的索引
func (bc *BlockChain) getSignerIndex(pubKey []byte, height uint32) int {
	node := deputynode.Instance().GetNodeByNodeID(height, pubKey)
	if node != nil {
		return int(node.Rank)
	}
	return -1
}

// GetConfirmPackage 获取指定区块的确认包
func (bc *BlockChain) GetConfirmPackage(query *protocol.GetConfirmInfo) []types.SignData {
	res, err := bc.dbOpe.GetConfirmPackage(query.Hash)
	if err != nil {
		log.Warn(fmt.Sprintf("can't GetConfirmPackage. hash:%s height:%d", query.Hash.Hex(), query.Height))
		return nil
	}
	return res
}

// ReceiveConfirmPackage 接收到区块确认包
func (bc *BlockChain) ReceiveConfirmPackage(pack protocol.BlockConfirmPackage) {
	if pack.Hash != (common.Hash{}) && pack.Pack != nil && len(pack.Pack) > 0 {
		bc.dbOpe.SetConfirmPackage(pack.Hash, pack.Pack)
	}
}

// Stop 停止block chain
func (bc *BlockChain) Stop() {
	if !atomic.CompareAndSwapInt32(&bc.running, 0, 1) {
		return
	}
	close(bc.quitCh)
	log.Info("Blockchain stopped")
}

func (bc *BlockChain) Db() db.ChainDB {
	return bc.dbOpe
}
