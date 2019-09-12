package consensus

import (
	"crypto/ecdsa"
	cryptoRand "crypto/rand"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/transaction"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto/secp256k1"
	"github.com/LemoFoundationLtd/lemochain-core/common/merkle"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

const (
	block01MinerAddress = "0x015780F8456F9c1532645087a19DcF9a7e0c7F97"
	deputy01Privkey     = "0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"
)

func TestNewBlockAssembler(t *testing.T) {
	dm := deputynode.NewManager(5, &testBlockLoader{})

	ba := NewBlockAssembler(nil, dm, &transaction.TxProcessor{}, createCandidateLoader())
	assert.Equal(t, dm, ba.dm)
}

func TestBlockAssembler_PrepareHeader(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), "", "")
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	dm := initDeputyManager(5)

	ba := NewBlockAssembler(am, dm, &transaction.TxProcessor{}, createCandidateLoader())
	parentHeader := &types.Header{Height: 100, Time: 1001}

	// I'm a miner
	now := uint32(time.Now().Unix())
	newHeader, err := ba.PrepareHeader(parentHeader, []byte{12, 34})
	assert.NoError(t, err)
	assert.Equal(t, parentHeader.Hash(), newHeader.ParentHash)
	assert.Equal(t, testDeputies[0].MinerAddress, newHeader.MinerAddress)
	assert.Equal(t, parentHeader.Height+1, newHeader.Height)
	assert.Equal(t, params.GenesisGasLimit, newHeader.GasLimit)
	assert.Equal(t, []byte{12, 34}, newHeader.Extra)
	assert.Equal(t, true, newHeader.Time >= now)

	// I'm not miner
	privateBackup := deputynode.GetSelfNodeKey()
	private, _ := ecdsa.GenerateKey(secp256k1.S256(), cryptoRand.Reader)
	deputynode.SetSelfNodeKey(private)
	_, err = ba.PrepareHeader(parentHeader, nil)
	assert.Equal(t, ErrNotDeputy, err)
	deputynode.SetSelfNodeKey(privateBackup)
}

func TestCalcGasLimit(t *testing.T) {
	parentHeader := &types.Header{GasLimit: 0, GasUsed: 0}
	limit := calcGasLimit(parentHeader)
	assert.Equal(t, true, limit > params.MinGasLimit)
	assert.Equal(t, params.TargetGasLimit, limit)

	parentHeader = &types.Header{GasLimit: params.TargetGasLimit + 100000000000, GasUsed: params.TargetGasLimit + 100000000000}
	limit = calcGasLimit(parentHeader)
	assert.Equal(t, true, limit > params.MinGasLimit)
	assert.Equal(t, true, limit > params.TargetGasLimit)

	parentHeader = &types.Header{GasLimit: params.TargetGasLimit + 100000000000, GasUsed: 0}
	limit = calcGasLimit(parentHeader)
	assert.Equal(t, true, limit > params.MinGasLimit)
	assert.Equal(t, true, limit > params.TargetGasLimit)
}

// 对区块进行签名的函数
func signTestBlock(deputyPrivate string, block *types.Block) {
	// 对区块签名
	private, err := crypto.ToECDSA(common.FromHex(deputyPrivate))
	if err != nil {
		panic(err)
	}
	signBlock, err := crypto.Sign(block.Hash().Bytes(), private)
	if err != nil {
		panic(err)
	}
	block.Header.SignData = signBlock
}

func TestBlockAssembler_Seal(t *testing.T) {
	canLoader := createCandidateLoader(0, 1, 3)
	ba := NewBlockAssembler(nil, nil, nil, canLoader)

	// not snapshot block
	header := &types.Header{Height: 100}
	txs := []*types.Transaction{
		types.NewTransaction(common.HexToAddress("0x123"), common.Address{}, common.Big1, 2000000, common.Big2, []byte{12}, 0, 100, 1538210391, "aa", "aaa"),
		{},
	}
	product := &account.TxsProduct{
		Txs:         txs,
		GasUsed:     123,
		ChangeLogs:  types.ChangeLogSlice{{LogType: 101}, {}},
		VersionRoot: common.HexToHash("0x124"),
	}
	confirms := []types.SignData{{0x12}, {0x34}}
	block01 := ba.Seal(header, product, confirms)
	assert.Equal(t, product.VersionRoot, block01.VersionRoot())
	assert.Equal(t, product.ChangeLogs.MerkleRootSha(), block01.LogRoot())
	assert.Equal(t, product.Txs.MerkleRootSha(), block01.TxRoot())
	assert.Equal(t, product.GasUsed, block01.GasUsed())
	assert.Equal(t, product.Txs, block01.Txs)
	assert.Equal(t, product.ChangeLogs, block01.ChangeLogs)
	assert.Equal(t, confirms, block01.Confirms)

	// snapshot block
	header = &types.Header{Height: params.TermDuration}
	product = &account.TxsProduct{}
	confirms = []types.SignData{}
	block02 := ba.Seal(header, product, confirms)
	assert.NotEqual(t, block01.Hash(), block02.Hash())
	deputies := types.DeputyNodes(canLoader)
	assert.Equal(t, deputies, block02.DeputyNodes)
	deputyRoot := deputies.MerkleRootSha()
	assert.Equal(t, deputyRoot[:], block02.DeputyRoot())
	assert.Equal(t, merkle.EmptyTrieHash, block02.LogRoot())
	assert.Equal(t, merkle.EmptyTrieHash, block02.TxRoot())
	assert.Equal(t, uint64(0), block02.GasUsed())
	assert.Empty(t, block02.Txs)
	assert.Empty(t, block02.ChangeLogs)
	assert.Empty(t, block02.Confirms)
}

// func TestBlockAssembler_Finalize(t *testing.T) {
// 	dm := deputynode.NewManager(5, &testBlockLoader{})
//
// 	// 第0届 一个deputy node
// 	nodes := pickNodes(0)
// 	dm.SaveSnapshot(0, nodes)
// 	// 第一届
// 	nodes = pickNodes(1, 2, 3)
// 	dm.SaveSnapshot(params.TermDuration, nodes)
// 	// 第二届
// 	nodes = pickNodes(1, 3, 4, 5, 0)
// 	dm.SaveSnapshot(params.TermDuration*2, nodes)
//
// 	dpovp := loadDpovp(dm)
// 	defer dpovp.db.Close()
// 	am := account.NewManager(common.Hash{}, dpovp.db)
//
// 	// 设置前0,1,2届的矿工换届奖励
// 	rewardMap := make(params.RewardsMap)
// 	num00, _ := new(big.Int).SetString("55555555555555555555", 10)
// 	num01, _ := new(big.Int).SetString("66666666666666666666", 10)
// 	num02, _ := new(big.Int).SetString("77777777777777777777", 10)
// 	rewardMap[0] = &params.Reward{
// 		Term:  0,
// 		Value: num00,
// 		Times: 1,
// 	}
// 	rewardMap[1] = &params.Reward{
// 		Term:  1,
// 		Value: num01,
// 		Times: 1,
// 	}
// 	rewardMap[2] = &params.Reward{
// 		Term:  2,
// 		Value: num02,
// 		Times: 1,
// 	}
// 	data, err := json.Marshal(rewardMap)
// 	assert.NoError(t, err)
// 	rewardAcc := am.GetAccount(params.TermRewardContract)
// 	rewardAcc.SetStorageState(params.TermRewardContract.Hash(), data)
// 	// 设置deputy node的income address
// 	term00, err := dm.GetTermByHeight(0)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 1, len(term00.Nodes))
//
// 	term01, err := dm.GetTermByHeight(params.TermDuration + params.InterimDuration + 1)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 3, len(term01.Nodes))
//
// 	term02, err := dm.GetTermByHeight(2*params.TermDuration + params.InterimDuration + 1)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 5, len(term02.Nodes))
//
// 	// miner
// 	minerAddr00 := term02.Nodes[0].MinerAddress
// 	minerAddr01 := term02.Nodes[1].MinerAddress
// 	minerAddr02 := term02.Nodes[2].MinerAddress
// 	minerAddr03 := term02.Nodes[3].MinerAddress
// 	minerAddr04 := term02.Nodes[4].MinerAddress
// 	minerAcc00 := am.GetAccount(minerAddr00)
// 	minerAcc01 := am.GetAccount(minerAddr01)
// 	minerAcc02 := am.GetAccount(minerAddr02)
// 	minerAcc03 := am.GetAccount(minerAddr03)
// 	minerAcc04 := am.GetAccount(minerAddr04)
// 	// 设置income address
// 	incomeAddr00 := common.HexToAddress("0x10000")
// 	incomeAddr01 := common.HexToAddress("0x10001")
// 	incomeAddr02 := common.HexToAddress("0x10002")
// 	incomeAddr03 := common.HexToAddress("0x10003")
// 	incomeAddr04 := common.HexToAddress("0x10004")
// 	profile := make(map[string]string)
// 	profile[types.CandidateKeyIncomeAddress] = incomeAddr00.String()
// 	minerAcc00.SetCandidate(profile)
// 	profile[types.CandidateKeyIncomeAddress] = incomeAddr01.String()
// 	minerAcc01.SetCandidate(profile)
// 	profile[types.CandidateKeyIncomeAddress] = incomeAddr02.String()
// 	minerAcc02.SetCandidate(profile)
// 	profile[types.CandidateKeyIncomeAddress] = incomeAddr03.String()
// 	minerAcc03.SetCandidate(profile)
// 	profile[types.CandidateKeyIncomeAddress] = incomeAddr04.String()
// 	minerAcc04.SetCandidate(profile)
//
// 	// 为第0届发放奖励
// 	err = dpovp.Finalize(params.InterimDuration+params.TermDuration+1, am)
// 	assert.NoError(t, err)
// 	// 查看第0届的deputy node 收益地址的balance. 只有第一个deputy node
// 	incomeAcc00 := am.GetAccount(incomeAddr00)
// 	value1, _ := new(big.Int).SetString("55000000000000000000", 10)
// 	assert.Equal(t, value1, incomeAcc00.GetBalance())
//
// 	// 	为第二届发放奖励
// 	err = dpovp.Finalize(2*params.TermDuration+params.InterimDuration+1, am)
// 	assert.NoError(t, err)
// 	// 查看第二届的deputy node 收益地址的balance.前三个deputy node
// 	value2, _ := new(big.Int).SetString("79000000000000000000", 10)
// 	assert.Equal(t, value2, incomeAcc00.GetBalance())
//
// 	incomeAcc01 := am.GetAccount(incomeAddr01)
// 	value3, _ := new(big.Int).SetString("22000000000000000000", 10)
// 	assert.Equal(t, value3, incomeAcc01.GetBalance())
//
// 	incomeAcc02 := am.GetAccount(incomeAddr02)
// 	value4, _ := new(big.Int).SetString("20000000000000000000", 10)
// 	assert.Equal(t, value4, incomeAcc02.GetBalance())
//
// 	// 	为第三届的deputy nodes 发放奖励 5个deputy node
// 	err = dpovp.Finalize(3*params.TermDuration+params.InterimDuration+1, am)
// 	assert.NoError(t, err)
// 	//
// 	value5, _ := new(big.Int).SetString("97000000000000000000", 10)
// 	assert.Equal(t, value5, incomeAcc00.GetBalance())
//
// 	value6, _ := new(big.Int).SetString("39000000000000000000", 10)
// 	assert.Equal(t, value6, incomeAcc01.GetBalance())
//
// 	value7, _ := new(big.Int).SetString("35000000000000000000", 10)
// 	assert.Equal(t, value7, incomeAcc02.GetBalance())
//
// 	incomeAcc03 := am.GetAccount(incomeAddr03)
// 	value8, _ := new(big.Int).SetString("13000000000000000000", 10)
// 	assert.Equal(t, value8, incomeAcc03.GetBalance())
//
// 	incomeAcc04 := am.GetAccount(incomeAddr04)
// 	value9, _ := new(big.Int).SetString("12000000000000000000", 10)
// 	assert.Equal(t, value9, incomeAcc04.GetBalance())
//
// }
//
// func Test_calculateSalary(t *testing.T) {
// 	tests := []struct {
// 		Expect, TotalSalary, DeputyVotes, TotalVotes, Precision int64
// 	}{
// 		// total votes=100
// 		{0, 100, 0, 100, 1},
// 		{1, 100, 1, 100, 1},
// 		{2, 100, 2, 100, 1},
// 		{100, 100, 100, 100, 1},
// 		// total votes=100, precision=10
// 		{0, 100, 1, 100, 10},
// 		{10, 100, 10, 100, 10},
// 		{10, 100, 11, 100, 10},
// 		// total votes=1000
// 		{0, 100, 1, 1000, 1},
// 		{0, 100, 9, 1000, 1},
// 		{1, 100, 10, 1000, 1},
// 		{1, 100, 11, 1000, 1},
// 		{100, 100, 1000, 1000, 1},
// 		// total votes=1000, precision=10
// 		{10, 100, 100, 1000, 10},
// 		{10, 100, 120, 1000, 10},
// 		{20, 100, 280, 1000, 10},
// 		// total votes=10
// 		{0, 100, 0, 10, 1},
// 		{10, 100, 1, 10, 1},
// 		{100, 100, 10, 10, 1},
// 		// total votes=10, precision=10
// 		{10, 100, 1, 10, 10},
// 		{100, 100, 10, 10, 10},
// 	}
// 	for _, test := range tests {
// 		expect := big.NewInt(test.Expect)
// 		totalSalary := big.NewInt(test.TotalSalary)
// 		deputyVotes := big.NewInt(test.DeputyVotes)
// 		totalVotes := big.NewInt(test.TotalVotes)
// 		precision := big.NewInt(test.Precision)
// 		assert.Equalf(t, 0, calculateSalary(totalSalary, deputyVotes, totalVotes, precision).Cmp(expect), "calculateSalary(%v, %v, %v, %v)", totalSalary, deputyVotes, totalVotes, precision)
// 	}
// }
//
// // Test_DivideSalary test total salary with random data
// func Test_DivideSalary(t *testing.T) {
// 	ClearData()
// 	db := store.NewChainDataBase(GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
// 	defer db.Close()
// 	am := account.NewManager(common.Hash{}, db)
//
// 	r := rand.New(rand.NewSource(time.Now().Unix()))
// 	for i := 0; i < 100; i++ {
// 		nodeCount := r.Intn(49) + 1 // [1, 50]
// 		nodes := GenerateDeputies(nodeCount)
// 		registerDeputies(nodes, am)
// 		for _, node := range nodes {
// 			node.Votes = randomBigInt(r)
// 		}
//
// 		totalSalary := randomBigInt(r)
// 		term := &deputynode.TermRecord{TermIndex: 0, Nodes: nodes}
//
// 		salaries := DivideSalary(totalSalary, am, term)
// 		assert.Len(t, salaries, nodeCount)
//
// 		// 验证income是否相同
// 		for j := 0; j < len(nodes); j++ {
// 			if getIncomeAddressFromDeputyNode(am, nodes[j]) != salaries[j].Address {
// 				panic("income address no equal")
// 			}
// 		}
// 		actualTotal := new(big.Int)
// 		for _, s := range salaries {
// 			actualTotal.Add(actualTotal, s.Salary)
// 		}
// 		totalVotes := new(big.Int)
// 		for _, v := range nodes {
// 			totalVotes.Add(totalVotes, v.Votes)
// 		}
// 		// 比较每个deputy node salary
// 		for k := 0; k < len(nodes); k++ {
// 			if salaries[k].Salary.Cmp(calculateSalary(totalSalary, nodes[k].Votes, totalVotes, minPrecision)) != 0 {
// 				panic("deputy node salary no equal")
// 			}
// 		}
//
// 		// errRange = nodeCount * minPrecision
// 		// actualTotal must be in range [totalSalary - errRange, totalSalary]
// 		errRange := new(big.Int).Mul(big.NewInt(int64(nodeCount)), minPrecision)
// 		assert.Equal(t, true, actualTotal.Cmp(new(big.Int).Sub(totalSalary, errRange)) >= 0)
// 		assert.Equal(t, true, actualTotal.Cmp(totalSalary) <= 0)
// 	}
// }

// newSignedBlock 用来生成符合测试用例所用的区块
// func newSignedBlock(dpovp *DPoVP, parentHash common.Hash, author common.Address, txs []*types.Transaction, time uint32, signPrivate string, save bool) *types.Block {
// 	newBlockInfo := test.blockInfo{
// 		parentHash: parentHash,
// 		author:     author,
// 		txList:     txs,
// 		gasLimit:   500000000,
// 		time:       time,
// 	}
// 	parent, err := dpovp.db.GetBlockByHash(parentHash)
// 	if err != nil {
// 		// genesis block
// 		newBlockInfo.height = 0
// 	} else {
// 		newBlockInfo.height = parent.Height() + 1
// 	}
// 	testBlock := test.makeBlock(dpovp.db, newBlockInfo, save)
// 	if save {
// 		if err := dpovp.UpdateStable(testBlock); err != nil {
// 			panic(err)
// 		}
// 	}
// 	// 对区块进行签名
// 	signTestBlock(signPrivate, testBlock)
// 	return testBlock
// }

func registerDeputies(deputies types.DeputyNodes, am *account.Manager) {
	for _, node := range deputies {
		profile := make(map[string]string)
		// 设置deputy node 的income address
		minerAcc := am.GetAccount(node.MinerAddress)
		// 设置income address 为minerAddress
		profile[types.CandidateKeyIncomeAddress] = node.MinerAddress.String()
		minerAcc.SetCandidate(profile)
	}
}

func randomBigInt(r *rand.Rand) *big.Int {
	return new(big.Int).Mul(big.NewInt(r.Int63()), big.NewInt(r.Int63()))
}
