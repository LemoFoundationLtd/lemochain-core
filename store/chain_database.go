package store

import (
	"bytes"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-go/store/leveldb"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

var max_candidate_count = 20

type ChainDatabase struct {
	LastConfirm     *CBlock
	UnConfirmBlocks map[common.Hash]*CBlock
	Context         *RunContext
	LevelDB         *leveldb.LevelDBDatabase
	Beansdb         *BeansDB
	BizDB           *BizDatabase
	rw              sync.RWMutex
}

func checkHome(home string) error {
	isExist, err := IsExist(home)
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

func NewChainDataBase(home string, driver string, dns string) *ChainDatabase {
	err := checkHome(home)
	if err != nil {
		panic("check home: " + home + "|error: " + err.Error())
	}

	db := &ChainDatabase{
		UnConfirmBlocks: make(map[common.Hash]*CBlock),
		Context:         NewRunContext(home),
		LevelDB:         leveldb.NewLevelDBDatabase(filepath.Join(home, "index"), 16, 16),
	}

	db.BizDB = NewBizDatabase(db, NewMySqlDB(driver, dns), db.LevelDB)
	db.Beansdb = NewBeansDB(home, 2, db.LevelDB, db.AfterScan)

	db.LastConfirm = NewGenesisBlock(db.Context.GetStableBlock(), db.Beansdb)
	candidates, err := db.Context.Candidates.GetCandidates()
	if err != nil {
		panic("get candidates err: " + err.Error())
	} else {
		db.LastConfirm.Top.Rank(max_candidate_count, candidates)
	}

	return db
}

func (database *ChainDatabase) GetLastConfirm() *CBlock {
	return database.LastConfirm
}

func (database *ChainDatabase) AfterScan(flag uint, key []byte, val []byte) error {
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
	log.Error("block commit : " + hash.Hex())
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

	batch := database.Beansdb.NewBatch(hash[:])

	// store block
	buf, err := rlp.EncodeToBytes(cItem.Block)
	if err != nil {
		return err
	}

	batch.Put(CACHE_FLG_BLOCK, leveldb.GetBlockHashKey(hash), buf)
	batch.Put(CACHE_FLG_BLOCK_HEIGHT, leveldb.GetCanonicalKey(cItem.Block.Height()), hash[:])

	// store account
	decode := func(account *types.AccountData, batch Batch) error {
		buf, err = rlp.EncodeToBytes(account)
		if err != nil {
			return err
		} else {
			batch.Put(CACHE_FLG_ACT, leveldb.GetAddressKey(account.Address), buf)
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
		err := database.Context.SetCandidates(candidates)
		if err != nil {
			return err
		}

		database.Context.SetStableBlock(cItem.Block)
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

	val, err := database.Beansdb.Get(leveldb.GetBlockHashKey(hash))
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, ErrNotExist
	}

	var block types.Block
	err = rlp.DecodeBytes(val, &block)
	if err != nil {
		return nil, err
	} else {
		return &block, nil
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
		return database.Beansdb.Put(CACHE_FLG_BLOCK, hash[:], leveldb.GetBlockHashKey(hash), buf)
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
	val, err := database.Beansdb.Get(hash[:])
	if err != nil {
		return -1, err
	} else {
		return len(val), nil
	}
}

func (database *ChainDatabase) GetBlock(hash common.Hash, height uint32) (*types.Block, error) {
	database.rw.Lock()
	defer database.rw.Unlock()

	block, err := database.getBlock(hash)
	if err != nil {
		return nil, err
	} else {
		if block.Height() != height {
			return nil, ErrNotExist
		} else {
			return block, nil
		}
	}
}

func (database *ChainDatabase) GetBlockByHeight(height uint32) (*types.Block, error) {
	database.rw.Lock()
	defer database.rw.Unlock()

	val, err := database.Beansdb.Get(leveldb.GetCanonicalKey(height))
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, ErrNotExist
	}

	block, err := database.getBlock(common.BytesToHash(val))
	if err != nil {
		return nil, err
	} else {
		if block.Height() != height {
			return nil, ErrNotExist
		} else {
			return block, nil
		}
	}
}

func (database *ChainDatabase) GetBlockByHash(hash common.Hash) (*types.Block, error) {
	database.rw.Lock()
	defer database.rw.Unlock()

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

	return database.Beansdb.Has(leveldb.GetBlockHashKey(hash))
}

func (database *ChainDatabase) IsExistByHash(hash common.Hash) (bool, error) {
	database.rw.Lock()
	defer database.rw.Unlock()

	return database.isExistByHash(hash)
}

func (database *ChainDatabase) SetBlock(hash common.Hash, block *types.Block) error {
	database.rw.Lock()
	defer database.rw.Unlock()

	isExist, err := database.isExistByHash(hash)
	if err != nil {
		return err
	}

	if isExist {
		log.Debug("[store]set block error:the block is exist.hash:%s", hash.String())
		return ErrExist
	}

	if database.LastConfirm.Block == nil {
		if (block.Height() != 0) || (block.ParentHash() != common.Hash{}) {
			panic("(database.LastConfirm.Block == nil) && (block.Height() != 0) && (block.ParentHash() != common.Hash{})")
		}
	} else {
		if (block.Height() == 0) || (block.ParentHash() == common.Hash{}) {
			panic("(block.Height() == 0) || (block.ParentHash() == common.Hash{})")
		}

		if block.Height() <= database.LastConfirm.Block.Height() {
			bheight := strconv.Itoa(int(block.Height()))
			bhash := block.Hash().Hex()
			lheight := strconv.Itoa(int(database.LastConfirm.Block.Height()))
			lhash := database.LastConfirm.Block.Hash().Hex()
			panic("block'height:" + bheight + "|bhash:" + bhash + "|confirm'height:" + lheight + "|lhash:" + lhash)
		}
	}

	// genesis block
	if (block.ParentHash() == common.Hash{}) {
		database.UnConfirmBlocks[hash] = NewGenesisBlock(block, database.Beansdb)
		return nil
	}

	pHash := block.Header.ParentHash
	pBlock := database.UnConfirmBlocks[pHash]
	if pBlock == nil {
		if database.LastConfirm.Block.Header.Hash() != pHash {
			bheight := strconv.Itoa(int(block.Height()))
			bhash := block.Hash().Hex()
			lheight := strconv.Itoa(int(database.LastConfirm.Block.Height()))
			lhash := database.LastConfirm.Block.Hash().Hex()
			panic("block'height:" + bheight + "|bhash:" + bhash + "|confirm'height:" + lheight + "|lhash:" + lhash)
		}

		if database.LastConfirm.Block.Height()+1 != block.Height() {
			bheight := strconv.Itoa(int(block.Height()))
			bhash := block.Hash().Hex()
			lheight := strconv.Itoa(int(database.LastConfirm.Block.Height()))
			lhash := database.LastConfirm.Block.Hash().Hex()
			panic("block'height:" + bheight + "|bhash:" + bhash + "|confirm'height:" + lheight + "|lhash:" + lhash)
		}

		database.UnConfirmBlocks[hash] = NewNormalBlock(block, database.LastConfirm.AccountTrieDB, database.LastConfirm.CandidateTrieDB, database.LastConfirm.Top)
	} else {
		if pBlock.Block.Height()+1 != block.Height() {
			bheight := strconv.Itoa(int(block.Height()))
			bhash := block.Hash().Hex()
			lheight := strconv.Itoa(int(database.LastConfirm.Block.Height()))
			lhash := database.LastConfirm.Block.Hash().Hex()
			panic("block'height:" + bheight + "|bhash:" + bhash + "|confirm'height:" + lheight + "|lhash:" + lhash)
		}

		database.UnConfirmBlocks[hash] = NewNormalBlock(block, pBlock.AccountTrieDB, pBlock.CandidateTrieDB, pBlock.Top)
	}

	return nil
}

func (database *ChainDatabase) appendConfirm(block *types.Block, confirms []types.SignData) {
	if (block == nil) || (confirms == nil) {
		return
	}

	if block.Confirms == nil {
		block.SetConfirms(confirms)
	} else {
		for i := 0; i < len(confirms); i++ {
			j := 0
			for ; j < len(block.Confirms); j++ {
				if bytes.Compare(confirms[i][:], block.Confirms[j][:]) == 0 {
					break
				}
			}

			if j == len(block.Confirms) {
				block.Confirms = append(block.Confirms, confirms[i])
			}
		}
	}
}

func (database *ChainDatabase) setConfirm(hash common.Hash, confirms []types.SignData) error {
	item := database.UnConfirmBlocks[hash]
	if (item != nil) && (item.Block != nil) {
		database.appendConfirm(item.Block, confirms)
		return nil
	} else {
		block, err := database.getBlock4DB(hash)
		if err != nil {
			return err
		} else {
			database.appendConfirm(block, confirms)
			return database.setBlock2DB(hash, block)
		}
	}
}

// 设置区块的确认信息 每次收到一个
func (database *ChainDatabase) SetConfirm(hash common.Hash, signData types.SignData) error {
	database.rw.Lock()
	defer database.rw.Unlock()

	confirms := make([]types.SignData, 0)
	confirms = append(confirms, signData)
	return database.setConfirm(hash, confirms)
}

func (database *ChainDatabase) SetConfirms(hash common.Hash, pack []types.SignData) error {
	database.rw.Lock()
	defer database.rw.Unlock()

	return database.setConfirm(hash, pack)
}

func (database *ChainDatabase) GetConfirms(hash common.Hash) ([]types.SignData, error) {
	database.rw.Lock()
	defer database.rw.Unlock()

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

func (database *ChainDatabase) SetStableBlock(hash common.Hash) error {
	database.rw.Lock()
	defer database.rw.Unlock()

	log.Error("set stable block : " + hash.Hex())
	cItem := database.UnConfirmBlocks[hash]
	if cItem == nil {
		panic("set stable block error:the block is not exist. hash:" + hash.Hex())
	}

	blocks := make([]*CBlock, 0)
	blocks = append(blocks, cItem)
	collected := func() []*CBlock {
		for {
			cParent := cItem.Block.ParentHash()
			cItem = database.UnConfirmBlocks[cParent]
			if (cItem != nil) && (cItem.Block != nil) {
				blocks = append(blocks, cItem)
			} else {
				break
			}
		}

		return blocks
	}

	confirm := func(item *CBlock) {
		last := database.LastConfirm
		if last == nil || last.Block == nil || last.AccountTrieDB == nil {
			database.LastConfirm = item
		} else {
			// last.Trie.DelDye(last.Block.Height())
			database.LastConfirm = item
		}
	}

	commit := func() error {
		for index := len(blocks) - 1; index >= 0; index-- {
			cItem := blocks[index]
			err := database.blockCommit(cItem.Block.Hash())
			if err != nil {
				return err
			} else {
				confirm(cItem)
			}
		}

		return nil
	}

	clear := func(max uint32) {
		for k, v := range database.UnConfirmBlocks {
			if v.Block.Height() <= database.LastConfirm.Block.Height() {
				// v.Trie.DelDye(v.Block.Height())
				delete(database.UnConfirmBlocks, k)
			}
		}
	}

	blocks = collected()
	err := commit()
	if err != nil {
		return err
	} else {
		clear(blocks[len(blocks)-1].Block.Height())
		return nil
	}
}

// GetAccount loads account from cache or db
func (database *ChainDatabase) GetAccount(addr common.Address) (*types.AccountData, error) {
	val, err := database.Beansdb.Get(leveldb.GetAddressKey(addr))
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, ErrNotExist
	}

	var account types.AccountData
	err = rlp.DecodeBytes(val, &account)
	if err != nil {
		return nil, err
	} else {
		return &account, nil
	}
}

func (database *ChainDatabase) GetTrieDatabase() *TrieDatabase {
	return NewTrieDatabase(NewLDBDatabase(database.Beansdb))
}

func (database *ChainDatabase) GetActDatabase(hash common.Hash) (*AccountTrieDB, error) {
	if (hash == common.Hash{}) {
		return NewAccountTrieDB(NewEmptyDatabase(), database.Beansdb), nil
	}

	item := database.UnConfirmBlocks[hash]
	if item == nil {
		_, err := database.getBlock4DB(hash)
		if err == ErrNotExist {
			panic("the block not exist. check the block is set.")
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

// func (database *ChainDatabase) GetBizDatabase() BizDb {
// 	return database.BizDB
// }

// GetContractCode loads contract's code from db.
func (database *ChainDatabase) GetContractCode(hash common.Hash) (types.Code, error) {
	val, err := database.Beansdb.Get(hash[:])
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
	return database.Beansdb.Put(CACHE_FLG_CODE, hash[:], hash[:], code[:])
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

func (database *ChainDatabase) CandidatesRanking(hash common.Hash) {
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

	cItem.Ranking()
}

func (database *ChainDatabase) GetAssetCode(code common.Hash) (common.Address, error) {
	return leveldb.GetAssetCode(database.LevelDB, code)
}

func (database *ChainDatabase) GetAssetID(id common.Hash) (common.Address, error) {
	return leveldb.GetAssetID(database.LevelDB, id)
}

func (database *ChainDatabase) Close() error {
	if database.LevelDB != nil {
		database.LevelDB.Close()
		database.LevelDB = nil
	}

	database.Beansdb.Close()

	return nil
}
