package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
)

type sBlock struct {
	Header      *types.Header
	Txs         []*types.Transaction
	ChangeLogs  []*types.ChangeLog
	Events      []*types.Event
	Confirms    []types.SignData
	DeputyNodes deputynode.DeputyNodes
}

type CacheChain struct {
	ConfirmNum int64
	Blocks     map[common.Hash]*types.Block
	Accounts   map[common.Hash]map[common.Address]*types.AccountData
	LmDataBase *LmDataBase
}

func btoSb(block *types.Block) (*sBlock, error) {
	if (block == nil) || (block.Header == nil) {
		return nil, ErrArgInvalid
	}

	sb := &sBlock{
		Header:      block.Header,
		Txs:         block.Txs,
		ChangeLogs:  block.ChangeLogs,
		Events:      block.Events,
		Confirms:    block.ConfirmPackage,
		DeputyNodes: block.DeputyNodes,
	}

	if block.ConfirmPackage == nil {
		sb.Confirms = make([]types.SignData, 0)
	}

	return sb, nil
}

func sBtoB(sb *sBlock) (*types.Block, error) {
	if (sb == nil) || (sb.Header == nil) {
		return nil, ErrArgInvalid
	}

	block := &types.Block{}
	block.SetHeader(sb.Header)
	block.SetTxs(sb.Txs)
	block.SetChangeLogs(sb.ChangeLogs)
	block.SetEvents(sb.Events)
	block.SetConfirmPackage(sb.Confirms)
	block.SetDeputyNodes(sb.DeputyNodes)

	return block, nil
}

func NewCacheChain(path string) (*CacheChain, error) {
	cacheChain := &CacheChain{}
	lmDataBase, err := NewLmDataBase(path)
	if err != nil {
		return nil, err
	}

	cacheChain.LmDataBase = lmDataBase
	cacheChain.ConfirmNum = -1
	cacheChain.Blocks = make(map[common.Hash]*types.Block, 65536)
	cacheChain.Accounts = make(map[common.Hash]map[common.Address]*types.AccountData, 1024)
	return cacheChain, nil
}

func (chain *CacheChain) setBlock(hash common.Hash, block *types.Block) error {
	sb, err := btoSb(block)
	if err != nil {
		return err
	}

	buf, err := rlp.EncodeToBytes(sb)
	if err != nil {
		return err
	}

	err = chain.LmDataBase.Put(hash.Bytes(), buf)
	if err != nil {
		return err
	} else {
		indexHash := encodeBlockNumber2Hash(block.Height())
		if err != nil {
			return err
		} else {
			return chain.LmDataBase.Put(indexHash.Bytes(), hash.Bytes())
		}
	}
}

func (chain *CacheChain) setGeneric(hash common.Hash, block *types.Block) error {
	err := chain.setBlock(hash, block)
	if err != nil {
		return err
	} else {
		chain.Blocks[hash] = block
		return nil
	}
}

func (chain *CacheChain) writeChain(hash common.Hash) error {
	block := chain.Blocks[hash]
	if (block == nil) || (int64(block.Height()) <= chain.ConfirmNum) {
		return nil
	}

	sb, err := btoSb(block)
	if err != nil {
		return err
	}

	buf, err := rlp.EncodeToBytes(sb)
	if err != nil {
		return err
	}

	err = rlp.DecodeBytes(buf, sb)

	err = chain.LmDataBase.SetCurrentBlock(buf)
	if err != nil {
		return err
	}

	allA := make(map[common.Address]*types.AccountData)
	allB := make(map[common.Hash]*types.Block)
	for {
		block := chain.Blocks[hash]
		if (block == nil) || (int64(block.Height()) <= chain.ConfirmNum) {
			break
		}

		if allB[hash] == nil {
			allB[hash] = block
		}

		accounts := chain.Accounts[hash]
		if accounts != nil {
			for k, v := range accounts {
				if allA[k] == nil {
					allA[k] = v
				}
			}
			delete(chain.Accounts, hash)
		}

		delete(chain.Blocks, hash)
		hash = block.ParentHash()
	}

	items := make([]*BatchItem, 10000)
	index := 0
	for _, v := range allA {
		item := new(BatchItem)
		item.Key = v.Address.Bytes()

		val, err := rlp.EncodeToBytes(v)
		if err != nil {
			return err
		} else {
			var act types.AccountData
			err = rlp.DecodeBytes(val, &act)
			if err != nil {
				//
			}
			item.Val = val
		}
		items[index] = item
		index = index + 1
	}

	for _, v := range allB {
		item1 := new(BatchItem)
		item1.Key = v.Hash().Bytes()

		sb, err := btoSb(v)
		if err != nil {
			return err
		}

		val, err := rlp.EncodeToBytes(sb)
		if err != nil {
			return err
		} else {
			item1.Val = val
		}

		items[index] = item1
		index = index + 1

		height := v.Height()
		hash := v.Hash()
		item2 := new(BatchItem)
		item2.Key = encodeBlockNumber2Hash(height).Bytes()
		item2.Val = hash.Bytes()
		items[index] = item2
		index = index + 1
	}

	return chain.LmDataBase.Commit(items)
}

func (chain *CacheChain) mergeSign(src []types.SignData, dst []types.SignData) []types.SignData {
	set := make(map[string]bool, len(src)+len(dst))
	result := make([]types.SignData, 0)
	for index1 := 0; index1 < len(src); index1++ {
		if !set[string(src[index1][:])] {
			result = append(result, src[index1])
			set[string(src[index1][:])] = true
		}
	}

	for index2 := 0; index2 < len(dst); index2++ {
		if !set[string(src[index2][:])] {
			result = append(result, src[index2])
			set[string(src[index2][:])] = true
		}
	}

	return result
}

func (chain *CacheChain) getBlock(hash common.Hash) (*types.Block, error) {
	if (hash == common.Hash{}) {
		log.Errorf("[store]GET BLOCK FROM CACHE ERROR.%s", hash.String())
		return nil, ErrNotExist
	}

	block := chain.Blocks[hash]
	if block != nil {
		return block, nil
	}

	val, err := chain.LmDataBase.Get(hash.Bytes())
	if err != nil {
		log.Errorf("[store]GET BLOCK FROM CACHE ERROR.HASH：%s, ERR:%s", hash.String(), err.Error())
		return nil, err
	} else {
		var sb sBlock
		err = rlp.DecodeBytes(val, &sb)
		if err != nil {
			fmt.Println("[store]GET BLOCK FROM CACHE ERROR.HASH：", fmt.Sprintf("[%s][%s]", hash.Hex(), err.Error()))
			return nil, err
		} else {
			block, err := sBtoB(&sb)
			if err != nil {
				fmt.Println("[store]GET BLOCK FROM CACHE ERROR.HASH：", fmt.Sprintf("[%s][%s]", hash.Hex(), err.Error()))
				return nil, err
			} else {
				chain.Blocks[hash] = block
				return block, nil
			}
		}
	}
}

// 设置区块
func (chain *CacheChain) SetBlock(hash common.Hash, block *types.Block) error {
	header := block.Header
	isExist, err := chain.IsExistByHash(hash)
	if err != nil {
		return err
	}

	if isExist {
		log.Errorf("[store]set block error:the block is exist.hash:%s", hash.String())
		return ErrExist
	}

	parent := header.ParentHash
	if (parent == common.Hash{}) {
		log.Errorf("[store]INSERT BLOCK TO CACHE.HASH：%s", hash.String())
		chain.Blocks[hash] = block
	} else {
		_, err = chain.getBlock(parent)
		if err == ErrNotExist {
			log.Errorf("the block's parent is not exist.")
			return ErrAncestorsNotExist
		} else if err != nil {
			log.Errorf("get block's parent error.%s", err.Error())
			return err
		}

		accounts := chain.Accounts[parent]
		if len(accounts) != 0 {
			chain.Accounts[hash] = chain.cloneAccounts(accounts)
		}

		log.Infof("[store]INSERT BLOCK TO CACHE.HASH：%s", hash.String())
		chain.Blocks[hash] = block
	}

	return nil
}

// 获取区块 优先根据hash与height同时获取，若hash为空则根据Height获取 获取不到返回：nil,原因
func (chain *CacheChain) GetBlock(hash common.Hash, height uint32) (*types.Block, error) {
	block, err := chain.getBlock(hash)
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

func (chain *CacheChain) GetBlockByHeight(height uint32) (*types.Block, error) {
	indexHash := encodeBlockNumber2Hash(height)

	val, err := chain.LmDataBase.Get(indexHash.Bytes())
	if err != nil {
		return nil, err
	} else {
		return chain.GetBlockByHash(common.BytesToHash(val))
	}
}

func (chain *CacheChain) GetBlockByHash(hash common.Hash) (*types.Block, error) {
	return chain.getBlock(hash)
}

func (chain *CacheChain) IsExistByHash(hash common.Hash) (bool, error) {
	if (hash == common.Hash{}) {
		return false, nil
	}

	block := chain.Blocks[hash]
	if block != nil {
		return true, nil
	}

	val, err := chain.LmDataBase.Get(hash.Bytes())
	if err == ErrNotExist {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	var sb sBlock
	err = rlp.DecodeBytes(val, &sb)
	if err != nil {
		return false, err
	} else {
		block, err := sBtoB(&sb)
		if err != nil {
			return false, err
		} else {
			chain.Blocks[hash] = block
			return true, nil
		}
	}
}

// 设置区块的确认信息 每次收到一个
func (chain *CacheChain) SetConfirmInfo(hash common.Hash, signData types.SignData) error {
	block := chain.Blocks[hash]
	if block == nil {
		log.Errorf("set confirm error:the block is not exist. hash: %s", hash.String())
		return ErrNotExist
	}

	confirms := block.ConfirmPackage
	if confirms == nil {
		confirms = make([]types.SignData, 0)
	}

	for index := 0; index < len(confirms); index++ {
		if bytes.Equal(confirms[index][:], signData[:]) {
			return nil
		}
	}

	confirms = append(confirms, signData)
	block.SetConfirmPackage(confirms)
	return nil
}

func (chain *CacheChain) SetConfirmPackage(hash common.Hash, pack []types.SignData) error {
	block, err := chain.GetBlockByHash(hash)
	if err != nil {
		return err
	}

	if block != nil {
		block.SetConfirmPackage(pack)
		err = chain.setBlock(hash, block)
		if err != nil {
			return err
		}
	}

	if err == ErrNotExist {
		block = chain.Blocks[hash]
		if block != nil {
			block.SetConfirmPackage(pack)
		}
	}

	return nil
}

func (chain *CacheChain) AppendConfirmInfo(hash common.Hash, signData types.SignData) error {
	delete(chain.Blocks, hash)

	val, err := chain.LmDataBase.Get(hash.Bytes())
	if err != nil {
		return err
	} else {
		var sb sBlock
		err = rlp.DecodeBytes(val, &sb)
		if err != nil {
			return err
		} else {
			sb.Confirms = append(sb.Confirms, signData)
			val, err := rlp.EncodeToBytes(sb)
			if err != nil {
				return err
			} else {
				return chain.LmDataBase.Put(hash.Bytes(), val)
			}
		}
	}
}

func (chain *CacheChain) AppendConfirmPackage(hash common.Hash, pack []types.SignData) error {
	delete(chain.Blocks, hash)

	val, err := chain.LmDataBase.Get(hash.Bytes())
	if err != nil {
		return err
	} else {
		var sb sBlock
		err = rlp.DecodeBytes(val, &sb)
		if err != nil {
			return err
		} else {
			sb.Confirms = pack
			val, err := rlp.EncodeToBytes(sb)
			if err != nil {
				return err
			} else {
				return chain.LmDataBase.Put(hash.Bytes(), val)
			}
		}
	}
}

// 获取区块的确认包 获取不到返回：nil,原因
func (chain *CacheChain) GetConfirmPackage(hash common.Hash) ([]types.SignData, error) {
	block, err := chain.GetBlockByHash(hash)
	if err != nil {
		return nil, err
	} else {
		return block.ConfirmPackage, nil
	}
}

func encodeBlockNumber2Hash(number uint32) common.Hash {
	enc := make([]byte, 4)
	binary.BigEndian.PutUint32(enc, number)

	prefix := []byte("height-hash-")
	hash := append(prefix, enc...)
	return common.BytesToHash(hash)
}

func (chain *CacheChain) LoadLatestBlock() (*types.Block, error) {
	val := chain.LmDataBase.CurrentBlock()
	if val == nil {
		return nil, ErrNotExist
	} else {
		var sb sBlock
		err := rlp.DecodeBytes(val, &sb)
		if err != nil {
			return nil, err
		} else {
			return sBtoB(&sb)
		}
	}
}

// 区块得到共识
func (chain *CacheChain) SetStableBlock(hash common.Hash) error {
	block := chain.Blocks[hash]
	if block == nil {
		log.Errorf("set stable block error:the block is not exist. hash: %s", hash.String())
		return ErrNotExist
	}

	err := chain.writeChain(hash)
	if err != nil {
		return err
	}

	chain.ConfirmNum = int64(block.Height())

	return nil
}

func (chain *CacheChain) getAccount(address common.Address) (*types.AccountData, error) {
	val, err := chain.LmDataBase.Get(address.Bytes())
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

// GetAccount loads account from cache or db
func (chain *CacheChain) GetAccount(blockHash common.Hash, address common.Address) (*types.AccountData, error) {
	if (blockHash == common.Hash{}) {
		return nil, ErrNotExist
	}

	if chain.Accounts != nil {
		accounts := chain.Accounts[blockHash]
		if accounts != nil && len(accounts) > 0 {
			account := accounts[address]
			if account != nil {
				return account, nil
			}
		}
	}

	return chain.getAccount(address)
}

func (chain *CacheChain) GetCanonicalAccount(address common.Address) (*types.AccountData, error) {
	return chain.getAccount(address)
}

func (chain *CacheChain) cloneAccounts(src map[common.Address]*types.AccountData) map[common.Address]*types.AccountData {
	if len(src) <= 0 {
		return make(map[common.Address]*types.AccountData)
	} else {
		dst := make(map[common.Address]*types.AccountData)
		for k, v := range src {
			dst[k] = v.Copy()
		}

		return dst
	}
}

// SetAccounts saves dirty accounts generated by a block
func (chain *CacheChain) SetAccounts(blockHash common.Hash, accounts []*types.AccountData) error {
	block := chain.Blocks[blockHash]
	if block == nil {
		log.Errorf("set accounts error:this block is not exist.")
		return ErrNotExist
	}

	parentHash := block.ParentHash()
	if (parentHash == common.Hash{}) && (block.Height() == 0) {
		tmp := make(map[common.Address]*types.AccountData)
		for index := 0; index < len(accounts); index++ {
			address := accounts[index].Address
			tmp[address] = accounts[index]
		}
		chain.Accounts[blockHash] = tmp
	} else {
		parentBlock := chain.Blocks[parentHash]
		if parentBlock == nil {
			parentBlock, err := chain.getBlock(parentHash)
			if err != nil {
				return err
			}

			if parentBlock == nil {
				return ErrAncestorsNotExist
			}

			tmp := make(map[common.Address]*types.AccountData)
			for index := 0; index < len(accounts); index++ {
				address := accounts[index].Address
				tmp[address] = accounts[index]
			}
			chain.Accounts[blockHash] = tmp
		} else {
			oldAccounts := chain.Accounts[parentHash]
			if oldAccounts == nil {
				chain.Accounts[blockHash] = make(map[common.Address]*types.AccountData)
			} else {
				chain.Accounts[blockHash] = chain.cloneAccounts(oldAccounts)
			}

			for index := 0; index < len(accounts); index++ {
				address := accounts[index].Address
				chain.Accounts[blockHash][address] = accounts[index]
			}
		}
	}

	return nil
}

func (chain *CacheChain) DelAccount(address common.Address) error {
	return chain.LmDataBase.Delete(address.Bytes())
}

// OpenStorageTrie opens the storage trie of an account.
func (chain *CacheChain) GetTrieDatabase() *TrieDatabase {
	db := NewLDBDatabase(chain.LmDataBase, 256, 256)
	return NewTrieDatabase(db)
}

// GetContractCode loads contract's code from db.
func (chain *CacheChain) GetContractCode(codeHash common.Hash) (types.Code, error) {
	val, err := chain.LmDataBase.Get(codeHash.Bytes())
	if err != nil {
		return nil, err
	} else {
		var code types.Code = val
		return code, nil
	}
}

// SetContractCode saves contract's code
func (chain *CacheChain) SetContractCode(codeHash common.Hash, code types.Code) error {
	return chain.LmDataBase.Put(codeHash.Bytes(), code[:])
}

func (chain *CacheChain) Close() error {
	return nil
}
