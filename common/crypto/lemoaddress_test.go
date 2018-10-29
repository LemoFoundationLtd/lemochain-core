package crypto

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestWallet_GenerateAddress function test
func TestWallet_GenerateAddress(t *testing.T) {
	// Call ten times function and print result
	for i := 0; i < 10; i++ {
		_, err := GenerateAddress()
		assert.Nil(t, err)
		// t.Logf("LemoAddress=%v,\n publicKey=%v,\n privateKey=%v\n", addressKeyPair.LemoAddress, addressKeyPair.PublicKey, addressKeyPair.PrivateKey)

	}
}
