package chain

import (
	"bytes"
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
	db                   db.ChainDB
	am                   *account.Manager
	currentBlock         atomic.Value           // latest block in current chain
	stableBlock          atomic.Value           // latest stable block in current chain
	genesisBlock         *types.Block           // genesis block
	BroadcastConfirmInfo broadcastConfirmInfoFn // callback of broadcast confirm info
	BroadcastStableBlock broadcastBlockFn       // callback of broadcast stable block

	chainForksHead map[common.Hash]*types.Block // total latest header of different fork chain
	chainForksLock sync.Mutex

	engine     Engine       // consensus engine
	processor  *TxProcessor // state processor
	running    int32
	newBlockCh chan *types.Block // receive new block channel
	quitCh     chan struct{}
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

// loadLastState load latest state in starting
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

// Genesis genesis block
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

// HasBlock has special block in local
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

// CurrentBlock get latest current block
func (bc *BlockChain) CurrentBlock() *types.Block {
	return bc.currentBlock.Load().(*types.Block)
}

// StableBlock get latest stable block
func (bc *BlockChain) StableBlock() *types.Block {
	return bc.stableBlock.Load().(*types.Block)
}

// SaveMinedBlock 挖到新块
func (bc *BlockChain) SaveMinedBlock(block *types.Block) error {
	if err := bc.db.SetBlock(block.Hash(), block); err != nil {
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
func (bc *BlockChain) InsertChain(block *types.Block, isSynchronising bool) (err error) {
	if err := bc.Verify(block); err != nil {
		log.Errorf("block verify failed: %v", err)
		return err
	}

	hash := block.Hash()
	parentHash := block.ParentHash()
	currentHash := bc.currentBlock.Load().(*types.Block).Hash()
	// save
	block.SetEvents(bc.AccountManager().GetEvents())
	block.SetChangeLogs(bc.AccountManager().GetChangeLogs())
	if err = bc.db.SetBlock(hash, block); err != nil {
		log.Errorf("can't insert block to cache. height:%d hash:%s", block.Height(), hash.Hex())
		return err
	}
	if !isSynchronising {
		log.Infof("Insert block to chain. height: %d. hash: %s", block.Height(), block.Hash().String())
	}
	if err := bc.AccountManager().Save(hash); err != nil {
		log.Error("save account error!", "height", block.Height(), "hash", hash.Hex(), "err", err)
		return err
	}
	// is synchronise from net or deputy nodes less than 3
	nodeCount := deputynode.Instance().GetDeputiesCount()
	if nodeCount < 3 {
		defer bc.SetStableBlock(hash, block.Height(), isSynchronising)
	} else {
		minCount := int(math.Ceil(float64(nodeCount) * 2.0 / 3.0))
		if len(block.ConfirmPackage) >= minCount {
			defer bc.SetStableBlock(hash, block.Height(), isSynchronising)
		}
	}

	bc.chainForksLock.Lock()
	defer func() {
		bc.chainForksLock.Unlock()
		// only broadcast confirm info within one hour
		currentTime := uint64(time.Now().Unix())
		if currentTime-block.Time().Uint64() < 60*60 {
			time.AfterFunc(2*time.Second, func() { // todo
				bc.BroadcastConfirmInfo(hash, block.Height())
			})
		}
	}()

	// normal, in same chain
	if bytes.Compare(parentHash[:], currentHash[:]) == 0 {
		bc.currentBlock.Store(block)
		delete(bc.chainForksHead, currentHash) // remove old record from fork container
		bc.chainForksHead[hash] = block        // record new fork
		if !isSynchronising {                  // if synchronising, don't notify
			bc.newBlockNotify(block)
		}
		return nil
	}
	// new block height higher than current block, switch fork.
	curHeight := bc.currentBlock.Load().(*types.Block).Height()
	if block.Height() > curHeight {
		bc.currentBlock.Store(block)
		delete(bc.chainForksHead, parentHash)
		log.Warnf("chain forked! current block: height(%d), hash(%s)", block.Height(), block.Hash().Hex())
	} else if curHeight == block.Height() { // two block with same height, priority of lower alphabet order
		if hash.Big().Cmp(currentHash.Big()) < 0 {
			bc.currentBlock.Store(block)
			delete(bc.chainForksHead, parentHash)
			log.Warnf("chain forked! current block: height(%d), hash(%s)", block.Height(), block.Hash().Hex())
		}
	} else {
		if _, ok := bc.chainForksHead[parentHash]; ok {
			delete(bc.chainForksHead, parentHash)
		}
	}
	bc.chainForksHead[hash] = block
	if !isSynchronising {
		bc.newBlockNotify(block)
	}
	return nil
}

// SetStableBlock
func (bc *BlockChain) SetStableBlock(hash common.Hash, height uint32, logLess bool) error {
	block := bc.GetBlockByHash(hash)
	if block == nil {
		return fmt.Errorf("block not exist. height: %d hash: %s", height, hash.String())
	}
	// set stable
	if err := bc.db.SetStableBlock(hash); err != nil {
		log.Errorf("SetStableBlock error. height:%d hash:%s", height, common.ToHex(hash[:]))
		return err
	}
	bc.stableBlock.Store(block)
	defer func() {
		if !logLess {
			log.Infof("block has consensus. height:%d hash:%s", block.Height(), block.Hash().Hex())
		}
	}()

	// get parent block
	parBlock := bc.currentBlock.Load().(*types.Block)
	for parBlock.Height() > height {
		parBlock = bc.GetBlockByHash(parBlock.ParentHash())
	}
	if parBlock.Hash() == hash {
		return nil
	}
	// fork
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
			if highest < fBlock.Height() { // height priority
				highest = fBlock.Height()
				curBlock = fBlock
			}
		} else {
			delete(bc.chainForksHead, fHash)
		}
	}
	// same height: Sort in dictionary order
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
	// verify header
	if err := bc.engine.VerifyHeader(block); err != nil {
		return err
	}
	// verify body
	if err := bc.verifyBody(block); err != nil {
		return err
	}

	hash := block.Hash()
	// execute tx
	newHeader, err := bc.processor.Process(block)
	if err == ErrInvalidTxInBlock {
		return err
	} else if err == nil {
	} else {
		panic(fmt.Sprintf("internal error: %v", err))
	}

	// verify block hash
	if newHeader.Hash() != hash {
		return fmt.Errorf("verify block error! hash:%s", hash.Hex())
	}
	return nil
}

// verifyBody verify block body
func (bc *BlockChain) verifyBody(block *types.Block) error {
	header := block.Header
	// verify txRoot
	if hash := types.DeriveTxsSha(block.Txs); hash != header.TxRoot {
		return fmt.Errorf("verify block failed. hash:%s height:%d", block.Hash(), block.Height())
	}
	// verify deputyRoot
	if len(block.DeputyNodes) > 0 {
		hash := types.DeriveDeputyRootSha(block.DeputyNodes)
		root := block.Header.DeputyRoot
		if bytes.Compare(hash[:], root) != 0 {
			return fmt.Errorf("verify block failed. deputyRoot not match. header's root: %s, check root: %s", common.ToHex(root), hash.String())
		}
	}
	return nil
}

// ReceiveConfirm
func (bc *BlockChain) ReceiveConfirm(info *protocol.BlockConfirmData) (err error) {
	// recover public key
	pubKey, err := crypto.Ecrecover(info.Hash[:], info.SignInfo[:])
	if err != nil {
		log.Warnf("Unavailable confirm info. Can't recover signer. hash:%s SignInfo:%s", info.Hash.Hex(), common.ToHex(info.SignInfo[:]))
		return err
	}
	// get index of signer
	index := bc.getSignerIndex(pubKey[1:], info.Height)
	if index < 0 {
		log.Warnf("Unavailable confirm info. from: %s", common.ToHex(pubKey[1:]))
		return fmt.Errorf("unavailable confirm info. from: %s", common.ToHex(pubKey[1:]))
	}

	// has block consensus
	stableBlock := bc.stableBlock.Load().(*types.Block)
	if stableBlock.Height() >= info.Height { // stable block's confirm info
		if ok, err := bc.hasEnoughConfirmInfo(info.Hash); err == nil && !ok {
			bc.db.AppendConfirmInfo(info.Hash, info.SignInfo)
		}
		return nil
	}

	// cache confirm info
	if err = bc.db.SetConfirmInfo(info.Hash, info.SignInfo); err != nil {
		log.Errorf("can't SetConfirmInfo. hash:%s", info.Hash.Hex())
		return err
	}

	if ok, _ := bc.hasEnoughConfirmInfo(info.Hash); ok {
		return bc.SetStableBlock(info.Hash, info.Height, false)
	}
	return nil
}

func (bc *BlockChain) hasEnoughConfirmInfo(hash common.Hash) (bool, error) {
	confirmCount, err := bc.getConfirmCount(hash)
	if err != nil {
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
		log.Errorf("Can't GetConfirmInfo. hash:%s. error: %v", hash.Hex(), err)
		return -1, err
	}
	return len(pack), nil
}

// get index of signer in deputy nodes list
func (bc *BlockChain) getSignerIndex(pubKey []byte, height uint32) int {
	node := deputynode.Instance().GetDeputyByNodeID(height, pubKey)
	if node != nil {
		return int(node.Rank)
	}
	return -1
}

// GetConfirmPackage get all confirm info of special block
func (bc *BlockChain) GetConfirmPackage(query *protocol.GetConfirmInfo) []types.SignData {
	res, err := bc.db.GetConfirmPackage(query.Hash)
	if err != nil {
		log.Warnf("Can't GetConfirmPackage. hash:%s height:%d. error: %v", query.Hash.Hex(), query.Height, err)
		return nil
	}
	return res
}

// ReceiveConfirmPackage receive confirm package from net connection
func (bc *BlockChain) ReceiveConfirmPackage(pack protocol.BlockConfirmPackage) {
	if pack.Hash != (common.Hash{}) && pack.Pack != nil && len(pack.Pack) > 0 {
		bc.db.SetConfirmPackage(pack.Hash, pack.Pack)
	}
}

// Stop stop block chain
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
