package store

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/stretchr/testify/assert"
	"math/big"
	"strconv"
	"testing"
	"time"
)

func TestChainDatabase_Test(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())
	defer cacheChain.Close()

	result160, _ := cacheChain.GetBlockByHeight(160)
	result161, _ := cacheChain.GetBlockByHeight(161)
	assert.Equal(t, result160, result161)
}

func TestCacheChain_SetBlock(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())
	defer cacheChain.Close()

	result, err := cacheChain.GetBlockByHeight(0)
	assert.Equal(t, err, ErrBlockNotExist)

	// set genesis
	block0 := GetBlock0()
	err = cacheChain.SetBlock(block0.Hash(), block0)
	assert.NoError(t, err)
	unconfirmCB0 := cacheChain.UnConfirmBlocks[block0.Hash()]
	assert.Equal(t, cacheChain.LastConfirm, unconfirmCB0.Parent)
	assert.Len(t, cacheChain.LastConfirm.Children, 1)
	assert.Equal(t, unconfirmCB0, cacheChain.LastConfirm.Children[0])

	// set genesis stable
	blocks, err := cacheChain.SetStableBlock(block0.Hash())
	assert.NoError(t, err)
	assert.Len(t, blocks, 0)
	assert.Len(t, cacheChain.UnConfirmBlocks, 0)
	assert.Equal(t, unconfirmCB0, cacheChain.LastConfirm)

	result, err = cacheChain.GetBlockByHash(block0.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block0.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHeight(0)
	assert.NoError(t, err)
	assert.Equal(t, block0.ParentHash(), result.ParentHash())

	// set 3 blocks
	block1 := GetBlock1()
	err = cacheChain.SetBlock(block1.Hash(), block1)
	assert.NoError(t, err)

	unconfirmCB1 := cacheChain.UnConfirmBlocks[block1.Hash()]
	assert.Equal(t, unconfirmCB0, unconfirmCB1.Parent)
	assert.Len(t, unconfirmCB0.Children, 1)
	assert.Equal(t, unconfirmCB1, unconfirmCB0.Children[0])

	result, err = cacheChain.GetBlockByHash(block1.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block1.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHeight(block1.Height())
	assert.Equal(t, err, ErrBlockNotExist)

	block2 := GetBlock2()
	err = cacheChain.SetBlock(block2.Hash(), block2)
	assert.NoError(t, err)

	result, err = cacheChain.GetBlockByHash(block2.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block2.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHeight(block2.Height())
	assert.Equal(t, err, ErrBlockNotExist)

	block3 := GetBlock3()
	err = cacheChain.SetBlock(block3.Hash(), block3)
	assert.NoError(t, err)

	// set 2 blocks stable
	oldStable := cacheChain.LastConfirm
	blocks, err = cacheChain.SetStableBlock(block2.Hash())
	assert.NoError(t, err)
	assert.Empty(t, blocks)
	assert.Empty(t, oldStable.Children)
	assert.Empty(t, cacheChain.LastConfirm.Parent)
	assert.NotEqual(t, oldStable, cacheChain.LastConfirm)
	time.Sleep(1 * time.Second)

	result, err = cacheChain.GetBlockByHash(block2.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block2.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHeight(block2.Height())
	assert.NoError(t, err)
	assert.Equal(t, block2.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHash(block3.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block3.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHeight(block3.Height())
	assert.Equal(t, err, ErrBlockNotExist)

}

func TestCacheChain_SetStableBlock(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())
	defer cacheChain.Close()

	// set genesis
	block0 := GetBlock0()
	err := cacheChain.SetBlock(block0.Hash(), block0)
	assert.NoError(t, err)
	blocks, err := cacheChain.SetStableBlock(block0.Hash())
	assert.NoError(t, err)
	assert.Len(t, blocks, 0)

	// set fork blocks
	//         ┌─1──2
	// genesis─┤
	//         └─1OnFork──2OnFork
	block1 := GetBlock1()
	err = cacheChain.SetBlock(block1.Hash(), block1)
	assert.NoError(t, err)
	block1OnFork := GetBlock1()
	block1OnFork.Header.Time = 12345
	err = cacheChain.SetBlock(block1OnFork.Hash(), block1OnFork)
	assert.NoError(t, err)
	block2 := GetBlock2()
	err = cacheChain.SetBlock(block2.Hash(), block2)
	assert.NoError(t, err)
	block2OnFork := GetBlock2()
	block2OnFork.Header.ParentHash = block1OnFork.Hash()
	err = cacheChain.SetBlock(block2OnFork.Hash(), block2OnFork)
	assert.NoError(t, err)

	// set stable
	blocks, err = cacheChain.SetStableBlock(block2OnFork.Hash())
	assert.NoError(t, err)
	assert.Len(t, blocks, 2)
	assert.Equal(t, block2OnFork.Hash(), cacheChain.LastConfirm.Block.Hash())
	assert.Len(t, cacheChain.UnConfirmBlocks, 0)
}

func TestCacheChain_SetBlockError(t *testing.T) {
	// ERROR #1
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())
	// defer cacheChain.Close()

	block1 := GetBlock1()

	err := cacheChain.SetBlock(block1.Hash(), block1)
	assert.Equal(t, err, ErrArgInvalid)

	block0 := GetBlock0()
	cacheChain.SetBlock(block0.Hash(), block0)
	cacheChain.SetStableBlock(block0.Hash())

	err = cacheChain.SetBlock(block0.Hash(), block0)
	assert.Equal(t, ErrExist, err)

	block1.Header.Height = 0
	err = cacheChain.SetBlock(block1.Hash(), block1)
	assert.Equal(t, err, ErrArgInvalid)

	block1.Header.Height = 1
	cacheChain.SetBlock(block1.Hash(), block1)
	cacheChain.SetStableBlock(block1.Hash())

	block2 := GetBlock2()
	block2.Header.Height = 1
	err = cacheChain.SetBlock(block2.Hash(), block2)
	assert.Equal(t, err, ErrArgInvalid)

	cacheChain.Close()

	// ERROR #2
	cacheChain = NewChainDataBase(GetStorePath())
	block0 = GetBlock0()
	block1 = GetBlock1()
	block2 = GetBlock2()
	// block3 := GetBlock3()

	cacheChain.SetBlock(block0.Hash(), block0)
	_, err = cacheChain.SetStableBlock(block0.Hash())
	assert.Equal(t, err, ErrArgInvalid)

	hash := block1.Hash()
	block1.Header.Height = 2
	err = cacheChain.SetBlock(hash, block1)
	assert.Equal(t, err, ErrExist)

	cacheChain.SetBlock(block2.Hash(), block2)

	block1 = GetBlock1()
	cacheChain.SetBlock(block1.Hash(), block1)
	hash = block2.Hash()
	block2.Header.Height = 3
	err = cacheChain.SetBlock(block2.Hash(), block2)
	assert.Equal(t, err, ErrArgInvalid)

	cacheChain.Close()
}

func TestCacheChain_IsExistByHash(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())

	isExist, err := cacheChain.IsExistByHash(common.Hash{})
	assert.NoError(t, err)
	assert.Equal(t, false, isExist)

	block := GetBlock0()
	err = cacheChain.SetBlock(block.Hash(), block)
	assert.NoError(t, err)

	isExist, err = cacheChain.IsExistByHash(block.Hash())
	assert.NoError(t, err)
	assert.Equal(t, true, isExist)

	_, err = cacheChain.SetStableBlock(block.Hash())
	assert.NoError(t, err)

	isExist, err = cacheChain.IsExistByHash(block.Hash())
	assert.NoError(t, err)
	assert.Equal(t, true, isExist)

	cacheChain.Close()
	cacheChain = NewChainDataBase(GetStorePath())
	isExist, err = cacheChain.IsExistByHash(block.Hash())
	assert.NoError(t, err)
	assert.Equal(t, true, isExist)
	cacheChain.Close()
}

// func TestCacheChain_WriteChainBatch(t *testing.T) {
// 	for index := 0; index < 200; index++{
// 		TestCacheChain_WriteChain(t)
// 	}
// }

func TestCacheChain_WriteChain(t *testing.T) {
	block0 := GetBlock0()
	block1 := GetBlock1()
	block2 := GetBlock2()
	block3 := GetBlock3()
	block4 := GetBlock4()

	ClearData()
	log.Errorf("STEP.1")
	cacheChain := NewChainDataBase(GetStorePath())
	cacheChain.SetBlock(block0.Hash(), block0)
	_, err := cacheChain.SetStableBlock(block0.Hash())
	assert.NoError(t, err)

	log.Errorf("STEP.2")
	// 1, 2#, 3
	cacheChain.SetBlock(block1.Hash(), block1)
	cacheChain.SetBlock(block2.Hash(), block2)
	cacheChain.SetBlock(block3.Hash(), block3)
	_, err = cacheChain.SetStableBlock(block2.Hash())
	assert.NoError(t, err)
	result, err := cacheChain.GetBlockByHeight(1)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), block1.Hash())

	result, err = cacheChain.GetBlockByHeight(2)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), block2.Hash())

	result, err = cacheChain.GetBlockByHeight(3)
	assert.Equal(t, ErrBlockNotExist, err)
	cacheChain.Close()

	log.Errorf("STEP.3")
	// from db
	cacheChain = NewChainDataBase(GetStorePath())

	result, err = cacheChain.GetBlockByHeight(1)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), block1.Hash())

	result, err = cacheChain.GetBlockByHeight(2)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), block2.Hash())

	result, err = cacheChain.GetBlockByHeight(3)
	assert.Equal(t, ErrBlockNotExist, err)
	cacheChain.Close()

	log.Errorf("STEP.4")
	cacheChain = NewChainDataBase(GetStorePath())

	// 1, 2, 3#
	err = cacheChain.SetBlock(block1.Hash(), block1)
	assert.Equal(t, err, ErrExist)
	err = cacheChain.SetBlock(block2.Hash(), block2)
	assert.Equal(t, err, ErrExist)
	err = cacheChain.SetBlock(block3.Hash(), block3)
	assert.NoError(t, err)

	_, err = cacheChain.SetStableBlock(block3.Hash())
	assert.NoError(t, err)
	result, err = cacheChain.GetBlockByHeight(1)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), block1.Hash())

	result, err = cacheChain.GetBlockByHeight(2)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), block2.Hash())

	result, err = cacheChain.GetBlockByHeight(3)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), block3.Hash())
	cacheChain.Close()

	log.Errorf("STEP.5")
	cacheChain = NewChainDataBase(GetStorePath())
	result, err = cacheChain.GetBlockByHeight(1)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), block1.Hash())

	result, err = cacheChain.GetBlockByHeight(2)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), block2.Hash())

	result, err = cacheChain.GetBlockByHeight(3)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), block3.Hash())

	// error block
	_, err = cacheChain.SetStableBlock(block1.Hash())
	assert.Equal(t, err, ErrArgInvalid)

	_, err = cacheChain.SetStableBlock(block3.Hash())
	assert.Equal(t, err, ErrArgInvalid)

	_, err = cacheChain.SetStableBlock(block4.Hash())
	assert.Equal(t, err, ErrArgInvalid)

	cacheChain.Close()
}

func TestChainDatabase_SetContractCode(t *testing.T) {
	ClearData()
	database := NewChainDataBase(GetStorePath())

	code := types.Code("this  is code")
	hash := common.HexToHash("this is code")

	err := database.SetContractCode(hash, code)
	assert.NoError(t, err)

	result, err := database.GetContractCode(hash)
	assert.NoError(t, err)
	assert.Equal(t, code, result)
	database.Close()

	database = NewChainDataBase(GetStorePath())
	result, err = database.GetContractCode(hash)
	assert.NoError(t, err)
	assert.Equal(t, code, result)
	database.Close()
}

func TestCacheChain_LastConfirm(t *testing.T) {

	block0 := GetBlock0()
	block1 := GetBlock1()
	block2 := GetBlock2()
	block3 := GetBlock3()
	block4 := GetBlock4()
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())

	lastConfirmBlock := cacheChain.LastConfirm.Block
	assert.Nil(t, lastConfirmBlock)

	cacheChain.SetBlock(block0.Hash(), block0)
	cacheChain.SetStableBlock(block0.Hash())
	lastConfirmBlock = cacheChain.LastConfirm.Block
	assert.Equal(t, block0.Hash(), lastConfirmBlock.Hash())

	cacheChain.SetBlock(block1.Hash(), block1)
	cacheChain.SetBlock(block2.Hash(), block2)
	cacheChain.SetBlock(block3.Hash(), block3)
	cacheChain.SetStableBlock(block2.Hash())
	lastConfirmBlock = cacheChain.LastConfirm.Block
	assert.Equal(t, block2.Hash(), lastConfirmBlock.Hash())
	cacheChain.Close()

	cacheChain = NewChainDataBase(GetStorePath())
	cacheChain.SetBlock(block1.Hash(), block1)
	cacheChain.SetBlock(block2.Hash(), block2)
	cacheChain.SetBlock(block3.Hash(), block3)
	cacheChain.SetBlock(block4.Hash(), block4)
	cacheChain.SetStableBlock(block4.Hash())

	lastConfirmBlock = cacheChain.LastConfirm.Block
	assert.Equal(t, block4.Hash(), lastConfirmBlock.Hash())
	cacheChain.Close()

	cacheChain = NewChainDataBase(GetStorePath())
	lastConfirmBlock = cacheChain.LastConfirm.Block
	assert.Equal(t, block4.Hash(), lastConfirmBlock.Hash())
	cacheChain.Close()
}

func TestCacheChain_SetConfirm1(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())
	defer cacheChain.Close()

	signs, err := CreateSign(16)
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)
	_, err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)
	newBlock, err := cacheChain.SetConfirms(parentBlock.Hash(), signs)
	assert.NoError(t, err)
	assert.Equal(t, 16, len(newBlock.Confirms))

	result, err := cacheChain.GetConfirms(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 16, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])

	result, err = cacheChain.GetConfirms(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 16, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])
}

func TestCacheChain_SetConfirm2(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())
	defer cacheChain.Close()

	signs, err := CreateSign(16)
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	_, err = cacheChain.SetConfirms(parentBlock.Hash(), []types.SignData{signs[0]})
	assert.Equal(t, err, ErrBlockNotExist)

	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)
	newBlock, err := cacheChain.SetConfirms(parentBlock.Hash(), []types.SignData{signs[0]})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(newBlock.Confirms))
	newBlock, err = cacheChain.SetConfirms(parentBlock.Hash(), []types.SignData{signs[1]})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(newBlock.Confirms))
	newBlock, err = cacheChain.SetConfirms(parentBlock.Hash(), []types.SignData{signs[2]})
	assert.NoError(t, err)
	assert.Equal(t, 3, len(newBlock.Confirms))
	newBlock, err = cacheChain.SetConfirms(parentBlock.Hash(), []types.SignData{signs[3]})
	assert.NoError(t, err)
	assert.Equal(t, 4, len(newBlock.Confirms))

	result, err := cacheChain.GetConfirms(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 4, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])
	assert.Equal(t, signs[3], result[3])
	assert.Equal(t, signs[3], result[3])
	assert.Equal(t, signs[3], result[3])

	_, err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)
	result, err = cacheChain.GetConfirms(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 4, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])

	// cacheChain = NewChainDataBase(GetStorePath())

	result, err = cacheChain.GetConfirms(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 4, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])
}

func TestCacheChain_AppendConfirm(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())
	defer cacheChain.Close()

	signs, err := CreateSign(16)
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)

	_, err = cacheChain.SetConfirms(parentBlock.Hash(), []types.SignData{signs[0]})
	assert.NoError(t, err)

	_, err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)

	block, err := cacheChain.GetBlockByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 1, len(block.Confirms))
	assert.Equal(t, signs[0], block.Confirms[0])

	_, err = cacheChain.SetConfirms(parentBlock.Hash(), []types.SignData{signs[1]})
	assert.NoError(t, err)

	block, err = cacheChain.GetBlockByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(block.Confirms))
	assert.Equal(t, signs[0], block.Confirms[0])
	assert.Equal(t, signs[1], block.Confirms[1])

	_, err = cacheChain.SetConfirms(parentBlock.Hash(), signs)
	assert.NoError(t, err)

	block, err = cacheChain.GetBlockByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 16, len(block.Confirms))
	assert.Equal(t, signs[0], block.Confirms[0])
	assert.Equal(t, signs[1], block.Confirms[1])
	assert.Equal(t, signs[14], block.Confirms[14])
	assert.Equal(t, signs[15], block.Confirms[15])
}

func TestChainDatabase_Commit(t *testing.T) {
	ClearData()
	chain := NewChainDataBase(GetStorePath())
	defer chain.Close()

	// rand.Seed(time.Now().Unix())
	blocks := NewBlockBatch(10)
	chain.SetBlock(blocks[0].Hash(), blocks[0])
	_, err := chain.SetStableBlock(blocks[0].Hash())
	assert.NoError(t, err)
	for index := 1; index < 10; index++ {
		chain.SetBlock(blocks[index].Hash(), blocks[index])
		_, err = chain.SetStableBlock(blocks[index].Hash())
		assert.NoError(t, err)
		if index > 10 {
			val, err := chain.GetBlockByHeight(uint32(index - 7))
			assert.NoError(t, err)
			if val == nil {
				panic("val :" + strconv.Itoa(index))
			}
		}
		log.Errorf("index:" + strconv.Itoa(index))
	}
}

func TestChainDatabase_CandidatesRanking(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath())

	block0 := GetBlock0()
	cacheChain.SetBlock(block0.Hash(), block0)
	cacheChain.SetStableBlock(block0.Hash())

	block1 := GetBlock1()
	cacheChain.SetBlock(block1.Hash(), block1)

	count := 100
	candidates := NewAccountDataBatch(100)
	actDatabase, _ := cacheChain.GetActDatabase(block1.Hash())
	voteLogs := make(types.ChangeLogSlice, 0, 100)
	for index := 0; index < count; index++ {
		actDatabase.Put(candidates[index], 1)
		log := newVoteLog(candidates[index].Address, big.NewInt(int64(index)))
		voteLogs = append(voteLogs, log)
	}
	cacheChain.CandidatesRanking(block1.Hash(), voteLogs)
	cacheChain.SetStableBlock(block1.Hash())

	top := cacheChain.GetCandidatesTop(block1.Hash())
	assert.Equal(t, max_candidate_count, len(top))

	page, total, err := cacheChain.GetCandidatesPage(0, 10)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(page))
	assert.Equal(t, uint32(count), total)

	page, total, err = cacheChain.GetCandidatesPage(10, 100)
	assert.NoError(t, err)
	assert.Equal(t, 90, len(page))
	assert.Equal(t, uint32(count), total)

	page, total, err = cacheChain.GetCandidatesPage(100, 10)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(page))
	assert.Equal(t, uint32(count), total)
	cacheChain.Close()

	cacheChain = NewChainDataBase(GetStorePath())
	top = cacheChain.GetCandidatesTop(block1.Hash())
	assert.Equal(t, max_candidate_count, len(top))

	last, err := cacheChain.LoadLatestBlock()
	assert.NoError(t, err)
	assert.Equal(t, block1.Hash(), last.Hash())

	account, err := cacheChain.GetAccount(common.HexToAddress(fmt.Sprintf("%x", 99)))
	assert.NoError(t, err)
	assert.Equal(t, account.Address, common.HexToAddress(fmt.Sprintf("%x", 99)))

	all, err := cacheChain.Context.GetCandidates()
	assert.NoError(t, err)
	assert.Equal(t, count, len(all))

	page, total, err = cacheChain.GetCandidatesPage(0, 10)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(page))
	assert.Equal(t, uint32(count), total)

	page, total, err = cacheChain.GetCandidatesPage(10, 100)
	assert.NoError(t, err)
	assert.Equal(t, 90, len(page))
	assert.Equal(t, uint32(count), total)

	page, total, err = cacheChain.GetCandidatesPage(100, 10)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(page))
	assert.Equal(t, uint32(count), total)
	cacheChain.Close()
}
