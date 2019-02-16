package store

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"strconv"
	"testing"
)

func NewAccountData(address common.Address, isCandidate bool) *types.AccountData {
	account := &types.AccountData{
		Address:       address,
		Balance:       new(big.Int).SetInt64(100),
		NewestRecords: make(map[types.ChangeLogType]types.VersionRecord),
	}

	account.Candidate.Profile = make(types.CandidateProfile)
	if isCandidate {
		account.Candidate.Votes = new(big.Int).SetInt64(50)
		account.Candidate.Profile[types.CandidateKeyIsCandidate] = "true"
	}
	account.Candidate.Votes = new(big.Int)

	return account
}

func NewAccountDataBatch(count int) []*types.AccountData {
	result := make([]*types.AccountData, count, 100)
	for index := 0; index < count; index++ {
		result[index] = NewAccountData(common.HexToAddress(strconv.Itoa(index)), true)
	}
	return result
}

func equal(src []*types.AccountData, size int, dst []*Candidate) bool {
	if len(src) == 0 && len(dst) == 0 {
		return true
	}

	if len(src) != len(dst) && size != len(dst) {
		return false
	}

	for index := 0; index < size; index++ {
		if src[index].Address != dst[index].address {
			return false
		}
	}

	return true
}

func modifyCandidates(srcPos int, dstPos int, val int64, accounts []*types.AccountData) []*types.AccountData {
	if srcPos < dstPos {
		panic("src pos greater dst pos.")
	}

	if srcPos == dstPos {
		accounts[srcPos].Candidate.Votes = new(big.Int).SetInt64(val)
		return accounts
	} else {
		srcVal := accounts[srcPos]
		for index := srcPos; index > dstPos; index-- {
			accounts[index] = accounts[index-1]
		}
		accounts[dstPos] = srcVal
		accounts[dstPos].Candidate.Votes = new(big.Int).SetInt64(val)
		return accounts
	}
}

func clone(src []*types.AccountData) []*types.AccountData {
	if len(src) <= 0 {
		return make([]*types.AccountData, 0)
	}

	dst := make([]*types.AccountData, len(src))
	for index := 0; index < len(src); index++ {
		dst[index] = src[index].Copy()
	}
	return dst
}

func TestCBlock_RankingNo1(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	cblock1 := NewGenesisBlock(GetBlock1(), cacheChain.Beansdb)

	account1 := NewAccountData(common.HexToAddress(strconv.Itoa(0x1000)), false)
	cblock1.AccountTrieDB.Put(account1, 1)
	account2 := NewAccountData(common.HexToAddress(strconv.Itoa(0x1001)), false)
	cblock1.AccountTrieDB.Put(account2, 1)

	count := 29
	candidates := NewAccountDataBatch(count)
	for index := 0; index < count; index++ {
		cblock1.AccountTrieDB.Put(candidates[index], 1)
	}

	cblock1.Ranking()
	top30 := cblock1.Top30
	assert.Equal(t, true, equal(candidates, count, top30))
}

func TestCBlock_RankingNo2(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	cblock1 := NewGenesisBlock(GetBlock1(), cacheChain.Beansdb)

	count := 29
	candidates := NewAccountDataBatch(count)

	account1 := NewAccountData(common.HexToAddress(strconv.Itoa(0x1000)), false)
	account1.VoteFor = candidates[0].Address
	account1.Balance.SetInt64(50000)
	cblock1.AccountTrieDB.Put(account1, 1)
	candidates = modifyCandidates(0, 0, 50000, candidates)

	account2 := NewAccountData(common.HexToAddress(strconv.Itoa(0x1001)), false)
	account2.VoteFor = candidates[9].Address
	account2.Balance.SetInt64(40000)
	cblock1.AccountTrieDB.Put(account2, 1)
	candidates = modifyCandidates(9, 1, 40000, candidates)

	for index := 0; index < count; index++ {
		cblock1.AccountTrieDB.Put(candidates[index], 1)
	}

	cblock1.Ranking()
	top30 := cblock1.Top30
	assert.Equal(t, true, equal(candidates, count, top30))
}

func TestCBlock_RankingNo3(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	cblock1 := NewGenesisBlock(GetBlock1(), cacheChain.Beansdb)

	count := 29
	candidates := NewAccountDataBatch(count)

	account1 := NewAccountData(common.HexToAddress(strconv.Itoa(0x1000)), false)
	account1.VoteFor = candidates[0].Address
	account1.Balance.SetInt64(50000)
	cblock1.AccountTrieDB.Put(account1, 1)
	candidates = modifyCandidates(0, 0, 50000, candidates)

	account2 := NewAccountData(common.HexToAddress(strconv.Itoa(0x1001)), false)
	account2.VoteFor = candidates[9].Address
	account2.Balance.SetInt64(40000)
	cblock1.AccountTrieDB.Put(account2, 1)
	candidates = modifyCandidates(9, 1, 40000, candidates)

	for index := 0; index < count; index++ {
		cblock1.AccountTrieDB.Put(candidates[index], 1)
	}

	cblock1.Ranking()
	cblock1Candidates := clone(candidates)

	cblock2 := NewNormalBlock(GetBlock2(), cblock1.AccountTrieDB, cblock1.CandidateTrieDB, cblock1.Top30)
	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count)), true))
	candidates = modifyCandidates(count, 2, 30000, candidates)
	cblock2.AccountTrieDB.Put(candidates[2], 2)

	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count+1)), true))
	candidates = modifyCandidates(count+1, 3, 20000, candidates)
	cblock2.AccountTrieDB.Put(candidates[3], 2)
	cblock2.Ranking()

	block2Top30 := cblock2.Top30
	assert.Equal(t, true, equal(candidates, 30, block2Top30))
	assert.Equal(t, 31, len(cblock2.CandidateTrieDB.GetAll()))

	block1Top30 := cblock1.Top30
	assert.Equal(t, true, equal(cblock1Candidates, 29, block1Top30))
	assert.Equal(t, 29, len(cblock1.CandidateTrieDB.GetAll()))
}

//
func TestCBlock_RankingNo10(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	cblock1 := NewGenesisBlock(GetBlock1(), cacheChain.Beansdb)

	count := 50
	candidates := NewAccountDataBatch(count)

	account1 := NewAccountData(common.HexToAddress(strconv.Itoa(0x1000)), false)
	account1.VoteFor = candidates[0].Address
	account1.Balance.SetInt64(50000)
	cblock1.AccountTrieDB.Put(account1, 1)
	candidates = modifyCandidates(0, 0, 50000, candidates)

	account2 := NewAccountData(common.HexToAddress(strconv.Itoa(0x1001)), false)
	account2.VoteFor = candidates[9].Address
	account2.Balance.SetInt64(40000)
	cblock1.AccountTrieDB.Put(account2, 1)
	candidates = modifyCandidates(9, 1, 40000, candidates)

	for index := 0; index < count; index++ {
		cblock1.AccountTrieDB.Put(candidates[index], 1)
	}

	cblock1.Ranking()
	cblock1Candidates := clone(candidates)

	cblock2 := NewNormalBlock(GetBlock2(), cblock1.AccountTrieDB, cblock1.CandidateTrieDB, cblock1.Top30)
	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count)), true))
	candidates = modifyCandidates(count, 2, 30000, candidates)
	cblock2.AccountTrieDB.Put(candidates[2], 2)

	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count+1)), true))
	candidates = modifyCandidates(count+1, 3, 20000, candidates)
	cblock2.AccountTrieDB.Put(candidates[3], 2)
	cblock2.Ranking()

	block2Top30 := cblock2.Top30
	assert.Equal(t, true, equal(candidates, 30, block2Top30))
	assert.Equal(t, 52, len(cblock2.CandidateTrieDB.GetAll()))

	block1Top30 := cblock1.Top30
	assert.Equal(t, true, equal(cblock1Candidates, 30, block1Top30))
	assert.Equal(t, 50, len(cblock1.CandidateTrieDB.GetAll()))
}
