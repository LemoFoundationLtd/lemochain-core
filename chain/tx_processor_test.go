package chain

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewTxProcessor(t *testing.T) {
	chain := newChain()
	p := NewTxProcessor(chain)
	assert.NotEqual(t, (*vm.Config)(nil), p.cfg)
	assert.Equal(t, false, p.cfg.Debug)

	chain, _ = NewBlockChain(uint64(chainID), NewDpovp(10, 3), chain.dbOpe, chain.newBlockCh, map[string]string{
		common.Debug: "1",
	})
	p = NewTxProcessor(chain)
	assert.Equal(t, true, p.cfg.Debug)
}

func TestTxProcessor_Process(t *testing.T) {
	p := NewTxProcessor(newChain())

	// last not stable block
	block := defaultBlocks[2]
	newHeader, err := p.Process(block)
	assert.NoError(t, err)
	assert.Equal(t, block.Header.Bloom, newHeader.Bloom)
	assert.Equal(t, block.Header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, block.Header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, block.Header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, block.Header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, block.Header.LogsRoot, newHeader.LogsRoot)
	assert.Equal(t, block.Hash(), newHeader.Hash())

	// block not in db
	block = newestBlock
	newHeader, err = p.Process(block)
	assert.NoError(t, err)
	assert.Equal(t, block.Header.Bloom, newHeader.Bloom)
	assert.Equal(t, block.Header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, block.Header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, block.Header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, block.Header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, block.Header.LogsRoot, newHeader.LogsRoot)
	assert.Equal(t, block.Hash(), newHeader.Hash())

	// genesis block
	block = defaultBlocks[0]
	newHeader, err = p.Process(block)
	assert.NoError(t, err)
	assert.Equal(t, block.Header.Bloom, newHeader.Bloom)
	assert.Equal(t, block.Header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, block.Header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, block.Header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, block.Header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, block.Header.LogsRoot, newHeader.LogsRoot)
	assert.Equal(t, block.Hash(), newHeader.Hash())
}

func TestTxProcessor_ApplyTxs(t *testing.T) {
	t.Error("not implemented")
}
