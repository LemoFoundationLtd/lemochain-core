package transaction

import (
	"crypto/ecdsa"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
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

type Config struct {
	RewardManager common.Address
	ChainID       uint16
}

var (
	chainID uint16 = 100
	config         = Config{
		RewardManager: godAddr,
		ChainID:       chainID,
	}
	godPrivate, _ = crypto.HexToECDSA("c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa") // 测试中的16亿lemo地址
	godAddr       = crypto.PubkeyToAddress(godPrivate.PublicKey)
	godRawAddr    = "0x015780F8456F9c1532645087a19DcF9a7e0c7F97"
	godLemoAddr   = "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG"
)

func init() {
	log.Setup(log.LevelInfo, false, false)
}

func GetStorePath() string {
	return "../../testdata/transaction"
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
	return store.NewChainDataBase(GetStorePath())
}

// newDBToBlockHeight 返回存储了指定高度个稳定区块的db和当前状态的account manager
func newDBToBlockHeight(height uint32) (db protocol.ChainDB, am *account.Manager) {
	db, _ = newCoverGenesisDB()
	dm := deputynode.NewManager(5, db)
	for i := uint32(1); i <= height; i++ {
		parentBlock, _ := db.GetBlockByHeight(i - 1)
		am = account.NewManager(parentBlock.Hash(), db)
		newBlockForTest(i, nil, am, dm, db, true)
	}
	return db, am
}

// 新建一个初始化了创世块的db
func newCoverGenesisDB() (db protocol.ChainDB, genesisHash common.Hash) {
	db = newDB()
	am := account.NewManager(common.Hash{}, db)
	total, _ := new(big.Int).SetString("1600000000000000000000000000", 10) // 1.6 billion
	am.GetAccount(godAddr).SetBalance(total)
	genesis := newBlockForTest(0, nil, am, nil, db, true)
	genesisHash = genesis.Hash()
	return db, genesisHash
}

func createBlock(height uint32, am *account.Manager, db protocol.ChainDB) *types.Block {
	block := &types.Block{
		Header: &types.Header{
			MinerAddress: common.HexToAddress("0x1100"),
			Height:       height,
			GasLimit:     5100000000,
		},
	}
	// 保存db
	hash := block.Hash()
	db.SetBlock(hash, block)
	// am.Save(hash)
	db.SetStableBlock(hash)
	return block
}

// newBlockForTest 只能按照高度顺序创建区块
func newBlockForTest(height uint32, txs types.Transactions, am *account.Manager, dm *deputynode.Manager, db protocol.ChainDB, stable bool) *types.Block {
	var (
		parentHash common.Hash
		gasUsed    uint64
		selectTxs  types.Transactions
	)
	p := NewTxProcessor(config.RewardManager, config.ChainID, newTestChain(db), am, db, dm)
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
		GasLimit:     5100000000,
		TxRoot:       txs.MerkleRootSha(),
	}
	// 执行交易
	if len(txs) != 0 {
		selectTxs, _, gasUsed = p.ApplyTxs(header, txs, 1000)
	}

	am.Finalise()
	logs := am.GetChangeLogs()
	log.Infof("logs: %s", logs)
	header.GasUsed = gasUsed
	header.LogRoot = logs.MerkleRootSha()
	header.VersionRoot = am.GetVersionRoot()

	block := types.NewBlock(header, selectTxs, logs)
	// 给快照块添加共识节点列表
	if block.Height()%params.TermDuration == 0 {
		deputyNodes := make(types.DeputyNodes, 0)
		for i := 0; i < 5; i++ {
			node := &types.DeputyNode{
				MinerAddress: godAddr,
				NodeID:       common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"),
				Rank:         uint32(i),
				Votes:        new(big.Int).SetInt64(int64(5 - i)), // 初始的代理节点列表中的votes都为0，因为初始的时候没有一个账户中有lemo.
			}
			deputyNodes = append(deputyNodes, node)
		}
		block.DeputyNodes = deputyNodes
	}
	hash := block.Hash()
	db.SetBlock(hash, block)
	am.Save(hash)
	if stable {
		db.SetStableBlock(hash)
	}

	return block
}

// 实现BlockLoader接口
type testChain struct {
	db protocol.ChainDB
}

func newTestChain(db protocol.ChainDB) *testChain {
	return &testChain{db}
}

func (t *testChain) GetParentByHeight(height uint32, sonBlockHash common.Hash) *types.Block {
	block, err := t.db.GetBlockByHeight(height)
	if err != nil {
		return nil
	}
	return block
}

func makeTx(fromPrivate *ecdsa.PrivateKey, from, to common.Address, data []byte, txType uint16, amount *big.Int) *types.Transaction {
	return makeTransaction(fromPrivate, from, to, data, txType, amount, common.Big1, uint64(time.Now().Unix()+30*60), 1000000)
}

func makeTransaction(fromPrivate *ecdsa.PrivateKey, from, to common.Address, data []byte, txType uint16, amount *big.Int, gasPrice *big.Int, expiration uint64, gasLimit uint64) *types.Transaction {
	tx := types.NewTransaction(from, to, amount, gasLimit, gasPrice, data, txType, chainID, expiration, "", string("aaa"))
	return signTransaction(tx, fromPrivate)
}

func signTransaction(tx *types.Transaction, private *ecdsa.PrivateKey) *types.Transaction {
	tx, err := types.DefaultSigner{}.SignTx(tx, private)
	if err != nil {
		panic(err)
	}
	return tx
}
