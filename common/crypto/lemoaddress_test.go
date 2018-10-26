package crypto

import (
	"github.com/stretchr/testify/assert"
	"strings"
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

// RestoreOriginalAddress function test
func TestRestoreOriginalAddress(t *testing.T) {
	tests := []struct {
		LemoAddress string
		Native      string
	}{
		{"Lemo44HKF7J49KY3ZDGG8YA2DWKKJJA73WFK997", "0x01d88fa5d7b95e3749891097f58990faff42840a"},
		{"Lemo46RGWZPB2R4N85K4Y6T4JDHGT3BPWRHF6JR", "0x01ebd94fee207ab64c99c0212276a832c3f936a3"},
		{"Lemo35ZFT9ZPZB9QCNNJ285B3HSJ8AJD4A8A8CH", "0x0103a851dfe5ec1a649ecb87eb2b391f5cb9b9b0"},
	}
	for _, test := range tests {
		nativeAddress := RestoreOriginalAddress(test.LemoAddress)
		LowerNativeAddress := strings.ToLower(nativeAddress.Hex())
		assert.Equal(t, test.Native, LowerNativeAddress)
	}
}
