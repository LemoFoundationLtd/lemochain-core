package store

import (
	"bytes"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
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

	account.Candidate.Profile = make(types.Profile)
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

	account.Candidate.Profile = make(types.Profile)
	account.Candidate.Votes = new(big.Int).SetInt64(val)
	account.Candidate.Profile[types.CandidateKeyIsCandidate] = "true"

	return account
}

func NewAccountDataBatch(count int) []*types.AccountData {
	result := make([]*types.AccountData, count)
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

func sortAccount(src []*types.AccountData) []*types.AccountData {
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

	tmp := sortAccount(accounts)
	for index := 0; index < 5; index++ {
		assert.Equal(t, result[index].Address, tmp[index].Address)
	}
}

func TestCBlock_RankingNo1(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())
	defer cacheChain.Close()

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
	top30 := cblock1.Top.GetTop()
	assert.Equal(t, count, len(top30))
	assert.Equal(t, true, equal(candidates, count, top30))

}

func TestCBlock_RankingNo2(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())
	defer cacheChain.Close()

	cblock1 := NewGenesisBlock(GetBlock1(), cacheChain.Beansdb)

	count := 19
	candidates := NewAccountDataBatch(count)
	candidates[0].Candidate.Votes.SetInt64(50000)
	candidates[9].Candidate.Votes.SetInt64(40000)
	for index := 0; index < count; index++ {
		cblock1.AccountTrieDB.Put(candidates[index], 1)
	}

	cblock1.Ranking()
	top30 := cblock1.Top.GetTop()
	assert.Equal(t, count, len(top30))
	assert.Equal(t, true, equal(sortAccount(candidates), count, top30))
}

func TestCBlock_RankingNo3(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())
	defer cacheChain.Close()

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

	cblock2 := NewNormalBlock(GetBlock2(), cblock1.AccountTrieDB, cblock1.CandidateTrieDB, cblock1.Top)
	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count+1)), true))
	candidates[count].Candidate.Votes.SetInt64(30000)
	cblock2.AccountTrieDB.Put(candidates[count], 2)

	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count+2)), true))
	candidates[count+1].Candidate.Votes.SetInt64(30000)
	cblock2.AccountTrieDB.Put(candidates[count+1], 2)
	cblock2.Ranking()

	block1Top30 := cblock1.Top.GetTop()
	assert.Equal(t, true, equal(sortAccount(cblock1Candidates), count, block1Top30))
	assert.Equal(t, count, len(cblock1.CandidateTrieDB.GetAll()))

	block2Top30 := cblock2.Top.GetTop()
	assert.Equal(t, max_candidate_count, len(block2Top30))
	assert.Equal(t, count+2, len(cblock2.CandidateTrieDB.GetAll()))
	assert.Equal(t, true, equal(sortAccount(candidates), max_candidate_count, block2Top30))
}

//
func TestCBlock_RankingNo10(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())
	defer cacheChain.Close()

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

	cblock2 := NewNormalBlock(GetBlock2(), cblock1.AccountTrieDB, cblock1.CandidateTrieDB, cblock1.Top)
	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count)), true))
	candidates[count].Candidate.Votes.SetInt64(50000)
	cblock2.AccountTrieDB.Put(candidates[count], 2)

	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count+1)), true))
	candidates[count+1].Candidate.Votes.SetInt64(50000)
	cblock2.AccountTrieDB.Put(candidates[count+1], 2)
	cblock2.Ranking()

	block1Top30 := cblock1.Top.GetTop()
	assert.Equal(t, true, equal(sortAccount(cblock1Candidates), max_candidate_count, block1Top30))
	assert.Equal(t, count, len(cblock1.CandidateTrieDB.GetAll()))

	block2Top30 := cblock2.Top.GetTop()
	assert.Equal(t, true, equal(sortAccount(candidates), max_candidate_count, block2Top30))
	assert.Equal(t, count+2, len(cblock2.CandidateTrieDB.GetAll()))
}

func TestCBlock_RankingNo11(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())
	defer cacheChain.Close()

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

	cblock2 := NewNormalBlock(GetBlock2(), cblock1.AccountTrieDB, cblock1.CandidateTrieDB, cblock1.Top)
	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count)), true))
	candidates[count].Candidate.Votes.SetInt64(20)
	cblock2.AccountTrieDB.Put(candidates[count], 2)

	candidates = append(candidates, NewAccountData(common.HexToAddress(strconv.Itoa(count+1)), true))
	candidates[count+1].Candidate.Votes.SetInt64(20)
	cblock2.AccountTrieDB.Put(candidates[count+1], 2)
	cblock2.Ranking()

	block1Top30 := cblock1.Top.GetTop()
	assert.Equal(t, true, equal(sortAccount(cblock1Candidates), max_candidate_count, block1Top30))
	assert.Equal(t, count, len(cblock1.CandidateTrieDB.GetAll()))

	block2Top30 := cblock2.Top.GetTop()
	assert.Equal(t, true, equal(sortAccount(candidates), max_candidate_count, block2Top30))
	assert.Equal(t, count+2, len(cblock2.CandidateTrieDB.GetAll()))
}

func TestCBlock_RankingNo12(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())
	defer cacheChain.Close()

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

	cblock2 := NewNormalBlock(GetBlock2(), cblock1.AccountTrieDB, cblock1.CandidateTrieDB, cblock1.Top)
	candidates[18].Candidate.Votes.SetInt64(40)
	cblock2.AccountTrieDB.Put(candidates[18], 2)

	candidates[19].Candidate.Votes.SetInt64(30)
	cblock2.AccountTrieDB.Put(candidates[19], 2)
	cblock2.Ranking()

	block2Top30 := cblock2.Top.GetTop()
	assert.Equal(t, true, equal(sortAccount(candidates), max_candidate_count, block2Top30))
	assert.Equal(t, count, len(cblock2.CandidateTrieDB.GetAll()))

	block1Top30 := cblock1.Top.GetTop()
	assert.Equal(t, true, equal(sortAccount(cblock1Candidates), max_candidate_count, block1Top30))
	assert.Equal(t, count, len(cblock1.CandidateTrieDB.GetAll()))
}

func TestCBlock_BeChildOf(t *testing.T) {
	block1 := &CBlock{}
	block2 := &CBlock{}

	// normal
	block1.BeChildOf(block2)
	assert.Equal(t, block2, block1.Parent)
	assert.Len(t, block2.Children, 1)
	assert.Equal(t, block1, block2.Children[0])

	// is child already
	block1.BeChildOf(block2)
	assert.Equal(t, block2, block1.Parent)
	assert.Len(t, block2.Children, 1)
	assert.Equal(t, block1, block2.Children[0])

	// nil
	block1.BeChildOf(nil)
	assert.Equal(t, (*CBlock)(nil), block1.Parent)
}

func TestCBlock_IsSameBlock(t *testing.T) {
	rawBlock1 := &types.Block{Header: &types.Header{Height: 1}}
	block1 := &CBlock{Block: rawBlock1}
	rawBlock2 := &types.Block{Header: &types.Header{Height: 2}}
	block2 := &CBlock{Block: rawBlock2}
	block3 := &CBlock{Block: rawBlock1}

	assert.Equal(t, true, block1.IsSameBlock(block1))
	assert.Equal(t, false, block1.IsSameBlock(block2))
	assert.Equal(t, true, block1.IsSameBlock(block3))
	assert.Equal(t, false, block1.IsSameBlock(nil))
	assert.Equal(t, true, (*CBlock)(nil).IsSameBlock(nil))
}

// makeBlocks make blocks and setup the tree struct like this:
//      ┌─2
// 0──1─┼─3
//      └─4
func makeBlocks() []*CBlock {
	rawBlock0 := &types.Block{Header: &types.Header{Height: 100}}
	block0 := &CBlock{Block: rawBlock0}
	rawBlock1 := &types.Block{Header: &types.Header{Height: 101}}
	block1 := &CBlock{Block: rawBlock1}
	rawBlock2 := &types.Block{Header: &types.Header{Height: 102, Time: 123}}
	block2 := &CBlock{Block: rawBlock2}
	rawBlock3 := &types.Block{Header: &types.Header{Height: 102, Time: 234}}
	block3 := &CBlock{Block: rawBlock3}
	rawBlock4 := &types.Block{Header: &types.Header{Height: 102, Time: 345}}
	block4 := &CBlock{Block: rawBlock4}

	block1.BeChildOf(block0)
	block2.BeChildOf(block1)
	block3.BeChildOf(block1)
	block4.BeChildOf(block1)
	return []*CBlock{block0, block1, block2, block3, block4}
}

func TestCBlock_CollectToParent(t *testing.T) {
	b := makeBlocks()

	assert.Equal(t, []*CBlock{b[0]}, b[0].CollectToParent(nil))
	assert.Equal(t, []*CBlock{}, b[0].CollectToParent(b[0]))
	assert.Equal(t, []*CBlock{b[1]}, b[1].CollectToParent(b[0]))
	assert.Equal(t, []*CBlock{b[2], b[1]}, b[2].CollectToParent(b[0]))
	assert.Equal(t, []*CBlock{b[3], b[1]}, b[3].CollectToParent(b[0]))
}

func TestCBlock_Walk(t *testing.T) {
	b := makeBlocks()

	makeTestFn := func(expect []*CBlock) func(*CBlock) {
		index := 0
		return func(node *CBlock) {
			assert.Equal(t, expect[index], node, fmt.Sprintf("index=%d", index))
			index++
		}
	}
	b[3].Walk(makeTestFn([]*CBlock{}), nil)
	b[3].Walk(makeTestFn([]*CBlock{}), b[3])
	b[1].Walk(makeTestFn([]*CBlock{b[2], b[3], b[4]}), nil)
	b[1].Walk(makeTestFn([]*CBlock{b[2], b[3], b[4]}), b[1])
	b[1].Walk(makeTestFn([]*CBlock{b[3], b[4]}), b[2])
	b[0].Walk(makeTestFn([]*CBlock{b[1], b[2], b[3], b[4]}), nil)
	b[0].Walk(makeTestFn([]*CBlock{}), b[1])
	b[0].Walk(makeTestFn([]*CBlock{b[1], b[3], b[4]}), b[2])
}
