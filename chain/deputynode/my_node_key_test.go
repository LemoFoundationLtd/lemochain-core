package deputynode

import (
	"crypto/ecdsa"
	"crypto/rand"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto/secp256k1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_SelfNodeKey_GetSelfNodeKey(t *testing.T) {
	key, _ := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	SetSelfNodeKey(key)
	assert.Equal(t, key, GetSelfNodeKey())
}

func Test_GetSelfNodeID(t *testing.T) {
	hash := common.HexToHash("0x1234567890")
	for i := 0; i < 100; i++ {
		key, _ := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
		SetSelfNodeKey(key)

		signInfo, err := crypto.Sign(hash[:], GetSelfNodeKey())
		assert.NoError(t, err)
		pub, err := crypto.Ecrecover(hash[:], signInfo)
		assert.NoError(t, err)
		assert.Equal(t, pub[1:], GetSelfNodeID())
	}
}
