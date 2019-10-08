package deputynode

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_SelfNodeKey_GetSelfNodeKey(t *testing.T) {
	key, _ := crypto.GenerateKey()
	SetSelfNodeKey(key)
	assert.Equal(t, key, GetSelfNodeKey())
}

func Test_GetSelfNodeID(t *testing.T) {
	hash := common.HexToHash("0x1234567890")
	for i := 0; i < 100; i++ {
		key, _ := crypto.GenerateKey()
		SetSelfNodeKey(key)

		signInfo, err := crypto.Sign(hash[:], GetSelfNodeKey())
		assert.NoError(t, err)
		pub, err := crypto.Ecrecover(hash[:], signInfo)
		assert.NoError(t, err)
		assert.Equal(t, pub[1:], GetSelfNodeID())
	}
}
