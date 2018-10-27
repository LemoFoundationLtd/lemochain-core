package types

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestSignTx(t *testing.T) {
	V := testTx.data.V
	assert.Empty(t, testTx.data.R)
	assert.Empty(t, testTx.data.S)
	txV, err := SignTx(testTx, testSigner, testPrivate)
	assert.NoError(t, err)
	assert.NotEqual(t, V, txV.data.V)
	assert.NotEmpty(t, txV.data.R)
	assert.NotEmpty(t, txV.data.S)
}

func TestDefaultSigner_GetSender(t *testing.T) {
	txV, err := SignTx(testTx, testSigner, testPrivate)
	assert.NoError(t, err)

	addr, err := testSigner.GetSender(txV)
	assert.NoError(t, err)
	assert.Equal(t, testAddr, addr)

	// longer V
	tx := &Transaction{data: txV.data}
	var ok bool
	tx.data.V, ok = new(big.Int).SetString("3de7cbaaff085cfc1db7d1f31bea6819413d2391d9c5f81684faaeb9835df8774727d43924a0eb18621076607211edd7062c413d1663f29eadda0b0ee3c467fe01", 16)
	assert.Equal(t, true, ok)
	assert.NotEqual(t, txV.data.V, tx.data.V)
	addr, err = testSigner.GetSender(tx)
	assert.Equal(t, ErrInvalidSig, err)

	// empty R,S
	addr, err = testSigner.GetSender(testTx)
	assert.Equal(t, ErrInvalidSig, err)

	// invalid R
	tx = &Transaction{data: txV.data}
	tx.data.R = big.NewInt(0)
	addr, err = testSigner.GetSender(tx)
	assert.Equal(t, ErrInvalidSig, err)
	assert.NotEqual(t, testAddr, addr)
}

func TestDefaultSigner_ParseSignature(t *testing.T) {
	txV, err := SignTx(testTx, testSigner, testPrivate)
	assert.NoError(t, err)
	r, s, v, err := testSigner.ParseSignature(testTx, common.FromHex("0x158b80d695e7d543ddb3ae09ed89b0fdd0c9f72b95a96e5f2b5e67a4d6d71a882b893b663e36f997df1e3f489b98d001cf615ee1e32b3c28ce6364f5cc681d5c01"))
	assert.NoError(t, err)
	assert.Equal(t, txV.data.R, r)
	assert.Equal(t, txV.data.S, s)
	assert.Equal(t, txV.data.V, v)

	// invalid length
	assert.PanicsWithValue(t, "wrong size for signature: got 0, want 65", func() {
		testSigner.ParseSignature(testTx, common.FromHex(""))
	})
}

func TestDefaultSigner_Hash(t *testing.T) {
	assert.Equal(t, "0x9f79748da47a0c32d2d268a5cfbe3a2a7d6c29d1a2f0534f416f3d2157933808", testSigner.Hash(testTx).Hex())
}
