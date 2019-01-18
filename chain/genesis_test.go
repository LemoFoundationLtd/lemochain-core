package chain

import (
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetupGenesisBlock(t *testing.T) {
	genesis := DefaultGenesisBlock()
	store.ClearData()

	cacheChain := store.NewChainDataBase(store.GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)

	hash, err := SetupGenesisBlock(cacheChain, genesis)
	assert.NoError(t, err)

	block, err := cacheChain.GetBlockByHeight(0)
	assert.NoError(t, err)

	assert.Equal(t, hash, block.Hash())
}
