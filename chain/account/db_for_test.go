package account

import (
	"bytes"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"github.com/LemoFoundationLtd/lemochain-go/store/trie"
	"math/big"
	"os"
	"path/filepath"
)

type blockInfo struct {
	hash        common.Hash
	versionRoot common.Hash
	time        *big.Int
}

var (
	defaultBlocks     = make([]*types.Block, 0)
	newestBlock       = new(types.Block)
	defaultBlockInfos = []blockInfo{
		// genesis block
		{
			hash:        common.HexToHash("0xac6efe56783d22fa7542abc82202e7c0b7bcaaf611b3bcdb217e23a9366f6677"),
			versionRoot: common.HexToHash("0xac5efb21e3de5900ef965fcfca8bd43c4e84e22d1b66bb5bf3d8418c976a853c"),
			time:        big.NewInt(1538209751),
		},
		// block 1 is stable block
		{
			hash:        common.HexToHash("0x0c38287d372f3f580f1f8b019433ff8070552af423e0fdbb93de8332430cc941"),
			versionRoot: common.HexToHash("0xac5efb21e3de5900ef965fcfca8bd43c4e84e22d1b66bb5bf3d8418c976a853c"),
			time:        big.NewInt(1538209755),
		},
		// block 2 is not stable block
		{
			hash:        common.HexToHash("0x5337bafc55b9f0e8deefb6d4b1061af787aba6e7a9b2f1bad9c4e31d39d16ca2"),
			versionRoot: common.HexToHash("0xac5efb21e3de5900ef965fcfca8bd43c4e84e22d1b66bb5bf3d8418c976a853c"),
			time:        big.NewInt(1538209758),
		},
	}
	// this account data is written with genesis block
	defaultAccounts = []*types.AccountData{
		{
			Address:     common.HexToAddress("0x10000"),
			Balance:     big.NewInt(100),
			CodeHash:    common.HexToHash("0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e"),
			StorageRoot: common.HexToHash("0xcbeb7c7e36b846713bc99b8fa527e8d552e31bfaa1ac0f2b773958cda3aba3ed"),
			NewestRecords: map[types.ChangeLogType]types.VersionRecord{
				BalanceLog: {Version: 100, Height: 1},
				CodeLog:    {Version: 101, Height: 2},
			},
			TxHashList: []common.Hash{common.HexToHash("0x11"), common.HexToHash("0x22")},
		},
	}
	defaultCodes = []struct {
		hash common.Hash
		code *types.Code
	}{
		{
			hash: common.HexToHash("0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e"),
			code: &types.Code{12, 34},
		},
	}
	defaultStorage = []struct {
		key   common.Hash
		value []byte
	}{
		{
			key:   k(10000),
			value: []byte{11},
		},
		{
			key:   k(10001),
			value: []byte{22, 33, 44},
		},
	}
	emptyTrieRoot = common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
)

func init() {
	// clear db
	filepath.Walk(store.GetStorePath(), func(path string, f os.FileInfo, err error) error {
		return os.RemoveAll(path)
	})
}

// newDB creates db for test account module
func newDB() protocol.ChainDB {
	db, err := store.NewCacheChain(store.GetStorePath())
	if err != nil {
		panic(err)
	}

	for i, _ := range defaultBlockInfos {
		// use pointer for repairing incorrect hash
		saveBlock(db, i, &defaultBlockInfos[i])
	}
	saveAccount(db)
	err = db.SetStableBlock(defaultBlockInfos[1].hash)
	if err != nil {
		panic(err)
	}
	testStorageTrieGet(db)
	return db
}

func saveBlock(db protocol.ChainDB, blockIndex int, info *blockInfo) {
	// version trie
	trieDB := db.GetTrieDatabase()
	tr, err := trie.NewSecure(common.Hash{}, trieDB, MaxTrieCacheGen)
	if err != nil {
		panic(err)
	}
	for _, account := range defaultAccounts {
		addr := account.Address.Bytes()
		for logType, record := range account.NewestRecords {
			k := append(addr, big.NewInt(int64(logType)).Bytes()...)
			err = tr.TryUpdate(k, big.NewInt(int64(record.Version)).Bytes())
			if err != nil {
				panic(err)
			}
		}
	}
	hash, err := tr.Commit(nil)
	if err != nil {
		panic(err)
	}
	if hash != info.versionRoot {
		fmt.Printf("%d version root error. except: %s, got: %s\n", blockIndex, info.versionRoot.Hex(), hash.Hex())
		info.versionRoot = hash
	}
	err = trieDB.Commit(hash, false)
	if err != nil {
		panic(err)
	}
	// header
	header := &types.Header{
		VersionRoot: info.versionRoot,
		Height:      uint32(blockIndex),
		Time:        info.time,
	}
	if blockIndex > 0 {
		header.ParentHash = defaultBlockInfos[blockIndex-1].hash
	}
	blockHash := header.Hash()
	if blockHash != info.hash {
		fmt.Printf("%d block hash error. except: %s, got: %s\n", blockIndex, info.hash.Hex(), blockHash.Hex())
		info.hash = blockHash
	}
	// block
	block := &types.Block{}
	block.SetHeader(header)
	defaultBlocks = append(defaultBlocks, block)
	newestBlock = block
	err = db.SetBlock(info.hash, block)
	if err != nil && err != store.ErrExist {
		panic(err)
	}
}

func saveAccount(db protocol.ChainDB) {
	trieDB := db.GetTrieDatabase()
	// save account (to db cache, not to file)
	err := db.SetAccounts(defaultBlockInfos[0].hash, defaultAccounts)
	if err != nil {
		panic(err)
	}
	// save code
	for _, codeInfo := range defaultCodes {
		hash := crypto.Keccak256Hash(*codeInfo.code)
		if hash != codeInfo.hash {
			panic(fmt.Errorf("code hash error. except: %s, got: %s", codeInfo.hash.Hex(), hash.Hex()))
		}
		err = db.SetContractCode(codeInfo.hash, codeInfo.code)
		if err != nil {
			panic(err)
		}
	}
	// save contract storage (to file)
	tr, err := trie.NewSecure(common.Hash{}, trieDB, MaxTrieCacheGen)
	if err != nil {
		panic(err)
	} else {
		for _, info := range defaultStorage {
			v := bytes.TrimLeft(info.value, "\x00")
			err = tr.TryUpdate(info.key[:], v)
			if err != nil {
				panic(err)
			}
		}
		hash, err := tr.Commit(nil)
		if err != nil {
			panic(err)
		}
		if hash != defaultAccounts[0].StorageRoot {
			panic(fmt.Errorf("storage root error. except: %s, got: %s", defaultAccounts[0].StorageRoot.Hex(), hash.Hex()))
		}
		err = trieDB.Commit(hash, false)
		if err != nil {
			panic(err)
		}
	}
}

// testStorageTrieGet creates a new trie to make sure the data is accessible
func testStorageTrieGet(db protocol.ChainDB) {
	value, err := ReadTrie(db, defaultAccounts[0].StorageRoot, defaultStorage[0].key[:])
	if err != nil {
		panic(err)
	} else if bytes.Compare(value, defaultStorage[0].value) != 0 {
		panic(fmt.Errorf("Data has changed! Expect %v, got %v\n", defaultStorage[0].value, value))
	}
}

// h returns hash for test
func h(i int64) common.Hash { return common.HexToHash(fmt.Sprintf("0xa%x", i)) }

// b returns block hash for test
func b(i int64) common.Hash { return common.HexToHash(fmt.Sprintf("0xb%x", i)) }

// c returns code hash for test
func c(i int64) common.Hash { return common.HexToHash(fmt.Sprintf("0xc%x", i)) }

// k returns storage key hash for test
func k(i int64) common.Hash { return common.HexToHash(fmt.Sprintf("0xd%x", i)) }

// t returns transaction hash for test
func th(i int64) common.Hash { return common.HexToHash(fmt.Sprintf("0xe%x", i)) }

func ReadTrie(db protocol.ChainDB, root common.Hash, key []byte) (value []byte, err error) {
	var tr *trie.SecureTrie
	tr, err = trie.NewSecure(root, db.GetTrieDatabase(), MaxTrieCacheGen)
	if err == nil {
		value, err = tr.TryGet(key)
	}
	return
}
