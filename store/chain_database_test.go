package store

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"testing"
	// "github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"strconv"
)

func TestCacheChain_SetBlock(t *testing.T) {
	ClearData()

	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	result, err := cacheChain.GetBlockByHeight(0)
	assert.Equal(t, err, ErrNotExist)

	//
	block0 := GetBlock0()
	err = cacheChain.SetBlock(block0.Hash(), block0)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(block0.Hash())
	assert.NoError(t, err)

	result, err = cacheChain.GetBlockByHash(block0.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block0.ParentHash(), result.ParentHash())

	//
	block1 := GetBlock1()
	err = cacheChain.SetBlock(block1.Hash(), block1)
	assert.NoError(t, err)

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

	result, err = cacheChain.GetBlockByHash(block3.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block3.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHeight(block3.Height())
	assert.Equal(t, err, ErrNotExist)

	//
	err = cacheChain.SetStableBlock(block2.Hash())
	assert.NoError(t, err)

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
	block1 := GetBlock1()

	assert.PanicsWithValue(t, "(database.LastConfirm.Block == nil) && (block.Height() != 0) && (block.ParentHash() != common.Hash{})", func() {
		cacheChain.SetBlock(block1.Hash(), block1)
	})

	block0 := GetBlock0()
	cacheChain.SetBlock(block0.Hash(), block0)
	cacheChain.SetStableBlock(block0.Hash())

	err := cacheChain.SetBlock(block0.Hash(), block0)
	assert.Equal(t, ErrExist, err)

	block1.Header.Height = 0
	assert.PanicsWithValue(t, "(block.Height() == 0) || (block.ParentHash() == common.Hash{})", func() {
		cacheChain.SetBlock(block1.Hash(), block1)
	})
	block1.Header.Height = 1
	cacheChain.SetBlock(block1.Hash(), block1)
	cacheChain.SetStableBlock(block1.Hash())

	block2 := GetBlock2()
	block2.Header.Height = 1
	assert.PanicsWithValue(t, "(database.LastConfirm.Block != nil) && (height < database.LastConfirm.Block.Height())", func() {
		cacheChain.SetBlock(block2.Hash(), block2)
	})

	// ERROR #2
	ClearData()
	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	block0 = GetBlock0()
	block1 = GetBlock1()
	block2 = GetBlock2()
	// block3 := GetBlock3()

	cacheChain.SetBlock(block0.Hash(), block0)
	cacheChain.SetStableBlock(block0.Hash())

	hash := block1.Hash()
	block1.Header.Height = 2
	assert.PanicsWithValue(t, "database.LastConfirm.Block.Height() + 1 != block.Height()", func() {
		cacheChain.SetBlock(hash, block1)
	})

	assert.PanicsWithValue(t, "database.LastConfirm.Block.Header.Hash() != pHash", func() {
		cacheChain.SetBlock(block2.Hash(), block2)
	})

	cacheChain.SetBlock(block1.Hash(), block1)
	hash = block2.Hash()
	block2.Header.Height = 3
	assert.PanicsWithValue(t, "pBlock.Block.Height() + 1 != block.Height()", func() {
		cacheChain.SetBlock(block2.Hash(), block2)
	})
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

	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	isExist, err = cacheChain.IsExistByHash(block.Hash())
	assert.NoError(t, err)
	assert.Equal(t, true, isExist)
}

func TestCacheChain_WriteChain(t *testing.T) {
	ClearData()
	block0 := GetBlock0()
	block1 := GetBlock1()
	block2 := GetBlock2()
	block3 := GetBlock3()
	block4 := GetBlock4()

	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	cacheChain.SetBlock(block0.Hash(), block0)
	err := cacheChain.SetStableBlock(block0.Hash())
	assert.NoError(t, err)

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

	ClearData()
	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	cacheChain.SetBlock(block0.Hash(), block0)
	err = cacheChain.SetStableBlock(block0.Hash())
	assert.NoError(t, err)

	// 1, 2, 3#
	cacheChain.SetBlock(block1.Hash(), block1)
	cacheChain.SetBlock(block2.Hash(), block2)
	cacheChain.SetBlock(block3.Hash(), block3)
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
	assert.PanicsWithValue(t, "set stable block error:the block is not exist. hash:"+block1.Hash().Hex(), func() {
		cacheChain.SetStableBlock(block1.Hash())
	})

	assert.PanicsWithValue(t, "set stable block error:the block is not exist. hash:"+block3.Hash().Hex(), func() {
		cacheChain.SetStableBlock(block3.Hash())
	})

	assert.PanicsWithValue(t, "set stable block error:the block is not exist. hash:"+block4.Hash().Hex(), func() {
		cacheChain.SetStableBlock(block4.Hash())
	})
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

	database = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	result, err = database.GetContractCode(hash)
	assert.NoError(t, err)
	assert.Equal(t, code, result)
}

func TestCacheChain_LastConfirm(t *testing.T) {
	ClearData()

	block0 := GetBlock0()
	block1 := GetBlock1()
	block2 := GetBlock2()
	block3 := GetBlock3()
	block4 := GetBlock4()
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

	ClearData()
	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	cacheChain.SetBlock(block0.Hash(), block0)
	cacheChain.SetStableBlock(block0.Hash())

	cacheChain.SetBlock(block1.Hash(), block1)
	cacheChain.SetBlock(block2.Hash(), block2)
	cacheChain.SetBlock(block3.Hash(), block3)
	cacheChain.SetBlock(block4.Hash(), block4)
	cacheChain.SetStableBlock(block4.Hash())

	lastConfirmBlock = cacheChain.LastConfirm.Block
	assert.Equal(t, block4.Hash(), lastConfirmBlock.Hash())

	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	lastConfirmBlock = cacheChain.LastConfirm.Block
	assert.Equal(t, block4.Hash(), lastConfirmBlock.Hash())
}

func TestCacheChain_SetConfirm1(t *testing.T) {
	ClearData()

	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	signs, err := CreateSign(16)
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)
	err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)
	err = cacheChain.SetConfirms(parentBlock.Hash(), signs)
	assert.NoError(t, err)

	result, err := cacheChain.GetConfirms(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 16, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])

	//cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

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

	//cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

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
