package txprocessor

import (
	"crypto/ecdsa"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"math/big"
	"os"
	"time"
)

var (
	chainID     uint16 = 100
	timeOutTime uint64 = 10
	config             = Config{
		LogForks:      false,
		RewardManager: godAddr,
		ChainID:       chainID,
		MineTimeout:   timeOutTime,
	}
	godPrivate, _ = crypto.HexToECDSA("432a86ab8765d82415a803e29864dcfc1ed93dac949abf6f95a583179f27e4bb") // 测试中的16亿lemo地址
	godAddr       = crypto.PubkeyToAddress(godPrivate.PublicKey)
)

func init() {
	log.Setup(log.LevelInfo, false, false)
}

func GetStorePath() string {
	return "../testdata/blockchain"
}

func ClearData() {
	err := os.RemoveAll(GetStorePath())
	failCnt := 1
	for err != nil {
		log.Errorf("CLEAR DATA BASE FAIL.%s, SLEEP(%ds) AND CONTINUE", err.Error(), failCnt)
		time.Sleep(time.Duration(failCnt) * time.Second)
		err = os.RemoveAll(GetStorePath())
		failCnt = failCnt + 1
	}
}

// newDB db for test
func newDB() protocol.ChainDB {
	return store.NewChainDataBase(GetStorePath(), "", "")
}

// 新建一个初始化了创世块的db
func newCoverGenesisDB() (db protocol.ChainDB, genesisHash common.Hash) {
	db = newDB()
	am := account.NewManager(common.Hash{}, db)
	total, _ := new(big.Int).SetString("1600000000000000000000000000", 10) // 1.6 billion
	am.GetAccount(godAddr).SetBalance(total)
	genesis := addBlockToDB(0, nil, am, db)
	genesisHash = genesis.Hash()
	return db, genesisHash
}

// 加入block 到db
func addBlockToDB(height uint32, txs types.Transactions, am *account.Manager, db protocol.ChainDB) *types.Block {
	var (
		parentHash common.Hash
		gasUsed    uint64
	)
	p := NewTxProcessor(config, newTestChain(db), am, db)
	// 判断创世块
	if height == 0 {
		parentHash = common.Hash{}
	} else {
		parent, _ := db.GetBlockByHeight(height - 1)
		parentHash = parent.Hash()
	}

	header := &types.Header{
		ParentHash:   parentHash,
		MinerAddress: common.HexToAddress("0x1100"),
		Height:       height,
		GasLimit:     510000000,
		TxRoot:       txs.MerkleRootSha(),
	}
	// 执行交易
	if len(txs) != 0 {
		_, _, gasUsed = p.ApplyTxs(header, txs, 10)
	}

	am.Finalise()
	logs := am.GetChangeLogs()
	log.Infof("logs: %s", logs)
	header.GasUsed = gasUsed
	header.LogRoot = logs.MerkleRootSha()
	header.VersionRoot = am.GetVersionRoot()

	block := types.NewBlock(header, txs, logs)
	hash := block.Hash()
	db.SetBlock(hash, block)
	am.Save(hash)
	db.SetStableBlock(hash)

	return block
}

// 实现BlockLoader接口
type testChain struct {
	db protocol.ChainDB
}

func newTestChain(db protocol.ChainDB) *testChain {
	return &testChain{db}
}
func (t *testChain) GetBlockByHash(hash common.Hash) *types.Block {
	block, err := t.db.GetBlockByHash(hash)
	if err != nil {
		return nil
	}
	return block
}

func (t *testChain) GetParentByHeight(height uint32, sonBlockHash common.Hash) *types.Block {
	block, err := t.db.GetBlockByHeight(height)
	if err != nil {
		return nil
	}
	return block
}

func makeTx(fromPrivate *ecdsa.PrivateKey, to common.Address, txType uint16, amount *big.Int) *types.Transaction {
	return makeTransaction(fromPrivate, to, txType, amount, common.Big1, uint64(time.Now().Unix()+300), 1000000)
}

func makeTransaction(fromPrivate *ecdsa.PrivateKey, to common.Address, txType uint16, amount *big.Int, gasPrice *big.Int, expiration uint64, gasLimit uint64) *types.Transaction {
	tx := types.NewTransaction(to, amount, gasLimit, gasPrice, []byte{}, txType, chainID, expiration, "", string("aaa"))
	return signTransaction(tx, fromPrivate)
}

func signTransaction(tx *types.Transaction, private *ecdsa.PrivateKey) *types.Transaction {
	tx, err := types.DefaultSigner{}.SignTx(tx, private)
	if err != nil {
		panic(err)
	}
	return tx
}
