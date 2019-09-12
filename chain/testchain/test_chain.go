package testchain

import (
	"crypto/ecdsa"
	"github.com/LemoFoundationLtd/lemochain-core/chain"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/consensus"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/transaction"
	"github.com/LemoFoundationLtd/lemochain-core/chain/txpool"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"math/big"
	"os"
	"time"

	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/flag"
)

func GetStorePath() string {
	return "../../testdata/testchain"
}

func ClearData() {
	err := os.RemoveAll(GetStorePath())
	failCnt := 1
	for err != nil {
		log.Errorf("CLEAR DATA BASE FAIL.%s, SLEEP(%ds) AND CONTINUE", err.Error(), failCnt)
		time.Sleep(time.Duration(failCnt) * time.Second)
		err = os.RemoveAll(GetStorePath())
		failCnt++
	}
}

type parentLoader struct {
	Db protocol.ChainDB
}

func (t *parentLoader) GetParentByHeight(height uint32, sonBlockHash common.Hash) *types.Block {
	block, err := t.Db.GetUnConfirmByHeight(height, sonBlockHash)
	if err == store.ErrNotExist {
		block, err = t.Db.GetBlockByHeight(height)
	}

	if err != nil {
		log.Error("load block by height fail", "height", height, "err", err)
		return nil
	}
	return block
}

type candidateLoader struct {
	db protocol.ChainDB
	dm *deputynode.Manager
	am *account.Manager
}

func (cl *candidateLoader) LoadTopCandidates(blockHash common.Hash) types.DeputyNodes {
	// TODO simplify
	result := make(types.DeputyNodes, 0, cl.dm.DeputyCount)
	list := cl.db.GetCandidatesTop(blockHash)
	if len(list) > cl.dm.DeputyCount {
		list = list[:cl.dm.DeputyCount]
	}

	for i, n := range list {
		acc := cl.am.GetAccount(n.GetAddress())
		candidate := acc.GetCandidate()
		strID := candidate[types.CandidateKeyNodeID]
		dn, err := types.NewDeputyNode(n.GetTotal(), uint32(i), n.GetAddress(), strID)
		if err != nil {
			continue
		}
		result = append(result, dn)
	}
	return result
}

func (dp *candidateLoader) LoadRefundCandidates() ([]common.Address, error) {
	return []common.Address{}, nil
}

type dbStatus uint32

const (
	Stable   dbStatus = iota
	Unstable          // unstable but still in store
	NotInStore
)

type blockInfo struct {
	height      uint32
	author      common.Address
	txList      types.Transactions
	gasLimit    uint64
	time        uint32
	deputyNodes types.DeputyNodes
	status      dbStatus
}

var (
	chainID           uint16 = 200
	bigNumber                = common.Lemo2Mo("1000")
	testSigner               = types.DefaultSigner{}
	FounderPrivate, _        = crypto.HexToECDSA("432a86ab8765d82415a803e29864dcfc1ed93dac949abf6f95a583179f27e4bb") // secp256k1.V = 1
	FounderAddr              = crypto.PubkeyToAddress(FounderPrivate.PublicKey)                                      // Lemo836BQKCBZ8Z7B7N4G4N4SNGBT24ZZSJQD24D
	defaultAccounts          = []common.Address{
		common.HexToAddress("0x10000"), common.HexToAddress("0x20000"), FounderAddr,
	}
	defaultBlocks     = make([]*types.Block, 0)
	defaultBlockInfos = []blockInfo{
		// genesis block must no transactions
		{
			height:      0,
			author:      defaultAccounts[0],
			time:        1538209751,
			deputyNodes: chain.BuildDeputyNodes(chain.DefaultDeputyNodesInfo),
			status:      Stable,
		},
		// block 1
		{
			height: 1,
			author: common.HexToAddress("0x20000"),
			txList: []*types.Transaction{
				// FounderAddr -> defaultAccounts[0] 1
				SignTx(types.NewTransaction(FounderAddr, defaultAccounts[0], common.Big1, 2000000, common.Big2, []byte{12}, 0, chainID, uint64(1538210391), "aa", string("aaa")), FounderPrivate),
				// FounderAddr -> defaultAccounts[1] 1
				MakeTx(FounderPrivate, defaultAccounts[1], common.Big1, 2000000, common.Big2, 1538210491),
			},
			gasLimit: 20000000,
			time:     1538209755,
			status:   Stable,
		},
		// block 2
		{
			height: 2,
			author: defaultAccounts[0],
			txList: []*types.Transaction{
				// FounderAddr -> defaultAccounts[0] 2
				MakeTx(FounderPrivate, defaultAccounts[0], bigNumber, 2000000, common.Big2, 1538210395),
			},
			time:     1538209758,
			gasLimit: 20000000,
			status:   Unstable,
		},
		// block 3
		{
			height: 3,
			author: defaultAccounts[0],
			txList: []*types.Transaction{
				// FounderAddr -> defaultAccounts[0] 2
				MakeTx(FounderPrivate, defaultAccounts[0], common.Big2, 30000, common.Big2, 1538210398),
				// FounderAddr -> defaultAccounts[1] 2
				MakeTx(FounderPrivate, defaultAccounts[1], common.Big2, 30000, common.Big3, 1538210425),
			},
			time:     1538209761,
			gasLimit: 20000000,
			status:   NotInStore,
		},
	}
)

func init() {
	ClearData()
}

// NewTestChain creates chain for test
func NewTestChain() (*chain.BlockChain, protocol.ChainDB) {
	deputynode.SetSelfNodeKey(FounderPrivate)
	db := store.NewChainDataBase(GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
	initGenesis(db)
	// must save genesis before new deputy manager
	dm := deputynode.NewManager(5, db)
	initBlocks(db, dm)
	bc, err := chain.NewBlockChain(chain.Config{chainID, 10000}, dm, db, flag.CmdFlags{}, txpool.NewTxPool())
	if err != nil {
		panic(err)
	}
	return bc, db
}

func CloseTestChain(bc *chain.BlockChain, db protocol.ChainDB) {
	if bc != nil {
		bc.Stop()
		// time.Sleep(500 * time.Millisecond)
	}
	if db != nil {
		if err := db.Close(); err != nil {
			log.Errorf("close db fail: %v", err)
		}
		// time.Sleep(500 * time.Millisecond)
	}
	ClearData()
}

func initGenesis(db protocol.ChainDB) {
	am := account.NewManager(common.Hash{}, db)

	genesis := chain.DefaultGenesisConfig()
	genesis.Time = defaultBlockInfos[0].time
	genesis.Founder = defaultBlockInfos[0].author
	// copy and set first miner address
	firstCandidate := *genesis.DeputyNodesInfo[0]
	firstCandidate.MinerAddress = FounderAddr
	firstCandidate.NodeID = crypto.PrivateKeyToNodeID(FounderPrivate)
	genesis.DeputyNodesInfo[0] = &firstCandidate
	block, err := genesis.ToBlock(am)
	if err != nil {
		panic(err)
	}
	saveBlock(db, am, block, defaultBlockInfos[0].status)

	defaultBlocks = []*types.Block{block}
}

func initBlocks(db protocol.ChainDB, dm *deputynode.Manager) {
	for i, info := range defaultBlockInfos {
		if i == 0 {
			continue // skip genesis block
		}
		parentHeader := defaultBlocks[i-1].Header
		newBlock := makeBlock(db, dm, info, parentHeader)
		defaultBlocks = append(defaultBlocks, newBlock)
	}
}

func makeBlock(db protocol.ChainDB, dm *deputynode.Manager, info blockInfo, parentHeader *types.Header) *types.Block {
	am := account.NewManager(parentHeader.Hash(), db)
	processor := transaction.NewTxProcessor(FounderAddr, chainID, &parentLoader{db}, am, db, dm)
	assembler := consensus.NewBlockAssembler(am, dm, processor, &candidateLoader{db, dm, am})
	// account
	header, err := assembler.PrepareHeader(parentHeader, nil)
	if err != nil {
		panic(err)
	}
	block, validTxs, err := assembler.MineBlock(header, info.txList, 2)
	if err != nil {
		panic(err)
	}
	if len(validTxs) != len(info.txList) {
		panic("invalid tx")
	}

	saveBlock(db, am, block, info.status)
	return block
}

func saveBlock(db protocol.ChainDB, am *account.Manager, block *types.Block, dbOp dbStatus) {
	blockHash := block.Hash()
	if dbOp != NotInStore {
		err := db.SetBlock(blockHash, block)
		if err != nil && err != store.ErrExist {
			panic(err)
		}
		err = am.Save(blockHash)
		if err != nil {
			panic(err)
		}
	}
	if dbOp == Stable {
		_, err := db.SetStableBlock(blockHash)
		if err != nil {
			panic(err)
		}
	}
}

func LoadDefaultBlock(index int) *types.Block {
	return defaultBlocks[index]
}

func MakeTransferTx(fromPrivate *ecdsa.PrivateKey, to common.Address, amount *big.Int) *types.Transaction {
	return MakeTx(fromPrivate, to, amount, 1000000, common.Big1, uint64(time.Now().Unix()+300))
}

func MakeTx(fromPrivate *ecdsa.PrivateKey, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, expiration uint64) *types.Transaction {
	from := crypto.PubkeyToAddress(fromPrivate.PublicKey)
	tx := types.NewTransaction(from, to, amount, gasLimit, gasPrice, []byte{}, params.OrdinaryTx, chainID, expiration, "", string("aaa"))
	return SignTx(tx, fromPrivate)
}

func SignTx(tx *types.Transaction, private *ecdsa.PrivateKey) *types.Transaction {
	tx, err := testSigner.SignTx(tx, private)
	if err != nil {
		panic(err)
	}
	return tx
}
