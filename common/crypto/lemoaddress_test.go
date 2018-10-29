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
		{"Lemo3W3BPY6WQYY679PFNCA8RTCJ3DQ422PYBP2", "0x01a8d70aab17d3befeb865c9763f6d1287f56d1e"},
		{"Lemo392JP6H7WAFPYGK7AT38H5KK4BTGSKKSAP3", "0x011e0274e6c17f6763fe80c50c55eefe27786287"},
		{"Lemo35KT5PY5BRNQN58GCTFJJPB22Z7JB7JHDJP", "0x0100c749f8ca92f758bdd1b0b7a415df42cfaa06"},
	}
	for _, test := range tests {
		nativeAddress, err := RestoreOriginalAddress(test.LemoAddress)
		assert.Nil(t, err)
		LowerNativeAddress := strings.ToLower(nativeAddress.Hex())
		assert.Equal(t, test.Native, LowerNativeAddress)
	}

	// addpair, _ := GenerateAddress()
	// a := addpair.NativePubkey
	// t.Log(a)
	// t.Log(addpair.LemoAddress)
	// addressByte := common.FromHex(a)
	// t.Log(len(addressByte))
	//
	// address := common.BytesToAddress(addressByte)
	// lemo := address.String()
	// t.Log(len(lemo))
	//
	// fullPayload := append(addressByte, '1')
	// t.Log(len(fullPayload))
	// t.Log(fullPayload)
	// lemoAddress := append([]byte("logo"), fullPayload...)
	// t.Log(string(lemoAddress))

}
