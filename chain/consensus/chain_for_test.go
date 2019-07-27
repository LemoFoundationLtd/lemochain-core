package consensus

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/consensus"
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

var (
	ErrNotExist = errors.New("item does not exist")
)

type testChain struct {
	Db protocol.ChainDB
}

func NewTestChain(db protocol.ChainDB) *testChain {
	return &testChain{
		Db: db,
	}
}
func (t *testChain) GetBlockByHash(hash common.Hash) *types.Block {
	block, err := t.Db.GetBlockByHash(hash)
	if err != nil {
		return nil
	}
	return block
}
func (t *testChain) GetBlockByHeight(height uint32) *types.Block {
	block, err := t.Db.GetBlockByHeight(height)
	if err != nil {
		return nil
	}
	return block
}
func (t *testChain) GetParentByHeight(height uint32, sonBlockHash common.Hash) *types.Block {
	var block *types.Block
	var err error
	block, err = t.Db.GetUnConfirmByHeight(height, sonBlockHash)
	if err == ErrNotExist {
		block, err = t.Db.GetBlockByHeight(height)
	}

	if err != nil {
		log.Error("load block by height fail", "height", height, "err", err)
		return nil
	}
	return block
}

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
	deputyNodes types.DeputyNodes
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
		// block 3 is not store in blockLoader
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
	return "../testdata/consensus"
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
	bc, err := NewBlockChain(chainID, consensus.NewDpovp(10*1000, dm, db), dm, db, flag.CmdFlags{})
	if err != nil {
		panic(err)
	}
	return bc
}

// newDB creates blockLoader for test chain module
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

// makeBlock for test
func makeBlock(db protocol.ChainDB, info blockInfo, save bool) *types.Block {
	start := time.Now().UnixNano()
	am := account.NewManager(info.parentHash, db)
	// 计算交易root
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
		owner := am.GetAccount(testAddr)
		// 1 million
		owner.SetBalance(new(big.Int).Set(totalLEMO))
	}
	var GasUsed uint64 = 0
	// 执行交易，这里忽略智能合约交易
	for _, tx := range info.txList {
		// 	交易发送者
		from, err := tx.From()
		if err != nil {
			panic(err)
		}
		fromAcc := am.GetAccount(from)
		// 取得from的初始balance
		initFromBalance := fromAcc.GetBalance()
		// to
		var (
			toAcc         types.AccountAccessor
			initToBalance *big.Int
		)
		to := tx.To()
		if to != nil {
			toAcc = am.GetAccount(*to)
			initToBalance = toAcc.GetBalance()
		}

		// tx消耗的gas
		gasUsed, err := IntrinsicGas(tx.Data(), false)
		// 计算总的gas used
		GasUsed = GasUsed + gasUsed
		// gas 费用
		gasFee := new(big.Int).Mul(new(big.Int).SetUint64(gasUsed), tx.GasPrice())
		if err != nil {
			panic(err)
		}
		// 得到gas payer
		gasPayer, err := tx.GasPayer()
		if err != nil {
			panic(err)
		}
		// gas payer支付gas费用
		gasPayerAcc := am.GetAccount(gasPayer)
		gasPayerAcc.SetBalance(new(big.Int).Sub(gasPayerAcc.GetBalance(), gasFee))

		// minerAddress
		minerAddr := info.author
		minerAcc := am.GetAccount(minerAddr)
		// 获取打包交易奖励
		minerAcc.SetBalance(new(big.Int).Add(minerAcc.GetBalance(), gasFee))

		// 转账导致的账户balance变化
		if tx.Amount().Cmp(big.NewInt(0)) != 0 && to != nil {
			fromAcc.SetBalance(new(big.Int).Sub(fromAcc.GetBalance(), tx.Amount()))
			toAcc.SetBalance(new(big.Int).Add(toAcc.GetBalance(), tx.Amount()))
		}
		// 对于普通交易，到这里就没有changlog生成了，下面是对特殊交易生成changlog
		switch tx.Type() {
		case params.OrdinaryTx:
			break
		case params.VoteTx: // 投票交易，to的票数增加，过去投的账户票数减少
			oldCandidate := fromAcc.GetVoteFor()
			if (oldCandidate != common.Address{}) {
				oldCandidateAcc := am.GetAccount(oldCandidate)
				// 减少oldCandidate票数,票数为from的balance
				oldCandidateAcc.SetVotes(new(big.Int).Sub(oldCandidateAcc.GetVotes(), initFromBalance))
			}
			newCandidateAcc := toAcc
			// 更新voteFor地址
			fromAcc.SetVoteFor(*to)
			// 增加newCandidateAcc的票数
			newCandidateAcc.SetVotes(new(big.Int).Add(newCandidateAcc.GetVotes(), initFromBalance))

		case params.RegisterTx: // 注册candidate交易
			data := tx.Data()
			profile := make(types.Profile)
			err := json.Unmarshal(data, &profile)
			if err != nil {
				panic(err)
			}
			// 减去之前投票的票数,此时的票数为初始balance
			oldCandidate := fromAcc.GetVoteFor()
			if (oldCandidate != common.Address{}) {
				oldCandidateAcc := am.GetAccount(oldCandidate)
				oldCandidateAcc.SetVotes(new(big.Int).Sub(oldCandidateAcc.GetVotes(), initFromBalance))
			}
			// 设置from的balance
			fromAcc.SetBalance(new(big.Int).Sub(fromAcc.GetBalance(), params.RegisterCandidatePledgeAmount))
			// 把自己投给自己
			fromAcc.SetVoteFor(from)
			fromAcc.SetVotes(initFromBalance)
			// 设置candidate信息
			fromAcc.SetCandidate(profile)
		case params.CreateAssetTx: // 创建资产交易
			data := tx.Data()
			asset := &types.Asset{}
			err = json.Unmarshal(data, asset)
			if err != nil {
				panic(err)
			}
			asset.Issuer = from
			asset.AssetCode = tx.Hash()
			asset.TotalSupply = big.NewInt(0)
			fromAcc.SetAssetCode(asset.AssetCode, asset)
		case params.IssueAssetTx: // 发行资产交易
			issueAsset := &types.IssueAsset{}
			err := json.Unmarshal(tx.Data(), issueAsset)
			if err != nil {
				panic(err)
			}
			asset, err := fromAcc.GetAssetCode(issueAsset.AssetCode)
			// get asset总量
			oldTotalSupply := asset.TotalSupply
			var newTotalSupply *big.Int
			// 判断是否为可分割的资产
			if !asset.IsDivisible {
				// 不可分割，则每次发行资产总量加1
				newTotalSupply = new(big.Int).Add(oldTotalSupply, big.NewInt(1))
			} else {
				newTotalSupply = new(big.Int).Add(oldTotalSupply, issueAsset.Amount)
			}
			// 设置asset的总量
			err = fromAcc.SetAssetCodeTotalSupply(asset.AssetCode, newTotalSupply)
			if err != nil {
				panic(err)
			}
			// 	发行的资产到to账户上
			equity := &types.AssetEquity{}
			equity.AssetCode = asset.AssetCode
			equity.Equity = issueAsset.Amount
			// 判断资产类型
			AssType := asset.Category
			if AssType == types.Asset01 { // ERC20
				equity.AssetId = asset.AssetCode
			} else if AssType == types.Asset02 || AssType == types.Asset03 { // ERC721 or ERC721+20
				equity.AssetId = tx.Hash()
			}
			err = toAcc.SetEquityState(equity.AssetId, equity)
			if err != nil {
				panic(err)
			}
			err = toAcc.SetAssetIdState(equity.AssetId, issueAsset.MetaData)
			if err != nil {
				panic(err)
			}
		case params.ReplenishAssetTx: // 增发资产
			break
		case params.ModifyAssetTx: // 修改资产信息
			break
		case params.TransferAssetTx: // 交易资产
			// todo
		}
		// 	一笔交易执行完之后balance的变化导致的候选节点的票数的变化
		endFromBalance := fromAcc.GetBalance()
		endToBalance := toAcc.GetBalance()
		// from 和 to 的banlance变化的值
		fromChangeBalance := new(big.Int).Sub(endFromBalance, initFromBalance)
		toChangeBalance := new(big.Int).Sub(endToBalance, initToBalance)
		// from and to 账户的投票者
		if (fromAcc.GetVoteFor() != common.Address{}) {
			fromVoteAcc := am.GetAccount(fromAcc.GetVoteFor())
			fromVoteAcc.SetVotes(new(big.Int).Add(fromVoteAcc.GetVotes(), fromChangeBalance))
		}
		if (toAcc.GetVoteFor() != common.Address{}) {
			toVoteAcc := am.GetAccount(toAcc.GetVoteFor())
			toVoteAcc.SetVotes(new(big.Int).Add(toVoteAcc.GetVotes(), toChangeBalance))
		}
		// 	如果有代付者，则改变代付者的投票的状态
		if len(tx.GasPayerSig()) != 0 {
			if (gasPayerAcc.GetVoteFor() != common.Address{}) {
				gasPayerVoteAcc := am.GetAccount(gasPayerAcc.GetVoteFor())
				gasPayerVoteAcc.SetVotes(new(big.Int).Sub(gasPayerVoteAcc.GetVotes(), gasFee))
			}
		}
	}

	editAccountsEnd := time.Now().UnixNano()
	// 得到所有的changlog之后，merge changlog
	am.MergeChangeLogs()
	err := am.Finalise()
	if err != nil {
		panic(err)
	}
	finaliseEnd := time.Now().UnixNano()

	// 生成区块头
	changlogs := am.GetChangeLogs()
	header := &types.Header{
		ParentHash:   info.parentHash,
		MinerAddress: info.author,
		VersionRoot:  am.GetVersionRoot(),
		TxRoot:       txRoot,
		LogRoot:      changlogs.MerkleRootSha(),
		Height:       info.height,
		GasLimit:     info.gasLimit,
		GasUsed:      GasUsed,
		Time:         info.time,
		SignData:     nil,
		DeputyRoot:   nil,
		Extra:        nil,
	}
	newBlock := &types.Block{
		Header:      header,
		Txs:         info.txList,
		ChangeLogs:  changlogs,
		Confirms:    nil,
		DeputyNodes: nil,
	}
	blockHash := newBlock.Hash()

	buildBlockEnd := time.Now().UnixNano()
	// 设置稳定块
	if save {
		err := db.SetBlock(blockHash, newBlock)
		if err != nil {
			panic(err)
		}
		err = am.Save(blockHash)
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
	return newBlock
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
