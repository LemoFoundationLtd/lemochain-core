package store

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCacheChain_SetBlock(t *testing.T) {
	ClearData()

	cacheChain, err := NewCacheChain(GetStorePath())
	assert.NoError(t, err)

	block := GetBlock0()
	err = cacheChain.SetBlock(block.Hash(), block)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(block.Hash())
	assert.NoError(t, err)

	result, err := cacheChain.GetBlockByHeight(0)
	assert.NoError(t, err)
	assert.Equal(t, block.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHash(block.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block.ParentHash(), result.ParentHash())

	//
	block = GetBlock1()
	err = cacheChain.SetBlock(block.Hash(), block)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(block.Hash())
	assert.NoError(t, err)

	result, err = cacheChain.GetBlockByHeight(1)
	assert.NoError(t, err)
	assert.Equal(t, block.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHash(block.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block.ParentHash(), result.ParentHash())
}

func TestCacheChain_WriteChain1(t *testing.T) {
	ClearData()

	cacheChain, err := NewCacheChain(GetStorePath())
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)

	childBlock := GetBlock1()
	err = cacheChain.SetBlock(childBlock.Hash(), childBlock)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(childBlock.Hash())
	assert.NoError(t, err)

	//
	result, err := cacheChain.GetBlockByHeight(0)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), parentBlock.Hash())

	result, err = cacheChain.GetBlockByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), parentBlock.Hash())

	result, err = cacheChain.GetBlockByHeight(1)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), childBlock.Hash())

	result, err = cacheChain.GetBlockByHash(childBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), childBlock.Hash())
}

func TestCacheChain_WriteChain2(t *testing.T) {
	ClearData()

	cacheChain, err := NewCacheChain(GetStorePath())
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)

	childBlock := GetBlock1()
	err = cacheChain.SetBlock(childBlock.Hash(), childBlock)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(childBlock.Hash())
	assert.NoError(t, err)

	//
	result, err := cacheChain.GetBlockByHeight(0)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), parentBlock.Hash())

	result, err = cacheChain.GetBlockByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), parentBlock.Hash())

	result, err = cacheChain.GetBlockByHeight(1)
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), childBlock.Hash())

	result, err = cacheChain.GetBlockByHash(childBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), childBlock.Hash())
}

func TestCacheChain_repeat(t *testing.T) {
	ClearData()

	cacheChain, err := NewCacheChain(GetStorePath())
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)

	childBlock := GetBlock1()
	err = cacheChain.SetBlock(childBlock.Hash(), childBlock)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(childBlock.Hash())
	assert.NoError(t, err)

	//
	parentBlock = GetBlock0()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.Equal(t, err, ErrExist)

	childBlock = GetBlock1()
	err = cacheChain.SetBlock(childBlock.Hash(), childBlock)
	assert.Equal(t, err, ErrExist)
}

func TestCacheChain_Load4Hit(t *testing.T) {
	ClearData()

	cacheChain, err := NewCacheChain(GetStorePath())
	assert.NoError(t, err)

	block := GetBlock0()
	err = cacheChain.SetBlock(block.Hash(), block)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(block.Hash())
	assert.NoError(t, err)

	block = GetBlock1()
	err = cacheChain.SetBlock(block.Hash(), block)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(block.Hash())
	assert.NoError(t, err)

	cacheChain, err = NewCacheChain(GetStorePath())
	assert.NoError(t, err)

	parentBlock := GetBlock0()

	result, err := cacheChain.GetBlockByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, parentBlock.ParentHash(), result.ParentHash())

	//
	childBlock := GetBlock1()
	result, err = cacheChain.GetBlockByHash(childBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, childBlock.ParentHash(), parentBlock.Hash())

	result, err = cacheChain.GetBlockByHeight(1)
	assert.NoError(t, err)
	assert.Equal(t, childBlock.ParentHash(), parentBlock.Hash())
}

func TestCacheChain_ContractCode(t *testing.T) {
	ClearData()

	cacheChain, err := NewCacheChain(GetStorePath())
	assert.NoError(t, err)

	code := types.Code("this  is code")
	hash := common.HexToHash("this is code")

	err = cacheChain.SetContractCode(hash, &code)
	assert.NoError(t, err)

	result, err := cacheChain.GetContractCode(hash)
	assert.NoError(t, err)
	assert.Equal(t, &code, result)
}

func TestCacheChain_SetAccounts(t *testing.T) {
	ClearData()

	cacheChain, err := NewCacheChain(GetStorePath())
	assert.NoError(t, err)

	// block is not exist
	accounts := GetAccounts()
	err = cacheChain.SetAccounts(common.Hash{}, accounts)
	assert.Equal(t, err, ErrNotExist)

	// the block's parent is nil
	parentBlock := GetBlock0()
	childBlock := GetBlock1()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)
	err = cacheChain.SetBlock(childBlock.Hash(), childBlock)
	assert.NoError(t, err)

	// TODO:the child block have no this accounts
	err = cacheChain.SetAccounts(parentBlock.Hash(), accounts)
	assert.NoError(t, err)

	account, err := cacheChain.GetAccount(parentBlock.Hash(), accounts[0].Address)
	assert.NoError(t, err)
	assert.Equal(t, account.Version, accounts[0].Version)

	account, err = cacheChain.GetAccount(childBlock.Hash(), accounts[0].Address)
	assert.Equal(t, err, ErrNotExist)
	assert.Nil(t, account)

	account, err = cacheChain.GetCanonicalAccount(accounts[0].Address)
	assert.Equal(t, err, ErrNotExist)
	assert.Nil(t, account)

	err = cacheChain.SetStableBlock(childBlock.Hash())
	assert.NoError(t, err)

	account, err = cacheChain.GetCanonicalAccount(accounts[0].Address)
	assert.NoError(t, err)
	assert.Equal(t, account.Version, accounts[0].Version)

	account, err = cacheChain.GetCanonicalAccount(accounts[1].Address)
	assert.NoError(t, err)
	assert.Equal(t, account.Version, accounts[1].Version)
}

func TestCacheChain_LoadLatestBlock1(t *testing.T) {
	ClearData()

	cacheChain, err := NewCacheChain(GetStorePath())
	assert.NoError(t, err)

	result, err := cacheChain.LoadLatestBlock()
	assert.Equal(t, err, ErrNotExist)
	assert.Nil(t, result)

	parentBlock := GetBlock0()
	childBlock := GetBlock1()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)
	err = cacheChain.SetBlock(childBlock.Hash(), childBlock)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)

	result, err = cacheChain.LoadLatestBlock()
	assert.NoError(t, err)
	assert.Equal(t, parentBlock.ParentHash(), result.ParentHash())

	err = cacheChain.SetStableBlock(childBlock.Hash())
	assert.NoError(t, err)

	result, err = cacheChain.LoadLatestBlock()
	assert.NoError(t, err)
	assert.Equal(t, childBlock.ParentHash(), result.ParentHash())

	cacheChain, err = NewCacheChain(GetStorePath())
	assert.NoError(t, err)

	result, err = cacheChain.LoadLatestBlock()
	assert.NoError(t, err)
	assert.Equal(t, childBlock.ParentHash(), result.ParentHash())
}

func TestCacheChain_LoadLatestBlock2(t *testing.T) {
	ClearData()

	cacheChain, err := NewCacheChain(GetStorePath())
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	childBlock := GetBlock1()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)
	err = cacheChain.SetBlock(childBlock.Hash(), childBlock)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(childBlock.Hash())
	assert.NoError(t, err)

	result, err := cacheChain.LoadLatestBlock()
	assert.NoError(t, err)
	assert.Equal(t, childBlock.ParentHash(), result.ParentHash())

	err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.Equal(t, err, ErrNotExist)

	result, err = cacheChain.LoadLatestBlock()
	assert.NoError(t, err)
	assert.Equal(t, childBlock.ParentHash(), result.ParentHash())
}

func TestCacheChain_SetConfirm1(t *testing.T) {
	ClearData()

	cacheChain, err := NewCacheChain(GetStorePath())
	assert.NoError(t, err)

	signs, err := CreateSign(16)
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)
	err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)
	err = cacheChain.SetConfirmPackage(parentBlock.Hash(), signs)
	assert.NoError(t, err)

	result, err := cacheChain.GetConfirmPackage(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 16, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])

	cacheChain, err = NewCacheChain(GetStorePath())
	assert.NoError(t, err)

	result, err = cacheChain.GetConfirmPackage(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 16, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])
}

func TestCacheChain_SetConfirm2(t *testing.T) {
	ClearData()

	cacheChain, err := NewCacheChain(GetStorePath())
	assert.NoError(t, err)

	signs, err := CreateSign(16)
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	err = cacheChain.SetConfirmInfo(parentBlock.Hash(), signs[0])
	assert.Equal(t, err, ErrNotExist)

	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)
	err = cacheChain.SetConfirmInfo(parentBlock.Hash(), signs[0])
	assert.NoError(t, err)
	err = cacheChain.SetConfirmInfo(parentBlock.Hash(), signs[1])
	assert.NoError(t, err)
	err = cacheChain.SetConfirmInfo(parentBlock.Hash(), signs[2])
	assert.NoError(t, err)
	err = cacheChain.SetConfirmInfo(parentBlock.Hash(), signs[3])
	assert.NoError(t, err)

	result, err := cacheChain.GetConfirmPackage(parentBlock.Hash())
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
	result, err = cacheChain.GetConfirmPackage(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 4, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])

	cacheChain, err = NewCacheChain(GetStorePath())
	assert.NoError(t, err)

	result, err = cacheChain.GetConfirmPackage(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 4, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])
}
