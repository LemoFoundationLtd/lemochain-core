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

	chain, _ = NewBlockChain(uint64(chainID), chain.dbOpe, chain.newBlockCh, map[string]string{
		common.Debug: "1",
	})
	p = NewTxProcessor(chain)
	assert.Equal(t, true, p.cfg.Debug)
}

func TestTxProcessor_Process(t *testing.T) {
	p := NewTxProcessor(newChain())
	newHeader, err := p.Process(newestBlock)
	assert.NoError(t, err)
	assert.Equal(t, newestBlock.Hash(), newHeader.Hash())
}

func TestTxProcessor_ApplyTxs(t *testing.T) {
	t.Error("not implemented")
}
