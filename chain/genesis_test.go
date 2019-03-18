package chain

import (
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestSetupGenesisBlock(t *testing.T) {
	genesis := DefaultGenesisBlock()
	store.ClearData()

	cacheChain := store.NewChainDataBase(store.GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
	defer cacheChain.Close()

	hash, err := SetupGenesisBlock(cacheChain, genesis)
	assert.NoError(t, err)

	block, err := cacheChain.GetBlockByHeight(0)
	assert.NoError(t, err)
	assert.Equal(t, hash, block.Hash())

	founder, err := cacheChain.GetAccount(genesis.Founder)
	assert.NoError(t, err)
	assert.Equal(t, true, founder.Balance.Cmp(new(big.Int)) > 0)
}
