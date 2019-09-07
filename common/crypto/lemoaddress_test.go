package crypto

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// Test_GenerateAddress
func Test_GenerateAddress(t *testing.T) {
	// account1
	account1, err := GenerateAddress()
	assert.NoError(t, err)

	// sign and recover test
	msg := Keccak256([]byte("foo"))
	priKey, err := HexToECDSA(account1.Private[2:])
	assert.NoError(t, err)
	sig, err := Sign(msg, priKey)
	assert.NoError(t, err)
	recoveredPub, err := Ecrecover(msg, sig)
	assert.NoError(t, err)
	assert.Equal(t, account1.Address, PubkeyToAddress(*ToECDSAPub(recoveredPub)))

	// account2
	account2, err := GenerateAddress()
	assert.NoError(t, err)
	// they are different
	assert.NotEqual(t, account1.Private, account2.Private)
	assert.NotEqual(t, account1.Public, account2.Public)
	assert.NotEqual(t, account1.Address, account2.Address)
}
