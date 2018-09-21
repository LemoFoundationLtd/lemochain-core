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

var (
	defaultBlock = struct {
		hash        common.Hash
		versionRoot common.Hash
	}{
		hash:        common.HexToHash("0x973d217e0e17173c9ee5b62c225b5caca70116169b9233fcb60686e0a67d9497"),
		versionRoot: common.HexToHash("0x7f9f6d86f2653404d59d39099b5e39eec80cdbcaa263fee5456b87ebc13b37bc"),
	}
	defaultAccounts = []*types.AccountData{
		{
			Address:     common.HexToAddress("0x10000"),
			Balance:     big.NewInt(100),
			Version:     100,
			CodeHash:    common.HexToHash("0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e"),
			StorageRoot: common.HexToHash("0xcbeb7c7e36b846713bc99b8fa527e8d552e31bfaa1ac0f2b773958cda3aba3ed"),
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
	filepath.Walk("../../db", func(path string, f os.FileInfo, err error) error {
		return os.RemoveAll(path)
	})
}

// newDB creates db for test account module
func newDB() protocol.ChainDB {
	db, err := store.NewCacheChain("../../db")
	if err != nil {
		panic(err)
	}

	saveBlock(db)
	saveAccount(db)
	err = db.SetStableBlock(defaultBlock.hash)
	if err != nil {
		panic(err)
	}
	testStorageTrieGet(db)
	return db
}

func saveBlock(db protocol.ChainDB) {
	// block
	block := &types.Block{}
	block.SetHeader(&types.Header{VersionRoot: defaultBlock.versionRoot})
	hash := block.Hash()
	if hash != defaultBlock.hash {
		panic(fmt.Errorf("block hash error. except: %s, got: %s", defaultBlock.hash.Hex(), hash.Hex()))
	}
	err := db.SetBlock(defaultBlock.hash, block)
	if err != nil && err != store.ErrExist {
		panic(err)
	}
	// version trie
	trieDB := db.GetTrieDatabase()
	tr, err := trie.NewSecure(common.Hash{}, trieDB, MaxTrieCacheGen)
	if err != nil {
		panic(err)
	} else {
		for _, account := range defaultAccounts {
			v := bytes.TrimLeft(account.Address.Bytes(), "\x00")
			err = tr.TryUpdate(big.NewInt(int64(account.Version)).Bytes(), v)
			if err != nil {
				panic(err)
			}
		}
		hash, err := tr.Commit(nil)
		if err != nil {
			panic(err)
		}
		if hash != defaultBlock.versionRoot {
			panic(fmt.Errorf("version root error. except: %s, got: %s", defaultBlock.versionRoot.Hex(), hash.Hex()))
		}
		err = trieDB.Commit(hash, false)
		if err != nil {
			panic(err)
		}
	}
}

func saveAccount(db protocol.ChainDB) {
	trieDB := db.GetTrieDatabase()
	// save account (to db cache, not to file)
	err := db.SetAccounts(defaultBlock.hash, defaultAccounts)
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
	tr, err := trie.NewSecure(defaultAccounts[0].StorageRoot, db.GetTrieDatabase(), MaxTrieCacheGen)
	if err != nil {
		panic(err)
	} else {
		value, err := tr.TryGet(defaultStorage[0].key[:])
		if err != nil {
			panic(err)
		} else if bytes.Compare(value, defaultStorage[0].value) != 0 {
			panic(fmt.Errorf("Data has changed! Expect %v, got %v\n", defaultStorage[0].value, value))
		}
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
