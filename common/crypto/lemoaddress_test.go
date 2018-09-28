package crypto

import (
	"testing"
)

// TestWallet_GenerateAddress function test
func TestWallet_GenerateAddress(t *testing.T) {
	// Call ten times function and print result
	for i := 0; i < 10; i++ {
		addressKeyPair, _ := GenerateAddress()
		t.Logf("LemoAddress=%v,\n publicKey=%v,\n privateKey=%v\n", addressKeyPair.LemoAddress, addressKeyPair.PublicKey, addressKeyPair.PrivateKet)
	}
}

// RestoreOriginalAddress function test
func TestRestoreOriginalAddress(t *testing.T) {
	tests := []struct {
		LemoAddress string
	}{
		{"Lemo4CQG3RWZG8DSQFTY6P78K24RCK88TTRNFSP6"},
		{"Lemo454NG9K7BFDQZR6J93T2QWPAG6Q4N9FWBR7J"},
		{"Lemo454NG9K7BFDQZRQW6J93T2PAG6Q4N9FWBR7J"},
		{"Lemo7BFDQZR6J93T45G9K2QWG6Q4N4N9FWBR7JPA"},
		{"Lemo454NG9K7BFDQZR6J93T2QWPAG6Q4N9FWBRZB"},
	}
	for _, test := range tests {
		nativeAddress := RestoreOriginalAddress(test.LemoAddress)
		t.Log(nativeAddress.Bytes())
	}
}
