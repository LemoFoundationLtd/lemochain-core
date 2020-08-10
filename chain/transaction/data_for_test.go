package transaction

import (
	"crypto/ecdsa"
	"encoding/json"
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
	db, _ = newDBWithGenesis()
	dm := deputynode.NewManager(5, db)
	for i := uint32(1); i <= height; i++ {
		parentBlock, _ := db.GetBlockByHeight(i - 1)
		am = account.NewManager(parentBlock.Hash(), db)
		newBlockForTest(i, nil, am, dm, db, true)
	}
	return db, am
}

// 新建一个初始化了创世块的db
func newDBWithGenesis() (db protocol.ChainDB, genesisHash common.Hash) {
	db = newDB()
	am := account.NewManager(common.Hash{}, db)
	am.GetAccount(godAddr).SetBalance(common.Lemo2Mo("1600000000"))
	err := am.Finalise()
	if err != nil {
		panic(err)
	}
	logs := am.GetChangeLogs()

	header := &types.Header{
		ParentHash:   common.Hash{},
		MinerAddress: godAddr,
		TxRoot:       (types.Transactions{}).MerkleRootSha(),
		Height:       0,
		GasLimit:     params.GenesisGasLimit,
		Time:         uint32(time.Now().Unix()),
		VersionRoot:  am.GetVersionRoot(),
		LogRoot:      logs.MerkleRootSha(),
	}
	block := types.NewBlock(header, nil, logs)
	block.DeputyNodes = generateDeputies(5)
	root := block.DeputyNodes.MerkleRootSha()
	block.Header.DeputyRoot = root[:]
	saveBlock(db, am, block, true)
	return db, block.Hash()
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

func saveBlock(db protocol.ChainDB, am *account.Manager, block *types.Block, setStable bool) {
	blockHash := block.Hash()
	err := db.SetBlock(blockHash, block)
	if err != nil && err != store.ErrExist {
		panic(err)
	}
	err = am.Save(blockHash)
	if err != nil {
		panic(err)
	}
	if setStable {
		_, err := db.SetStableBlock(blockHash)
		if err != nil {
			panic(err)
		}
	}
}

// newBlockForTest 只能按照高度顺序创建区块
func newBlockForTest(height uint32, txs types.Transactions, am *account.Manager, dm *deputynode.Manager, db protocol.ChainDB, stable bool) *types.Block {
	var (
		parentHash common.Hash
		gasUsed    uint64
		selectTxs  types.Transactions
	)
	p := NewTxProcessor(config.RewardManager, config.ChainID, newTestParentLoader(db), am, db, dm)
	parent, _ := db.GetBlockByHeight(height - 1)
	parentHash = parent.Hash()

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

	err := am.Finalise()
	if err != nil {
		panic(err)
	}
	logs := am.GetChangeLogs()
	log.Infof("logs: %s", logs)
	header.GasUsed = gasUsed
	header.LogRoot = logs.MerkleRootSha()
	header.VersionRoot = am.GetVersionRoot()

	block := types.NewBlock(header, selectTxs, logs)
	// 给快照块添加共识节点列表
	if block.Height()%params.TermDuration == 0 {
		block.DeputyNodes = generateDeputies(5)
		root := block.DeputyNodes.MerkleRootSha()
		block.Header.DeputyRoot = root[:]
	}

	saveBlock(db, am, block, stable)

	return block
}

// generateDeputies generate random deputy nodes
func generateDeputies(num int) types.DeputyNodes {
	result := make(types.DeputyNodes, num)
	for i := 0; i < num; i++ {
		private, _ := crypto.GenerateKey()
		result[i] = &types.DeputyNode{
			MinerAddress: crypto.PubkeyToAddress(private.PublicKey),
			NodeID:       crypto.PrivateKeyToNodeID(private),
			Rank:         uint32(i),
			Votes:        big.NewInt(int64(10000000000 - i)),
		}
	}
	return result
}

// 实现BlockLoader接口
type testParentLoader struct {
	db protocol.ChainDB
}

func newTestParentLoader(db protocol.ChainDB) *testParentLoader {
	return &testParentLoader{db}
}

func (t *testParentLoader) GetParentByHeight(height uint32, sonBlockHash common.Hash) *types.Block {
	block, err := t.db.GetUnConfirmByHeight(height, sonBlockHash)
	if err == store.ErrBlockNotExist {
		block, err = t.db.GetBlockByHeight(height)
	}

	if err != nil {
		log.Error("load block by height fail", "height", height, "err", err)
		return nil
	}
	return block
}

func makeTx(fromPrivate *ecdsa.PrivateKey, from, to common.Address, data []byte, txType uint16, amount *big.Int) *types.Transaction {
	return makeTransaction(fromPrivate, from, to, data, txType, amount, common.Big1, uint64(time.Now().Unix()+30*60), 1000000)
}

func makeTransaction(fromPrivate *ecdsa.PrivateKey, from, to common.Address, data []byte, txType uint16, amount *big.Int, gasPrice *big.Int, expiration uint64, gasLimit uint64) *types.Transaction {
	tx := types.NewTransaction(from, to, amount, gasLimit, gasPrice, data, txType, chainID, expiration, "", string("aaa"))
	gas, _ := IntrinsicGas(tx.Type(), tx.Data(), tx.Message())
	tx.SetGasUsed(gas)
	return signTransaction(tx, fromPrivate)
}

func signTransaction(tx *types.Transaction, private *ecdsa.PrivateKey) *types.Transaction {
	tx, err := types.DefaultSigner{}.SignTx(tx, private)
	if err != nil {
		panic(err)
	}
	return tx
}

func makeCreateAssetTx(name string) *types.Transaction {
	profile := make(types.Profile)
	profile[types.AssetName] = name
	profile[types.AssetSymbol] = "DT"
	profile[types.AssetDescription] = "test issue token"
	profile[types.AssetFreeze] = "false"
	profile[types.AssetSuggestedGasLimit] = "60000"
	asset := &types.Asset{
		Category:        types.TokenAsset,
		IsDivisible:     true,
		Decimal:         18,
		TotalSupply:     big.NewInt(100000),
		IsReplenishable: true,
		Profile:         profile,
	}
	data, err := json.Marshal(asset)
	if err != nil {
		panic(err)
	}
	return makeTx(godPrivate, godAddr, common.HexToAddress("0x1"), data, params.CreateAssetTx, big.NewInt(1))
}

func makeIssueAssetTx(assetCode common.Hash) *types.Transaction {
	amount, _ := new(big.Int).SetString("1000111000000000000000000", 10)
	issue := &types.IssueAsset{
		AssetCode: assetCode,
		MetaData:  "demo",
		Amount:    amount,
	}
	data, err := json.Marshal(issue)
	if err != nil {
		panic(err)
	}
	return makeTx(godPrivate, godAddr, godAddr, data, params.IssueAssetTx, big.NewInt(1))
}

func makeTransferAssetTx(assetId common.Hash) *types.Transaction {
	amount, _ := new(big.Int).SetString("1000111000000000000000000", 10)
	transfer := &types.TransferAsset{
		AssetId: assetId,
		Amount:  amount,
		Input:   nil,
	}
	data, err := json.Marshal(transfer)
	if err != nil {
		panic(err)
	}
	return makeTx(godPrivate, godAddr, common.HexToAddress("0x2"), data, params.TransferAssetTx, big.NewInt(1))
}
