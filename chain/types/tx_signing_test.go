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
	r, s, v, err := testSigner.ParseSignature(testTx, common.FromHex("0x3de7cbaaff085cfc1db7d1f31bea6819413d2391d9c5f81684faaeb9835df8774727d43924a0eb18621076607211edd7062c413d1663f29eadda0b0ee3c467fe01"))
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
	assert.Equal(t, "0x7fcf879493a5026b781cb6a2aa9158ccfb6cae0feb77fddf0a08d7f229e3c607", testSigner.Hash(testTx).Hex())
}
