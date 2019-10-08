package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSignBlock(t *testing.T) {
	key, _ := crypto.GenerateKey()
	deputynode.SetSelfNodeKey(key)
	block := types.Block{Header: &types.Header{}}
	hash := block.Hash()

	// sign and recover
	sig, err := SignBlock(hash)
	assert.NoError(t, err)
	block.Header.SignData = sig
	nodeID, err := block.SignerNodeID()
	assert.NoError(t, err)
	assert.Equal(t, crypto.PrivateKeyToNodeID(key), nodeID)

	// sign another hash
	block.Header.Height++
	sig2, err := SignBlock(block.Hash())
	assert.NoError(t, err)
	assert.NotEqual(t, sig, sig2)
}

func BenchmarkSignBlock(b *testing.B) {
	key, _ := crypto.GenerateKey()
	deputynode.SetSelfNodeKey(key)
	block := types.Block{Header: &types.Header{}}
	hash := block.Hash()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 400; j++ {
			_, err := SignBlock(hash)
			assert.NoError(b, err)
		}
	}
}
