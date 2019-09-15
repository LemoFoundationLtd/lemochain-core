package consensus

import (
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/transaction"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/merkle"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"math/rand"
	"testing"
	"time"
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
	private, err := crypto.GenerateKey()
	assert.NoError(t, err)
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

// createAssembler clear account manager then make some new change logs
func createAssembler(db *store.ChainDatabase, fillSomeLogs bool) *BlockAssembler {
	am := account.NewManager(common.Hash{}, db)

	deputyCount := 5
	dm := deputynode.NewManager(deputyCount, db)
	deputies := generateDeputies(deputyCount)
	dm.SaveSnapshot(0, deputies)
	dm.SaveSnapshot(params.TermDuration, deputies)

	if fillSomeLogs {
		// This will make 5 CandidateLogs and 5 VotesLogs
		for _, deputy := range deputies {
			profile := types.Profile{
				types.CandidateKeyIsCandidate:   params.IsCandidateNode,
				types.CandidateKeyDepositAmount: "100",
			}
			deputyAccount := am.GetAccount(deputy.MinerAddress)
			deputyAccount.SetCandidate(profile)
			deputyAccount.SetVotes(deputy.Votes)
		}
		// set reward pool balance. It will make 1 BalanceLog
		am.GetAccount(params.DepositPoolAddress).SetBalance(big.NewInt(int64(100 * deputyCount)))

		// make balance logs. It will make 3 BalanceLogs, but they could be merged to 2 logs
		account1 := am.GetAccount(common.HexToAddress("0x123"))
		account1.SetBalance(big.NewInt(10000))
		account1.SetBalance(big.NewInt(9999))
		account2 := am.GetAccount(common.HexToAddress("234"))
		account2.SetBalance(big.NewInt(100))

		// set reward for term 0. It will make 1 StorageLog. And 1 StorageRootLog after accountManager.Finalise()
		rewardAccont := am.GetAccount(params.TermRewardContract)
		rewardsMap := params.RewardsMap{
			0: &params.Reward{Term: 0, Value: common.Lemo2Mo("10000")},
			1: &params.Reward{Term: 1, Value: common.Lemo2Mo("0")},
		}
		storageVal, err := json.Marshal(rewardsMap)
		if err != nil {
			panic(err)
		}
		err = rewardAccont.SetStorageState(params.TermRewardContract.Hash(), storageVal)
		if err != nil {
			panic(err)
		}

		// unregister first deputy It will make 1 CandidateStateLog
		deputy := deputies[0]
		am.GetAccount(deputy.MinerAddress).SetCandidateState(types.CandidateKeyIsCandidate, params.NotCandidateNode)
	}

	return NewBlockAssembler(am, dm, &transaction.TxProcessor{}, testCandidateLoader{deputies[0]})
}

func TestBlockAssembler_Finalize(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), "", "")
	defer db.Close()

	finalizeAndCountLogs := func(ba *BlockAssembler, height uint32, logsCount int) {
		err := ba.Finalize(height) // genesis block
		assert.NoError(t, err)
		logs := ba.am.GetChangeLogs()
		assert.Equal(t, logsCount, len(logs))
	}
	checkReward := func(am *account.Manager, dm *deputynode.Manager) {
		deputy := dm.GetDeputiesByHeight(1)[1]
		deputyAccount := am.GetAccount(deputy.MinerAddress)
		assert.Equal(t, -1, big.NewInt(0).Cmp(deputyAccount.GetBalance()))
	}
	checkRefund := func(am *account.Manager, dm *deputynode.Manager) {
		deputy := dm.GetDeputiesByHeight(1)[0]
		deputyAccount := am.GetAccount(deputy.MinerAddress)
		assert.Equal(t, params.NotCandidateNode, deputyAccount.GetCandidateState(types.CandidateKeyIsCandidate))
		assert.Equal(t, "", deputyAccount.GetCandidateState(types.CandidateKeyDepositAmount))
		assert.Equal(t, -1, big.NewInt(0).Cmp(deputyAccount.GetVotes()))
	}

	// nothing to finalize and no reward
	ba := createAssembler(db, false)
	finalizeAndCountLogs(ba, 0, 0) // genesis block
	ba = createAssembler(db, false)
	finalizeAndCountLogs(ba, 1, 0) // normal block

	// many change logs
	ba = createAssembler(db, true)
	finalizeAndCountLogs(ba, 0, 16) // genesis block
	ba = createAssembler(db, true)
	finalizeAndCountLogs(ba, 1, 16) // normal block
	ba = createAssembler(db, true)
	finalizeAndCountLogs(ba, params.TermDuration+params.InterimDuration+1-params.RewardCheckHeight, 16) // check reward block
	ba = createAssembler(db, true)
	finalizeAndCountLogs(ba, params.TermDuration, 16) // snapshot block
	ba = createAssembler(db, true)
	finalizeAndCountLogs(ba, params.TermDuration+params.InterimDuration+1, 22) // reward and deposit refund block
	checkReward(ba.am, ba.dm)
	checkRefund(ba.am, ba.dm)
	ba = createAssembler(db, true)
	finalizeAndCountLogs(ba, params.TermDuration*2-100, 16) // 2st term normal block
	ba = createAssembler(db, true)
	finalizeAndCountLogs(ba, params.TermDuration*2+params.InterimDuration+1, 18) // 2st term reward block
	checkRefund(ba.am, ba.dm)
}

type errCanLoader struct {
}

func (cl errCanLoader) LoadTopCandidates(blockHash common.Hash) types.DeputyNodes {
	return types.DeputyNodes{}
}

func (cl errCanLoader) LoadRefundCandidates(height uint32) ([]common.Address, error) {
	return []common.Address{}, errors.New("refund error")
}

func TestBlockAssembler_Finalize2(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), "", "")
	defer db.Close()

	// issue term reward failed
	ba := createAssembler(db, true)
	rewardAccont := ba.am.GetAccount(params.TermRewardContract)
	err := rewardAccont.SetStorageState(params.TermRewardContract.Hash(), []byte{0x12})
	assert.NoError(t, err)
	err = ba.Finalize(params.TermDuration + params.InterimDuration + 1)
	assert.EqualError(t, err, "invalid character '\\x12' looking for beginning of value")

	// refund deposit failed
	ba = createAssembler(db, true)
	ba.canLoader = errCanLoader{}
	err = ba.Finalize(params.TermDuration + params.InterimDuration + 1)
	assert.Equal(t, errors.New("refund error"), err)

	// account manager finalise failed
	ba = createAssembler(db, true)
	storageRootLog := &types.ChangeLog{
		Version: 1,
		Address: params.TermRewardContract,
		LogType: account.StorageRootLog,
		NewVal:  common.HexToHash("0x12"),
	}
	err = ba.am.Rebuild(params.TermRewardContract, types.ChangeLogSlice{storageRootLog}) // break the storage root
	assert.NoError(t, err)
	err = ba.Finalize(1)
	assert.Equal(t, account.ErrTrieFail, err)
}

func TestCheckTermReward(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), "", "")
	defer db.Close()

	// TODO
}

func TestIssueTermReward(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), "", "")
	defer db.Close()

	// TODO
}

func TestRefundCandidateDeposit(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), "", "")
	defer db.Close()

	// TODO
}

func TestCalculateSalary(t *testing.T) {
	tests := []struct {
		Expect, TotalSalary, DeputyVotes, TotalVotes, Precision int64
	}{
		// total votes=100
		{0, 100, 0, 100, 1},
		{1, 100, 1, 100, 1},
		{2, 100, 2, 100, 1},
		{100, 100, 100, 100, 1},
		// total votes=100, precision=10
		{0, 100, 1, 100, 10},
		{10, 100, 10, 100, 10},
		{10, 100, 11, 100, 10},
		// total votes=1000
		{0, 100, 1, 1000, 1},
		{0, 100, 9, 1000, 1},
		{1, 100, 10, 1000, 1},
		{1, 100, 11, 1000, 1},
		{100, 100, 1000, 1000, 1},
		// total votes=1000, precision=10
		{10, 100, 100, 1000, 10},
		{10, 100, 120, 1000, 10},
		{20, 100, 280, 1000, 10},
		// total votes=10
		{0, 100, 0, 10, 1},
		{10, 100, 1, 10, 1},
		{100, 100, 10, 10, 1},
		// total votes=10, precision=10
		{10, 100, 1, 10, 10},
		{100, 100, 10, 10, 10},
	}
	for _, test := range tests {
		expect := big.NewInt(test.Expect)
		totalSalary := big.NewInt(test.TotalSalary)
		deputyVotes := big.NewInt(test.DeputyVotes)
		totalVotes := big.NewInt(test.TotalVotes)
		precision := big.NewInt(test.Precision)
		assert.Equalf(t, 0, calculateSalary(totalSalary, deputyVotes, totalVotes, precision).Cmp(expect), "calculateSalary(%v, %v, %v, %v)", totalSalary, deputyVotes, totalVotes, precision)
	}
}

func TestGetDeputyIncomeAddress(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)

	// no set income address
	minerAddr := common.HexToAddress("0x123")
	deputy := &types.DeputyNode{MinerAddress: minerAddr}
	incomeAddr := getDeputyIncomeAddress(am, deputy)
	assert.Equal(t, minerAddr, incomeAddr)

	// invalid income address
	candidate := am.GetAccount(minerAddr)
	candidate.SetCandidate(types.Profile{types.CandidateKeyIncomeAddress: "0x234"})
	incomeAddr = getDeputyIncomeAddress(am, deputy)
	assert.Equal(t, minerAddr, incomeAddr)

	// valid income address
	incomeAddrStr := "lemoqr"
	validIncomeAddr, _ := common.StringToAddress(incomeAddrStr)
	candidate.SetCandidate(types.Profile{types.CandidateKeyIncomeAddress: incomeAddrStr})
	incomeAddr = getDeputyIncomeAddress(am, deputy)
	assert.Equal(t, validIncomeAddr, incomeAddr)
}

// TestDivideSalary test total salary with random data
func TestDivideSalary(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)

	r := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < 100; i++ {
		nodeCount := r.Intn(49) + 1 // [1, 50]
		nodes := generateDeputies(nodeCount)
		registerDeputies(nodes, am)
		for _, node := range nodes {
			node.Votes = randomBigInt(r)
		}

		totalSalary := randomBigInt(r)
		term := &deputynode.TermRecord{TermIndex: 0, Nodes: nodes}

		salaries := DivideSalary(totalSalary, am, term)
		assert.Len(t, salaries, nodeCount)

		// 验证income是否相同
		for j := 0; j < len(nodes); j++ {
			if getDeputyIncomeAddress(am, nodes[j]) != salaries[j].Address {
				panic("income address no equal")
			}
		}
		actualTotal := new(big.Int)
		for _, s := range salaries {
			actualTotal.Add(actualTotal, s.Salary)
		}
		totalVotes := new(big.Int)
		for _, v := range nodes {
			totalVotes.Add(totalVotes, v.Votes)
		}
		// 比较每个deputy node salary
		for k := 0; k < len(nodes); k++ {
			if salaries[k].Salary.Cmp(calculateSalary(totalSalary, nodes[k].Votes, totalVotes, params.MinRewardPrecision)) != 0 {
				panic("deputy node salary no equal")
			}
		}

		// errRange = nodeCount * minPrecision
		// actualTotal must be in range [totalSalary - errRange, totalSalary]
		errRange := new(big.Int).Mul(big.NewInt(int64(nodeCount)), params.MinRewardPrecision)
		assert.Equal(t, true, actualTotal.Cmp(new(big.Int).Sub(totalSalary, errRange)) >= 0)
		assert.Equal(t, true, actualTotal.Cmp(totalSalary) <= 0)
	}
}

// 对区块进行签名的函数
// func signTestBlock(deputyPrivate string, block *types.Block) {
// 	// 对区块签名
// 	private, err := crypto.ToECDSA(common.FromHex(deputyPrivate))
// 	if err != nil {
// 		panic(err)
// 	}
// 	signBlock, err := crypto.Sign(block.Hash().Bytes(), private)
// 	if err != nil {
// 		panic(err)
// 	}
// 	block.Header.SignData = signBlock
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

func TestGetTermRewards(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	rewardAccont := am.GetAccount(params.TermRewardContract)

	// no reward
	result, err := getTermRewards(am, 0)
	assert.NoError(t, err)
	assert.Equal(t, *big.NewInt(0), *result)

	// load fail (account storage root is invalid)
	storageRootLog := &types.ChangeLog{
		Version: 1,
		Address: params.TermRewardContract,
		LogType: account.StorageRootLog,
		NewVal:  common.HexToHash("0x12"),
	}
	err = am.Rebuild(params.TermRewardContract, types.ChangeLogSlice{storageRootLog})
	assert.NoError(t, err)
	_, err = getTermRewards(am, 0)
	assert.Equal(t, account.ErrTrieFail, err)
	am.Reset(common.Hash{})
	rewardAccont = am.GetAccount(params.TermRewardContract)

	// set invalid reward
	err = rewardAccont.SetStorageState(params.TermRewardContract.Hash(), []byte{0x12})
	assert.NoError(t, err)
	result, err = getTermRewards(am, 0)
	assert.EqualError(t, err, "invalid character '\\x12' looking for beginning of value")
	am.Reset(common.Hash{})
	rewardAccont = am.GetAccount(params.TermRewardContract)

	// set rewards
	rewardAmount := big.NewInt(100)
	rewardsMap := params.RewardsMap{0: &params.Reward{Term: 0, Value: rewardAmount}}
	storageVal, err := json.Marshal(rewardsMap)
	assert.NoError(t, err)
	err = rewardAccont.SetStorageState(params.TermRewardContract.Hash(), storageVal)
	assert.NoError(t, err)
	result, err = getTermRewards(am, 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, rewardAmount.Cmp(result))
	// set rewards but not the term
	result, err = getTermRewards(am, 1)
	assert.NoError(t, err)
	assert.Equal(t, 0, big.NewInt(0).Cmp(result))
	am.Reset(common.Hash{})
	rewardAccont = am.GetAccount(params.TermRewardContract)

	// set empty rewards
	storageVal, err = json.Marshal(params.RewardsMap{})
	assert.NoError(t, err)
	err = rewardAccont.SetStorageState(params.TermRewardContract.Hash(), storageVal)
	assert.NoError(t, err)
	result, err = getTermRewards(am, 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, big.NewInt(0).Cmp(result))
}

func TestBlockAssembler_RunBlock(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	dm := deputynode.NewManager(5, db)
	deputies := generateDeputies(5)
	dm.SaveSnapshot(0, deputies)
	dm.SaveSnapshot(params.TermDuration, deputies)

	// genesis block
	processor := transaction.NewTxProcessor(deputies[0].MinerAddress, 100, &parentLoader{db}, am, db, dm)
	ba := NewBlockAssembler(am, dm, processor, testCandidateLoader{deputies[0]})
	tx := makeTx(deputies[0].MinerAddress, common.HexToAddress("0x88"), uint64(time.Now().Unix()))
	rawBlock := &types.Block{Header: &types.Header{Height: 1}, Txs: types.Transactions{tx}}
	newBlock, err := ba.RunBlock(rawBlock)
	assert.NoError(t, err)
	assert.NotEqual(t, rawBlock, newBlock)
	// TODO check gasused
}

func TestBlockAssembler_MineBlock(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)

	// TODO
	am.GetAccount(common.HexToAddress("0x"))
}
