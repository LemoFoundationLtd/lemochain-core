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
			author:      defaultAccounts[0],
			versionRoot: common.HexToHash("0x5a285bcfd4297d959e44cfc857e221695e12b088c5e01ad935e3eb2af62e3bcf"),
			txRoot:      common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"), // empty merkle
			logsRoot:    common.HexToHash("0x74450829ca5c4673011dd95266fbd78de05ec7d4a6bf9a22bc9f98c37823d1de"),
			time:        big.NewInt(1538209751),
		},
		// block 1 is stable block
		{
			hash:        common.HexToHash("0x6e1adde19556f281f9406a6a249f389deccaab7ca208c8a96228d6bacec0aee1"),
			author:      common.HexToAddress("0x20000"),
			versionRoot: common.HexToHash("0x695330e8317f42ae800b9c98096c698903fd3a9cc33e8228f42f67f30e5b23c8"),
			txRoot:      common.HexToHash("0xf044cc436950ef7470aca61053eb3f1ed46b9dcd501a5210f3673dc657c4fc88"),
			logsRoot:    common.HexToHash("0x520bb99fc1b8a23b60d8ded2a8b894d8c20364d25c544a0031b247d05f8c2997"),
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
			hash:        common.HexToHash("0xc84ede6118f3ce622ccb5900dc7df6e872bfa9916acbb72f2b50dd67a6421e02"),
			author:      defaultAccounts[0],
			versionRoot: common.HexToHash("0xb17f070e12aacafe07dabcae8b9333bb660cc55a3e06584a3f5a710b7f0a584a"),
			txRoot:      common.HexToHash("0x4558f847f8314dbc7e9d7d6fc84a9e75286040aa527b2d981f924a2ad75bca81"),
			logsRoot:    common.HexToHash("0x9e05a0471008bd9a617bf0679ef50b3ce92c8005c1883e1e48bebca33296e627"),
			txList: []*types.Transaction{
				// testAddr -> defaultAccounts[0] 2
				types.NewTransaction(defaultAccounts[0], common.Big2, 2000000, common.Big2, []byte{12}, chainID, big.NewInt(1538210395), "aa", []byte{34}),
			},
			time:     big.NewInt(1538209758),
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
	testTxRecover(bc)
	return bc
}

// newDB creates db for test chain module
func newDB() protocol.ChainDB {
	db, err := store.NewCacheChain("../../db")
	if err != nil {
		panic(err)
	}

	for i, _ := range defaultBlockInfos {
		// use pointer for repairing incorrect hash
		saveBlock(db, i, &defaultBlockInfos[i])
	}
	err = db.SetStableBlock(defaultBlockInfos[1].hash)
	if err != nil {
		panic(err)
	}
	return db
}

func saveBlock(db protocol.ChainDB, blockIndex int, info *blockInfo) {
	parentHash := common.Hash{}
	if blockIndex > 0 {
		parentHash = defaultBlockInfos[blockIndex-1].hash
	}
	manager := account.NewManager(parentHash, db)
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
		fmt.Printf("%d txRoot hash error. except: %s, got: %s\n", blockIndex, info.txRoot.Hex(), txRoot.Hex())
		info.txRoot = txRoot
	}
	// genesis coin
	if blockIndex == 0 {
		owner := manager.GetAccount(testAddr)
		owner.SetBalance(big.NewInt(1000000000))
	}
	// account
	for _, tx := range info.txList {
		gas := params.TxGas + params.TxDataNonZeroGas*uint64(len(tx.Data()))
		fromAddr, err := tx.From()
		if err != nil {
			panic(err)
		}
		// TODO the real change log is pay limit gas first, then refund the rest after contract execution
		from := manager.GetAccount(fromAddr)
		cost := new(big.Int).Add(tx.Value(), new(big.Int).SetUint64(gas))
		from.SetBalance(new(big.Int).Sub(from.GetBalance(), cost))
		to := manager.GetAccount(*tx.To())
		to.SetBalance(new(big.Int).Add(to.GetBalance(), tx.Value()))
		gasUsed += gas
	}
	miner := manager.GetAccount(info.author)
	miner.SetBalance(new(big.Int).Add(miner.GetBalance(), new(big.Int).SetUint64(gasUsed)))
	err = manager.Finalise()
	if err != nil {
		panic(err)
	}
	// header
	if manager.GetVersionRoot() != info.versionRoot {
		fmt.Printf("%d version root error. except: %s, got: %s\n", blockIndex, info.versionRoot.Hex(), manager.GetVersionRoot().Hex())
		info.versionRoot = manager.GetVersionRoot()
	}
	changeLogs := manager.GetChangeLogs()
	fmt.Printf("%d changeLogs %v\n", blockIndex, changeLogs)
	logsRoot := types.DeriveChangeLogsSha(changeLogs)
	if logsRoot != info.logsRoot {
		fmt.Printf("%d change logs root error. except: %s, got: %s\n", blockIndex, info.logsRoot.Hex(), logsRoot.Hex())
		info.logsRoot = logsRoot
	}
	header := &types.Header{
		LemoBase:    info.author,
		VersionRoot: info.versionRoot,
		TxRoot:      info.txRoot,
		LogsRoot:    info.logsRoot,
		EventRoot:   common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"), // empty merkle
		Height:      uint32(blockIndex),
		GasLimit:    info.gasLimit,
		GasUsed:     gasUsed,
		Time:        info.time,
		Extra:       []byte{},
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
	block := &types.Block{
		Txs:       info.txList,
		ChangeLog: changeLogs,
	}
	block.SetHeader(header)
	defaultBlocks = append(defaultBlocks, block)
	newestBlock = block
	err = db.SetBlock(blockHash, block)
	if err != nil && err != store.ErrExist {
		panic(err)
	}
	err = manager.Save(blockHash)
	if err != nil {
		panic(err)
	}
}

// testTxRecover load block to make sure the data is stored correctly
func testTxRecover(bc *BlockChain) {
	lastBlockInfo := defaultBlockInfos[len(defaultBlockInfos)-1]
	block := bc.GetBlockByHash(lastBlockInfo.hash)
	from, err := block.Txs[0].From()
	if err != nil {
		panic(err)
	}
	if from != testAddr {
		panic(fmt.Errorf("address recover fail! Expect %s, got %s\n", testAddr.Hex(), from.Hex()))
	}
	if block.Txs[0].Value().Cmp(lastBlockInfo.txList[0].Value()) != 0 {
		panic(fmt.Errorf("tx value load fail! Expect %v, got %v\n", lastBlockInfo.txList[0].Value(), block.Txs[0].Value()))
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
