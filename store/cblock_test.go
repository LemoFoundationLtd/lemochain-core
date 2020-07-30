package store

import (
	"bytes"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func NewAccountData(addressNumber int) *types.AccountData {
	address := common.HexToAddress(fmt.Sprintf("%x", addressNumber))
	account := &types.AccountData{
		Address:       address,
		Balance:       new(big.Int).SetInt64(100),
		NewestRecords: make(map[types.ChangeLogType]types.VersionRecord),
		Candidate: types.Candidate{
			Profile: make(types.Profile),
			Votes:   new(big.Int),
		},
	}

	return account
}

func NewCandidateAccountData(addressNumber int, val int64) *types.AccountData {
	account := NewAccountData(addressNumber)
	account.Candidate.Profile[types.CandidateKeyIsCandidate] = types.IsCandidateNode
	account.Candidate.Votes = new(big.Int).SetInt64(val)
	return account
}

func NewAccountDataBatch(count int) []*types.AccountData {
	result := make([]*types.AccountData, count)
	for index := 0; index < count; index++ {
		result[index] = NewCandidateAccountData(index, int64(index*2))
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
	accounts[0] = NewCandidateAccountData(0, 500)
	accounts[1] = NewCandidateAccountData(1, 400)
	accounts[2] = NewCandidateAccountData(2, 500)
	accounts[3] = NewCandidateAccountData(3, 200)
	accounts[4] = NewCandidateAccountData(4, 600)

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

func newVoteLog(addr common.Address, vote *big.Int) *types.ChangeLog {
	return &types.ChangeLog{
		LogType: 18,
		Address: addr,
		NewVal:  *vote,
	}
}

func TestCBlock_Ranking(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())
	defer cacheChain.Close()

	// 无候选节点
	cblock1 := NewGenesisBlock(GetBlock1(), cacheChain.Beansdb)
	cblock1.Ranking(types.ChangeLogSlice{})
	assert.Equal(t, 0, len(cblock1.Top.Top))

	// 3个候选节点
	account1 := NewCandidateAccountData(0x110, 100)
	account2 := NewCandidateAccountData(0x220, 200)
	account3 := NewCandidateAccountData(0x330, 300)
	log1 := newVoteLog(account1.Address, account1.Candidate.Votes)
	log2 := newVoteLog(account2.Address, account2.Candidate.Votes)
	log3 := newVoteLog(account3.Address, account3.Candidate.Votes)
	cblock2 := NewNormalBlock(GetBlock2(), cblock1.AccountTrieDB, cblock1.CandidateTrieDB, cblock1.Top)
	cblock2.BeChildOf(cblock1)
	cblock2.AccountTrieDB.Put(account1, cblock2.Block.Height())
	cblock2.AccountTrieDB.Put(account2, cblock2.Block.Height())
	cblock2.AccountTrieDB.Put(account3, cblock2.Block.Height())
	cblock2.Ranking(types.ChangeLogSlice{log1, log2, log3})
	assert.Equal(t, 3, len(cblock2.Top.Top))
	assert.Equal(t, log3.Address, cblock2.Top.Top[0].GetAddress())
	assert.Equal(t, log3.NewVal, *cblock2.Top.Top[0].GetTotal())
	assert.Equal(t, log2.Address, cblock2.Top.Top[1].GetAddress())
	assert.Equal(t, log2.NewVal, *cblock2.Top.Top[1].GetTotal())
	assert.Equal(t, log1.Address, cblock2.Top.Top[2].GetAddress())
	assert.Equal(t, log1.NewVal, *cblock2.Top.Top[2].GetTotal())

	// 注销候选节点
	log := newVoteLog(account2.Address, big.NewInt(0))
	cblock3 := NewNormalBlock(GetBlock3(), cblock2.AccountTrieDB, cblock2.CandidateTrieDB, cblock2.Top)
	cblock3.BeChildOf(cblock2)
	account2.Candidate.Profile[types.CandidateKeyIsCandidate] = types.NotCandidateNode
	cblock3.AccountTrieDB.Put(account2, cblock3.Block.Height())
	cblock3.Ranking(types.ChangeLogSlice{log})
	assert.Equal(t, 2, len(cblock3.Top.Top))
	assert.Equal(t, account3.Address, cblock3.Top.Top[0].GetAddress())
	assert.Equal(t, account1.Address, cblock3.Top.Top[1].GetAddress())

	// 排名发生了变化
	log = newVoteLog(account1.Address, big.NewInt(9999))
	cblock3.Ranking(types.ChangeLogSlice{log})
	cblock3.AccountTrieDB.Put(account1, cblock3.Block.Height())
	assert.Equal(t, 2, len(cblock3.Top.Top))
	assert.Equal(t, account1.Address, cblock3.Top.Top[0].GetAddress())
	assert.Equal(t, account3.Address, cblock3.Top.Top[1].GetAddress())
}

func TestCBlock_Ranking_full(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())
	defer cacheChain.Close()

	// candidate超过20之后top最多有20个
	cblock1 := NewGenesisBlock(GetBlock1(), cacheChain.Beansdb)
	accounts := NewAccountDataBatch(max_candidate_count + 10)
	logs := make(types.ChangeLogSlice, 0, len(accounts))
	for i := 0; i < len(accounts); i++ {
		log := newVoteLog(accounts[i].Address, big.NewInt(int64(i)))
		logs = append(logs, log)
		cblock1.AccountTrieDB.Put(accounts[i], cblock1.Block.Height())
	}
	cblock1.Ranking(logs)
	// top中只有20个candidate
	assert.Equal(t, max_candidate_count, len(cblock1.Top.Top))
	assert.Equal(t, accounts[max_candidate_count+9].Address, cblock1.Top.Top[0].GetAddress())

	// candidate票数变化之后影响的排名
	cblock2 := NewNormalBlock(GetBlock2(), cblock1.AccountTrieDB, cblock1.CandidateTrieDB, cblock1.Top)
	cblock2.BeChildOf(cblock1)
	changedAccount := accounts[max_candidate_count+5]
	changedAccount.Candidate.Votes = big.NewInt(10000)
	cblock2.AccountTrieDB.Put(changedAccount, cblock2.Block.Height())
	log := newVoteLog(changedAccount.Address, big.NewInt(10000))
	cblock2.Ranking(types.ChangeLogSlice{log})
	// 此时票数最多的为changedAccount
	assert.Equal(t, changedAccount.Address, cblock2.Top.Top[0].GetAddress())
	assert.Equal(t, big.NewInt(10000), cblock2.Top.Top[0].GetTotal())

	// 注销候选节点
	cblock3 := NewNormalBlock(GetBlock3(), cblock2.AccountTrieDB, cblock2.CandidateTrieDB, cblock2.Top)
	cblock3.BeChildOf(cblock2)
	log = newVoteLog(changedAccount.Address, big.NewInt(0))
	changedAccount.Candidate.Profile[types.CandidateKeyIsCandidate] = types.NotCandidateNode
	cblock3.AccountTrieDB.Put(changedAccount, cblock3.Block.Height())
	cblock3.Ranking(types.ChangeLogSlice{log})
	assert.Equal(t, max_candidate_count, len(cblock3.Top.Top))
	assert.NotEqual(t, changedAccount.Address, cblock3.Top.Top[0].GetAddress())

	// 新增加一个candidate进top，此时最后一个top会被挤出去
	cblock4 := NewNormalBlock(GetBlock4(), cblock3.AccountTrieDB, cblock3.CandidateTrieDB, cblock3.Top)
	cblock4.BeChildOf(cblock3)
	lastOneAddr := cblock4.Top.Top[max_candidate_count-1].GetAddress()
	lastOneVote := cblock4.Top.Top[max_candidate_count-1].GetTotal()
	account6 := NewCandidateAccountData(0x660, lastOneVote.Int64()+1)
	cblock4.AccountTrieDB.Put(account6, cblock4.Block.Height())
	log = newVoteLog(account6.Address, account6.Candidate.Votes)
	cblock4.Ranking(types.ChangeLogSlice{log})
	assert.Equal(t, account6.Address, cblock4.Top.Top[max_candidate_count-1].GetAddress())
	assert.Equal(t, big.NewInt(lastOneVote.Int64()+1), cblock4.Top.Top[max_candidate_count-1].GetTotal())

	// 一个candidate掉出top，此时lastOneAddr会回到top
	block5 := CreateBlock(common.HexToHash("0x55555"), cblock4.Block.Hash(), 4)
	cblock5 := NewNormalBlock(block5, cblock4.AccountTrieDB, cblock4.CandidateTrieDB, cblock4.Top)
	cblock5.BeChildOf(cblock4)
	loserAccount := NewCandidateAccountData(max_candidate_count+2, 1)
	cblock5.AccountTrieDB.Put(loserAccount, cblock5.Block.Height())
	log = newVoteLog(loserAccount.Address, big.NewInt(1))
	cblock5.Ranking(types.ChangeLogSlice{log})
	assert.Equal(t, lastOneAddr, cblock5.Top.Top[max_candidate_count-1].GetAddress())
	assert.Equal(t, lastOneVote, cblock5.Top.Top[max_candidate_count-1].GetTotal())

	// 验证前20是否按照票数由大到小顺序排序的
	for i := 1; i < len(cblock5.Top.Top)-1; i++ {
		assert.True(t, cblock5.Top.Top[i-1].Total.Cmp(cblock5.Top.Top[i].Total) >= 0)
	}
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
