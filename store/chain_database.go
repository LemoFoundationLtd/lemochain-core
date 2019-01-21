package store

import (
	"bytes"
	"encoding/binary"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"math/big"
	"os"
	"strconv"
	"sync"
)

type CBlock struct {
	Block *types.Block
	Trie  *PatriciaTrie
	Top30 *VoteTop
}

type ChainDatabase struct {
	LastConfirm     *CBlock
	UnConfirmBlocks map[common.Hash]*CBlock

	Beansdb        *BeansDB
	Context        *RunContext
	CandidatesRank *VoteRank

	rw sync.RWMutex
}

func NewChainDataBase(home string, driver string, dns string) *ChainDatabase {
	isExist, err := IsExist(home)
	if err != nil {
		panic("check home is exist error:" + err.Error())
	}

	if !isExist {
		err = os.MkdirAll(home, os.ModePerm)
		if err != nil {
			panic("mk dir is exist err : " + err.Error())
		}
	}

	db := &ChainDatabase{
		UnConfirmBlocks: make(map[common.Hash]*CBlock),
		Beansdb:         NewBeansDB(home, 2, driver, dns),
		Context:         NewRunContext(home),
	}

	db.LastConfirm = &CBlock{
		Block: db.Context.GetStableBlock(),
		Trie:  NewEmptyDatabase(db.Beansdb),
	}

	db.CandidatesRank = NewVoteRank()
	return db
}

func isCandidate(account *types.AccountData) bool {
	result, ok := account.Candidate.Profile[types.CandidateKeyIsCandidate]
	if !ok {
		return false
	} else {
		val, err := strconv.ParseBool(result)
		if err != nil {
			panic("to bool err : " + err.Error())
		}

		return val
	}
}

/**
 * 1. hash => block (chain)
 * 2. tx => block'hash and pos
 * 3. addr => txs
 * 4. addr => account(chain)
 * 5. height => block's hash(chain)
 */
func (database *ChainDatabase) blockCommit(hash common.Hash) error {
	cItem := database.UnConfirmBlocks[hash]
	if (cItem == nil) || (cItem.Block == nil) {
		// return nil
		panic("item or item'block is nil.")
	}

	if (database.LastConfirm.Block == nil) && (cItem.Block.Height() != 0) {
		panic("database.LastConfirm == nil && cItem.Block.Height() != 0")
	}

	if (database.LastConfirm.Block != nil) &&
		(cItem.Block.Height() < database.LastConfirm.Block.Height()) {
		panic("(database.LastConfirm.Block != nil) && (cItem.Block.Height() < database.LastConfirm.Block.Height())")
		// return nil
	}

	batch := database.Beansdb.NewBatch(hash[:])

	// store block
	buf, err := rlp.EncodeToBytes(cItem.Block)
	if err != nil {
		return err
	}

	batch.Put(CACHE_FLG_BLOCK, hash[:], buf)
	batch.Put(CACHE_FLG_BLOCK_HEIGHT, encodeBlockNumber2Hash(cItem.Block.Height()).Bytes(), hash[:])

	// store account
	decode := func(account *types.AccountData, batch Batch) error {
		buf, err = rlp.EncodeToBytes(account)
		if err != nil {
			return err
		} else {
			batch.Put(CACHE_FLG_ACT, account.Address[:], buf)
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

	filterCandidates := func(accounts []*types.AccountData) {
		for index := 0; index < len(accounts); index++ {
			account := accounts[index]
			if isCandidate(account) && !database.Context.CandidateIsExist(account.Address) {
				database.Context.SetCandidate(account.Address)
			}
		}
	}

	commitContext := func(block *types.Block, accounts []*types.AccountData) error {
		filterCandidates(accounts)
		database.Context.StableBlock = cItem.Block
		return database.Context.Flush()
	}

	accounts := cItem.Trie.Collected(cItem.Block.Height())
	err = decodeBatch(accounts, batch)
	if err != nil {
		return err
	}

	err = database.Beansdb.Commit(batch)
	if err != nil {
		return err
	}

	return commitContext(cItem.Block, accounts)
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

	val, err := database.Beansdb.Get(hash[:])
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
		return database.Beansdb.Put(CACHE_FLG_BLOCK, hash[:], hash[:], buf)
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

	val, err := database.Beansdb.Get(encodeBlockNumber2Hash(height).Bytes())
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

	return database.Beansdb.Has(hash[:])
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

	// genesis block
	if (block.ParentHash() == common.Hash{}) {
		database.UnConfirmBlocks[hash] = &CBlock{
			Block: block,
			Trie:  NewEmptyDatabase(database.Beansdb),
			Top30: NewVoteTop(),
		}
		return nil
	}

	pHash := block.Header.ParentHash
	pBlock := database.UnConfirmBlocks[pHash]
	if pBlock == nil {
		if database.LastConfirm.Block.Header.Hash() != pHash {
			panic("parent block is not exist.")
		} else {
			database.UnConfirmBlocks[hash] = &CBlock{
				Block: block,
				Trie:  NewActDatabase(database.Beansdb, database.LastConfirm.Trie),
				Top30: database.LastConfirm.Top30.Clone(),
			}
		}
	} else {
		database.UnConfirmBlocks[hash] = &CBlock{
			Block: block,
			Trie:  NewActDatabase(database.Beansdb, pBlock.Trie),
			Top30: pBlock.Top30.Clone(),
		}
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

func encodeBlockNumber2Hash(number uint32) common.Hash {
	enc := make([]byte, 4)
	binary.BigEndian.PutUint32(enc, number)

	prefix := []byte("height-hash-")
	hash := append(prefix, enc...)
	return common.BytesToHash(hash)
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

	cItem := database.UnConfirmBlocks[hash]
	if cItem == nil {
		log.Errorf("set stable block error:the block is not exist. hash: %s", hash.String())
		return ErrNotExist
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
		if last == nil || last.Block == nil || last.Trie == nil {
			database.LastConfirm = item
		} else {
			last.Trie.DelDye(last.Block.Height())
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
			if v.Block.Height() < database.LastConfirm.Block.Height() {
				v.Trie.DelDye(v.Block.Height())
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
	val, err := database.Beansdb.Get(addr[:])
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

func (database *ChainDatabase) GetActDatabase(hash common.Hash) *PatriciaTrie {
	item := database.UnConfirmBlocks[hash]
	if item == nil || item.Block == nil || item.Trie == nil {
		if database.LastConfirm == nil {
			return NewEmptyDatabase(database.Beansdb)
		} else {
			return database.LastConfirm.Trie
		}
	} else {
		return item.Trie
	}
}

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
	return database.CandidatesRank.GetTop()
}

func (database *ChainDatabase) CandidatesRanking(hash common.Hash) {
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
		// return nil
	}

	all := func(hash common.Hash) []*Candidate {
		db := database.GetActDatabase(hash)
		result := make([]*Candidate, len(database.Context.Candidates))
		for k, _ := range database.Context.Candidates {
			account := db.Find(k[:])
			result = append(result, &Candidate{
				address: account.Address,
				total:   new(big.Int).Set(account.Candidate.Votes),
			})
		}
		return result
	}

	accounts := cItem.Trie.Collected(cItem.Block.Height())
	if len(accounts) <= 0 {
		return
	} else {
		top30 := cItem.Top30
		lastMinCandidate := top30.Min()
		lastCount := top30.Count()

		lastCandidatesMap := top30.ToHashMap()
		incrCandidates := make([]*Candidate, 0)
		nextCandidates := make([]*Candidate, 0)
		for index := 0; index < len(accounts); index++ {
			account := accounts[index]

			if isCandidate(account) {
				nextCandidates = append(nextCandidates, &Candidate{
					address: account.Address,
					total:   new(big.Int).Set(account.Candidate.Votes),
				})
			}

			// cancle candidate
			_, ok := lastCandidatesMap[account.Address]
			isExist := database.Context.CandidateIsExist(account.Address)
			if isExist && !ok {
				top30.Del(account.Address)
				continue
			}

			// vote chanage
			if ok {
				lastCandidatesMap[account.Address].total.Set(account.Candidate.Votes)
				continue
			}

			// incr
			if !isExist && isCandidate(account) {
				incrCandidates = append(incrCandidates, &Candidate{
					address: account.Address,
					total:   new(big.Int).Set(account.Candidate.Votes),
				})
				continue
			}
		}

		top30.Rank()
		if (lastMinCandidate == nil) ||
			(lastCount > top30.Count()) ||
			(lastMinCandidate.total.Cmp(top30.Min().total) > 0) { // 如果本轮前30有代理节点取消，或前30代理的最少投票值变小，则+本轮新增代理，并全部重新计算
			candidates := all(hash)
			candidates = append(candidates, incrCandidates...)
			database.CandidatesRank.RankAll(candidates)
			cItem.Top30.Reset(database.CandidatesRank.GetTop())
		} else {
			database.CandidatesRank.Rank(nextCandidates) // 否则按照30 + 4000的规则进行计算
			cItem.Top30.Reset(database.CandidatesRank.GetTop())
		}
	}
}

func (database *ChainDatabase) Close() error {
	return nil
}