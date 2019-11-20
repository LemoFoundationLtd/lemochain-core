package store

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-core/store/leveldb"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

var max_candidate_count = 20

type ChainDatabase struct {
	LastConfirm     *CBlock                 // the newest confirm block, and the root of unconfirmed block tree
	UnConfirmBlocks map[common.Hash]*CBlock // unconfirmed block tree nodes
	Context         *RunContext
	LevelDB         *leveldb.LevelDBDatabase
	Beansdb         *BeansDB
	BizDB           *BizDatabase
	RW              sync.RWMutex
	BizRW           sync.RWMutex
}

func checkHome(home string) error {
	isExist, err := FileUtilsIsExist(home)
	if err != nil {
		panic("check home is exist error:" + err.Error())
	}

	if isExist {
		return nil
	}

	err = os.MkdirAll(home, os.ModePerm)
	if err != nil {
		panic("mk dir is exist err : " + err.Error())
	} else {
		return nil
	}
}

func NewChainDataBase(home string) *ChainDatabase {
	err := checkHome(home)
	if err != nil {
		panic("check home: " + home + "|error: " + err.Error())
	}

	db := &ChainDatabase{
		UnConfirmBlocks: make(map[common.Hash]*CBlock),
		Context:         NewRunContext(home),
		LevelDB:         leveldb.NewLevelDBDatabase(filepath.Join(home, "index"), 16, 16),
	}
	// 启动leveldb的metrics数据统计功能
	db.LevelDB.Meter()

	db.BizDB = NewBizDatabase(db, db.LevelDB)
	db.Beansdb = NewBeansDB(home, db.LevelDB)
	db.Beansdb.Start()

	stableBlock, err := db.GetStableBlock()
	if err != nil && err != ErrNotExist {
		panic("get stable block err: " + err.Error())
	}

	// if stableBlock == nil {
	// 	log.Errorf("stable block is nil.")
	// } else {
	// 	log.Errorf("stable block`height: " + strconv.Itoa(int(stableBlock.Height())))
	// }

	db.LastConfirm = NewGenesisBlock(stableBlock, db.Beansdb)
	candidates, err := db.Context.Candidates.GetCandidates()
	if err != nil {
		panic("get candidates err: " + err.Error())
	} else {

		// 把票数为0的candidate筛选掉，默认票数为0的candidate为注销的candidate
		newCandidate := make([]*Candidate, 0, len(candidates))
		for _, val := range candidates {
			accData, err := db.GetAccount(val.GetAddress())
			if err != nil {
				log.Errorf("getAccount from database. address: %s, error: %v", val.Address.String(), err)
				continue
			}
			if result, ok := accData.Candidate.Profile[types.CandidateKeyIsCandidate]; ok {
				if result == types.IsCandidateNode {
					newCandidate = append(newCandidate, val)
				}
			}
		}
		db.LastConfirm.Top.Rank(max_candidate_count, newCandidate)
	}
	return db
}

func (database *ChainDatabase) GetStableBlock() (*types.Block, error) {
	stableBlockHash, err := leveldb.GetCurrentBlock(database.LevelDB)
	if err != nil {
		return nil, err
	}

	if stableBlockHash == (common.Hash{}) {
		return nil, ErrNotExist
	}

	stableBlock, err := database.GetBlockByHash(stableBlockHash)
	if err != nil {
		return nil, err
	}

	return stableBlock, nil
}

func (database *ChainDatabase) GetLastConfirm() *CBlock {
	return database.LastConfirm
}

func (database *ChainDatabase) commitStableBlock(val []byte) error {
	database.BizRW.Lock()
	defer database.BizRW.Unlock()

	var block types.Block
	err := rlp.DecodeBytes(val, &block)
	if err != nil {
		return err
	}

	stableBlock, err := database.GetStableBlock()
	if err != nil && err != ErrNotExist {
		return err
	}

	if err == ErrNotExist {
		if block.Height() != 0 {
			panic("commit stable block. stable block is nil.and the block is not genesis")
		} else {
			return leveldb.SetCurrentBlock(database.LevelDB, block.Hash())
		}
	} else {
		log.Errorf("commit stable block.block height: " + strconv.Itoa(int(block.Height())))
		if block.Height() <= stableBlock.Height() {
			return nil
		} else {
			return leveldb.SetCurrentBlock(database.LevelDB, block.Hash())
		}
	}
}

func (database *ChainDatabase) isCandidate(account *types.AccountData) bool {
	if (account == nil) ||
		(len(account.Candidate.Profile) <= 0) {
		return false
	}

	result, ok := account.Candidate.Profile[types.CandidateKeyIsCandidate]
	if !ok {
		return false
	}

	val, err := strconv.ParseBool(result)
	if err != nil {
		panic("to bool err : " + err.Error())
	} else {
		return val
	}
}

func (database *ChainDatabase) commitCandidates(val []byte) error {
	var account types.AccountData
	err := rlp.DecodeBytes(val, &account)
	if err != nil {
		return err
	}

	if !database.isCandidate(&account) {
		return nil
	} else {
		candidates := make([]*Candidate, 1)
		candidates[0] = &Candidate{
			Address: account.Address,
			Total:   account.Candidate.Votes,
		}
		err := database.Context.SetCandidates(candidates)
		if err != nil {
			return err
		}

		return database.Context.Flush()
	}
}

func (database *ChainDatabase) AfterScan(flag uint32, key []byte, val []byte) error {
	if flag == leveldb.ItemFlagBlock {
		err := database.commitStableBlock(val)
		if err != nil {
			return err
		}
	}

	if flag == leveldb.ItemFlagAct {
		err := database.commitCandidates(val)
		if err != nil {
			return err
		}
	}

	return database.BizDB.AfterCommit(flag, key, val)
}

/**
 * 1. hash => block (chain)
 * 2. tx => block'hash and pos
 * 3. addr => txs
 * 4. addr => account(chain)
 * 5. height => block's hash(chain)
 */
func (database *ChainDatabase) blockCommit(hash common.Hash) error {
	// log.Error("block commit : " + hash.Hex())
	cItem := database.UnConfirmBlocks[hash]
	if (cItem == nil) || (cItem.Block == nil) {
		panic("item or item'block is nil.")
	}

	if (database.LastConfirm.Block == nil) && (cItem.Block.Height() != 0) {
		panic("database.LastConfirm == nil && cItem.Block.Height() != 0")
	}

	if (database.LastConfirm.Block != nil) &&
		(cItem.Block.Height() < database.LastConfirm.Block.Height()) {
		panic("(database.LastConfirm.Block != nil) && (cItem.Block.Height() < database.LastConfirm.Block.Height())")
	}

	batch := database.Beansdb.NewBatch()

	// store block
	buf, err := rlp.EncodeToBytes(cItem.Block)
	if err != nil {
		return err
	}

	batch.Put(leveldb.ItemFlagBlock, hash.Bytes(), buf)
	batch.Put(leveldb.ItemFlagBlockHeight, leveldb.EncodeNumber(cItem.Block.Height()), hash.Bytes())

	// store account
	decode := func(account *types.AccountData, batch Batch) error {
		buf, err = rlp.EncodeToBytes(account)
		if err != nil {
			return err
		} else {
			batch.Put(leveldb.ItemFlagAct, account.Address.Bytes(), buf)
			return nil
		}
	}

	decodeBatch := func(accounts []*types.AccountData, batch Batch) error {
		for index := 0; index < len(accounts); index++ {
			err := decode(accounts[index], batch)
			if err != nil {
				return err
			}
		}
		return nil
	}

	commitContext := func(block *types.Block, candidates []*Candidate) error {
		err = leveldb.SetCurrentBlock(database.LevelDB, cItem.Block.Hash())
		if err != nil {
			return err
		}

		if len(candidates) <= 0 {
			return nil
		}

		err := database.Context.SetCandidates(candidates)
		if err != nil {
			return err
		}

		return database.Context.Flush()
	}

	accounts := cItem.AccountTrieDB.Collect(cItem.Block.Height())
	err = decodeBatch(accounts, batch)
	if err != nil {
		return err
	}

	err = database.Beansdb.Commit(batch)
	if err != nil {
		return err
	}

	candidates := cItem.filterCandidates(accounts)
	// 注意这里即使是为注销候选节点不能删除记录，这里保存进去只是修改票数为0，因为在退还候选节点押金的地方要拉取所有的候选节点来判断注销的候选节点是否没有退还押金。
	return commitContext(cItem.Block, candidates)
}

func (database *ChainDatabase) getBlock4Cache(hash common.Hash) (*types.Block, error) {
	if (hash == common.Hash{}) {
		return nil, ErrNotExist
	}

	cBlock := database.UnConfirmBlocks[hash]
	if (cBlock != nil) && (cBlock.Block != nil) {
		return cBlock.Block, nil
	} else {
		return nil, ErrNotExist
	}
}

func (database *ChainDatabase) getBlock4DB(hash common.Hash) (*types.Block, error) {
	if (hash == common.Hash{}) {
		return nil, ErrNotExist
	}

	block, err := UtilsGetBlockByHash(database.Beansdb, hash)
	if err != nil {
		return nil, err
	}

	if block == nil {
		return nil, ErrNotExist
	} else {
		return block, nil
	}
}

func (database *ChainDatabase) setBlock2DB(hash common.Hash, block *types.Block) error {
	if (hash == common.Hash{}) || (block == nil) {
		return nil
	}

	buf, err := rlp.EncodeToBytes(block)
	if err != nil {
		return err
	} else {
		return database.Beansdb.Put(leveldb.ItemFlagBlock, hash.Bytes(), buf)
	}
}

func (database *ChainDatabase) getBlock(hash common.Hash) (*types.Block, error) {
	block, err := database.getBlock4Cache(hash)
	if err != nil {
		if err != ErrNotExist {
			return nil, err
		} else {
			return database.getBlock4DB(hash)
		}
	} else {
		return block, nil
	}
}

func (database *ChainDatabase) SizeOfValue(hash common.Hash) (int, error) {
	val, err := database.Beansdb.Get(leveldb.ItemFlagBlock, hash.Bytes())
	if err != nil {
		return -1, err
	} else {
		return len(val), nil
	}
}

func (database *ChainDatabase) GetBlockByHeight(height uint32) (*types.Block, error) {
	database.RW.Lock()
	defer database.RW.Unlock()

	block, err := UtilsGetBlockByHeight(database.Beansdb, height)
	if err != nil {
		return nil, err
	}

	if block == nil {
		return nil, ErrNotExist
	}

	return block, nil
}

func (database *ChainDatabase) GetBlockByHash(hash common.Hash) (*types.Block, error) {
	database.RW.Lock()
	defer database.RW.Unlock()

	return database.getBlock(hash)
}

func (database *ChainDatabase) isExistByHash(hash common.Hash) (bool, error) {
	if (hash == common.Hash{}) {
		return false, nil
	}

	item := database.UnConfirmBlocks[hash]
	if (item != nil) && (item.Block != nil) {
		return true, nil
	}

	return UtilsHashBlock(database.Beansdb, hash)
}

func (database *ChainDatabase) IsExistByHash(hash common.Hash) (bool, error) {
	database.RW.Lock()
	defer database.RW.Unlock()

	return database.isExistByHash(hash)
}

// GetUnConfirmByHeight find unconfirmed block by height. The leafBlockHash is a son block on the fork
func (database *ChainDatabase) GetUnConfirmByHeight(height uint32, leafBlockHash common.Hash) (*types.Block, error) {
	// confirmed block
	if height <= database.LastConfirm.Block.Height() {
		return nil, ErrNotExist
	}

	database.RW.Lock()
	defer database.RW.Unlock()

	// find the block parent by parent till reach the specific height
	leaf := database.UnConfirmBlocks[leafBlockHash]
	for leaf != nil && leaf.Block.Height() > height {
		leaf = leaf.Parent
	}

	if leaf == nil {
		return nil, ErrNotExist
	}
	return leaf.Block, nil
}

func (database *ChainDatabase) SetBlock(hash common.Hash, block *types.Block) error {
	database.RW.Lock()
	defer database.RW.Unlock()

	isExist, err := database.isExistByHash(hash)
	if err != nil {
		log.Errorf("block is exist. err: " + err.Error())
		return err
	}

	if isExist {
		log.Debug("[store]set block error:the block is exist.hash:%s", hash.String())
		return ErrExist
	}

	if database.LastConfirm.Block == nil {
		if (block.Height() != 0) || (block.ParentHash() != common.Hash{}) {
			log.Errorf("(database.LastConfirm.Block == nil) && (block.Height() != 0) && (block.ParentHash() != common.Hash{})")
			return ErrArgInvalid
		}
	} else {
		if (block.Height() == 0) || (block.ParentHash() == common.Hash{}) {
			log.Errorf("(block.Height() == 0) || (block.ParentHash() == common.Hash{})")
			return ErrArgInvalid
		}

		if block.Height() <= database.LastConfirm.Block.Height() {
			bheight := strconv.Itoa(int(block.Height()))
			bhash := hash.Hex()
			lheight := strconv.Itoa(int(database.LastConfirm.Block.Height()))
			lhash := database.LastConfirm.Block.Hash().Hex()
			log.Errorf("1.block'height:" + bheight + "|bhash:" + bhash + "|confirm'height:" + lheight + "|lhash:" + lhash)
			return ErrArgInvalid
		}
	}

	// genesis block
	if (block.ParentHash() == common.Hash{}) {
		newCBlock := NewGenesisBlock(block, database.Beansdb)
		newCBlock.BeChildOf(database.LastConfirm)
		database.UnConfirmBlocks[hash] = newCBlock
		log.Debug("block is genesis.height: " + strconv.Itoa(int(block.Height())))
		return nil
	}

	pHash := block.Header.ParentHash
	pBlock := database.UnConfirmBlocks[pHash]
	if pBlock == nil {
		if database.LastConfirm.Block.Header.Hash() != pHash {
			bheight := strconv.Itoa(int(block.Height()))
			bhash := hash.Hex()
			lheight := strconv.Itoa(int(database.LastConfirm.Block.Height()))
			lhash := database.LastConfirm.Block.Hash().Hex()
			log.Errorf("2.block'height:" + bheight + "|bhash:" + bhash + "|confirm'height:" + lheight + "|lhash:" + lhash)
			return ErrArgInvalid
		}

		if database.LastConfirm.Block.Height()+1 != block.Height() {
			bheight := strconv.Itoa(int(block.Height()))
			bhash := hash.Hex()
			lheight := strconv.Itoa(int(database.LastConfirm.Block.Height()))
			lhash := database.LastConfirm.Block.Hash().Hex()
			log.Errorf("3.block'height:" + bheight + "|bhash:" + bhash + "|confirm'height:" + lheight + "|lhash:" + lhash)
			return ErrArgInvalid
		}

		pBlock = database.LastConfirm
	} else {
		if pBlock.Block.Height()+1 != block.Height() {
			bheight := strconv.Itoa(int(block.Height()))
			bhash := hash.Hex()
			lheight := strconv.Itoa(int(database.LastConfirm.Block.Height()))
			lhash := database.LastConfirm.Block.Hash().Hex()
			log.Errorf("4.block'height:" + bheight + "|bhash:" + bhash + "|confirm'height:" + lheight + "|lhash:" + lhash)
			return ErrArgInvalid
		}
	}
	newCBlock := NewNormalBlock(block, pBlock.AccountTrieDB, pBlock.CandidateTrieDB, pBlock.Top)
	newCBlock.BeChildOf(pBlock)
	database.UnConfirmBlocks[hash] = newCBlock

	return nil
}

func (database *ChainDatabase) appendConfirm(block *types.Block, confirms []types.SignData) {
	if (block == nil) || (confirms == nil) {
		return
	}

	for _, confirm := range confirms {
		if !block.IsConfirmExist(confirm) {
			block.Confirms = append(block.Confirms, confirm)
		}
	}
}

func (database *ChainDatabase) setConfirm(hash common.Hash, confirms []types.SignData) (*types.Block, error) {
	item := database.UnConfirmBlocks[hash]
	if (item != nil) && (item.Block != nil) {
		database.appendConfirm(item.Block, confirms)
		return item.Block, nil
	} else {
		block, err := database.getBlock4DB(hash)
		if err != nil {
			return nil, err
		} else {
			database.appendConfirm(block, confirms)
			return block, database.setBlock2DB(hash, block)
		}
	}
}

// SetConfirms 设置区块的确认信息
func (database *ChainDatabase) SetConfirms(hash common.Hash, pack []types.SignData) (*types.Block, error) {
	database.RW.Lock()
	defer database.RW.Unlock()

	return database.setConfirm(hash, pack)
}

func (database *ChainDatabase) GetConfirms(hash common.Hash) ([]types.SignData, error) {
	database.RW.Lock()
	defer database.RW.Unlock()

	block, err := database.getBlock(hash)
	if err != nil {
		return nil, err
	} else {
		return block.Confirms, nil
	}
}

func (database *ChainDatabase) LoadLatestBlock() (*types.Block, error) {
	if database.LastConfirm == nil || database.LastConfirm.Block == nil {
		return nil, ErrNotExist
	} else {
		return database.LastConfirm.Block, nil
	}
}

// SetStableBlock set the state of the block to stable, then return pruned uncle blocks
func (database *ChainDatabase) SetStableBlock(hash common.Hash) ([]*types.Block, error) {
	database.RW.Lock()
	defer database.RW.Unlock()

	// log.Error("set stable block : " + hash.Hex())
	cItem := database.UnConfirmBlocks[hash]
	if cItem == nil {
		log.Errorf("set stable block error:the block is not exist. hash:" + hash.Hex())
		return nil, ErrArgInvalid
	}
	droppedBlocks := make([]*types.Block, 0)

	// clear the branches from root, except one branch
	clear := func(oldRoot, newRoot *CBlock) {
		removeList := make([]*CBlock, 0)
		// remove other brunch nodes
		oldRoot.Walk(func(node *CBlock) {
			// The walk method touch tree nodes from parent to child. We can't remove it in walk callback, because the node.Children will be set to nil in remove action
			removeList = append(removeList, node)
		}, newRoot)
		for _, node := range removeList {
			delete(database.UnConfirmBlocks, node.Block.Hash())
			node.Parent = nil
			node.Children = nil
			// collect pruned uncle blocks
			droppedBlocks = append(droppedBlocks, node.Block)
		}

		// remove old root from unconfirmed nodes map
		delete(database.UnConfirmBlocks, newRoot.Block.Hash())
		// cut the connection between old root and new root
		newRoot.Parent = nil
		oldRoot.Children = nil
	}

	commit := func(blocks []*CBlock) error {
		for index := len(blocks) - 1; index >= 0; index-- {
			cItem := blocks[index]
			err := database.blockCommit(cItem.Block.Hash())
			if err != nil {
				return err
			} else {
				oldLast := database.LastConfirm
				database.LastConfirm = cItem
				clear(oldLast, cItem)
			}
		}

		return nil
	}

	blocks := cItem.CollectToParent(database.LastConfirm)
	err := commit(blocks)
	return droppedBlocks, err
}

// GetAccount loads account from cache or db
func (database *ChainDatabase) GetAccount(addr common.Address) (*types.AccountData, error) {
	account, err := UtilsGetAccount(database.Beansdb, addr)
	if err != nil {
		return nil, err
	}

	if account == nil {
		return nil, ErrNotExist
	} else {
		return account, nil
	}
}

func (database *ChainDatabase) GetTrieDatabase() *TrieDatabase {
	return NewTrieDatabase(database.Beansdb)
}

func (database *ChainDatabase) GetActDatabase(hash common.Hash) (*AccountTrieDB, error) {
	if (hash == common.Hash{}) {
		return NewAccountTrieDB(NewEmptyDatabase(), database.Beansdb), nil
	}

	item := database.UnConfirmBlocks[hash]
	if item == nil {
		_, err := database.getBlock4DB(hash)
		if err == ErrNotExist {
			panic("the block not exist. check if the block is set.")
		}

		if err != nil {
			return nil, err
		}

		if database.LastConfirm == nil {
			return NewAccountTrieDB(NewEmptyDatabase(), database.Beansdb), nil
		} else {
			return database.LastConfirm.AccountTrieDB, nil
		}
	} else {
		return item.AccountTrieDB, nil
	}
}

// GetContractCode loads contract's code from db.
func (database *ChainDatabase) GetContractCode(hash common.Hash) (types.Code, error) {
	val, err := database.Beansdb.Get(leveldb.ItemFlagCode, hash.Bytes())
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, ErrNotExist
	} else {
		var code types.Code = val
		return code, nil
	}
}

// SetContractCode saves contract's code
func (database *ChainDatabase) SetContractCode(hash common.Hash, code types.Code) error {
	return database.Beansdb.Put(leveldb.ItemFlagCode, hash.Bytes(), code[:])
}

func (database *ChainDatabase) GetCandidatesTop(hash common.Hash) []*Candidate {
	cItem := database.UnConfirmBlocks[hash]
	if (cItem != nil) && (cItem.Block != nil) {
		if cItem.Top == nil {
			panic("item top30 is nil.")
		} else {
			return cItem.Top.GetTop()
		}
	}

	if database.LastConfirm.Block == nil { // all in cache
		panic("database.LastConfirm.Block == nil")
	}

	if hash == database.LastConfirm.Block.Hash() {
		return database.LastConfirm.Top.GetTop()
	} else {
		panic("hash != database.LastConfirm.Block.Hash()")
	}
}

func (database *ChainDatabase) GetCandidatesPage(index int, size int) ([]common.Address, uint32, error) {
	if (index < 0) || (size > 200) || (size <= 0) {
		return nil, 0, errors.New("argment error.")
	} else {
		return database.Context.GetCandidatePage(index, size)
	}
}

func (database *ChainDatabase) GetAllCandidates() ([]common.Address, error) {
	c, err := database.Context.GetCandidates()
	if err != nil {
		return nil, err
	}
	addresses := make([]common.Address, 0, len(c))
	for i := 0; i < len(c); i++ {
		addresses = append(addresses, c[i].Address)
	}
	return addresses, nil
}

func (database *ChainDatabase) CandidatesRanking(hash common.Hash, voteLogs types.ChangeLogSlice) {
	cItem := database.UnConfirmBlocks[hash]
	if (cItem == nil) || (cItem.Block == nil) {
		panic("item or item'block is nil.")
	}

	if ((cItem.Block.Height() == 0) && (cItem.Block.ParentHash() != common.Hash{})) ||
		((cItem.Block.ParentHash() == common.Hash{}) && (cItem.Block.Height() != 0)) {
		panic("（cItem.Block.Height() == 0) || (cItem.Block.ParentHash() != common.Hash{})")
	}

	if (database.LastConfirm.Block != nil) &&
		(cItem.Block.Height() <= database.LastConfirm.Block.Height()) {
		panic("(database.LastConfirm.Block != nil) && (cItem.Block.Height() < database.LastConfirm.Block.Height())")
	}

	cItem.Ranking(voteLogs)
}

func (database *ChainDatabase) GetAssetCode(code common.Hash) (common.Address, error) {
	return UtilsGetAssetCode(database.Beansdb, code)
}

func (database *ChainDatabase) GetAssetID(id common.Hash) (common.Address, error) {
	code, err := UtilsGetAssetId(database.Beansdb, id)
	if err != nil {
		return common.Address{}, err
	}

	if code == (common.Hash{}) {
		return common.Address{}, nil
	} else {
		return UtilsGetAssetCode(database.Beansdb, code)
	}
}

func (database *ChainDatabase) IterateUnConfirms(fn func(*types.Block)) {
	database.LastConfirm.Walk(func(block *CBlock) {
		fn(block.Block)
	}, nil)
}

func (database *ChainDatabase) SerializeForks(currentHash common.Hash) string {
	database.RW.RLock()
	defer database.RW.RUnlock()

	// Print forks string in a new line
	forkStr := SerializeForks(database.UnConfirmBlocks, currentHash)
	if len(forkStr) == 0 {
		forkStr = "No fork"
		if database.LastConfirm != nil && database.LastConfirm.Block != nil {
			hash := database.LastConfirm.Block.Hash()
			forkStr = fmt.Sprintf("%s. Last stable: [%d]%x", forkStr, database.LastConfirm.Block.Height(), hash[:3])
		}
	}
	return "Print forks\n" + forkStr
}

func (database *ChainDatabase) Close() error {
	database.Beansdb.Close()

	if database.LevelDB != nil {
		database.LevelDB.Close()
		database.LevelDB = nil
	}

	return nil
}
