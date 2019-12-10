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
		account.Candidate.Profile[types.CandidateKeyIsCandidate] = "false"
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
	addr01 := common.HexToAddress("0x110")
	addr02 := common.HexToAddress("0x220")
	addr03 := common.HexToAddress("0x330")
	addr04 := common.HexToAddress("0x440")

	cblock1 := NewGenesisBlock(GetBlock1(), cacheChain.Beansdb)
	// 注册候选节点
	cblock1.AccountTrieDB.Put(NewAccountData(addr01, true), 1)
	cblock1.AccountTrieDB.Put(NewAccountData(addr02, true), 1)
	cblock1.AccountTrieDB.Put(NewAccountData(addr03, true), 1)
	cblock1.AccountTrieDB.Put(NewAccountData(addr04, true), 1)

	// 构造changeLog
	log01 := newVoteLog(addr01, big.NewInt(110))
	log02 := newVoteLog(addr02, big.NewInt(220))
	log03 := newVoteLog(addr03, big.NewInt(330))
	log04 := newVoteLog(addr04, big.NewInt(440))

	// 1. 初始化top中的值
	votelogs := types.ChangeLogSlice{log01, log02, log03, log04}
	cblock1.Ranking(votelogs)
	// 判断top中的值
	assert.Equal(t, 4, len(cblock1.Top.Top))
	assert.Equal(t, log04.Address, cblock1.Top.Top[0].GetAddress())
	assert.Equal(t, log04.NewVal, *cblock1.Top.Top[0].GetTotal())
	assert.Equal(t, log03.Address, cblock1.Top.Top[1].GetAddress())
	assert.Equal(t, log03.NewVal, *cblock1.Top.Top[1].GetTotal())
	assert.Equal(t, log02.Address, cblock1.Top.Top[2].GetAddress())
	assert.Equal(t, log02.NewVal, *cblock1.Top.Top[2].GetTotal())
	assert.Equal(t, log01.Address, cblock1.Top.Top[3].GetAddress())
	assert.Equal(t, log01.NewVal, *cblock1.Top.Top[3].GetTotal())

	// 2. 测试注销候选节点的情况
	cblock2 := &CBlock{
		Block:           GetBlock2(),
		AccountTrieDB:   cblock1.AccountTrieDB,
		CandidateTrieDB: cblock1.CandidateTrieDB,
		Top:             cblock1.Top,
		Parent:          cblock1,
	}
	unregisterAcc1 := NewAccountData(addr01, false)
	cblock2.AccountTrieDB.Put(unregisterAcc1, 2)

	log := newVoteLog(addr01, big.NewInt(0))
	cblock2.Ranking(types.ChangeLogSlice{log})
	// 验证
	assert.Equal(t, 3, len(cblock2.Top.Top))
	assert.Equal(t, log04.Address, cblock2.Top.Top[0].GetAddress())
	assert.Equal(t, log03.Address, cblock2.Top.Top[1].GetAddress())
	assert.Equal(t, log02.Address, cblock2.Top.Top[2].GetAddress())

	// 3. 测试top中的排序发生了变化
	log = newVoteLog(addr02, big.NewInt(9999))
	cblock2.Ranking(types.ChangeLogSlice{log})

	// 验证
	assert.Equal(t, addr02, cblock2.Top.Top[0].GetAddress())
	assert.Equal(t, addr04, cblock2.Top.Top[1].GetAddress())
	assert.Equal(t, addr03, cblock2.Top.Top[2].GetAddress())

	// 4. 测试candidate超过20之后top最多有20个的情况
	logs := make(types.ChangeLogSlice, 0, 20)
	for i := 0; i < 20; i++ {
		log := newVoteLog(common.HexToAddress(strconv.Itoa(i*1000+1)), big.NewInt(int64(i+10)))
		logs = append(logs, log)
	}
	cblock2.Ranking(logs)
	// 验证top中只有20个candidate
	assert.Equal(t, max_candidate_count, len(cblock2.Top.Top))
	// 此时票数最多的还是addr02有9999票
	assert.Equal(t, addr02, cblock2.Top.Top[0].GetAddress())
	assert.Equal(t, big.NewInt(9999), cblock2.Top.Top[0].GetTotal())

	// 5. 测试candidate票数变化之后影响的排名
	log = newVoteLog(addr04, big.NewInt(10000)) // 把addr04对应的票数修改为10000
	cblock2.Ranking(types.ChangeLogSlice{log})
	// 此时票数最多的为addr04
	assert.Equal(t, addr04, cblock2.Top.Top[0].GetAddress())
	assert.Equal(t, addr02, cblock2.Top.Top[1].GetAddress())

	// 6. 验证新增加一个candidate进top，此时最后一个top会被挤出去
	top20Total := cblock2.Top.Top[19].GetTotal()
	log = newVoteLog(common.HexToAddress("0x8887777"), new(big.Int).Add(top20Total, big.NewInt(1)))
	cblock2.Ranking(types.ChangeLogSlice{log})
	assert.Equal(t, common.HexToAddress("0x8887777"), cblock2.Top.Top[19].GetAddress())

	// 验证前20是否按照票数由大到小顺序排序的
	for i := 0; i < len(cblock2.Top.Top)-1; i++ {
		for j := i + 1; j < len(cblock2.Top.Top); j++ {
			assert.True(t, cblock2.Top.Top[i].Total.Cmp(cblock2.Top.Top[j].Total) >= 0)
		}
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
