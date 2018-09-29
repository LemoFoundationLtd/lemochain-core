package chain

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/vm"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewTxProcessor(t *testing.T) {
	p := NewTxProcessor(newChain())
	assert.NotEqual(t, (*vm.Config)(nil), p.cfg)
}

func TestTxProcessor_Process(t *testing.T) {
	t.Error("not implemented")
}

func TestTxProcessor_ApplyTxs(t *testing.T) {
	t.Error("not implemented")
}
