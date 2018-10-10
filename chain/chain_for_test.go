package chain

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"math/big"
	"os"
	"path/filepath"
)

type blockInfo struct {
	hash        common.Hash
	parentHash  common.Hash
	height      uint32
	author      common.Address
	versionRoot common.Hash
	txRoot      common.Hash
	logsRoot    common.Hash
	txList      []*types.Transaction
	gasLimit    uint64
	time        *big.Int
}

var (
	chainID         uint16 = 200
	testSigner             = types.DefaultSigner{}
	testPrivate, _         = crypto.HexToECDSA("432a86ab8765d82415a803e29864dcfc1ed93dac949abf6f95a583179f27e4bb") // secp256k1.V = 1
	testAddr               = crypto.PubkeyToAddress(testPrivate.PublicKey)                                         // 0x0107134B9CdD7D89F83eFa6175F9b3552F29094c
	defaultAccounts        = []common.Address{
		common.HexToAddress("0x10000"), common.HexToAddress("0x20000"), testAddr,
	}
	defaultBlocks     = make([]*types.Block, 0)
	newestBlock       = new(types.Block)
	defaultBlockInfos = []blockInfo{
		// genesis block must no transactions
		{
			hash:        common.HexToHash("0xc9c3211bce591d47e4af4b598b1d35ddf552d8b34e569a791a484c1875c234cf"),
			height:      0,
			author:      defaultAccounts[0],
			versionRoot: common.HexToHash("0x5a285bcfd4297d959e44cfc857e221695e12b088c5e01ad935e3eb2af62e3bcf"),
			txRoot:      common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"), // empty merkle
			logsRoot:    common.HexToHash("0x74450829ca5c4673011dd95266fbd78de05ec7d4a6bf9a22bc9f98c37823d1de"),
			time:        big.NewInt(1538209751),
		},
		// block 1 is stable block
		{
			hash:        common.HexToHash("0x16c0160ba55378547c0fc6fb892191adabe459b27d6314f44c54aa1e9f69e326"),
			height:      1,
			author:      common.HexToAddress("0x20000"),
			versionRoot: common.HexToHash("0x695330e8317f42ae800b9c98096c698903fd3a9cc33e8228f42f67f30e5b23c8"),
			txRoot:      common.HexToHash("0xf044cc436950ef7470aca61053eb3f1ed46b9dcd501a5210f3673dc657c4fc88"),
			logsRoot:    common.HexToHash("0x5a0783cc3ff00fb0c6a3ce34be8251ccaef6bb1f3acff09322e42c58c51be91c"),
			txList: []*types.Transaction{
				// testAddr -> defaultAccounts[0] 1
				types.NewTransaction(defaultAccounts[0], common.Big1, 2000000, common.Big2, []byte{12}, chainID, big.NewInt(1538210391), "aa", []byte{34}),
				// testAddr -> defaultAccounts[1] 1
				types.NewTransaction(defaultAccounts[1], common.Big1, 2000000, common.Big2, []byte{}, chainID, big.NewInt(1538210491), "", []byte{}),
			},
			gasLimit: 20000000,
			time:     big.NewInt(1538209755),
		},
		// block 2 is not stable block
		{
			hash:        common.HexToHash("0x6de5576b743ac346c019793f24470aa82464cda92aad0509942158251a3035b0"),
			height:      2,
			author:      defaultAccounts[0],
			versionRoot: common.HexToHash("0xb17f070e12aacafe07dabcae8b9333bb660cc55a3e06584a3f5a710b7f0a584a"),
			txRoot:      common.HexToHash("0x4558f847f8314dbc7e9d7d6fc84a9e75286040aa527b2d981f924a2ad75bca81"),
			logsRoot:    common.HexToHash("0xec63d84a5f0e9fc43a05c49e76ff2d291686bfaca48d104b519b3a9984ed5398"),
			txList: []*types.Transaction{
				// testAddr -> defaultAccounts[0] 2
				types.NewTransaction(defaultAccounts[0], common.Big2, 2000000, common.Big2, []byte{12}, chainID, big.NewInt(1538210395), "aa", []byte{34}),
			},
			time:     big.NewInt(1538209758),
			gasLimit: 20000000,
		},
		// block 3 is not store in db
		{
			hash:        common.HexToHash("0x4b17abd7e65b34379d992a2e663b15d38e473126a8161c6e3f84568fb2474c0c"),
			height:      3,
			author:      defaultAccounts[0],
			versionRoot: common.HexToHash("0xfa256aeabd9bc66361aec3e149af6372b725e9cfdcf81de147ceb0ffc8904ab1"),
			txRoot:      common.HexToHash("0xef1ecb2eaa56f8719161d50e5a454b685c7079f686a58e65210be87a03c11bbf"),
			logsRoot:    common.HexToHash("0x80e1dd928e158c40494b1f76595ece38a5790c5894e15f436c5e86bb12700bc8"),
			txList: []*types.Transaction{
				// testAddr -> defaultAccounts[0] 3
				types.NewTransaction(defaultAccounts[0], common.Big2, 2000000, common.Big2, []byte{12}, chainID, big.NewInt(1538210398), "aa", []byte{34}),
				// testAddr -> defaultAccounts[1] 3
				types.NewTransaction(defaultAccounts[0], common.Big2, 3000000, common.Big3, []byte{}, chainID, big.NewInt(1538210425), "", []byte{}),
			},
			time:     big.NewInt(1538209761),
			gasLimit: 20000000,
		},
	}
)

func init() {
	// clear db
	filepath.Walk("../../db", func(path string, f os.FileInfo, err error) error {
		return os.RemoveAll(path)
	})
}

// newChain creates chain for test
func newChain() *BlockChain {
	db := newDB()
	newBlockCh := make(chan *types.Block)
	bc, err := NewBlockChain(uint64(chainID), NewDpovp(10, 3), db, newBlockCh, map[string]string{})
	if err != nil {
		panic(err)
	}
	return bc
}

// newDB creates db for test chain module
func newDB() protocol.ChainDB {
	db, err := store.NewCacheChain("../../db")
	if err != nil {
		panic(err)
	}

	for i, _ := range defaultBlockInfos {
		if i > 0 {
			defaultBlockInfos[i].parentHash = defaultBlocks[i-1].Hash()
		}
		newestBlock = makeBlock(db, defaultBlockInfos[i], i < 3)
		defaultBlocks = append(defaultBlocks, newestBlock)
	}
	err = db.SetStableBlock(defaultBlockInfos[1].hash)
	if err != nil {
		panic(err)
	}
	return db
}

func makeBlock(db protocol.ChainDB, info blockInfo, save bool) *types.Block {
	manager := account.NewManager(info.parentHash, db)
	// sign transactions
	var err error
	var gasUsed uint64 = 0
	for i, tx := range info.txList {
		info.txList[i], err = types.SignTx(tx, testSigner, testPrivate)
		if err != nil {
			panic(err)
		}
	}
	txRoot := types.DeriveTxsSha(info.txList)
	if txRoot != info.txRoot {
		if info.txRoot != (common.Hash{}) {
			fmt.Printf("%d txRoot hash error. except: %s, got: %s\n", info.height, info.txRoot.Hex(), txRoot.Hex())
		}
		info.txRoot = txRoot
	}
	// genesis coin
	if info.height == 0 {
		owner := manager.GetAccount(testAddr)
		owner.SetBalance(big.NewInt(1000000000))
	}
	// account
	salary := new(big.Int)
	for _, tx := range info.txList {
		gas := params.TxGas + params.TxDataNonZeroGas*uint64(len(tx.Data()))
		fromAddr, err := tx.From()
		if err != nil {
			panic(err)
		}
		from := manager.GetAccount(fromAddr)
		fee := new(big.Int).Mul(new(big.Int).SetUint64(gas), tx.GasPrice())
		cost := new(big.Int).Add(tx.Value(), fee)
		to := manager.GetAccount(*tx.To())
		// make sure the change log has right order
		if fromAddr.Hex() < tx.To().Hex() {
			from.SetBalance(new(big.Int).Sub(from.GetBalance(), cost))
			to.SetBalance(new(big.Int).Add(to.GetBalance(), tx.Value()))
		} else {
			to.SetBalance(new(big.Int).Add(to.GetBalance(), tx.Value()))
			from.SetBalance(new(big.Int).Sub(from.GetBalance(), cost))
		}
		gasUsed += gas
		salary.Add(salary, fee)
	}
	miner := manager.GetAccount(info.author)
	miner.SetBalance(new(big.Int).Add(miner.GetBalance(), salary))
	err = manager.Finalise()
	if err != nil {
		panic(err)
	}
	// header
	if manager.GetVersionRoot() != info.versionRoot {
		if info.versionRoot != (common.Hash{}) {
			fmt.Printf("%d version root error. except: %s, got: %s\n", info.height, info.versionRoot.Hex(), manager.GetVersionRoot().Hex())
		}
		info.versionRoot = manager.GetVersionRoot()
	}
	changeLogs := manager.GetChangeLogs()
	// fmt.Printf("%d changeLogs %v\n", height, changeLogs)
	logsRoot := types.DeriveChangeLogsSha(changeLogs)
	if logsRoot != info.logsRoot {
		if info.logsRoot != (common.Hash{}) {
			fmt.Printf("%d change logs root error. except: %s, got: %s\n", info.height, info.logsRoot.Hex(), logsRoot.Hex())
		}
		info.logsRoot = logsRoot
	}
	header := &types.Header{
		ParentHash:  info.parentHash,
		LemoBase:    info.author,
		VersionRoot: info.versionRoot,
		TxRoot:      info.txRoot,
		LogsRoot:    info.logsRoot,
		EventRoot:   common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"), // empty merkle
		Height:      info.height,
		GasLimit:    info.gasLimit,
		GasUsed:     gasUsed,
		Time:        info.time,
		Extra:       []byte{},
	}
	blockHash := header.Hash()
	if blockHash != info.hash {
		if info.hash != (common.Hash{}) {
			fmt.Printf("%d block hash error. except: %s, got: %s\n", info.height, info.hash.Hex(), blockHash.Hex())
		}
		info.hash = blockHash
	}
	// block
	block := &types.Block{
		Txs:       info.txList,
		ChangeLog: changeLogs,
	}
	block.SetHeader(header)
	if save {
		err = db.SetBlock(blockHash, block)
		if err != nil && err != store.ErrExist {
			panic(err)
		}
		err = manager.Save(blockHash)
		if err != nil {
			panic(err)
		}
	}
	return block
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
