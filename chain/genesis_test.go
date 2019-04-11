package chain

import (
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestSetupGenesisBlock(t *testing.T) {
	genesis := DefaultGenesisBlock()
	ClearData()

	cacheChain := store.NewChainDataBase(GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
	defer cacheChain.Close()

	genesisBlock := SetupGenesisBlock(cacheChain, genesis)
	block, err := cacheChain.GetBlockByHeight(0)
	assert.NoError(t, err)
	assert.Equal(t, genesisBlock.Hash(), block.Hash())

	founder, err := cacheChain.GetAccount(genesis.Founder)
	assert.NoError(t, err)
	assert.Equal(t, true, founder.Balance.Cmp(new(big.Int)) > 0)
}
