package chain

import (
	"encoding/json"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/flag"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"strconv"
	"testing"
	"time"
)

func TestNewTxProcessor(t *testing.T) {
	chain := newChain()
	p := NewTxProcessor(chain)
	assert.NotEqual(t, (*vm.Config)(nil), p.cfg)
	assert.Equal(t, false, p.cfg.Debug)

	flags := flag.CmdFlags{}
	flags.Set(common.Debug, "1")
	chain, _ = NewBlockChain(chainID, NewDpovp(10*1000, chain.db), chain.db, flags)
	p = NewTxProcessor(chain)
}

// test valid block processing
func TestTxProcessor_Process(t *testing.T) {
	store.ClearData()
	p := NewTxProcessor(newChain())

	sender := p.am.GetAccount(testAddr)
	// last not stable block
	block := defaultBlocks[2]
	newHeader, err := p.Process(block)
	assert.NoError(t, err)
	assert.Equal(t, block.Header.Bloom, newHeader.Bloom)
	assert.Equal(t, block.Header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, block.Header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, block.Header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, block.Header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, block.Header.LogRoot, newHeader.LogRoot)
	assert.Equal(t, block.Hash(), newHeader.Hash())
	sender = p.am.GetAccount(testAddr)
	assert.Equal(t, 3, len(sender.GetTxHashList()))
	assert.Equal(t, block.Txs[0].Hash(), sender.GetTxHashList()[2])

	// block not in db
	block = defaultBlocks[3]
	newHeader, err = p.Process(block)
	assert.NoError(t, err)
	assert.Equal(t, block.Header.Bloom, newHeader.Bloom)
	assert.Equal(t, block.Header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, block.Header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, block.Header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, block.Header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, block.Header.LogRoot, newHeader.LogRoot)
	assert.Equal(t, block.Hash(), newHeader.Hash())
	sender = p.am.GetAccount(testAddr)
	assert.Equal(t, 5, len(sender.GetTxHashList()))
	assert.Equal(t, block.Txs[0].Hash(), sender.GetTxHashList()[3])

	// genesis block
	block = defaultBlocks[0]
	newHeader, err = p.Process(block)
	assert.NoError(t, err)
	assert.Equal(t, block.Header.Bloom, newHeader.Bloom)
	assert.Equal(t, block.Header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, block.Header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, block.Header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, block.Header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, block.Header.LogRoot, newHeader.LogRoot)
	assert.Equal(t, block.Hash(), newHeader.Hash())

	// block on fork branch
	block = createNewBlock()
	newHeader, err = p.Process(block)
	assert.NoError(t, err)
	assert.Equal(t, block.Header.Bloom, newHeader.Bloom)
	assert.Equal(t, block.Header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, block.Header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, block.Header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, block.Header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, block.Header.LogRoot, newHeader.LogRoot)
	assert.Equal(t, block.Hash(), newHeader.Hash())
}

// test invalid block processing
func TestTxProcessor_Process2(t *testing.T) {
	store.ClearData()
	p := NewTxProcessor(newChain())

	// tamper with amount
	block := createNewBlock()
	rawTx, _ := rlp.EncodeToBytes(block.Txs[0])
	rawTx[29]++ // amount++
	cpy := new(types.Transaction)
	err := rlp.DecodeBytes(rawTx, cpy)
	assert.NoError(t, err)
	assert.Equal(t, new(big.Int).Add(block.Txs[0].Amount(), big.NewInt(1)), cpy.Amount())
	block.Txs[0] = cpy
	_, err = p.Process(block)
	assert.Equal(t, ErrInvalidTxInBlock, err)

	// invalid signature
	block = createNewBlock()
	rawTx, _ = rlp.EncodeToBytes(block.Txs[0])
	rawTx[43] = 0 // invalid S
	cpy = new(types.Transaction)
	err = rlp.DecodeBytes(rawTx, cpy)
	assert.NoError(t, err)
	block.Txs[0] = cpy
	_, err = p.Process(block)
	assert.Equal(t, ErrInvalidTxInBlock, err)

	// not enough gas (resign by another address)
	block = createNewBlock()
	private, _ := crypto.GenerateKey()
	origFrom, _ := block.Txs[0].From()
	block.Txs[0] = signTransaction(block.Txs[0], private)
	newFrom, _ := block.Txs[0].From()
	assert.NotEqual(t, origFrom, newFrom)
	block.Header.TxRoot = types.DeriveTxsSha(block.Txs)
	_, err = p.Process(block)
	assert.Equal(t, ErrInvalidTxInBlock, err)

	// exceed block gas limit
	block = createNewBlock()
	block.Header.GasLimit = 1
	_, err = p.Process(block)
	assert.Equal(t, ErrInvalidTxInBlock, err)

	// used gas reach limit in some tx
	block = createNewBlock()
	block.Txs[0] = makeTransaction(testPrivate, defaultAccounts[1], params.OrdinaryTx, big.NewInt(100), common.Big1, 0, 1)
	block.Header.TxRoot = types.DeriveTxsSha(block.Txs)
	_, err = p.Process(block)
	assert.Equal(t, ErrInvalidTxInBlock, err)

	// balance not enough
	block = createNewBlock()
	balance := p.am.GetAccount(testAddr).GetBalance()
	block.Txs[0] = makeTx(testPrivate, defaultAccounts[1], params.OrdinaryTx, new(big.Int).Add(balance, big.NewInt(1)))
	block.Header.TxRoot = types.DeriveTxsSha(block.Txs)
	_, err = p.Process(block)
	assert.Equal(t, ErrInvalidTxInBlock, err)

	// TODO test create or call contract fail
}

func createNewBlock() *types.Block {
	db := store.NewChainDataBase(store.GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
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
	store.ClearData()
	p := NewTxProcessor(newChain())

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
	newHeader, selectedTxs, invalidTxs, err := p.ApplyTxs(emptyHeader, txs)
	assert.NoError(t, err)
	assert.Equal(t, header.Bloom, newHeader.Bloom)
	assert.Equal(t, header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, header.LogRoot, newHeader.LogRoot)
	assert.Empty(t, newHeader.DeputyRoot)
	assert.Equal(t, header.Hash(), newHeader.Hash())
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
	newHeader, selectedTxs, invalidTxs, err = p.ApplyTxs(emptyHeader, txs)
	assert.NoError(t, err)
	assert.Equal(t, header.Bloom, newHeader.Bloom)
	assert.Equal(t, header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, header.LogRoot, newHeader.LogRoot)
	assert.Equal(t, header.Hash(), newHeader.Hash())
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
	newHeader, selectedTxs, invalidTxs, err = p.ApplyTxs(emptyHeader, nil)
	assert.NoError(t, err)
	assert.Equal(t, types.Bloom{}, newHeader.Bloom)
	emptyTrieHash := common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")
	assert.Equal(t, emptyTrieHash, newHeader.EventRoot)
	assert.Equal(t, emptyTrieHash, newHeader.TxRoot)
	assert.Equal(t, emptyTrieHash, newHeader.LogRoot)
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
	newHeader, selectedTxs, invalidTxs, err = p.ApplyTxs(emptyHeader, txs)
	assert.NoError(t, err)
	assert.Equal(t, header.Bloom, newHeader.Bloom)
	assert.Equal(t, header.EventRoot, newHeader.EventRoot)
	assert.NotEqual(t, header.GasUsed, newHeader.GasUsed)
	assert.NotEqual(t, header.TxRoot, newHeader.TxRoot)
	assert.NotEqual(t, header.VersionRoot, newHeader.VersionRoot)
	assert.NotEqual(t, header.LogRoot, newHeader.LogRoot)
	assert.NotEqual(t, header.Hash(), newHeader.Hash())
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
	newHeader, selectedTxs, invalidTxs, err = p.ApplyTxs(emptyHeader, txs)
	assert.NoError(t, err)
	assert.Equal(t, header.Bloom, newHeader.Bloom)
	assert.Equal(t, header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, header.LogRoot, newHeader.LogRoot)
	assert.Equal(t, header.Hash(), newHeader.Hash())
	assert.Equal(t, len(txs)-1, len(selectedTxs))
	assert.Equal(t, 1, len(invalidTxs))
}

// TODO move these cases to evm
// test different transactions
func TestTxProcessor_ApplyTxs2(t *testing.T) {
	store.ClearData()
	p := NewTxProcessor(newChain())

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
	newHeader, _, _, err := p.ApplyTxs(emptyHeader, txs)
	assert.NoError(t, err)
	assert.Equal(t, params.TxGas, newHeader.GasUsed)
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
	newHeader, _, _, err = p.ApplyTxs(emptyHeader, txs)
	assert.NoError(t, err)
	assert.Equal(t, params.TxGas, newHeader.GasUsed)
	newSenderBalance = p.am.GetAccount(testAddr).GetBalance()
	cost = txs[0].GasPrice().Mul(txs[0].GasPrice(), big.NewInt(int64(params.TxGas)))
	assert.Equal(t, senderBalance.Sub(senderBalance, cost), newSenderBalance)
}

//  Test_voteAndRegisteTx测试投票交易和注册候选节点交易
func Test_voteAndRegisteTx(t *testing.T) {
	store.ClearData()
	p := NewTxProcessor(newChain())

	// 接收注册候选节点1000LEMO的地址
	strAddress := "0x1001"
	to, _ := common.StringToAddress(strAddress)

	// 申请第一个候选节点(testAddr)信息data
	var cand00 = &deputynode.CandidateNode{
		IsCandidate:  true,
		MinerAddress: common.HexToAddress("0x10000"),
		NodeID:       common.FromHex("0x34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"),
		Host:         "0.0.0.0",
		Port:         0000,
	}
	candData00, _ := json.Marshal(cand00)
	// 申请第二个候选节点(testAddr02)信息data
	var cand02 = &deputynode.CandidateNode{
		IsCandidate:  true,
		MinerAddress: common.HexToAddress("0x222222"),
		NodeID:       common.FromHex("0x7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2d0347692f36ef736f342fb5ce4710f7e337f062cc2110d134b63a9575f78cb167bfae2f43"),
		Host:         "2.2.2.2",
		Port:         2222,
	}
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
	registerTx01 := signTransaction(types.NewTransaction(to, params.RegisterCandidateNodeFees, 200000, common.Big1, candData00, params.RegisterTx, chainID, uint64(time.Now().Unix()+300), "", ""), testPrivate)

	parentBlock := p.chain.currentBlock.Load().(*types.Block)
	header01 := &types.Header{
		ParentHash:   parentBlock.Hash(),
		MinerAddress: parentBlock.MinerAddress(),
		VersionRoot:  parentBlock.VersionRoot(),
		Height:       parentBlock.Height() + 1,
		GasLimit:     parentBlock.GasLimit(),
		GasUsed:      0,
		Time:         parentBlock.Time() + 4,
	}
	tx01 := types.Transactions{registerTx01, getBalanceTx01, getBalanceTx02, getBalanceTx03}
	newHeader01, _, _, err := p.ApplyTxs(header01, tx01)
	if err != nil {
		fmt.Printf(" apply register tx err : %s \n", err)
	}
	Block01 := &types.Block{
		Txs:         tx01,
		ChangeLogs:  p.am.GetChangeLogs(),
		Events:      p.am.GetEvents(),
		Confirms:    nil,
		DeputyNodes: nil,
	}
	Block01.SetHeader(newHeader01)
	blockHash := newHeader01.Hash()
	err = p.chain.db.SetBlock(blockHash, Block01)
	if err != nil && err != store.ErrExist {
		panic(err)
	}
	err = p.am.Save(blockHash)
	if err != nil {
		panic(err)
	}
	err = p.chain.db.SetStableBlock(blockHash)
	if err != nil {
		panic(err)
	}

	// 	验证注册代理节点交易信息
	testAddr, _ := registerTx01.From()
	account00 := p.am.GetCanonicalAccount(testAddr)
	assert.Equal(t, testAddr, account00.GetVoteFor())                               // 投给自己
	assert.Equal(t, account00.GetBalance().String(), account00.GetVotes().String()) // 初始票数为自己的Balance
	profile := account00.GetCandidateProfile()
	assert.Equal(t, cand00.MinerAddress.Hex(), profile[types.CandidateKeyMinerAddress])
	assert.Equal(t, cand00.Host, profile[types.CandidateKeyHost])
	assert.Equal(t, strconv.Itoa(int(cand00.Port)), profile[types.CandidateKeyPort])
	assert.Equal(t, common.ToHex(cand00.NodeID), profile[types.CandidateKeyNodeID])

	// ---Block02-----------------------------------------------------------------------
	//  2. 测试发送投票交易,testAddr01账户为testAddr候选节点账户投票,并注册testAddr02为候选节点
	p.am.Reset(Block01.Hash())
	// 投票交易
	voteTx01 := makeTx(testPrivate01, testAddr, params.VoteTx, big.NewInt(0))
	// 注册testAddr02为候选节点的交易
	registerTx02 := signTransaction(types.NewTransaction(to, params.RegisterCandidateNodeFees, 200000, common.Big1, candData02, params.RegisterTx, chainID, uint64(time.Now().Unix()+300), "", ""), testPrivate02)

	header02 := &types.Header{
		ParentHash:   Block01.Hash(),
		MinerAddress: Block01.MinerAddress(),
		VersionRoot:  Block01.VersionRoot(),
		Height:       Block01.Height() + 1,
		GasLimit:     Block01.GasLimit(),
		Time:         Block01.Time() + 4,
	}
	txs02 := types.Transactions{voteTx01, registerTx02}
	newHeader02, _, _, err := p.ApplyTxs(header02, txs02)
	if err != nil {
		fmt.Printf(" apply vote tx err : %s \n", err)
	}
	Block02 := &types.Block{
		Header:      newHeader02,
		Txs:         txs02,
		ChangeLogs:  p.am.GetChangeLogs(),
		Events:      p.am.GetEvents(),
		Confirms:    nil,
		DeputyNodes: nil,
	}

	Hash02 := Block02.Hash()
	err = p.chain.db.SetBlock(Hash02, Block02)
	if err != nil && err != store.ErrExist {
		panic(err)
	}
	err = p.am.Save(Hash02)
	if err != nil {
		panic(err)
	}
	err = p.chain.db.SetStableBlock(Hash02)
	if err != nil {
		panic(err)
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
	profile02 := account02.GetCandidateProfile()
	assert.Equal(t, cand02.MinerAddress.Hex(), profile02[types.CandidateKeyMinerAddress])
	assert.Equal(t, cand02.Host, profile02[types.CandidateKeyHost])
	assert.Equal(t, strconv.Itoa(int(cand02.Port)), profile02[types.CandidateKeyPort])
	assert.Equal(t, common.ToHex(cand02.NodeID), profile02[types.CandidateKeyNodeID])
	// ---Block03-----------------------------------------------------------------------------
	// 3. testAddr01从候选节点testAddr 转投 给候选节点testAddr02; 候选节点testAddr修改注册信息
	p.am.Reset(Block02.Hash())
	// 	投票交易
	voteTx02 := makeTx(testPrivate01, address02, params.VoteTx, big.NewInt(0))
	// 修改候选节点profile交易
	changeCand00 := &deputynode.CandidateNode{
		IsCandidate:  true,
		MinerAddress: cand00.MinerAddress,
		NodeID:       cand00.NodeID,
		Host:         "www.changeIndo.org",
		Port:         30303,
	}
	changeCandData00, _ := json.Marshal(changeCand00)
	registerTx03 := signTransaction(types.NewTransaction(to, params.RegisterCandidateNodeFees, 200000, common.Big1, changeCandData00, params.RegisterTx, chainID, uint64(time.Now().Unix()+300), "", ""), testPrivate)
	// 生成block
	header03 := &types.Header{
		ParentHash:   Block02.Hash(),
		MinerAddress: Block02.MinerAddress(),
		VersionRoot:  Block02.VersionRoot(),
		Height:       Block02.Height() + 1,
		GasLimit:     Block02.GasLimit(),
		Time:         Block02.Time() + 4,
	}
	txs03 := types.Transactions{voteTx02, registerTx03}
	newHeader03, _, _, err := p.ApplyTxs(header03, txs03)
	if err != nil {
		fmt.Printf(" apply vote tx err : %s \n", err)
	}
	Block03 := &types.Block{
		Header:      newHeader03,
		Txs:         txs03,
		ChangeLogs:  p.am.GetChangeLogs(),
		Events:      p.am.GetEvents(),
		Confirms:    nil,
		DeputyNodes: nil,
	}

	Hash03 := Block03.Hash()
	err = p.chain.db.SetBlock(Hash03, Block03)
	if err != nil && err != store.ErrExist {
		panic(err)
	}
	err = p.am.Save(Hash03)
	if err != nil {
		panic(err)
	}
	err = p.chain.db.SetStableBlock(Hash03)
	if err != nil {
		panic(err)
	}
	// 	验证1. 候选节点testAddr票数减少量 = testAddr01的Balance，候选节点testAddr02票数增加量 = testAddr01的Balance
	latestAccount00 := p.am.GetCanonicalAccount(testAddr)
	block03testAddr00Votes := latestAccount00.GetVotes()
	subVote00 := new(big.Int).Sub(block02testAddr00Votes, block03testAddr00Votes)
	testAccount01 := p.am.GetCanonicalAccount(testAddr01)
	assert.Equal(t, subVote00, new(big.Int).Add(testAccount01.GetBalance(), big.NewInt(21000)))

	latestAccount02 := p.am.GetCanonicalAccount(testAddr02)
	block03testAddr02Votes := latestAccount02.GetVotes()
	addVotes02 := new(big.Int).Sub(block03testAddr02Votes, block02testAddr02Votes)
	assert.Equal(t, addVotes02, testAccount01.GetBalance())

	// 	验证2. 候选节点testAddr修改后的信息
	pro := latestAccount00.GetCandidateProfile()
	assert.Equal(t, strconv.Itoa(int(changeCand00.Port)), pro[types.CandidateKeyPort])
	assert.Equal(t, common.ToHex(changeCand00.NodeID), pro[types.CandidateKeyNodeID])
}
