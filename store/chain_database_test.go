package store

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

func TestChainDatabase_Test(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	defer cacheChain.Close()

	result160, _ := cacheChain.GetBlockByHeight(160)
	result161, _ := cacheChain.GetBlockByHeight(161)
	assert.Equal(t, result160, result161)
}

func TestCacheChain_SetBlock(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	defer cacheChain.Close()

	result, err := cacheChain.GetBlockByHeight(0)
	assert.Equal(t, err, ErrNotExist)

	// set genesis
	block0 := GetBlock0()
	err = cacheChain.SetBlock(block0.Hash(), block0)
	assert.NoError(t, err)
	unconfirmCB0 := cacheChain.UnConfirmBlocks[block0.Hash()]
	assert.Equal(t, cacheChain.LastConfirm, unconfirmCB0.Parent)
	assert.Len(t, cacheChain.LastConfirm.Children, 1)
	assert.Equal(t, unconfirmCB0, cacheChain.LastConfirm.Children[0])

	err = cacheChain.SetStableBlock(block0.Hash())
	assert.NoError(t, err)
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
	assert.Equal(t, err, ErrNotExist)

	block2 := GetBlock2()
	err = cacheChain.SetBlock(block2.Hash(), block2)
	assert.NoError(t, err)

	result, err = cacheChain.GetBlockByHash(block2.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block2.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHeight(block2.Height())
	assert.Equal(t, err, ErrNotExist)

	block3 := GetBlock3()
	err = cacheChain.SetBlock(block3.Hash(), block3)
	assert.NoError(t, err)

	// set 2 blocks stable
	err = cacheChain.SetStableBlock(block2.Hash())
	assert.NoError(t, err)

	time.Sleep(5000)

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
	assert.Equal(t, err, ErrNotExist)

}

func TestCacheChain_SetBlockError(t *testing.T) {
	// ERROR #1
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
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
	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	block0 = GetBlock0()
	block1 = GetBlock1()
	block2 = GetBlock2()
	// block3 := GetBlock3()

	cacheChain.SetBlock(block0.Hash(), block0)
	err = cacheChain.SetStableBlock(block0.Hash())
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
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	isExist, err := cacheChain.IsExistByHash(common.Hash{})
	assert.NoError(t, err)
	assert.Equal(t, false, isExist)

	block := GetBlock0()
	err = cacheChain.SetBlock(block.Hash(), block)
	assert.NoError(t, err)

	isExist, err = cacheChain.IsExistByHash(block.Hash())
	assert.NoError(t, err)
	assert.Equal(t, true, isExist)

	err = cacheChain.SetStableBlock(block.Hash())
	assert.NoError(t, err)

	isExist, err = cacheChain.IsExistByHash(block.Hash())
	assert.NoError(t, err)
	assert.Equal(t, true, isExist)

	cacheChain.Close()
	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
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
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	cacheChain.SetBlock(block0.Hash(), block0)
	err := cacheChain.SetStableBlock(block0.Hash())
	assert.NoError(t, err)

	log.Errorf("STEP.2")
	// 1, 2#, 3
	cacheChain.SetBlock(block1.Hash(), block1)
	cacheChain.SetBlock(block2.Hash(), block2)
	cacheChain.SetBlock(block3.Hash(), block3)
	err = cacheChain.SetStableBlock(block2.Hash())
	assert.NoError(t, err)
	result, err := cacheChain.GetBlockByHeight(1)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), block1.Hash())

	result, err = cacheChain.GetBlockByHeight(2)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), block2.Hash())

	result, err = cacheChain.GetBlockByHeight(3)
	assert.Equal(t, ErrNotExist, err)
	cacheChain.Close()

	log.Errorf("STEP.3")
	// from db
	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	result, err = cacheChain.GetBlockByHeight(1)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), block1.Hash())

	result, err = cacheChain.GetBlockByHeight(2)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), block2.Hash())

	result, err = cacheChain.GetBlockByHeight(3)
	assert.Equal(t, ErrNotExist, err)
	cacheChain.Close()

	log.Errorf("STEP.4")
	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	// 1, 2, 3#
	err = cacheChain.SetBlock(block1.Hash(), block1)
	assert.Equal(t, err, ErrExist)
	err = cacheChain.SetBlock(block2.Hash(), block2)
	assert.Equal(t, err, ErrExist)
	err = cacheChain.SetBlock(block3.Hash(), block3)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(block3.Hash())
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
	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
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
	err = cacheChain.SetStableBlock(block1.Hash())
	assert.Equal(t, err, ErrArgInvalid)

	err = cacheChain.SetStableBlock(block3.Hash())
	assert.Equal(t, err, ErrArgInvalid)

	err = cacheChain.SetStableBlock(block4.Hash())
	assert.Equal(t, err, ErrArgInvalid)

	cacheChain.Close()
}

func TestChainDatabase_SetContractCode(t *testing.T) {
	ClearData()
	database := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	code := types.Code("this  is code")
	hash := common.HexToHash("this is code")

	err := database.SetContractCode(hash, code)
	assert.NoError(t, err)

	result, err := database.GetContractCode(hash)
	assert.NoError(t, err)
	assert.Equal(t, code, result)
	database.Close()

	database = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
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
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

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

	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	cacheChain.SetBlock(block1.Hash(), block1)
	cacheChain.SetBlock(block2.Hash(), block2)
	cacheChain.SetBlock(block3.Hash(), block3)
	cacheChain.SetBlock(block4.Hash(), block4)
	cacheChain.SetStableBlock(block4.Hash())

	lastConfirmBlock = cacheChain.LastConfirm.Block
	assert.Equal(t, block4.Hash(), lastConfirmBlock.Hash())
	cacheChain.Close()

	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	lastConfirmBlock = cacheChain.LastConfirm.Block
	assert.Equal(t, block4.Hash(), lastConfirmBlock.Hash())
	cacheChain.Close()
}

func TestCacheChain_SetConfirm1(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	defer cacheChain.Close()

	signs, err := CreateSign(16)
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)
	err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)
	err = cacheChain.SetConfirms(parentBlock.Hash(), signs)
	log.Errorf("set confirms end!")
	assert.NoError(t, err)

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
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	defer cacheChain.Close()

	signs, err := CreateSign(16)
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	err = cacheChain.SetConfirm(parentBlock.Hash(), signs[0])
	assert.Equal(t, err, ErrNotExist)

	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)
	err = cacheChain.SetConfirm(parentBlock.Hash(), signs[0])
	assert.NoError(t, err)
	err = cacheChain.SetConfirm(parentBlock.Hash(), signs[1])
	assert.NoError(t, err)
	err = cacheChain.SetConfirm(parentBlock.Hash(), signs[2])
	assert.NoError(t, err)
	err = cacheChain.SetConfirm(parentBlock.Hash(), signs[3])
	assert.NoError(t, err)

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

	err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)
	result, err = cacheChain.GetConfirms(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 4, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])

	// cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

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
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	defer cacheChain.Close()

	signs, err := CreateSign(16)
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)

	err = cacheChain.SetConfirm(parentBlock.Hash(), signs[0])
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)

	block, err := cacheChain.GetBlockByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, len(block.Confirms), 1)
	assert.Equal(t, block.Confirms[0], signs[0])

	err = cacheChain.SetConfirm(parentBlock.Hash(), signs[1])
	assert.NoError(t, err)

	block, err = cacheChain.GetBlockByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, len(block.Confirms), 2)
	assert.Equal(t, block.Confirms[0], signs[0])
	assert.Equal(t, block.Confirms[1], signs[1])

	err = cacheChain.SetConfirms(parentBlock.Hash(), signs)
	assert.NoError(t, err)

	block, err = cacheChain.GetBlockByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, len(block.Confirms), 16)
	assert.Equal(t, block.Confirms[0], signs[0])
	assert.Equal(t, block.Confirms[1], signs[1])
	assert.Equal(t, block.Confirms[14], signs[14])
	assert.Equal(t, block.Confirms[15], signs[15])
}

func TestChainDatabase_Commit(t *testing.T) {
	ClearData()
	chain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	defer chain.Close()

	// rand.Seed(time.Now().Unix())
	blocks := NewBlockBatch(10)
	chain.SetBlock(blocks[0].Hash(), blocks[0])
	err := chain.SetStableBlock(blocks[0].Hash())
	assert.NoError(t, err)
	for index := 1; index < 10; index++ {
		chain.SetBlock(blocks[index].Hash(), blocks[index])
		err = chain.SetStableBlock(blocks[index].Hash())
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

func TestChainDatabase_GetBlock(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	block0 := GetBlock0()
	cacheChain.SetBlock(block0.Hash(), block0)
	cacheChain.SetStableBlock(block0.Hash())

	result, err := cacheChain.GetBlock(block0.Hash(), block0.Height())
	assert.NoError(t, err)
	assert.Equal(t, block0.Hash(), result.Hash())

	_, err = cacheChain.GetBlock(block0.Hash(), block0.Height()+1)
	assert.Equal(t, ErrNotExist, err)
	cacheChain.Close()

	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	result, err = cacheChain.GetBlock(block0.Hash(), block0.Height())
	assert.NoError(t, err)
	assert.Equal(t, block0.Hash(), result.Hash())

	_, err = cacheChain.GetBlock(block0.Hash(), block0.Height()+1)
	assert.Equal(t, ErrNotExist, err)
	cacheChain.Close()
}

func TestChainDatabase_CandidatesRanking(t *testing.T) {
	ClearData()
	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	block0 := GetBlock0()
	cacheChain.SetBlock(block0.Hash(), block0)
	cacheChain.SetStableBlock(block0.Hash())

	block1 := GetBlock1()
	cacheChain.SetBlock(block1.Hash(), block1)

	count := 100
	candidates := NewAccountDataBatch(100)
	actDatabase, _ := cacheChain.GetActDatabase(block1.Hash())
	for index := 0; index < count; index++ {
		actDatabase.Put(candidates[index], 1)
	}
	cacheChain.CandidatesRanking(block1.Hash())
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

	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	top = cacheChain.GetCandidatesTop(block1.Hash())
	assert.Equal(t, max_candidate_count, len(top))

	last, err := cacheChain.LoadLatestBlock()
	assert.NoError(t, err)
	assert.Equal(t, block1.Hash(), last.Hash())

	account, err := cacheChain.GetAccount(common.HexToAddress(strconv.Itoa(99)))
	assert.NoError(t, err)
	assert.Equal(t, account.Address, common.HexToAddress(strconv.Itoa(99)))

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
