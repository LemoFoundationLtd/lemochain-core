package store

import (
	"bytes"
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
		account.Candidate.Votes = new(big.Int).SetInt64(500)
		account.Candidate.Profile[types.CandidateKeyIsCandidate] = "true"
	} else {
		account.Candidate.Votes = new(big.Int)
	}

	return account
}

func NewAccountDataWithVotes(address common.Address, val int64) *types.AccountData {
	account := &types.AccountData{
		Address:       address,
		Balance:       new(big.Int).SetInt64(100),
		NewestRecords: make(map[types.ChangeLogType]types.VersionRecord),
	}

	account.Candidate.Profile = make(types.CandidateProfile)
	account.Candidate.Votes = new(big.Int).SetInt64(val)
	account.Candidate.Profile[types.CandidateKeyIsCandidate] = "true"

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
		if src[index].Address != dst[index].Address {
			return false
		}
	}

	return true
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

func sort(src []*types.AccountData) []*types.AccountData {
	if len(src) <= 0 {
		return make([]*types.AccountData, 0)
	}

	for i := 0; i < len(src); i++ {
		for j := i + 1; j < len(src); j++ {
			val := src[i].Candidate.Votes.Cmp(src[j].Candidate.Votes)

			if val < 0 {
				src[i], src[j] = src[j], src[i]
			} else {
				if (val == 0) &&
					(bytes.Compare(src[i].Address[:], src[j].Address[:]) > 0) {
					src[i], src[j] = src[j], src[i]
				}
			}
		}
	}
	return src
}

func TestSort(t *testing.T) {
	accounts := make([]*types.AccountData, 5)
	accounts[0] = NewAccountDataWithVotes(common.HexToAddress(strconv.Itoa(0)), 500)
	accounts[1] = NewAccountDataWithVotes(common.HexToAddress(strconv.Itoa(1)), 400)
	accounts[2] = NewAccountDataWithVotes(common.HexToAddress(strconv.Itoa(2)), 500)
	accounts[3] = NewAccountDataWithVotes(common.HexToAddress(strconv.Itoa(3)), 200)
	accounts[4] = NewAccountDataWithVotes(common.HexToAddress(strconv.Itoa(4)), 600)

	result := make([]*types.AccountData, 5)
	result[0] = accounts[4]
	result[1] = accounts[0]
	result[2] = accounts[2]
	result[3] = accounts[1]
	result[4] = accounts[3]

	tmp := sort(accounts)
	for index := 0; index < 5; index++ {
		assert.Equal(t, result[index].Address, tmp[index].Address)
	}
}

func TestCBlock_RankingNo1(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	cblock1 := NewGenesisBlock(GetBlock1(), cacheChain.Beansdb)

	account1 := NewAccountData(common.HexToAddress(strconv.Itoa(0x1000)), false)
	cblock1.AccountTrieDB.Put(account1, 1)
	account2 := NewAccountData(common.HexToAddress(strconv.Itoa(0x1001)), false)
	cblock1.AccountTrieDB.Put(account2, 1)

	count := 19
	candidates := NewAccountDataBatch(count)
	for index := 0; index < count; index++ {
		cblock1.AccountTrieDB.Put(candidates[index], 1)
	}

	cblock1.Ranking()
	top30 := cblock1.Top30
	assert.Equal(t, count, len(top30))
	assert.Equal(t, true, equal(candidates, count, top30))
}

func TestCBlock_RankingNo2(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	cblock1 := NewGenesisBlock(GetBlock1(), cacheChain.Beansdb)

	count := 19
	candidates := NewAccountDataBatch(count)
	candidates[0].Candidate.Votes.SetInt64(50000)
	candidates[9].Candidate.Votes.SetInt64(40000)
	for index := 0; index < count; index++ {
		cblock1.AccountTrieDB.Put(candidates[index], 1)
	}

	cblock1.Ranking()
	top30 := cblock1.Top30
	assert.Equal(t, count, len(top30))
	assert.Equal(t, true, equal(sort(candidates), count, top30))
}

func TestCBlock_RankingNo3(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	cblock1 := NewGenesisBlock(GetBlock1(), cacheChain.Beansdb)

	count := 19
	candidates := NewAccountDataBatch(count)
	candidates[0].Candidate.Votes.SetInt64(50000)
	candidates[9].Candidate.Votes.SetInt64(40000)
	for index := 0; index < count; index++ {
		cblock1.AccountTrieDB.Put(candidates[index], 1)
	}
	cblock1.Ranking()
	cblock1Candidates := clone(candidates)

	cblock2 := NewNormalBlock(GetBlock2(), cblock1.AccountTrieDB, cblock1.CandidateTrieDB, cblock1.Top30)
	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count+1)), true))
	candidates[count].Candidate.Votes.SetInt64(30000)
	cblock2.AccountTrieDB.Put(candidates[count], 2)

	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count+2)), true))
	candidates[count+1].Candidate.Votes.SetInt64(30000)
	cblock2.AccountTrieDB.Put(candidates[count+1], 2)
	cblock2.Ranking()

	block1Top30 := cblock1.Top30
	assert.Equal(t, true, equal(sort(cblock1Candidates), count, block1Top30))
	assert.Equal(t, count, len(cblock1.CandidateTrieDB.GetAll()))

	block2Top30 := cblock2.Top30
	assert.Equal(t, max_candidate_count, len(block2Top30))
	assert.Equal(t, count+2, len(cblock2.CandidateTrieDB.GetAll()))
	assert.Equal(t, true, equal(sort(candidates), max_candidate_count, block2Top30))
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
	candidates[0].Candidate.Votes.SetInt64(50000)

	account2 := NewAccountData(common.HexToAddress(strconv.Itoa(0x1001)), false)
	account2.VoteFor = candidates[9].Address
	account2.Balance.SetInt64(40000)
	cblock1.AccountTrieDB.Put(account2, 1)
	candidates[9].Candidate.Votes.SetInt64(50000)

	for index := 0; index < count; index++ {
		cblock1.AccountTrieDB.Put(candidates[index], 1)
	}

	cblock1.Ranking()
	cblock1Candidates := clone(candidates)

	cblock2 := NewNormalBlock(GetBlock2(), cblock1.AccountTrieDB, cblock1.CandidateTrieDB, cblock1.Top30)
	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count)), true))
	candidates[count].Candidate.Votes.SetInt64(50000)
	cblock2.AccountTrieDB.Put(candidates[count], 2)

	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count+1)), true))
	candidates[count+1].Candidate.Votes.SetInt64(50000)
	cblock2.AccountTrieDB.Put(candidates[count+1], 2)
	cblock2.Ranking()

	block1Top30 := cblock1.Top30
	assert.Equal(t, true, equal(sort(cblock1Candidates), max_candidate_count, block1Top30))
	assert.Equal(t, count, len(cblock1.CandidateTrieDB.GetAll()))

	block2Top30 := cblock2.Top30
	assert.Equal(t, true, equal(sort(candidates), max_candidate_count, block2Top30))
	assert.Equal(t, count+2, len(cblock2.CandidateTrieDB.GetAll()))
}

func TestCBlock_RankingNo11(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	cblock1 := NewGenesisBlock(GetBlock1(), cacheChain.Beansdb)

	count := 50
	candidates := NewAccountDataBatch(count)
	candidates[0].Candidate.Votes.SetInt64(50000)
	candidates[9].Candidate.Votes.SetInt64(40000)
	for index := 0; index < count; index++ {
		cblock1.AccountTrieDB.Put(candidates[index], 1)
	}
	cblock1.Ranking()
	cblock1Candidates := clone(candidates)

	cblock2 := NewNormalBlock(GetBlock2(), cblock1.AccountTrieDB, cblock1.CandidateTrieDB, cblock1.Top30)
	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count)), true))
	candidates[count].Candidate.Votes.SetInt64(20)
	cblock2.AccountTrieDB.Put(candidates[count], 2)

	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count+1)), true))
	candidates[count+1].Candidate.Votes.SetInt64(20)
	cblock2.AccountTrieDB.Put(candidates[count+1], 2)
	cblock2.Ranking()

	block1Top30 := cblock1.Top30
	assert.Equal(t, true, equal(sort(cblock1Candidates), max_candidate_count, block1Top30))
	assert.Equal(t, count, len(cblock1.CandidateTrieDB.GetAll()))

	block2Top30 := cblock2.Top30
	assert.Equal(t, true, equal(sort(candidates), max_candidate_count, block2Top30))
	assert.Equal(t, count+2, len(cblock2.CandidateTrieDB.GetAll()))
}

func TestCBlock_RankingNo12(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	cblock1 := NewGenesisBlock(GetBlock1(), cacheChain.Beansdb)

	count := 50
	candidates := NewAccountDataBatch(count)
	candidates[0].Candidate.Votes.SetInt64(50000)
	candidates[9].Candidate.Votes.SetInt64(40000)
	for index := 0; index < count; index++ {
		cblock1.AccountTrieDB.Put(candidates[index], 1)
	}
	cblock1.Ranking()
	cblock1Candidates := clone(candidates)

	cblock2 := NewNormalBlock(GetBlock2(), cblock1.AccountTrieDB, cblock1.CandidateTrieDB, cblock1.Top30)
	candidates[18].Candidate.Votes.SetInt64(40)
	cblock2.AccountTrieDB.Put(candidates[18], 2)

	candidates[19].Candidate.Votes.SetInt64(30)
	cblock2.AccountTrieDB.Put(candidates[19], 2)
	cblock2.Ranking()

	block2Top30 := cblock2.Top30
	assert.Equal(t, true, equal(sort(candidates), max_candidate_count, block2Top30))
	assert.Equal(t, count, len(cblock2.CandidateTrieDB.GetAll()))

	block1Top30 := cblock1.Top30
	assert.Equal(t, true, equal(sort(cblock1Candidates), max_candidate_count, block1Top30))
	assert.Equal(t, count, len(cblock1.CandidateTrieDB.GetAll()))
}
