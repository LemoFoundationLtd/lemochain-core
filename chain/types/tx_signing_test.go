package types

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestSignTx(t *testing.T) {

	assert.Empty(t, testTx.data.R)
	assert.Empty(t, testTx.data.S)
	// the specific testTx and testPrivate makes recovery == 1
	txV, err := SignTx(testTx, testSigner, testPrivate)
	assert.NoError(t, err)
	assert.NotEmpty(t, txV.data.R)
	assert.NotEmpty(t, txV.data.S)

	// reimbursed gas transaction
	assert.Empty(t, reimbursementTx.Tx.data.R)
	assert.Empty(t, reimbursementTx.Tx.data.S)
	// 	reimbursed gas transaction first times sign
	txW, err := SignTx(reimbursementTx.Tx, MakeReimbursementTxSigner(), testPrivate)
	assert.NoError(t, err)
	assert.NotEmpty(t, txW.data.R)
	assert.NotEmpty(t, txW.data.S)
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
	r, s, v, err := testSigner.ParseSignature(testTx, common.FromHex("0x8c0499083cb3d27bead4f21994aeebf8e75fa11df6bfe01c71cad583fc9a3c70778a437607d072540719a866adb630001fabbfb6b032d1a8dfbffac7daed8f0201"))
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

	assert.Equal(t, "0x81f8b8f725a9342a9ad85f31d2a6009afd52e43c5f86199a2089a32ea81913e6", testSigner.Hash(testTx).Hex())
}
func TestReimbursementTxSigner_GetSender(t *testing.T) {
	txW, err := SignTx(reimbursementTx.Tx, MakeReimbursementTxSigner(), testPrivate)
	assert.NoError(t, err)

	addr, err := MakeReimbursementTxSigner().GetSender(txW)
	assert.NoError(t, err)
	assert.Equal(t, testAddr, addr)
}
func TestReimbursementTxSigner_Hash(t *testing.T) {
	assert.Equal(t, "0xa46464124179dce00d7a1b16f356a2b7f80495eb2229049d86007aac9f12b0f6", MakeReimbursementTxSigner().Hash(reimbursementTx.Tx).Hex())
}

func TestGasPayerSigner_GasPayerSignTx(t *testing.T) {
	firstSignTx, err := SignTx(reimbursementTx.Tx, MakeReimbursementTxSigner(), testPrivate)
	assert.NoError(t, err)
	assert.Equal(t, []byte{}, firstSignTx.GasPayerSign())
	assert.Empty(t, firstSignTx.GasPrice())
	assert.Empty(t, firstSignTx.GasLimit())
	lastSignTx, err := MakeGasPayerSigner().GasPayerSignTx(firstSignTx, common.Big2, 2222, gasPayerPrivate)
	assert.NoError(t, err)
	assert.NotEmpty(t, lastSignTx.GasPayerSign())
	assert.Equal(t, 65, len(lastSignTx.GasPayerSign()))
	assert.Equal(t, common.Big2, lastSignTx.GasPrice())
	assert.Equal(t, uint64(2222), lastSignTx.GasLimit())

}
func TestGasPayerSigner_SignHash(t *testing.T) {
	firstSignTx, err := SignTx(reimbursementTx.Tx, MakeReimbursementTxSigner(), testPrivate)
	assert.NoError(t, err)
	hash, err := MakeGasPayerSigner().SignHash(firstSignTx, common.Big2, 2222)
	assert.NoError(t, err)
	assert.Equal(t, "0xbae908458c9cc208e599167f90576ad9c5d970e5d5d6aef88365694695c0d667", hash.String())
}
func TestGasPayerSigner_GasPayer(t *testing.T) {
	firstSignTx, err := SignTx(reimbursementTx.Tx, MakeReimbursementTxSigner(), testPrivate)
	assert.NoError(t, err)
	lastSignTx, err := MakeGasPayerSigner().GasPayerSignTx(firstSignTx, common.Big2, 2222, gasPayerPrivate)
	assert.NoError(t, err)
	tx_gasPayer, err := lastSignTx.GasPayer()
	assert.NoError(t, err)
	payer, err := MakeGasPayerSigner().GasPayer(lastSignTx)
	assert.NoError(t, err)
	assert.Equal(t, gasPayerAddr, tx_gasPayer, payer)
}
