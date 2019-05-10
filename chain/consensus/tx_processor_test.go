package consensus

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/flag"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"github.com/stretchr/testify/assert"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

var config = Config{
	LogForks:      false,
	RewardManager: common.HexToAddress("0x11001"),
	ChainID:       chainID,
	MineTimeout:   timeOutTime,
}

// 创世块的hash for test
var (
	testBlockHash    = common.HexToHash("0x1111111111")
	testMinerAddress = common.HexToAddress("0x1111111")
)

// newDbForTest db for test
func newDbForTest() protocol.ChainDB {
	return store.NewChainDataBase(GetStorePath(), "", "")
}

func TestNewTxProcessor(t *testing.T) {
	ClearData()
	db := newDbForTest()
	defer db.Close()
	am := account.NewManager(testBlockHash, db)

	p := NewTxProcessor(config, nil, am, db)
	assert.Equal(t, chainID, p.ChainID)
	assert.Equal(t, config.RewardManager, p.cfg.RewardManager)
	assert.False(t, p.cfg.Debug)
}

// test valid block processing
func TestTxProcessor_Process(t *testing.T) {
	ClearData()
	db := newDbForTest()
	defer db.Close()
	info := blockInfo{
		parentHash:  testBlockHash,
		height:      1,
		author:      testMinerAddress,
		versionRoot: common.Hash{},
		txRoot:      common.Hash{},
		logRoot:     common.Hash{},
		txList:      nil,
		gasLimit:    0,
		time:        0,
		deputyRoot:  nil,
		deputyNodes: nil,
	}
	block := makeBlock(db)

	p.am.GetAccount(testAddr)
	// last not stable block
	block := defaultBlocks[2]
	gasUsed, err := p.Process(block.Header, block.Txs)
	assert.NoError(t, err)
	assert.Equal(t, block.Header.GasUsed, gasUsed)
	err = p.am.Finalise()
	assert.NoError(t, err)
	assert.Equal(t, block.Header.VersionRoot, p.am.GetVersionRoot())
	assert.Equal(t, len(block.ChangeLogs), len(p.am.GetChangeLogs()))
	p.am.GetAccount(testAddr)

	// block not in db
	block = defaultBlocks[3]
	gasUsed, err = p.Process(block.Header, block.Txs)
	assert.NoError(t, err)
	err = p.am.Finalise()
	assert.NoError(t, err)
	assert.Equal(t, block.Header.GasUsed, gasUsed)
	// TODO these test is fail because Account.GetNextVersion always +1, so that the ChangeLog is not continuous. This will be fixed if we refactor the logic of ChangeLog merging to makes all version continuous in account.
	assert.Equal(t, block.Header.VersionRoot, p.am.GetVersionRoot())
	assert.Equal(t, len(block.ChangeLogs), len(p.am.GetChangeLogs()))
	p.am.GetAccount(testAddr)

	// genesis block
	block = defaultBlocks[0]
	_, err = p.Process(block.Header, block.Txs)
	assert.Equal(t, err, ErrInvalidGenesis)

	// block on fork branch
	block = createNewBlock(bc.db)
	gasUsed, err = p.Process(block.Header, block.Txs)
	assert.NoError(t, err)
	err = p.am.Finalise()
	assert.NoError(t, err)
	assert.Equal(t, block.Header.GasUsed, gasUsed)
	assert.Equal(t, block.Header.VersionRoot, p.am.GetVersionRoot())
	assert.Equal(t, len(block.ChangeLogs), len(p.am.GetChangeLogs()))
}

// test invalid block processing
func TestTxProcessor_Process2(t *testing.T) {
	ClearData()
	bc := newChain()
	defer bc.db.Close()
	p := NewTxProcessor(bc)

	// tamper with amount
	block := createNewBlock(bc.db)
	rawTx, _ := rlp.EncodeToBytes(block.Txs[0])
	rawTx[29]++ // amount++
	cpy := new(types.Transaction)
	err := rlp.DecodeBytes(rawTx, cpy)
	assert.NoError(t, err)
	assert.Equal(t, new(big.Int).Add(block.Txs[0].Amount(), big.NewInt(1)), cpy.Amount())
	block.Txs[0] = cpy
	_, err = p.Process(block.Header, block.Txs)
	assert.Equal(t, ErrInvalidTxInBlock, err)

	// invalid signature
	block = createNewBlock(bc.db)
	rawTx, _ = rlp.EncodeToBytes(block.Txs[0])
	rawTx[43] = 0 // invalid S
	cpy = new(types.Transaction)
	err = rlp.DecodeBytes(rawTx, cpy)
	assert.NoError(t, err)
	block.Txs[0] = cpy
	_, err = p.Process(block.Header, block.Txs)
	assert.Equal(t, ErrInvalidTxInBlock, err)

	// not enough gas (resign by another address)
	block = createNewBlock(bc.db)
	private, _ := crypto.GenerateKey()
	origFrom, _ := block.Txs[0].From()
	block.Txs[0] = signTransaction(block.Txs[0], private)
	newFrom, _ := block.Txs[0].From()
	assert.NotEqual(t, origFrom, newFrom)
	block.Header.TxRoot = block.Txs.MerkleRootSha()
	_, err = p.Process(block.Header, block.Txs)
	assert.Equal(t, ErrInvalidTxInBlock, err)

	// exceed block gas limit
	block = createNewBlock(bc.db)
	block.Header.GasLimit = 1
	_, err = p.Process(block.Header, block.Txs)
	assert.Equal(t, ErrInvalidTxInBlock, err)

	// used gas reach limit in some tx
	block = createNewBlock(bc.db)
	block.Txs[0] = makeTransaction(testPrivate, defaultAccounts[1], params.OrdinaryTx, big.NewInt(100), common.Big1, 0, 1)
	block.Header.TxRoot = block.Txs.MerkleRootSha()
	_, err = p.Process(block.Header, block.Txs)
	assert.Equal(t, ErrInvalidTxInBlock, err)

	// balance not enough
	block = createNewBlock(bc.db)
	balance := p.am.GetAccount(testAddr).GetBalance()
	block.Txs[0] = makeTx(testPrivate, defaultAccounts[1], params.OrdinaryTx, new(big.Int).Add(balance, big.NewInt(1)))
	block.Header.TxRoot = block.Txs.MerkleRootSha()
	_, err = p.Process(block.Header, block.Txs)
	assert.Equal(t, ErrInvalidTxInBlock, err)

	// TODO test create or call contract fail
}

func createNewBlock(db protocol.ChainDB) *types.Block {
	// db := store.NewChainDataBase(store.GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
	return makeBlock(db, blockInfo{
		height:     2,
		parentHash: defaultBlocks[1].Hash(),
		author:     testAddr,
		txList: []*types.Transaction{
			makeTx(testPrivate, defaultAccounts[1], params.OrdinaryTx, big.NewInt(100)),
		}}, false)
}

// test tx picking logic
func TestTxProcessor_ApplyTxs(t *testing.T) {
	ClearData()
	bc := newChain()
	defer bc.db.Close()
	p := NewTxProcessor(bc)

	// 1 txs
	header := defaultBlocks[2].Header
	txs := defaultBlocks[2].Txs
	emptyHeader := &types.Header{
		ParentHash:   header.ParentHash,
		MinerAddress: header.MinerAddress,
		Height:       header.Height,
		GasLimit:     header.GasLimit,
		GasUsed:      header.GasUsed,
		Time:         header.Time,
	}
	selectedTxs, invalidTxs, gasUsed := p.ApplyTxs(emptyHeader, txs, int64(10000))
	p.am.MergeChangeLogs()
	p.am.Finalise()
	assert.Equal(t, header.GasUsed, gasUsed)
	assert.Equal(t, header.VersionRoot, p.am.GetVersionRoot())
	assert.Equal(t, defaultBlocks[2].ChangeLogs, p.am.GetChangeLogs())
	assert.Equal(t, len(txs), len(selectedTxs))
	assert.Equal(t, 0, len(invalidTxs))

	// 2 txs
	header = defaultBlocks[3].Header
	txs = defaultBlocks[3].Txs
	emptyHeader = &types.Header{
		ParentHash:   header.ParentHash,
		MinerAddress: header.MinerAddress,
		Height:       header.Height,
		GasLimit:     header.GasLimit,
		GasUsed:      header.GasUsed,
		Time:         header.Time,
	}
	selectedTxs, invalidTxs, gasUsed = p.ApplyTxs(emptyHeader, txs, int64(10000))
	p.am.MergeChangeLogs()
	p.am.Finalise()
	assert.Equal(t, header.GasUsed, gasUsed)
	assert.Equal(t, header.VersionRoot, p.am.GetVersionRoot())
	assert.Equal(t, defaultBlocks[3].ChangeLogs, p.am.GetChangeLogs())
	assert.Equal(t, len(txs), len(selectedTxs))
	assert.Equal(t, 0, len(invalidTxs))

	// 0 txs
	header = defaultBlocks[3].Header
	emptyHeader = &types.Header{
		ParentHash:   header.ParentHash,
		MinerAddress: header.MinerAddress,
		Height:       header.Height,
		GasLimit:     header.GasLimit,
		GasUsed:      header.GasUsed,
		Time:         header.Time,
	}
	p.am.Reset(emptyHeader.ParentHash)
	author := p.am.GetAccount(header.MinerAddress)
	origBalance := author.GetBalance()
	selectedTxs, invalidTxs, gasUsed = p.ApplyTxs(emptyHeader, nil, int64(10000))
	p.am.MergeChangeLogs()
	p.am.Finalise()
	assert.Equal(t, uint64(0), gasUsed)
	assert.Equal(t, defaultBlocks[2].VersionRoot(), p.am.GetVersionRoot()) // last block version root
	assert.Equal(t, 0, len(selectedTxs))
	assert.Equal(t, *origBalance, *author.GetBalance())
	assert.Equal(t, 0, len(p.am.GetChangeLogs()))

	// too many txs
	header = defaultBlocks[3].Header
	txs = defaultBlocks[3].Txs
	emptyHeader = &types.Header{
		ParentHash:   header.ParentHash,
		MinerAddress: header.MinerAddress,
		Height:       header.Height,
		GasLimit:     45000, // Every transaction's gasLimit is 30000. So the block only contains one transaction.
		GasUsed:      header.GasUsed,
		Time:         header.Time,
	}
	selectedTxs, invalidTxs, gasUsed = p.ApplyTxs(emptyHeader, txs, int64(10000))
	p.am.MergeChangeLogs()
	p.am.Finalise()
	assert.NotEqual(t, header.GasUsed, gasUsed)
	assert.NotEqual(t, header.VersionRoot, p.am.GetVersionRoot())
	assert.NotEqual(t, true, len(defaultBlocks[3].ChangeLogs), len(p.am.GetChangeLogs()))
	assert.NotEqual(t, len(txs), len(selectedTxs))
	assert.Equal(t, 0, len(invalidTxs))

	// balance not enough
	header = defaultBlocks[3].Header
	txs = defaultBlocks[3].Txs
	emptyHeader = &types.Header{
		ParentHash:   header.ParentHash,
		MinerAddress: header.MinerAddress,
		Height:       header.Height,
		GasLimit:     header.GasLimit,
		GasUsed:      header.GasUsed,
		Time:         header.Time,
	}
	p.am.Reset(emptyHeader.ParentHash)
	balance := p.am.GetAccount(testAddr).GetBalance()
	txs = types.Transactions{
		txs[0],
		makeTx(testPrivate, defaultAccounts[1], params.OrdinaryTx, new(big.Int).Add(balance, big.NewInt(1))),
		txs[1],
	}
	selectedTxs, invalidTxs, _ = p.ApplyTxs(emptyHeader, txs, int64(10000))
	p.am.MergeChangeLogs()
	p.am.Finalise()
	assert.Equal(t, len(txs)-1, len(selectedTxs))
	assert.Equal(t, 1, len(invalidTxs))
}

// TODO move these cases to evm
// test different transactions
func TestTxProcessor_ApplyTxs2(t *testing.T) {
	ClearData()
	bc := newChain()
	defer bc.db.Close()
	p := NewTxProcessor(bc)

	// transfer to other
	header := defaultBlocks[3].Header
	emptyHeader := &types.Header{
		ParentHash:   header.ParentHash,
		MinerAddress: header.MinerAddress,
		Height:       header.Height,
		GasLimit:     header.GasLimit,
		GasUsed:      header.GasUsed,
		Time:         header.Time,
	}
	p.am.Reset(emptyHeader.ParentHash)
	senderBalance := p.am.GetAccount(testAddr).GetBalance()
	minerBalance := p.am.GetAccount(defaultAccounts[0]).GetBalance()
	recipientBalance := p.am.GetAccount(defaultAccounts[1]).GetBalance()
	txs := types.Transactions{
		makeTx(testPrivate, defaultAccounts[1], params.OrdinaryTx, common.Big1),
	}
	_, _, GasUsed := p.ApplyTxs(emptyHeader, txs, int64(10000))
	assert.Equal(t, params.TxGas, GasUsed)
	newSenderBalance := p.am.GetAccount(testAddr).GetBalance()
	newMinerBalance := p.am.GetAccount(defaultAccounts[0]).GetBalance()
	newRecipientBalance := p.am.GetAccount(defaultAccounts[1]).GetBalance()
	cost := txs[0].GasPrice().Mul(txs[0].GasPrice(), big.NewInt(int64(params.TxGas)))
	senderBalance.Sub(senderBalance, cost)
	senderBalance.Sub(senderBalance, common.Big1)
	assert.Equal(t, senderBalance, newSenderBalance)
	assert.Equal(t, minerBalance.Add(minerBalance, cost), newMinerBalance)
	assert.Equal(t, recipientBalance.Add(recipientBalance, common.Big1), newRecipientBalance)

	// transfer to self, only cost gas
	header = defaultBlocks[3].Header
	emptyHeader = &types.Header{
		ParentHash:   header.ParentHash,
		MinerAddress: header.MinerAddress,
		Height:       header.Height,
		GasLimit:     header.GasLimit,
		GasUsed:      header.GasUsed,
		Time:         header.Time,
	}
	p.am.Reset(emptyHeader.ParentHash)
	senderBalance = p.am.GetAccount(testAddr).GetBalance()
	txs = types.Transactions{
		makeTx(testPrivate, testAddr, params.OrdinaryTx, common.Big1),
	}
	_, _, GasUsed = p.ApplyTxs(emptyHeader, txs, int64(10000))
	assert.Equal(t, params.TxGas, GasUsed)
	newSenderBalance = p.am.GetAccount(testAddr).GetBalance()
	cost = txs[0].GasPrice().Mul(txs[0].GasPrice(), big.NewInt(int64(params.TxGas)))
	assert.Equal(t, senderBalance.Sub(senderBalance, cost), newSenderBalance)
}

func TestApplyTxsTimeoutTime(t *testing.T) {
	ClearData()
	bc := newChain()
	defer bc.db.Close()
	p := NewTxProcessor(bc)

	parentBlock := p.chain.stableBlock.Load().(*types.Block)
	header := &types.Header{
		ParentHash:   parentBlock.Hash(),
		MinerAddress: parentBlock.MinerAddress(),
		Height:       parentBlock.Height() + 1,
		GasLimit:     parentBlock.GasLimit(),
	}

	tx01 := makeTx(testPrivate, defaultAccounts[1], params.OrdinaryTx, common.Big1)
	txNum := 500
	txs := make([]*types.Transaction, 0, txNum)
	for i := 0; i < txNum; i++ {
		txs = append(txs, tx01)
	}
	selectedTxs01, _, _ := p.ApplyTxs(header, txs, int64(0))
	assert.NotEqual(t, len(selectedTxs01), txNum)
	selectedTxs02, _, _ := p.ApplyTxs(header, txs, int64(2))
	assert.NotEqual(t, len(selectedTxs02), txNum)
	selectedTxs03, _, _ := p.ApplyTxs(header, txs, int64(100))
	assert.Equal(t, len(selectedTxs03), txNum)
}

// TestTxProcessor_candidateTX 打包特殊交易测试
func TestTxProcessor_candidateTX(t *testing.T) {
	ClearData()
	bc := newChain()
	defer bc.db.Close()
	p := NewTxProcessor(bc)

	candData00 := createCandidateData(params.IsCandidateNode, "34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f", "0.0.0.0", "0.0.0.0", "0x10000")

	testAddr01, _ := common.StringToAddress("Lemo83W59DHT7FD4KSB3HWRJ5T4JD82TZW27ZKHJ")
	value := new(big.Int).Mul(params.RegisterCandidateNodeFees, big.NewInt(2)) // 转账为2000LEMO
	getBalanceTx01 := makeTx(testPrivate, testAddr01, params.OrdinaryTx, value)
	//
	registerTx01 := signTransaction(types.NewTransaction(params.FeeReceiveAddress, params.RegisterCandidateNodeFees, 200000, common.Big1, candData00, params.RegisterTx, chainID, uint64(time.Now().Unix()+300), "", ""), testPrivate)

	parentBlock := p.chain.stableBlock.Load().(*types.Block)
	txs01 := types.Transactions{registerTx01, getBalanceTx01}
	Block01, invalidTxs := newNextBlock(p, parentBlock, txs01, false)
	if len(invalidTxs) != 0 {
		panic("has invalid txs 01")
	}
	bbb := Block01.ChangeLogs
	BB, _ := rlp.EncodeToBytes(bbb[2])
	fmt.Println("rlp: ", common.ToHex(BB))
	t.Log(Block01.MinerAddress().String())
	GasUsed, err := p.Process(Block01.Header, Block01.Txs)
	if err != nil {
		fmt.Println("Process: ", err)
	}

	cc := p.am.GetChangeLogs()
	CC, _ := rlp.EncodeToBytes(cc[2])
	fmt.Println("rlp: ", common.ToHex(CC))
	assert.Equal(t, Block01.Header.GasUsed, GasUsed)
	assert.Equal(t, bbb, cc)
	assert.Equal(t, CC, BB)
	// 	临界值测试
	// candData01 := createCandidateData(params.NotCandidateNode)
}

func TestGetHashFn(t *testing.T) {
	ClearData()
	chain := newChain()
	defer chain.db.Close()
	p := NewTxProcessor(chain)

	candiAddr01, err := common.StringToAddress("Lemo8493289P3N6STKGRAPAT278FWD32S95S95ZS")
	assert.NoError(t, err)
	acc01 := p.am.GetAccount(candiAddr01)
	acc01.SetBalance(big.NewInt(5000000))
	p.am.Finalise()
	t.Log(acc01.GetBalance())

}

// create register candidate node tx data
func createCandidateData(isCandidata, nodeID, host, port, minerAdd string) []byte {
	pro := make(types.Profile)
	pro[types.CandidateKeyIsCandidate] = isCandidata
	pro[types.CandidateKeyNodeID] = nodeID
	pro[types.CandidateKeyHost] = host
	pro[types.CandidateKeyPort] = port
	pro[types.CandidateKeyIncomeAddress] = minerAdd
	data, err := json.Marshal(pro)
	if err != nil {
		return nil
	}
	return data
}

//  Test_voteAndRegisteTx 测试投票交易和注册候选节点交易
func Test_voteAndRegisteTx(t *testing.T) {
	ClearData()
	chain := newChain()
	defer chain.db.Close()
	p := NewTxProcessor(chain)

	// 申请第一个候选节点(testAddr)信息data
	cand00 := make(types.Profile)
	cand00[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	cand00[types.CandidateKeyPort] = "0000"
	cand00[types.CandidateKeyNodeID] = "34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"
	cand00[types.CandidateKeyIncomeAddress] = common.HexToAddress("0x10000").String()
	cand00[types.CandidateKeyHost] = "0.0.0.0"
	candData00, _ := json.Marshal(cand00)

	// 申请第二个候选节点(testAddr02)信息data
	cand02 := make(types.Profile)
	cand02[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	cand02[types.CandidateKeyPort] = "2222"
	cand02[types.CandidateKeyNodeID] = "7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2d0347692f36ef736f342fb5ce4710f7e337f062cc2110d134b63a9575f78cb167bfae2f43"
	cand02[types.CandidateKeyIncomeAddress] = common.HexToAddress("0x222222").String()
	cand02[types.CandidateKeyHost] = "2.2.2.2"
	candData02, _ := json.Marshal(cand02)

	// 生成有balance的account
	testAddr01, _ := common.StringToAddress("Lemo83W59DHT7FD4KSB3HWRJ5T4JD82TZW27ZKHJ")
	testPrivate01, _ := crypto.HexToECDSA("7a720181f628d9b132af6730d797fc3486adfb2993f0796ac6854f5885697746")
	testAddr02, _ := common.StringToAddress("Lemo83F96RQR3J5GW8CS35JWP2A4QBQ3CYHHQJAK")
	testPrivate02, _ := crypto.HexToECDSA("5462a02f5fbac2ae8e157d95809aa57fc6f12095b14ee95b051aa9d47ad054f4")
	testAddr03, _ := common.StringToAddress("Lemo843A8K22PDK9BSZT8SDN95GASSRSDW2DJZ3S")
	// testPrivate03, _ := crypto.HexToECDSA("197e8f49f38487f2b435bc8eb1f2d9fec5cde987d0c91926921d8ac8ac7f7261")
	value := new(big.Int).Mul(params.RegisterCandidateNodeFees, big.NewInt(2)) // 转账为2000LEMO
	// 给testAdd01账户转账
	getBalanceTx01 := makeTx(testPrivate, testAddr01, params.OrdinaryTx, value)
	// 给testAdd02账户转账
	getBalanceTx02 := makeTx(testPrivate, testAddr02, params.OrdinaryTx, value)
	// 给testAdd03账户转账
	getBalanceTx03 := makeTx(testPrivate, testAddr03, params.OrdinaryTx, value)

	// ---Block01----------------------------------------------------------------------
	// 1. testAddr 账户申请候选节点交易，包含给testAddr01,testAddr02,testAddr03转账交易
	registerTx01 := signTransaction(types.NewTransaction(params.FeeReceiveAddress, params.RegisterCandidateNodeFees, 200000, common.Big1, candData00, params.RegisterTx, chainID, uint64(time.Now().Unix()+300), "", ""), testPrivate)

	parentBlock := p.chain.currentBlock.Load().(*types.Block)
	txs01 := types.Transactions{registerTx01, getBalanceTx01, getBalanceTx02, getBalanceTx03}
	Block01, invalidTx := newNextBlock(p, parentBlock, txs01, true)
	if len(invalidTx) != 0 {
		panic("have invalid txs 02")
	}
	blockHash := Block01.Hash()
	result := p.chain.db.GetCandidatesTop(blockHash)
	assert.Equal(t, 1, len(result))

	// 	验证注册代理节点交易信息
	testAddr, _ := registerTx01.From()
	account00 := p.am.GetCanonicalAccount(testAddr)
	assert.Equal(t, testAddr, account00.GetVoteFor())                               // 投给自己
	assert.Equal(t, account00.GetBalance().String(), account00.GetVotes().String()) // 初始票数为自己的Balance
	profile := account00.GetCandidate()
	assert.Equal(t, cand00[types.CandidateKeyIncomeAddress], profile[types.CandidateKeyIncomeAddress])
	assert.Equal(t, cand00[types.CandidateKeyHost], profile[types.CandidateKeyHost])
	assert.Equal(t, cand00[types.CandidateKeyPort], profile[types.CandidateKeyPort])
	assert.Equal(t, cand00[types.CandidateKeyNodeID], profile[types.CandidateKeyNodeID])
	log.Warn("account00Vote:", account00.GetVotes().String())
	// ---Block02-----------------------------------------------------------------------
	//  2. 测试发送投票交易,testAddr01账户为testAddr候选节点账户投票,并注册testAddr02为候选节点
	// p.am.Reset(Block01.Hash())
	// 投票交易
	voteTx01 := makeTx(testPrivate01, testAddr, params.VoteTx, big.NewInt(0))
	// 注册testAddr02为候选节点的交易
	registerTx02 := signTransaction(types.NewTransaction(params.FeeReceiveAddress, params.RegisterCandidateNodeFees, 200000, common.Big1, candData02, params.RegisterTx, chainID, uint64(time.Now().Unix()+300), "", ""), testPrivate02)
	txs02 := types.Transactions{voteTx01, registerTx02}
	Block02, invalidTx := newNextBlock(p, Block01, txs02, true)
	if len(invalidTx) != 0 {
		panic("have invalid txs 03")
	}
	// 	验证1. 投票交易后的结果
	account01 := p.am.GetCanonicalAccount(testAddr01)
	newAccount00 := p.am.GetCanonicalAccount(testAddr)
	assert.Equal(t, testAddr, account01.GetVoteFor()) // 是否投给了指定的address
	block02testAddr00Votes := newAccount00.GetVotes()
	assert.Equal(t, new(big.Int).Add(newAccount00.GetBalance(), account01.GetBalance()), block02testAddr00Votes) // 票数是否增加了期望的值
	// 验证2. testAddr02注册代理节点的结果
	address02, _ := registerTx02.From()
	account02 := p.am.GetCanonicalAccount(address02)
	block02testAddr02Votes := account02.GetVotes()
	assert.Equal(t, address02, account02.GetVoteFor())                                // 默认投给自己
	assert.Equal(t, account02.GetBalance().String(), block02testAddr02Votes.String()) // 初始票数为自己的Balance
	profile02 := account02.GetCandidate()
	assert.Equal(t, cand02[types.CandidateKeyIncomeAddress], profile02[types.CandidateKeyIncomeAddress])
	assert.Equal(t, cand02[types.CandidateKeyHost], profile02[types.CandidateKeyHost])
	assert.Equal(t, cand02[types.CandidateKeyPort], profile02[types.CandidateKeyPort])
	assert.Equal(t, cand02[types.CandidateKeyNodeID], profile02[types.CandidateKeyNodeID])
	// ---Block03-----------------------------------------------------------------------------
	// 3. testAddr01从候选节点testAddr 转投 给候选节点testAddr02; 候选节点testAddr修改注册信息
	// 	投票交易
	voteTx02 := makeTx(testPrivate01, address02, params.VoteTx, big.NewInt(0))
	// 修改候选节点profile交易
	changeCand00 := make(types.Profile)
	changeCand00[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	changeCand00[types.CandidateKeyPort] = "8080"
	changeCand00[types.CandidateKeyNodeID] = "34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"
	changeCand00[types.CandidateKeyIncomeAddress] = common.HexToAddress("0x222222").String()
	changeCand00[types.CandidateKeyHost] = "www.changeIndo.org"

	changeCandData00, _ := json.Marshal(changeCand00)
	registerTx03 := signTransaction(types.NewTransaction(params.FeeReceiveAddress, params.RegisterCandidateNodeFees, 200000, common.Big1, changeCandData00, params.RegisterTx, chainID, uint64(time.Now().Unix()+300), "", ""), testPrivate)
	txs03 := types.Transactions{voteTx02, registerTx03}

	Block03, invalidTx := newNextBlock(p, Block02, txs03, true)
	if len(invalidTx) != 0 {
		panic("have invalid txs 04")
	}
	assert.NotEmpty(t, Block03)

	// 	验证1. 候选节点testAddr票数减少量 = testAddr01的Balance，候选节点testAddr02票数增加量 = testAddr01的Balance
	latestAccount00 := p.am.GetCanonicalAccount(testAddr)
	block03testAddr00Votes := latestAccount00.GetVotes()
	log.Warn("block03testAddr00Votes:", block03testAddr00Votes.String())
	log.Warn("block02testAddr00Votes:", block02testAddr00Votes.String())

	// subVote00 := new(big.Int).Sub(block02testAddr00Votes, block03testAddr00Votes)
	testAccount01 := p.am.GetCanonicalAccount(testAddr01)
	// assert.Equal(t, new(big.Int).Sub(subVote00, new(big.Int).Add(big.NewInt(16210), params.RegisterCandidateNodeFees)), new(big.Int).Add(testAccount01.GetBalance(), big.NewInt(42000)))

	latestAccount02 := p.am.GetCanonicalAccount(testAddr02)
	block03testAddr02Votes := latestAccount02.GetVotes()
	addVotes02 := new(big.Int).Sub(block03testAddr02Votes, block02testAddr02Votes)
	assert.Equal(t, addVotes02, testAccount01.GetBalance())
}

// Test_CreatRegisterTxData 注册候选节点所用交易data
func Test_CreatRegisterTxData(t *testing.T) {
	pro1 := make(types.Profile)
	pro1[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	pro1[types.CandidateKeyPort] = "1111"
	pro1[types.CandidateKeyNodeID] = "5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"
	pro1[types.CandidateKeyIncomeAddress] = "Lemo83JZRYPYF97CFSZBBQBH4GW42PD8CFHT5ARN"
	pro1[types.CandidateKeyHost] = "1111"
	marPro1, _ := json.Marshal(pro1)
	fmt.Println("txData1:", common.ToHex(marPro1))

	pro2 := make(types.Profile)
	pro2[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	pro2[types.CandidateKeyPort] = "2222"
	pro2[types.CandidateKeyNodeID] = "34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"
	pro2[types.CandidateKeyIncomeAddress] = common.HexToAddress("0x222").String()
	pro2[types.CandidateKeyHost] = "2222"
	marPro2, _ := json.Marshal(pro2)
	fmt.Println("txData2:", common.ToHex(marPro2))

	pro3 := make(types.Profile)
	pro3[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	pro3[types.CandidateKeyPort] = "3333"
	pro3[types.CandidateKeyNodeID] = "5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"
	pro3[types.CandidateKeyIncomeAddress] = common.HexToAddress("0x333").String()
	pro3[types.CandidateKeyHost] = "3333"
	marPro3, _ := json.Marshal(pro3)
	fmt.Println("txData3:", common.ToHex(marPro3))

	pro4 := make(types.Profile)
	pro4[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	pro4[types.CandidateKeyPort] = "4444"
	pro4[types.CandidateKeyNodeID] = "ddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0"
	pro4[types.CandidateKeyIncomeAddress] = common.HexToAddress("0x444").String()
	pro4[types.CandidateKeyHost] = "4444"
	marPro4, _ := json.Marshal(pro4)
	fmt.Println("txData4:", common.ToHex(marPro4))

	pro5 := make(types.Profile)
	pro5[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	pro5[types.CandidateKeyPort] = "5555"
	pro5[types.CandidateKeyNodeID] = "7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2d0347692f36ef736f342fb5ce4710f7e337f062cc2110d134b63a9575f78cb167bfae2f43"
	pro5[types.CandidateKeyIncomeAddress] = common.HexToAddress("0x555").String()
	pro5[types.CandidateKeyHost] = "5555"
	marPro5, _ := json.Marshal(pro5)
	fmt.Println("txData5:", common.ToHex(marPro5))
}

// TestReimbursement_transaction 打包并验证代付gas交易测试
func TestReimbursement_transaction(t *testing.T) {
	var (
		senderPrivate, _   = crypto.HexToECDSA("c8fa12aa54fbcc249611e5fefa0967658a7ca06022e9d50b53ef6f5b050b697f")
		senderAddr         = crypto.PubkeyToAddress(senderPrivate.PublicKey)
		gasPayerPrivate, _ = crypto.HexToECDSA("57a0b0be5616e74c4315882e3649ade12c775db3b5023dcaa168d01825612c9b")
		gasPayerAddr       = crypto.PubkeyToAddress(gasPayerPrivate.PublicKey)
		Tx01               = makeTx(testPrivate, gasPayerAddr, params.OrdinaryTx, params.RegisterCandidateNodeFees) // 转账1000LEMO给gasPayerAddr
		Tx02               = makeTx(testPrivate, senderAddr, params.OrdinaryTx, params.RegisterCandidateNodeFees)   // 转账1000LEMO给senderAddr

		amountReceiver = common.HexToAddress("0x1234")
		TxV01          = types.NewReimbursementTransaction(amountReceiver, gasPayerAddr, params.RegisterCandidateNodeFees, []byte{}, params.OrdinaryTx, chainID, uint64(time.Now().Unix()+300), "", "")
	)
	ClearData()
	chain := newChain()
	defer chain.db.Close()
	p := NewTxProcessor(chain)

	// create a block contains two account which used to make reimbursement transaction
	parentBlock := p.chain.stableBlock.Load().(*types.Block)
	Block01, invalidTxs := newNextBlock(p, parentBlock, types.Transactions{Tx01, Tx02}, true)
	if len(invalidTxs) != 0 {
		panic("has invalid txs 05")
	}
	p.am.Reset(Block01.Hash())
	// check their balance
	gasPayerAcc := p.am.GetAccount(gasPayerAddr)
	senderAcc := p.am.GetAccount(senderAddr)
	initGasPayerBalance := gasPayerAcc.GetBalance()
	initTxSenderBalance := senderAcc.GetBalance()
	assert.Equal(t, params.RegisterCandidateNodeFees, initGasPayerBalance)
	assert.Equal(t, params.RegisterCandidateNodeFees, initTxSenderBalance)

	// sender transfer LEMO to receiver, payer pay for that transaction
	firstSignTxV, err := types.MakeReimbursementTxSigner().SignTx(TxV01, senderPrivate)
	assert.NoError(t, err)
	firstSignTxV = types.GasPayerSignatureTx(firstSignTxV, common.Big1, uint64(60000))
	lastSignTxV, err := types.MakeGasPayerSigner().SignTx(firstSignTxV, gasPayerPrivate)
	assert.NoError(t, err)
	newNextBlock(p, Block01, types.Transactions{lastSignTxV}, true)
	// check their balance
	endGasPayerBalance := p.am.GetAccount(gasPayerAddr).GetBalance()
	endTxSenderBalance := p.am.GetAccount(senderAddr).GetBalance()
	assert.Equal(t, big.NewInt(0), endTxSenderBalance)
	assert.Equal(t, endGasPayerBalance, new(big.Int).Sub(initGasPayerBalance, big.NewInt(int64(params.TxGas))))
	assert.Equal(t, params.RegisterCandidateNodeFees, p.am.GetAccount(amountReceiver).GetBalance())
}

// TestBlockChain_txData 生成调用设置换届奖励的预编译合约交易的data
func TestBlockChain_data(t *testing.T) {
	re := params.RewardJson{
		Term:  3,
		Value: big.NewInt(3330),
	}
	by, _ := json.Marshal(re)
	fmt.Println("tx data", common.ToHex(by))
	fmt.Println("预编译合约地址", common.BytesToAddress([]byte{9}).String())
}

// // newNextBlock new a block
// func newNextBlock(p *TxProcessor, parentBlock *types.Block, txs types.Transactions, save bool) *types.Block {
// 	blockInfo := blockInfo{
// 		parentHash: parentBlock.Hash(),
// 		height:     parentBlock.Height() + 1,
// 		author:     parentBlock.MinerAddress(),
// 		time:       parentBlock.Time() + 4,
// 		gasLimit:   parentBlock.GasLimit(),
// 	}
// 	newBlock := makeBlock(p.chain.db, blockInfo, save)
//
// 	gasUsed, err := p.Process(newBlock.Header, newBlock.Txs)
// 	if newBlock.GasUsed() != gasUsed {
// 		panic(fmt.Errorf("gasUsed is different. before process: %d, after: %d", newBlock.GasUsed(), gasUsed))
// 	}
// 	if save {
// 		err = p.chain.db.SetStableBlock(newBlock.Hash())
// 		p.am.Reset(newBlock.Hash())
// 		if err != nil {
// 			panic(err)
// 		}
// 	}
// 	return newBlock
// }

// newNextBlock new a block
func newNextBlock(p *TxProcessor, parentBlock *types.Block, txs types.Transactions, save bool) (*types.Block, types.Transactions) {
	header01 := &types.Header{
		ParentHash:   parentBlock.Hash(),
		MinerAddress: parentBlock.MinerAddress(),
		Height:       parentBlock.Height() + 1,
		GasLimit:     parentBlock.GasLimit(),
		Time:         parentBlock.Time() + 4,
	}
	packagedTxs, invalidTxs, gasUsed := p.ApplyTxs(header01, txs, int64(10000))
	err := p.chain.engine.Finalize(header01.Height, p.am)
	if err != nil {
		panic(err)
	}
	// seal block
	newBlock, err := p.chain.engine.Seal(header01, p.am.GetTxsProduct(packagedTxs, gasUsed), nil, nil)
	if err != nil {
		panic(err)
	}
	// save
	BlockHash := newBlock.Hash()
	err = p.chain.db.SetBlock(BlockHash, newBlock)
	if err != nil {
		panic(err)
	}
	p.am.Save(BlockHash)

	if save {
		err = p.chain.db.SetStableBlock(BlockHash)
		p.am.Reset(BlockHash)
		if err != nil {
			panic(err)
		}
	}
	return newBlock, invalidTxs
}

// TestCreateAssetTx create asset test
func TestCreateAssetTx(t *testing.T) {
	tx01, err := newCreateAssetTx(testPrivate, types.Asset01, true, true)
	assert.NoError(t, err)
	ClearData()
	chain := newChain()
	defer chain.db.Close()
	p := NewTxProcessor(chain)
	parentBlock := p.chain.stableBlock.Load().(*types.Block)
	txs := types.Transactions{tx01}
	block01, invalidTxs := newNextBlock(p, parentBlock, txs, true)
	if len(invalidTxs) != 0 {
		panic("has invalid txs 06")
	}
	// 	compare
	p.am.Reset(block01.Hash())
	senderAcc := p.am.GetAccount(crypto.PubkeyToAddress(testPrivate.PublicKey))
	asset, err := senderAcc.GetAssetCode(tx01.Hash())
	assert.NoError(t, err)
	assert.Equal(t, types.Asset01, asset.Category)
	assert.Equal(t, true, asset.IsDivisible, asset.IsReplenishable)
	assert.Equal(t, "Demo Token", asset.Profile[types.AssetName])
	assert.Equal(t, "DT", asset.Profile[types.AssetSymbol])
	assert.Equal(t, "test issue token", asset.Profile[types.AssetDescription])
	assert.Equal(t, "false", asset.Profile[types.AssetFreeze])
	assert.Equal(t, "60000", asset.Profile[types.AssetSuggestedGasLimit])

}

// newCreateAssetTx
func newCreateAssetTx(private *ecdsa.PrivateKey, category uint32, isReplenishable, isDivisible bool) (*types.Transaction, error) {
	issuer := crypto.PubkeyToAddress(private.PublicKey)
	profile := make(types.Profile)
	profile[types.AssetName] = "Demo Token"
	profile[types.AssetSymbol] = "DT"
	profile[types.AssetDescription] = "test issue token"
	profile[types.AssetFreeze] = "false"
	profile[types.AssetSuggestedGasLimit] = "60000"
	asset := &types.Asset{
		Category:        category,
		IsDivisible:     isDivisible,
		AssetCode:       common.Hash{},
		Decimal:         5,
		TotalSupply:     big.NewInt(10000000),
		IsReplenishable: isReplenishable,
		Issuer:          issuer,
		Profile:         profile,
	}
	data, _ := json.Marshal(asset)
	tx := types.NoReceiverTransaction(nil, uint64(500000), big.NewInt(1), data, params.CreateAssetTx, chainID, uint64(time.Now().Unix()+30*60), "", "create asset tx")
	return types.MakeSigner().SignTx(tx, private)
}

// TestIssueAssetTest issue asset tx test
func TestIssueAssetTest(t *testing.T) {
	ClearData()
	bc := newChain()
	defer bc.db.Close()
	p := NewTxProcessor(bc)
	parentBlock := p.chain.stableBlock.Load().(*types.Block)

	// 执行创建资产的交易的区块
	tx01, err := newCreateAssetTx(testPrivate, types.Asset01, false, true)
	assert.NoError(t, err)
	tx02, err := newCreateAssetTx(testPrivate, types.Asset02, false, false)
	assert.NoError(t, err)
	tx03, err := newCreateAssetTx(testPrivate, types.Asset03, true, true)
	Ctxs := types.Transactions{tx01, tx02, tx03}
	block01, invalidTxs := newNextBlock(p, parentBlock, Ctxs, true)
	if len(invalidTxs) != 0 {
		panic("has invalid txs 07")
	}
	asset01Code := tx01.Hash()
	asset02Code := tx02.Hash()
	asset03Code := tx03.Hash()

	// 执行发行资产的交易的区块
	receiver := common.HexToAddress("0x0813")
	issAsset01Tx, err := newIssueAssetTx(testPrivate, receiver, asset01Code, big.NewInt(110), "issue erc20 asset")
	assert.NoError(t, err)
	issAsset02Tx, err := newIssueAssetTx(testPrivate, receiver, asset02Code, big.NewInt(110), "issue erc721 asset")
	assert.NoError(t, err)
	issAsset03Tx, err := newIssueAssetTx(testPrivate, receiver, asset03Code, big.NewInt(110), "issue erc721+20 asset")
	assert.NoError(t, err)
	Itxs := types.Transactions{issAsset01Tx, issAsset02Tx, issAsset03Tx}
	block02, invalidTxs := newNextBlock(p, block01, Itxs, true)
	if len(invalidTxs) != 0 {
		panic("has invalid txs 08")
	}
	assert.Equal(t, block01.Height()+1, block02.Height())

	// 验证资产issuer中的资产totalSupply
	issuerAcc := p.am.GetAccount(testAddr)
	asset01Total, err := issuerAcc.GetAssetCodeTotalSupply(asset01Code)
	assert.NoError(t, err)
	asset02Total, err := issuerAcc.GetAssetCodeTotalSupply(asset02Code)
	assert.NoError(t, err)
	asset03Total, err := issuerAcc.GetAssetCodeTotalSupply(asset03Code)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(110), asset01Total)
	assert.Equal(t, big.NewInt(1), asset02Total)
	assert.Equal(t, big.NewInt(110), asset03Total)

	// 验证receiver中的资产余额
	asset01Id := asset01Code
	asset02Id := issAsset02Tx.Hash()
	asset03Id := issAsset03Tx.Hash()
	receiverAcc := p.am.GetAccount(receiver)
	equity01, err := receiverAcc.GetEquityState(asset01Id)
	assert.NoError(t, err)
	equity02, err := receiverAcc.GetEquityState(asset02Id)
	assert.NoError(t, err)
	equity03, err := receiverAcc.GetEquityState(asset03Id)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(110), equity01.Equity)
	assert.Equal(t, big.NewInt(110), equity02.Equity)
	assert.Equal(t, big.NewInt(110), equity03.Equity)

	// 	验证metaData
	metaData01, err := receiverAcc.GetAssetIdState(asset01Id)
	assert.NoError(t, err)
	metaData02, err := receiverAcc.GetAssetIdState(asset02Id)
	assert.NoError(t, err)
	metaData03, err := receiverAcc.GetAssetIdState(asset03Id)
	assert.NoError(t, err)
	assert.Equal(t, "issue erc20 asset", metaData01)
	assert.Equal(t, "issue erc721 asset", metaData02)
	assert.Equal(t, "issue erc721+20 asset", metaData03)
}

// new发行资产交易
func newIssueAssetTx(prv *ecdsa.PrivateKey, receiver common.Address, assetCode common.Hash, amount *big.Int, metaData string) (*types.Transaction, error) {
	issue := &types.IssueAsset{
		AssetCode: assetCode,
		MetaData:  metaData,
		Amount:    amount,
	}
	data, err := json.Marshal(issue)
	if err != nil {
		return nil, err
	}
	tx := types.NewTransaction(receiver, nil, uint64(500000), big.NewInt(1), data, params.IssueAssetTx, chainID, uint64(time.Now().Unix()+30*60), "", "issue asset tx")

	signTx, err := types.MakeSigner().SignTx(tx, prv)
	if err != nil {
		return nil, err
	}
	return signTx, nil
}

// relenishAsset tx test
func TestReplenishAssetTx(t *testing.T) {
	ClearData()
	bc := newChain()
	defer bc.db.Close()
	p := NewTxProcessor(bc)
	parentBlock := p.chain.stableBlock.Load().(*types.Block)

	// 执行创建资产的交易的区块
	tx01, err := newCreateAssetTx(testPrivate, types.Asset01, true, true) // 可增发
	assert.NoError(t, err)
	tx02, err := newCreateAssetTx(testPrivate, types.Asset02, false, false) // 不可增发
	assert.NoError(t, err)
	tx03, err := newCreateAssetTx(testPrivate, types.Asset03, false, true) // 不可增发
	Ctxs := types.Transactions{tx01, tx02, tx03}
	block01, invalidTxs := newNextBlock(p, parentBlock, Ctxs, true)
	if len(invalidTxs) != 0 {
		panic("has invalid txs 10")
	}
	asset01Code := tx01.Hash()
	asset02Code := tx02.Hash()
	asset03Code := tx03.Hash()
	// 执行发行资产的交易的区块
	receiver := common.HexToAddress("0x0813")
	issAsset01Tx, err := newIssueAssetTx(testPrivate, receiver, asset01Code, big.NewInt(110), "issue erc20 asset")
	assert.NoError(t, err)
	issAsset02Tx, err := newIssueAssetTx(testPrivate, receiver, asset02Code, big.NewInt(110), "issue erc721 asset")
	assert.NoError(t, err)
	issAsset03Tx, err := newIssueAssetTx(testPrivate, receiver, asset03Code, big.NewInt(110), "issue erc721+20 asset")
	assert.NoError(t, err)
	Itxs := types.Transactions{issAsset01Tx, issAsset02Tx, issAsset03Tx}
	block02, invalidTxs := newNextBlock(p, block01, Itxs, true)
	if len(invalidTxs) != 0 {
		panic("has invalid txs 11")
	}
	assert.Equal(t, block01.Height()+1, block02.Height())
	//
	asset01Id := asset01Code
	asset02Id := issAsset02Tx.Hash()
	asset03Id := issAsset03Tx.Hash()

	relpAsset01, err := newReplenishAssetTx(testPrivate, receiver, asset01Code, asset01Id, big.NewInt(90))
	assert.NoError(t, err)
	relpAsset02, err := newReplenishAssetTx(testPrivate, receiver, asset02Code, asset02Id, big.NewInt(90))
	assert.NoError(t, err)
	relpAsset03, err := newReplenishAssetTx(testPrivate, receiver, asset03Code, asset03Id, big.NewInt(90))
	assert.NoError(t, err)

	Rtxs := types.Transactions{relpAsset01, relpAsset02, relpAsset03}
	block03, invalidTxs := newNextBlock(p, block02, Rtxs, true)
	if len(invalidTxs) != 0 {
		panic("has invalid txs 12")
	}
	assert.Equal(t, block02.Height()+1, block03.Height())
	// 	验证增发后的totalSupply和接收者的equity
	issuerAcc := p.am.GetAccount(testAddr)
	total01, err := issuerAcc.GetAssetCodeTotalSupply(asset01Code)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(200), total01)
	total02, err := issuerAcc.GetAssetCodeTotalSupply(asset02Code)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(1), total02)
	total03, err := issuerAcc.GetAssetCodeTotalSupply(asset03Code)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(110), total03)
	receiverAcc := p.am.GetAccount(receiver)
	equity01, err := receiverAcc.GetEquityState(asset01Id)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(200), equity01.Equity)
	equity02, err := receiverAcc.GetEquityState(asset02Id)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(110), equity02.Equity)
	equity03, err := receiverAcc.GetEquityState(asset03Id)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(110), equity03.Equity)

}

// new 增发资产交易
func newReplenishAssetTx(private *ecdsa.PrivateKey, receiver common.Address, assetCode, assetId common.Hash, amount *big.Int) (*types.Transaction, error) {
	repl := &types.ReplenishAsset{
		AssetCode: assetCode,
		AssetId:   assetId,
		Amount:    amount,
	}
	data, err := json.Marshal(repl)
	if err != nil {
		return nil, err
	}
	tx := types.NewTransaction(receiver, nil, uint64(500000), big.NewInt(1), data, params.ReplenishAssetTx, chainID, uint64(time.Now().Unix()+30*60), "", "replenish asset tx")
	signTx, err := types.MakeSigner().SignTx(tx, private)
	return signTx, err
}

// TestModifyAssetProfile modify asset profile map
func TestModifyAssetProfile(t *testing.T) {
	ClearData()
	bc := newChain()
	defer bc.db.Close()
	p := NewTxProcessor(bc)
	parentBlock := p.chain.stableBlock.Load().(*types.Block)
	// 执行创建资产的交易的区块
	tx01, err := newCreateAssetTx(testPrivate, types.Asset01, true, true) // 可增发
	assert.NoError(t, err)
	tx02, err := newCreateAssetTx(testPrivate, types.Asset02, false, false) // 不可增发
	assert.NoError(t, err)
	tx03, err := newCreateAssetTx(testPrivate, types.Asset03, false, true) // 不可增发
	Ctxs := types.Transactions{tx01, tx02, tx03}
	block01, _ := newNextBlock(p, parentBlock, Ctxs, true)
	asset01Code := tx01.Hash()
	asset02Code := tx02.Hash()
	asset03Code := tx03.Hash()

	// 执行修改资产信息的交易区块
	modifyTx01, err := newModifyAssetTx(testPrivate, asset01Code)
	assert.NoError(t, err)
	modifyTx02, err := newModifyAssetTx(testPrivate, asset02Code)
	assert.NoError(t, err)
	modifyTx03, err := newModifyAssetTx(testPrivate, asset03Code)
	assert.NoError(t, err)
	mTxs := types.Transactions{modifyTx01, modifyTx02, modifyTx03}
	block02, _ := newNextBlock(p, block01, mTxs, true)
	assert.Equal(t, block01.Height()+1, block02.Height())

	// 验证
	issuerAcc := p.am.GetAccount(testAddr)
	name01, err := issuerAcc.GetAssetCodeState(asset01Code, "name")
	assert.NoError(t, err)
	lemo01, err := issuerAcc.GetAssetCodeState(asset01Code, "lemo")
	assert.NoError(t, err)
	zyj01, err := issuerAcc.GetAssetCodeState(asset01Code, "aaa")
	assert.NoError(t, err)
	assert.Equal(t, "modify", name01)
	assert.Equal(t, "lemo1", lemo01)
	assert.Equal(t, "aaa1", zyj01)
	//
	name02, err := issuerAcc.GetAssetCodeState(asset02Code, "name")
	assert.NoError(t, err)
	lemo02, err := issuerAcc.GetAssetCodeState(asset02Code, "lemo")
	assert.NoError(t, err)
	zyj02, err := issuerAcc.GetAssetCodeState(asset02Code, "aaa")
	assert.NoError(t, err)
	assert.Equal(t, "modify", name02)
	assert.Equal(t, "lemo1", lemo02)
	assert.Equal(t, "aaa1", zyj02)
	//
	name03, err := issuerAcc.GetAssetCodeState(asset03Code, "name")
	assert.NoError(t, err)
	lemo03, err := issuerAcc.GetAssetCodeState(asset03Code, "lemo")
	assert.NoError(t, err)
	zyj03, err := issuerAcc.GetAssetCodeState(asset03Code, "aaa")
	assert.NoError(t, err)
	assert.Equal(t, "modify", name03)
	assert.Equal(t, "lemo1", lemo03)
	assert.Equal(t, "aaa1", zyj03)

}

// new 修改资产交易
func newModifyAssetTx(private *ecdsa.PrivateKey, assetCode common.Hash) (*types.Transaction, error) {
	info := make(types.Profile)
	info["name"] = "modify"
	info["lemo"] = "lemo1"
	info["aaa"] = "aaa1"
	modify := &types.ModifyAssetInfo{
		AssetCode: assetCode,
		Info:      info,
	}
	data, err := json.Marshal(modify)
	if err != nil {
		return nil, err
	}
	tx := types.NoReceiverTransaction(nil, uint64(500000), big.NewInt(1), data, params.ModifyAssetTx, chainID, uint64(time.Now().Unix()+30*60), "", "modify asset tx")
	return types.MakeSigner().SignTx(tx, private)
}

// TestTransferAssetTx
func TestTransferAssetTx(t *testing.T) {
	private01, _ := crypto.HexToECDSA("08f4896eea38dd271b50baf7f3a711cef4ada76066dbe71b13601e5e2dc8e27f")
	private02, _ := crypto.HexToECDSA("1a7c5f98cf5519e638eae69932d0260570b7c9913b8abe5550f177fbf29c11c9")
	private03, _ := crypto.HexToECDSA("702aff687d34228aa696d32cf702844c4cbe619411250e864ea45826d8df6751")
	addr01 := crypto.PubkeyToAddress(private01.PublicKey)
	addr02 := crypto.PubkeyToAddress(private02.PublicKey)
	addr03 := crypto.PubkeyToAddress(private03.PublicKey)

	ClearData()
	bc := newChain()
	defer bc.db.Close()
	p := NewTxProcessor(bc)
	parentBlock := p.chain.stableBlock.Load().(*types.Block)
	// 执行创建资产的交易的区块
	tx01, err := newCreateAssetTx(testPrivate, types.Asset01, true, true) // 可增发
	assert.NoError(t, err)
	tx02, err := newCreateAssetTx(testPrivate, types.Asset02, false, false) // 不可增发
	assert.NoError(t, err)
	tx03, err := newCreateAssetTx(testPrivate, types.Asset03, false, true) // 不可增发
	// get balance tx
	tx04 := makeTx(testPrivate, addr01, params.OrdinaryTx, big.NewInt(500000))
	tx05 := makeTx(testPrivate, addr02, params.OrdinaryTx, big.NewInt(500000))
	tx06 := makeTx(testPrivate, addr03, params.OrdinaryTx, big.NewInt(500000))

	cTxs := types.Transactions{tx01, tx02, tx03, tx04, tx05, tx06}
	block01, _ := newNextBlock(p, parentBlock, cTxs, true)
	asset01Code := tx01.Hash()
	asset02Code := tx02.Hash()
	asset03Code := tx03.Hash()

	// 执行发行资产的交易的区块
	issAsset01Tx, err := newIssueAssetTx(testPrivate, addr01, asset01Code, big.NewInt(110), "issue erc20 asset")
	assert.NoError(t, err)
	issAsset02Tx, err := newIssueAssetTx(testPrivate, addr02, asset02Code, big.NewInt(110), "issue erc721 asset")
	assert.NoError(t, err)
	issAsset03Tx, err := newIssueAssetTx(testPrivate, addr03, asset03Code, big.NewInt(110), "issue erc721+20 asset")
	assert.NoError(t, err)
	Itxs := types.Transactions{issAsset01Tx, issAsset02Tx, issAsset03Tx}
	block02, _ := newNextBlock(p, block01, Itxs, true)
	acc01 := p.am.GetAccount(addr01)
	acc02 := p.am.GetAccount(addr02)
	acc03 := p.am.GetAccount(addr03)
	assetId01 := asset01Code
	assetId02 := issAsset02Tx.Hash()
	assetId03 := issAsset03Tx.Hash()
	equity01, err := acc01.GetEquityState(assetId01)
	assert.NoError(t, err)
	equity02, err := acc02.GetEquityState(assetId02)
	assert.NoError(t, err)
	equity03, err := acc03.GetEquityState(assetId03)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(110), equity01.Equity, equity02.Equity, equity03.Equity)
	// 执行交易资产的交易的区块
	trading01Tx, err := newTransferAssetTx(private01, addr02, assetId01, big.NewInt(100), nil)
	assert.NoError(t, err)
	trading02Tx, err := newTransferAssetTx(private02, addr03, assetId02, big.NewInt(10000000), nil)
	assert.NoError(t, err)
	trading03Tx, err := newTransferAssetTx(private03, addr01, assetId03, big.NewInt(100), nil)
	assert.NoError(t, err)
	tTxs := types.Transactions{trading01Tx, trading02Tx, trading03Tx}
	block03, _ := newNextBlock(p, block02, tTxs, true)
	assert.Equal(t, block02.Height()+1, block03.Height())
	// 验证
	newAcc1 := p.am.GetAccount(addr01)
	newAcc2 := p.am.GetAccount(addr02)
	newAcc3 := p.am.GetAccount(addr03)
	newEquity11, err := newAcc1.GetEquityState(assetId01)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(10), newEquity11.Equity)
	newEquity13, err := newAcc1.GetEquityState(assetId03)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(100), newEquity13.Equity)

	newEquity22, err := newAcc2.GetEquityState(assetId02)
	assert.Empty(t, newEquity22)
	assert.Equal(t, store.ErrNotExist, err)
	newEquity21, err := newAcc2.GetEquityState(assetId01)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(100), newEquity21.Equity)
	newEquity33, err := newAcc3.GetEquityState(assetId03)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(10), newEquity33.Equity)
	newEquity32, err := newAcc3.GetEquityState(assetId02)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(110), newEquity32.Equity)
}

// new 交易资产
func newTransferAssetTx(private *ecdsa.PrivateKey, to common.Address, assetId common.Hash, amount *big.Int, input []byte) (*types.Transaction, error) {
	trading := &types.TradingAsset{
		AssetId: assetId,
		Value:   amount,
		Input:   input,
	}
	data, err := json.Marshal(trading)
	if err != nil {
		return nil, err
	}
	tx := types.NewTransaction(to, amount, uint64(500000), big.NewInt(1), data, params.TransferAssetTx, chainID, uint64(time.Now().Unix()+30*60), "", "trading asset tx")
	return types.MakeSigner().SignTx(tx, private)
}

// test asset max marshal data length
func TestMaxAssetProfile(t *testing.T) {
	profile := make(types.Profile)
	profile["aaaaaaaaaaaaaaaaaaaa"] = "www.lemochain.com"
	profile["bbbbbbbbbbbbbbbbbbbb"] = "www.lemochain.com"
	profile["cccccccccccccccccccc"] = "www.lemochain.com"
	profile["dddddddddddddddddddd"] = "www.lemochain.com"
	profile["eeeeeeeeeeeeeeeeeeee"] = "www.lemochain.com"
	profile["ffffffffffffffffffff"] = "www.lemochain.com"
	profile["gggggggggggggggggggg"] = "www.lemochain.com"
	profile["hhhhhhhhhhhhhhhhhhhh"] = "www.lemochain.com"
	profile["iiiiiiiiiiiiiiiiiiii"] = "www.lemochain.com"
	profile["jjjjjjjjjjjjjjjjjjjj"] = "www.lemochain.com"
	asset := &types.Asset{
		Category:        1,
		IsDivisible:     false,
		AssetCode:       common.StringToHash("702aff687d34228aa696d32cf702844c4cbe619411250e864ea45826d8df6751"),
		Decimal:         18,
		TotalSupply:     big.NewInt(111111111111111111),
		IsReplenishable: false,
		Issuer:          common.HexToAddress("0x702aff687d34228aa69619411250e864ea45826d8df6751"),
		Profile:         profile,
	}
	data, err := json.Marshal(asset)
	assert.NoError(t, err)
	t.Logf("data length : %d", len(data))

	gasUsed, err := IntrinsicGas(data, false)
	assert.NoError(t, err)
	t.Logf("max gasUsed : %d", gasUsed)
}

// TestPrecomplieContract and send rewards for deputyNode
func TestPrecomplieContract(t *testing.T) {
	params.TermDuration = 4    // 换届间隔
	params.InterimDuration = 1 // 过渡期
	ClearData()
	bc := newChain()
	defer bc.db.Close()
	p := NewTxProcessor(bc)

	data := setRewardTxData(0, new(big.Int).Div(params.TermRewardPoolTotal, common.Big2))
	private, err := crypto.HexToECDSA("c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa")
	assert.NoError(t, err)
	TxV01 := types.NewReimbursementTransaction(params.TermRewardContract, testAddr, nil, data, params.OrdinaryTx, chainID, uint64(time.Now().Unix()+300), "", "")
	firstSignTxV, err := types.MakeReimbursementTxSigner().SignTx(TxV01, private)
	assert.NoError(t, err)
	firstSignTxV = types.GasPayerSignatureTx(firstSignTxV, common.Big1, uint64(60000))
	lastSignTxV, err := types.MakeGasPayerSigner().SignTx(firstSignTxV, testPrivate)
	assert.NoError(t, err)
	txs := types.Transactions{lastSignTxV}

	Block02, _ := newNextBlock(p, p.chain.stableBlock.Load().(*types.Block), txs, true)
	assert.NotEmpty(t, Block02)
	Acc := p.am.GetAccount(params.TermRewardContract)
	key := params.TermRewardContract.Hash()
	v, err := Acc.GetStorageState(key)
	assert.NoError(t, err)
	rewardMap := make(params.RewardsMap)
	err = json.Unmarshal(v, &rewardMap)
	assert.NoError(t, err)
	assert.Equal(t, uint32(0), rewardMap[0].Term)
	assert.Equal(t, new(big.Int).Div(params.TermRewardPoolTotal, common.Big2), rewardMap[0].Value)
	assert.Equal(t, uint32(1), rewardMap[0].Times)
	// genesisBlock := p.chain.GetBlockByHeight(0)
	Block03, _ := newNextBlock(p, Block02, nil, true)
	assert.NotEmpty(t, Block03)
	Block04, _ := newNextBlock(p, Block03, nil, true)
	assert.NotEmpty(t, Block04)
	Block05, _ := newNextBlock(p, Block04, nil, true)
	assert.NotEmpty(t, Block05)
	Block06, _ := newNextBlock(p, Block05, nil, true)
	assert.NotEmpty(t, Block06)
	// t.Log(p.am.GetAccount(DefaultDeputyNodes[1].MinerAddress).GetBalance())
	balance01, _ := new(big.Int).SetString("120000000000000000000000000", 10)
	assert.Equal(t, balance01, p.am.GetAccount(DefaultDeputyNodes[1].MinerAddress).GetBalance())
	// t.Log(p.am.GetAccount(DefaultDeputyNodes[2].MinerAddress).GetBalance())
	balance02, _ := new(big.Int).SetString("90000000000000000000000000", 10)
	assert.Equal(t, balance02, p.am.GetAccount(DefaultDeputyNodes[2].MinerAddress).GetBalance())
	// t.Log(p.am.GetAccount(DefaultDeputyNodes[3].MinerAddress).GetBalance())
	balance03, _ := new(big.Int).SetString("60000000000000000000000000", 10)
	assert.Equal(t, balance03, p.am.GetAccount(DefaultDeputyNodes[3].MinerAddress).GetBalance())
	// t.Log(p.am.GetAccount(DefaultDeputyNodes[4].MinerAddress).GetBalance())
	balance04, _ := new(big.Int).SetString("30000000000000000000000000", 10)
	assert.Equal(t, balance04, p.am.GetAccount(DefaultDeputyNodes[4].MinerAddress).GetBalance())

	data02 := setRewardTxData(1, new(big.Int).Div(params.TermRewardPoolTotal, common.Big2))
	TxV02 := types.NewReimbursementTransaction(params.TermRewardContract, testAddr, nil, data02, params.OrdinaryTx, chainID, uint64(time.Now().Unix()+300), "", "")
	firstSignTxV02, err := types.MakeReimbursementTxSigner().SignTx(TxV02, private)
	assert.NoError(t, err)
	firstSignTxV02 = types.GasPayerSignatureTx(firstSignTxV02, common.Big1, uint64(60000))
	lastSignTxV02, err := types.MakeGasPayerSigner().SignTx(firstSignTxV02, testPrivate)
	assert.NoError(t, err)
	txs02 := types.Transactions{lastSignTxV02}
	Block07, _ := newNextBlock(p, Block06, txs02, true)
	Block08, _ := newNextBlock(p, Block07, nil, true)
	// set next deputyNodeList
	bc.DeputyManager().SaveSnapshot(9, DefaultDeputyNodes)

	Block09, _ := newNextBlock(p, Block08, nil, true)
	assert.NotEmpty(t, Block09)
	// t.Log(p.am.GetAccount(DefaultDeputyNodes[1].MinerAddress).GetBalance())
	balance01, _ = new(big.Int).SetString("240000000000000000000000000", 10)
	assert.Equal(t, balance01, p.am.GetAccount(DefaultDeputyNodes[1].MinerAddress).GetBalance())
	// t.Log(p.am.GetAccount(DefaultDeputyNodes[2].MinerAddress).GetBalance())
	balance02, _ = new(big.Int).SetString("180000000000000000000000000", 10)
	assert.Equal(t, balance02, p.am.GetAccount(DefaultDeputyNodes[2].MinerAddress).GetBalance())
	// t.Log(p.am.GetAccount(DefaultDeputyNodes[3].MinerAddress).GetBalance())
	balance03, _ = new(big.Int).SetString("120000000000000000000000000", 10)
	assert.Equal(t, balance03, p.am.GetAccount(DefaultDeputyNodes[3].MinerAddress).GetBalance())
	// t.Log(p.am.GetAccount(DefaultDeputyNodes[4].MinerAddress).GetBalance())
	balance04, _ = new(big.Int).SetString("60000000000000000000000000", 10)
	assert.Equal(t, balance04, p.am.GetAccount(DefaultDeputyNodes[4].MinerAddress).GetBalance())

}

//
func setRewardTxData(term uint32, value *big.Int) []byte {
	re := params.RewardJson{
		Term:  term,
		Value: value,
	}
	by, err := json.Marshal(re)
	if err != nil {
		log.Warn(err.Error())
		return nil
	}
	return by
}

// TestBlockChain_txData 生成调用设置换届奖励的预编译合约交易的data
func TestBlockChain_txData(t *testing.T) {
	value, _ := new(big.Int).SetString("99999999999999999999", 10)
	re := params.RewardJson{
		Term:  0,
		Value: value,
	}
	by, _ := json.Marshal(re)
	fmt.Println("tx data", common.ToHex(by))
	fmt.Println("预编译合约地址", common.BytesToAddress([]byte{9}).String())
}

func TestIntrinsicGas(t *testing.T) {
	value, _ := new(big.Int).SetString("500000000000000000000000", 10)
	for i := 11; i < 26; i++ {
		re := params.RewardJson{
			Term:  uint32(i),
			Value: value,
		}
		by, _ := json.Marshal(re)
		fmt.Println(i, "term: ", common.ToHex(by))
	}
}

func Test_rlpBlock(t *testing.T) {
	ClearData()
	bc := newChain()
	defer bc.db.Close()
	p := NewTxProcessor(bc)

	block, _ := newNextBlock(p, p.chain.stableBlock.Load().(*types.Block), types.Transactions{}, true)
	t.Log("txRoot:", block.Header.TxRoot.String())
	t.Log("logRoot:", block.Header.LogRoot.String())

	buf, err := rlp.EncodeToBytes(block)
	assert.NoError(t, err)
	t.Log("rlp length:", len(buf))
	var decBlock types.Block

	err = rlp.DecodeBytes(buf, &decBlock)
	assert.NoError(t, err)
	assert.Equal(t, block.Header.Hash(), decBlock.Header.Hash())
}

func BenchmarkApplyTxs(b *testing.B) {
	ClearData()
	bc := newChain()
	defer bc.db.Close()
	p := NewTxProcessor(bc)

	// prepare account and balance
	blockHash, accountKeys := createAccounts(b.N, bc.db)
	bc.am.Reset(blockHash)
	header := &types.Header{
		ParentHash:   blockHash,
		MinerAddress: defaultAccounts[0],
		Height:       4,
		GasLimit:     2100000000,
		Time:         1538209762,
	}
	// make txs
	txs := make(types.Transactions, b.N, b.N)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < b.N; i++ {
		fromKey := accountKeys[r.Intn(b.N)]
		fromPrivate, _ := crypto.ToECDSA(common.FromHex(fromKey.Private))
		to := accountKeys[r.Intn(b.N)].Address
		fromBalance := p.am.GetAccount(fromKey.Address).GetBalance() // maybe 0
		amount := new(big.Int).Rand(r, fromBalance)                  // maybe too many if we make transaction more than twice from same address
		txs[i] = makeTx(fromPrivate, to, params.OrdinaryTx, amount)
	}

	start := time.Now().UnixNano()
	b.ResetTimer()
	selectedTxs, invalidTxs, _ := p.ApplyTxs(header, txs, int64(10000))
	fmt.Printf("BenchmarkApplyTxs cost %dms\n", (time.Now().UnixNano()-start)/1000000)
	fmt.Printf("%d transactions success, %d transactions fail\n", len(selectedTxs), len(invalidTxs))
}

func BenchmarkMakeBlock(b *testing.B) {
	ClearData()
	bc := newChain()
	defer bc.db.Close()
	p := NewTxProcessor(bc)
	balanceRecord := make(map[common.Address]*big.Int)

	// prepare account and balance
	blockHash, accountKeys := createAccounts(b.N, bc.db)
	bc.am.Reset(blockHash)
	bc.db.SetStableBlock(blockHash)
	// make txs
	txs := make(types.Transactions, b.N, b.N)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < b.N; i++ {
		fromKey := accountKeys[r.Intn(b.N)]
		fromPrivate, _ := crypto.ToECDSA(common.FromHex(fromKey.Private))
		to := accountKeys[r.Intn(b.N)].Address
		fromBalance, ok := balanceRecord[fromKey.Address]
		if !ok {
			fromBalance = p.am.GetAccount(fromKey.Address).GetBalance() // maybe 0
		}
		amount := new(big.Int).Rand(r, fromBalance)
		balanceRecord[fromKey.Address] = new(big.Int).Sub(fromBalance, amount)
		txs[i] = makeTx(fromPrivate, to, params.OrdinaryTx, amount)
	}

	start := time.Now().UnixNano()
	b.ResetTimer()
	newBlock := makeBlock(bc.db, blockInfo{
		height:     4,
		parentHash: blockHash,
		author:     defaultAccounts[0],
		time:       1538209762,
		txList:     txs,
		gasLimit:   2100000000,
	}, true)
	fmt.Printf("BenchmarkMakeBlock cost %dms\n", (time.Now().UnixNano()-start)/1000000)
	fmt.Printf("%d transactions success, %d transactions fail\n", len(newBlock.Txs), b.N-len(newBlock.Txs))

	startSave := time.Now().UnixNano()
	bc.db.SetStableBlock(newBlock.Hash())
	fmt.Printf("Saving stable to disk cost %dms\n", (time.Now().UnixNano()-startSave)/1000000)
	time.Sleep(3 * time.Second)
}

func BenchmarkSetBalance(b *testing.B) {
	fromAddr := testAddr
	fromBalance := new(big.Int)
	toBalance := new(big.Int)
	salary := new(big.Int)
	amount, _ := new(big.Int).SetString("1234857462837462918237", 10)
	tx := makeTx(testPrivate, common.HexToAddress("0x123"), params.OrdinaryTx, amount)
	for i := 0; i < b.N; i++ {
		gas := params.TxGas + params.TxDataNonZeroGas*uint64(len("abc"))
		// fromAddr, err := tx.From()
		// if err != nil {
		// 	panic(err)
		// }
		// from := manager.GetAccount(fromAddr)
		fee := new(big.Int).Mul(new(big.Int).SetUint64(gas), tx.GasPrice())
		cost := new(big.Int).Add(tx.Amount(), fee)
		// to := manager.GetAccount(*tx.To())
		// make sure the change log has right order
		if fromAddr.Hex() < tx.To().Hex() {
			fromBalance.Set(new(big.Int).Sub(fromBalance, cost))
			toBalance.Set(new(big.Int).Add(toBalance, tx.Amount()))
		} else {
			toBalance.Set(new(big.Int).Add(toBalance, tx.Amount()))
			fromBalance.Set(new(big.Int).Sub(fromBalance, cost))
		}
		salary.Add(salary, fee)
	}
}
