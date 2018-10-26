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
	"github.com/LemoFoundationLtd/lemochain-go/common/flag"
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
	flags                flag.CmdFlags
	db                   db.ChainDB // 数据库操作
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

func NewBlockChain(chainID uint16, engine Engine, db db.ChainDB, newBlockCh chan *types.Block, flags flag.CmdFlags) (bc *BlockChain, err error) {
	bc = &BlockChain{
		chainID:        chainID,
		db:             db,
		newBlockCh:     newBlockCh,
		flags:          flags,
		engine:         engine,
		chainForksHead: make(map[common.Hash]*types.Block, 16),
		quitCh:         make(chan struct{}),
	}
	bc.genesisBlock = bc.GetBlockByHeight(0)
	if bc.genesisBlock == nil {
		return nil, ErrNoGenesis
	}
	if err := bc.loadLastState(); err != nil {
		return nil, err
	}
	bc.processor = NewTxProcessor(bc)
	return bc, nil
}

func (bc *BlockChain) AccountManager() *account.Manager {
	return bc.am
}

// loadLastState 程序启动后初始化加载最新状态
func (bc *BlockChain) loadLastState() error {
	block, err := bc.db.LoadLatestBlock()
	if err != nil {
		log.Errorf("Can't load last state: %v", err)
		return err
	}
	bc.currentBlock.Store(block)
	bc.stableBlock.Store(block)
	bc.am = account.NewManager(block.Hash(), bc.db)
	return nil
}

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

func (bc *BlockChain) Flags() flag.CmdFlags {
	return bc.flags
}

// HasBlock 本地是否有某个块
func (bc *BlockChain) HasBlock(hash common.Hash) bool {
	if ok, _ := bc.db.IsExistByHash(hash); ok {
		return true
	}
	return false
}

func (bc *BlockChain) getGenesisFromDb() *types.Block {
	block, err := bc.db.GetBlockByHeight(0)
	if err != nil {
		panic("can't get genesis block")
	}
	return block
}

func (bc *BlockChain) GetBlockByHeight(height uint32) *types.Block {
	// genesis block
	if height == 0 {
		return bc.getGenesisFromDb()
	}

	// not genesis block
	block := bc.currentBlock.Load().(*types.Block)
	currentBlockHeight := block.Height()
	stableBlockHeight := bc.stableBlock.Load().(*types.Block).Height()
	var err error
	if stableBlockHeight >= height {
		block, err = bc.db.GetBlockByHeight(height)
		if err != nil {
			panic(fmt.Sprintf("can't get block. height:%d, err: %v", height, err))
		}
	} else if height <= currentBlockHeight {
		for i := currentBlockHeight - height; i > 0; i-- {
			block, err = bc.db.GetBlockByHash(block.ParentHash())
			if err != nil {
				panic(fmt.Sprintf("can't get block. height:%d, err: %v", height, err))
			}
		}
	} else {
		return nil
	}
	return block
}

func (bc *BlockChain) GetBlockByHash(hash common.Hash) *types.Block {
	block, err := bc.db.GetBlockByHash(hash)
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

// SaveMinedBlock 挖到新块
func (bc *BlockChain) SaveMinedBlock(block *types.Block) error {
	if err := bc.db.SetBlock(block.Hash(), block); err != nil { // 放入缓存中
		log.Error(fmt.Sprintf("can't insert block to cache. height:%d hash:%s", block.Height(), block.Hash().Hex()))
		return err
	}
	err := bc.AccountManager().Save(block.Hash())
	if err != nil {
		log.Error("save account error!", "hash", block.Hash().Hex(), "err", err)
		return err
	}
	nodeCount := deputynode.Instance().GetDeputiesCount()
	if nodeCount == 1 {
		bc.SetStableBlock(block.Hash(), block.Height(), false)
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
func (bc *BlockChain) InsertChain(block *types.Block, logLess bool) (err error) {
	if err := bc.Verify(block); err != nil {
		return err
	}

	// log.Debugf("start insert block to chain. height: %d", block.Height())
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
	block.SetChangeLogs(bc.AccountManager().GetChangeLogs())
	if err = bc.db.SetBlock(hash, block); err != nil { // 放入缓存中
		log.Error(fmt.Sprintf("can't insert block to cache. height:%d hash:%s", block.Height(), hash.Hex()))
		return err
	}
	if !logLess {
		log.Infof("Insert block to chain. height: %d. hash: %s", block.Height(), block.Hash().String())
	}
	err = bc.AccountManager().Save(hash)
	if err != nil {
		log.Error("save account error!", "height", block.Height(), "hash", hash.Hex(), "err", err)
		return err
	}

	nodeCount := deputynode.Instance().GetDeputiesCount()
	if nodeCount < 3 {
		defer bc.SetStableBlock(hash, block.Height(), logLess)
	} else {
		minCount := int(math.Ceil(float64(nodeCount) * 2.0 / 3.0))
		if len(block.ConfirmPackage) >= minCount {
			defer bc.SetStableBlock(hash, block.Height(), logLess)
		}
	}

	bc.chainForksLock.Lock()
	defer func() {
		bc.chainForksLock.Unlock()
		// log.Debugf("Insert block to db success. height:%d", block.Height())
		// only broadcast confirm info within one hour
		currentTime := uint64(time.Now().Unix())
		if currentTime-block.Time().Uint64() < 60*60 {
			time.AfterFunc(2*time.Second, func() { // todo
				bc.BroadcastConfirmInfo(hash, block.Height())
			})
		}
	}()

	// normal, in same chain
	if bytes.Compare(parHash[:], curHash[:]) == 0 {
		// needFork = false
		bc.currentBlock.Store(block)
		delete(bc.chainForksHead, curHash) // remove old record from fork container
		bc.chainForksHead[hash] = block    // record new fork
		if !logLess {
			bc.newBlockNotify(block)
		}
		return nil
	}
	// new block height higher than current block, switch fork.
	curHeight := bc.currentBlock.Load().(*types.Block).Height()
	if block.Height() > curHeight {
		// needFork = true
		bc.currentBlock.Store(block)
		delete(bc.chainForksHead, parHash)
		log.Warnf("chain forked! current block: height(%d), hash(%s)", block.Height(), block.Hash().Hex())
	} else if curHeight == block.Height() { // two block with same height, priority of lower alphabet order
		if hash.Big().Cmp(curHash.Big()) < 0 {
			bc.currentBlock.Store(block)
			delete(bc.chainForksHead, parHash)
			log.Warnf("chain forked! current block: height(%d), hash(%s)", block.Height(), block.Hash().Hex())
		}
	} else {
		if _, ok := bc.chainForksHead[parHash]; ok {
			delete(bc.chainForksHead, parHash)
		}
	}
	bc.chainForksHead[hash] = block
	if !logLess {
		bc.newBlockNotify(block)
	}
	return nil
}

// SetStableBlock 设置最新的稳定区块
func (bc *BlockChain) SetStableBlock(hash common.Hash, height uint32, logLess bool) error {
	if err := bc.db.SetStableBlock(hash); err != nil {
		log.Errorf("SetStableBlock error. height:%d hash:%s", height, common.ToHex(hash[:]))
		return err
	}
	block := bc.GetBlockByHash(hash)
	if block == nil {
		return errors.New("please sync latest block")
	}
	bc.stableBlock.Store(block)
	defer func() {
		if !logLess {
			log.Infof("block has consensus. height:%d hash:%s", block.Height(), block.Hash().Hex())
		}
	}()
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
	if !logLess {
		log.Infof("chain forked! current block: height(%d), hash(%s)", curBlock.Height(), curBlock.Hash().Hex())
	}
	return nil
}

// Verify verify block
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

// verifyBody verify block body
func (bc *BlockChain) verifyBody(block *types.Block) error {
	header := block.Header
	if hash := types.DeriveTxsSha(block.Txs); hash == header.TxRoot {
		return nil
	}
	// todo verify deputy root
	return fmt.Errorf("verify body failed. hash:%s height:%d", block.Hash(), block.Height())
}

// ReceiveConfirm
func (bc *BlockChain) ReceiveConfirm(info *protocol.BlockConfirmData) (err error) {
	// recover public key
	pubKey, err := crypto.Ecrecover(info.Hash[:], info.SignInfo[:])
	if err != nil {
		log.Warnf("Can't recover signer. hash:%s SignInfo:%s", info.Hash.Hex(), common.ToHex(info.SignInfo[:]))
		return err
	}
	// get index of node
	index := bc.getSignerIndex(pubKey[1:], info.Height)
	if index < 0 {
		log.Warnf("Unavailable confirm info. from: %s", common.ToHex(pubKey[1:]))
		return fmt.Errorf("unavailable confirm info. from: %s", common.ToHex(pubKey[1:]))
	}

	// has consensus?
	stableBlock := bc.stableBlock.Load().(*types.Block)
	if stableBlock.Height() >= info.Height { // stable block's confirm info
		ok, err := bc.hasEnoughConfirmInfo(info.Hash)
		if err != nil {
			return err
		}
		if !ok {
			bc.db.AppendConfirmInfo(info.Hash, info.SignInfo)
		}
		return nil
	}

	// cache confirm info
	if err = bc.db.SetConfirmInfo(info.Hash, info.SignInfo); err != nil {
		log.Errorf("can't SetConfirmInfo. hash:%s", info.Hash.Hex())
		return err
	}
	// log.Debugf("Receive confirm info. height: %d. hash: %s", info.Height, info.Hash.String())

	ok, err := bc.hasEnoughConfirmInfo(info.Hash)
	if err != nil {
		return err
	}
	if ok {
		return bc.SetStableBlock(info.Hash, info.Height, false)
	}
	return nil
}

func (bc *BlockChain) hasEnoughConfirmInfo(hash common.Hash) (bool, error) {
	confirmCount, err := bc.getConfirmCount(hash)
	if err != nil {
		log.Warnf("Can't GetConfirmInfo. hash:%s. error: %v", hash.Hex(), err)
		return false, err
	}
	nodeCount := deputynode.Instance().GetDeputiesCount()
	minCount := int(math.Ceil(float64(nodeCount) * 2.0 / 3.0))
	if confirmCount >= minCount {
		return true, nil
	}
	return false, nil
}

// getConfirmCount get confirm count by hash
func (bc *BlockChain) getConfirmCount(hash common.Hash) (int, error) {
	pack, err := bc.db.GetConfirmPackage(hash)
	if err != nil {
		return -1, err
	}
	return len(pack), nil
}

// 获取签名者在代理节点列表中的索引
func (bc *BlockChain) getSignerIndex(pubKey []byte, height uint32) int {
	node := deputynode.Instance().GetDeputyByNodeID(height, pubKey)
	if node != nil {
		return int(node.Rank)
	}
	return -1
}

// GetConfirmPackage 获取指定区块的确认包
func (bc *BlockChain) GetConfirmPackage(query *protocol.GetConfirmInfo) []types.SignData {
	res, err := bc.db.GetConfirmPackage(query.Hash)
	if err != nil {
		log.Warn(fmt.Sprintf("can't GetConfirmPackage. hash:%s height:%d", query.Hash.Hex(), query.Height))
		return nil
	}
	return res
}

// ReceiveConfirmPackage 接收到区块确认包
func (bc *BlockChain) ReceiveConfirmPackage(pack protocol.BlockConfirmPackage) {
	if pack.Hash != (common.Hash{}) && pack.Pack != nil && len(pack.Pack) > 0 {
		bc.db.SetConfirmPackage(pack.Hash, pack.Pack)
	}
}

// Stop 停止block chain
func (bc *BlockChain) Stop() {
	if !atomic.CompareAndSwapInt32(&bc.running, 0, 1) {
		return
	}
	close(bc.quitCh)
	log.Info("BlockChain stop")
}

func (bc *BlockChain) Db() db.ChainDB {
	return bc.db
}
