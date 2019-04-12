package chain

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/flag"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/merkle"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"math/big"
	"math/rand"
	"os"
	"time"
)

type blockInfo struct {
	hash        common.Hash
	parentHash  common.Hash
	height      uint32
	author      common.Address
	versionRoot common.Hash
	txRoot      common.Hash
	logRoot     common.Hash
	txList      types.Transactions
	gasLimit    uint64
	time        uint32
	deputyRoot  []byte
	deputyNodes deputynode.DeputyNodes
}

var (
	chainID         uint16 = 200
	bigNumber, _           = new(big.Int).SetString("1000000000000000000000", 10) // 1 thousand
	testSigner             = types.DefaultSigner{}
	testPrivate, _         = crypto.HexToECDSA("432a86ab8765d82415a803e29864dcfc1ed93dac949abf6f95a583179f27e4bb") // secp256k1.V = 1
	testAddr               = crypto.PubkeyToAddress(testPrivate.PublicKey)                                         // 0x0107134B9CdD7D89F83eFa6175F9b3552F29094c
	totalLEMO, _           = new(big.Int).SetString("1000000000000000000000000", 10)                               // 1 million
	defaultAccounts        = []common.Address{
		common.HexToAddress("0x10000"), common.HexToAddress("0x20000"), testAddr,
	}
	defaultBlocks     = make([]*types.Block, 0)
	defaultBlockInfos = []blockInfo{
		// genesis block must no transactions
		{
			height:      0,
			author:      defaultAccounts[0],
			txRoot:      merkle.EmptyTrieHash,
			time:        1538209751,
			deputyNodes: DefaultDeputyNodes,
		},
		// block 1 is stable block
		{
			height: 1,
			author: common.HexToAddress("0x20000"),
			txRoot: common.HexToHash("0xec3a193fd32f741372031854461b0413bf7a6136c5da8482f37d3bc42f75125d"),
			txList: []*types.Transaction{
				// testAddr -> defaultAccounts[0] 1
				signTransaction(types.NewTransaction(defaultAccounts[0], common.Big1, 2000000, common.Big2, []byte{12}, params.OrdinaryTx, chainID, 1538210391, "aa", "aaa"), testPrivate),
				// testAddr -> defaultAccounts[1] 1
				makeTransaction(testPrivate, defaultAccounts[1], params.OrdinaryTx, common.Big1, common.Big2, 1538210491, 2000000),
			},
			gasLimit: 20000000,
			time:     1538209755,
		},
		// block 2 is not stable block
		{
			height: 2,
			author: defaultAccounts[0],
			txRoot: common.HexToHash("0x05633a7bb926221425abcf4b3505f5c0e7cb60b5619b24ac66aea98c97c3f1da"),
			txList: []*types.Transaction{
				// testAddr -> defaultAccounts[0] 2
				makeTransaction(testPrivate, defaultAccounts[0], params.OrdinaryTx, bigNumber, common.Big2, 1538210395, 2000000),
			},
			time:     1538209758,
			gasLimit: 20000000,
		},
		// block 3 is not store in db
		{
			height: 3,
			author: defaultAccounts[0],
			txRoot: common.HexToHash("0x85400987768d585d45925e27bc64e0ecd8fc50f9d106832e352faa72eb6bd4fb"),
			txList: []*types.Transaction{
				// testAddr -> defaultAccounts[0] 2
				makeTransaction(testPrivate, defaultAccounts[0], params.OrdinaryTx, common.Big2, common.Big2, 1538210398, 30000),
				// testAddr -> defaultAccounts[1] 2
				makeTransaction(testPrivate, defaultAccounts[1], params.OrdinaryTx, common.Big2, common.Big3, 1538210425, 30000),
			},
			time:     1538209761,
			gasLimit: 20000000,
		},
	}
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

// newChain creates chain for test
func newChain() *BlockChain {
	db := newDB()
	dm := deputynode.NewManager(5)
	bc, err := NewBlockChain(chainID, NewDpovp(10*1000, dm, db), dm, db, flag.CmdFlags{})
	if err != nil {
		panic(err)
	}
	return bc
}

// newDB creates db for test chain module
func newDB() protocol.ChainDB {
	db := store.NewChainDataBase(GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
	for i, _ := range defaultBlockInfos {
		if i > 0 {
			defaultBlockInfos[i].parentHash = defaultBlocks[i-1].Hash()
		}
		newBlock := makeBlock(db, defaultBlockInfos[i], i < 3)
		if i == 0 || i == 1 {
			err := db.SetStableBlock(newBlock.Hash())
			if err != nil {
				panic(err)
			}
		}
		defaultBlocks = append(defaultBlocks, newBlock)
	}

	return db
}

func makeBlock(db protocol.ChainDB, info blockInfo, save bool) *types.Block {
	start := time.Now().UnixNano()
	manager := account.NewManager(info.parentHash, db)
	// sign transactions
	var err error
	var gasUsed uint64 = 0
	txRoot := info.txList.MerkleRootSha()
	if txRoot != info.txRoot {
		if info.txRoot != (common.Hash{}) {
			fmt.Printf("%d txRoot hash error. except: %s, got: %s\n", info.height, info.txRoot.Hex(), txRoot.Hex())
		}
		info.txRoot = txRoot
	}
	txMerkleEnd := time.Now().UnixNano()

	// genesis coin
	if info.height == 0 {
		owner := manager.GetAccount(testAddr)
		// 1 million
		owner.SetBalance(new(big.Int).Set(totalLEMO))
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
		cost := new(big.Int).Add(tx.Amount(), fee)
		to := manager.GetAccount(*tx.To())
		if tx.Type() == params.VoteTx || tx.Type() == params.RegisterTx {
			newProfile := make(map[string]string, 5)
			newProfile[types.CandidateKeyIsCandidate] = "true"
			to.SetCandidate(newProfile)
			to.SetVotes(big.NewInt(10))
		}
		// make sure the change log has right order
		if fromAddr.Hex() < tx.To().Hex() {
			from.SetBalance(new(big.Int).Sub(from.GetBalance(), cost))
			to.SetBalance(new(big.Int).Add(to.GetBalance(), tx.Amount()))
		} else {
			to.SetBalance(new(big.Int).Add(to.GetBalance(), tx.Amount()))
			from.SetBalance(new(big.Int).Sub(from.GetBalance(), cost))
		}
		gasUsed += gas
		salary.Add(salary, fee)
	}
	if salary.Cmp(new(big.Int)) != 0 {
		miner := manager.GetAccount(info.author)
		miner.SetBalance(new(big.Int).Add(miner.GetBalance(), salary))
	}
	editAccountsEnd := time.Now().UnixNano()

	manager.MergeChangeLogs(0)
	err = manager.Finalise()
	if err != nil {
		panic(err)
	}
	finaliseEnd := time.Now().UnixNano()

	// header
	if manager.GetVersionRoot() != info.versionRoot {
		if info.versionRoot != (common.Hash{}) {
			fmt.Printf("%d version root error. except: %s, got: %s\n", info.height, info.versionRoot.Hex(), manager.GetVersionRoot().Hex())
		}
		info.versionRoot = manager.GetVersionRoot()
	}

	changeLogs := manager.GetChangeLogs()
	fmt.Printf("%d changeLogs %v\n", info.height, changeLogs)
	logRoot := changeLogs.MerkleRootSha()
	if logRoot != info.logRoot {
		if info.logRoot != (common.Hash{}) {
			fmt.Printf("%d change logs root error. except: %s, got: %s\n", info.height, info.logRoot.Hex(), logRoot.Hex())
		}
		info.logRoot = logRoot
	}
	if info.time == 0 {
		info.time = uint32(time.Now().Unix())
	}
	if info.gasLimit == 0 {
		info.gasLimit = 1000000
	}
	header := &types.Header{
		ParentHash:   info.parentHash,
		MinerAddress: info.author,
		VersionRoot:  info.versionRoot,
		TxRoot:       info.txRoot,
		LogRoot:      info.logRoot,
		Height:       info.height,
		GasLimit:     info.gasLimit,
		GasUsed:      gasUsed,
		Time:         info.time,
		Extra:        []byte{},
	}

	var deputyRoot []byte
	// if len(info.deputyNodes) > 0 {
	// 	deputyRoot = types.DeriveDeputyRootSha(info.deputyNodes).Bytes()
	// 	deputynode.NewManager().SaveSnapshot(params.TermDuration, info.deputyNodes)
	// }
	if bytes.Compare(deputyRoot, info.deputyRoot) != 0 {
		if len(info.deputyNodes) > 0 || len(info.deputyRoot) != 0 {
			fmt.Printf("%d deputyRoot error. except: %s, got: %s\n", info.height, common.ToHex(info.deputyRoot), common.ToHex(deputyRoot))
		}
		info.deputyRoot = deputyRoot
	}
	if len(info.deputyRoot) > 0 {
		header.DeputyRoot = make([]byte, len(info.deputyRoot))
		copy(header.DeputyRoot[:], info.deputyRoot[:])
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
		Txs:         info.txList,
		ChangeLogs:  changeLogs,
		DeputyNodes: info.deputyNodes,
	}
	block.SetHeader(header)
	buildBlockEnd := time.Now().UnixNano()

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
	saveEnd := time.Now().UnixNano()

	fmt.Printf("Building tx merkle trie cost %dms. %d txs in total\n", (txMerkleEnd-start)/1000000, len(info.txList))
	fmt.Printf("Editing balance of accounts cost %dms\n", (editAccountsEnd-txMerkleEnd)/1000000)
	fmt.Printf("Finalising manager cost %dms\n", (finaliseEnd-editAccountsEnd)/1000000)
	fmt.Printf("Building block cost %dms\n", (buildBlockEnd-finaliseEnd)/1000000)
	fmt.Printf("Saving block and accounts cost %dms\n\n", (saveEnd-buildBlockEnd)/1000000)
	return block
}

func makeTx(fromPrivate *ecdsa.PrivateKey, to common.Address, txType uint16, amount *big.Int) *types.Transaction {
	return makeTransaction(fromPrivate, to, txType, amount, common.Big1, uint64(time.Now().Unix()+300), 1000000)
}

func makeTransaction(fromPrivate *ecdsa.PrivateKey, to common.Address, txType uint16, amount, gasPrice *big.Int, expiration uint64, gasLimit uint64) *types.Transaction {
	tx := types.NewTransaction(to, amount, gasLimit, gasPrice, []byte{}, txType, chainID, expiration, "", "")
	return signTransaction(tx, fromPrivate)
}

func signTransaction(tx *types.Transaction, private *ecdsa.PrivateKey) *types.Transaction {
	tx, err := testSigner.SignTx(tx, private)
	if err != nil {
		panic(err)
	}
	return tx
}

// createAccounts creates random accounts and transfer LEMO to them
func createAccounts(n int, db protocol.ChainDB) (common.Hash, []*crypto.AccountKey) {
	accountKeys := make([]*crypto.AccountKey, n, n)
	txs := make(types.Transactions, n, n)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	maxAmount := new(big.Int).Div(totalLEMO, big.NewInt(int64(n)))
	for i := 0; i < n; i++ {
		accountKey, err := crypto.GenerateAddress()
		if err != nil {
			panic(err)
		}
		accountKeys[i] = accountKey
		txs[i] = makeTx(testPrivate, accountKey.Address, params.OrdinaryTx, new(big.Int).Rand(r, maxAmount))
	}
	newBlock := makeBlock(db, blockInfo{
		height:     3,
		parentHash: defaultBlocks[2].Hash(),
		author:     defaultAccounts[0],
		time:       1538209761,
		txList:     txs,
		gasLimit:   2100000000,
	}, true)
	return newBlock.Hash(), accountKeys
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
