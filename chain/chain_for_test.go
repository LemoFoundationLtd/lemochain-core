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
	versionRoot common.Hash
	txRoot      common.Hash
	logsRoot    common.Hash
	txList      []*types.Transaction
	time        *big.Int
	author      common.Address
}

var (
	chainID         uint16 = 200
	testSigner             = types.DefaultSigner{}
	testPrivate, _         = crypto.HexToECDSA("432a86ab8765d82415a803e29864dcfc1ed93dac949abf6f95a583179f27e4bb") // secp256k1.V = 1
	testAddr               = crypto.PubkeyToAddress(testPrivate.PublicKey)
	defaultAccounts        = []common.Address{
		common.HexToAddress("0x10000"), common.HexToAddress("0x20000"), testAddr,
	}
	defaultBlocks     = make([]*types.Block, 0)
	newestBlock       = new(types.Block)
	defaultBlockInfos = []blockInfo{
		// genesis block must no transactions
		{
			hash:        common.HexToHash("0xcf6d309d2dc4ca770535325bdf6656a2894cdf7c4c30af6eebc101dc01316dc8"),
			versionRoot: common.HexToHash("0x5a285bcfd4297d959e44cfc857e221695e12b088c5e01ad935e3eb2af62e3bcf"),
			txRoot:      common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"), // empty merkle
			logsRoot:    common.HexToHash("0x74450829ca5c4673011dd95266fbd78de05ec7d4a6bf9a22bc9f98c37823d1de"),
			time:        big.NewInt(1538209751),
			author:      defaultAccounts[0],
		},
		// block 1 is stable block
		{
			hash:        common.HexToHash("0xb5d28398c8111094df24c7667074239c4640595a4afff953a2fb557d6c792ebe"),
			versionRoot: common.HexToHash("0xb6e5b4f59eaa3e521cda36f68fb1cef930aefca27309030637e68887cd843755"),
			txRoot:      common.HexToHash("0xf044cc436950ef7470aca61053eb3f1ed46b9dcd501a5210f3673dc657c4fc88"),
			logsRoot:    common.HexToHash("0xaaa73250c24238d7aac05b0a89ded03c37d8d3b0b9462a906cec01d7ec50195e"),
			txList: []*types.Transaction{
				// testAddr -> defaultAccounts[0] 1
				types.NewTransaction(defaultAccounts[0], common.Big1, 2000000, common.Big2, []byte{12}, chainID, big.NewInt(1538210391), "aa", []byte{34}),
				// testAddr -> defaultAccounts[1] 1
				types.NewTransaction(defaultAccounts[1], common.Big1, 2000000, common.Big2, []byte{}, chainID, big.NewInt(1538210491), "", []byte{}),
			},
			time:   big.NewInt(1538209755),
			author: common.HexToAddress("0x20000"),
		},
		// block 2 is not stable block
		{
			hash:        common.HexToHash("0x7b8607bede3899901e48c893acb2822b48eaf2399c3a11a82419e0758f375a6f"),
			versionRoot: common.HexToHash("0x45797dfc11799a26c3b128ba55edfd0dd7894703c2d9d0d620485c2b84f325b2"),
			txRoot:      common.HexToHash("0x4558f847f8314dbc7e9d7d6fc84a9e75286040aa527b2d981f924a2ad75bca81"),
			logsRoot:    common.HexToHash("0x712697f0d951552f31a618a640cf5fd79de65d74ed5b329e0063b928a8fa5609"),
			txList: []*types.Transaction{
				// testAddr -> defaultAccounts[0] 2
				types.NewTransaction(defaultAccounts[0], common.Big2, 2000000, common.Big2, []byte{12}, chainID, big.NewInt(1538210395), "aa", []byte{34}),
			},
			time:   big.NewInt(1538209758),
			author: defaultAccounts[0],
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
	bc, err := NewBlockChain(uint64(chainID), db, newBlockCh, map[string]string{})
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

	for i, blockInfo := range defaultBlockInfos {
		saveBlock(db, i, blockInfo)
	}
	err = db.SetStableBlock(defaultBlockInfos[1].hash)
	if err != nil {
		panic(err)
	}
	return db
}

func saveBlock(db protocol.ChainDB, blockIndex int, info blockInfo) {
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
	if types.DeriveTxsSha(info.txList) != info.txRoot {
		panic(fmt.Errorf("%d txRoot hash error. except: %s, got: %s", blockIndex, info.txRoot.Hex(), types.DeriveTxsSha(info.txList).Hex()))
	}
	// genesis coin
	if blockIndex == 0 {
		owner := manager.GetAccount(testAddr)
		owner.SetBalance(big.NewInt(1000000000))
	}
	// account
	for _, tx := range info.txList {
		gas := params.TxGas + params.TxDataNonZeroGas*uint64(len(tx.Data()))
		from := manager.GetAccount(testAddr)
		from.SetBalance(new(big.Int).Sub(from.GetBalance(), tx.Value()))
		from.SetBalance(new(big.Int).Sub(from.GetBalance(), new(big.Int).SetUint64(gas)))
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
		panic(fmt.Errorf("%d version root error. except: %s, got: %s", blockIndex, info.versionRoot.Hex(), manager.GetVersionRoot().Hex()))
	}
	changeLogs := manager.GetChangeLogs()
	logsRoot := types.DeriveChangeLogsSha(changeLogs)
	if logsRoot != info.logsRoot {
		panic(fmt.Errorf("%d change logs root error. except: %s, got: %s", blockIndex, info.logsRoot.Hex(), logsRoot.Hex()))
	}
	header := &types.Header{
		GasUsed:     gasUsed,
		TxRoot:      info.txRoot,
		VersionRoot: info.versionRoot,
		LogsRoot:    info.logsRoot,
		Time:        info.time,
		LemoBase:    info.author,
		Extra:       []byte{},
	}
	if blockIndex > 0 {
		header.ParentHash = defaultBlockInfos[blockIndex-1].hash
	}
	blockHash := header.Hash()
	if blockHash != info.hash {
		panic(fmt.Errorf("%d block hash error. except: %s, got: %s", blockIndex, info.hash.Hex(), blockHash.Hex()))
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
