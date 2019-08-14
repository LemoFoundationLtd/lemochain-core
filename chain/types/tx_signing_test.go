package types

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto/secp256k1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSignTx(t *testing.T) {

	assert.Empty(t, testTx.Sigs())
	assert.Empty(t, testTx.GasPayerSigs())
	// the specific testTx and testPrivate makes recovery == 1
	txV, err := testSigner.SignTx(testTx, testPrivate)
	assert.NoError(t, err)
	assert.NotEmpty(t, txV.Sigs())

	// reimbursed gas transaction
	assert.Empty(t, reimbursementTx.Sigs())
	// 	reimbursed gas transaction first times sign
	txW, err := MakeReimbursementTxSigner().SignTx(reimbursementTx, testPrivate)
	assert.NoError(t, err)
	assert.NotEmpty(t, txW.Sigs())
}

func TestDefaultSigner_GetSender(t *testing.T) {
	txV, err := testSigner.SignTx(testTx, testPrivate)
	assert.NoError(t, err)

	addr, err := testSigner.GetSigners(txV)
	assert.NoError(t, err)
	assert.Equal(t, testAddr, addr[0])

	// longer sig
	tx := &Transaction{data: txV.data}
	tx.data.Sigs = append(tx.data.Sigs, common.FromHex("3de7cbaaff085cfc1db7d1f31bea6819413d2391d9c5f81684faaeb9835df8774727d43924a0eb18621076607211edd7062c413d1663f29eadda0b0ee3c467fe0100"))

	addr, err = testSigner.GetSigners(tx)
	assert.Equal(t, secp256k1.ErrInvalidSignatureLen, err)

	// empty sig
	addr, err = testSigner.GetSigners(testTx)
	assert.Equal(t, ErrNoSignsData, err)
}

func TestDefaultSigner_Hash(t *testing.T) {
	hash, _ := testSigner.Hash(testTx)
	assert.Equal(t, "0x12c59cd1ba635a8a673e2276c870c912ce82a1157fae8dbed651a711682c260b", hash.Hex())
}
func TestReimbursementTxSigner_GetSender(t *testing.T) {
	txW, err := MakeReimbursementTxSigner().SignTx(reimbursementTx, testPrivate)
	assert.NoError(t, err)

	addr, err := MakeReimbursementTxSigner().GetSigners(txW)
	assert.NoError(t, err)
	assert.Equal(t, testAddr, addr[0])
}
func TestReimbursementTxSigner_Hash(t *testing.T) {
	hash, _ := MakeReimbursementTxSigner().Hash(reimbursementTx)
	assert.Equal(t, "0xc12595d1e15d445edd5b8653b69c8071794be7a8139cb21c2b0725c437803740", hash.Hex())
}

func TestGasPayerSigner_GasPayerSignTx(t *testing.T) {
	firstSignTx, err := MakeReimbursementTxSigner().SignTx(reimbursementTx, testPrivate)
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{}, firstSignTx.GasPayerSigs())
	assert.Empty(t, firstSignTx.GasPrice())
	assert.Empty(t, firstSignTx.GasLimit())
	firstSignTx = GasPayerSignatureTx(firstSignTx, common.Big2, 2222)
	lastSignTx, err := MakeGasPayerSigner().SignTx(firstSignTx, gasPayerPrivate)
	assert.NoError(t, err)
	assert.NotEmpty(t, lastSignTx.GasPayerSigs())
	assert.Equal(t, 65, len(lastSignTx.GasPayerSigs()[0]))
	assert.Equal(t, common.Big2, lastSignTx.GasPrice())
	assert.Equal(t, uint64(2222), lastSignTx.GasLimit())

}
func TestGasPayerSigner_SignHash(t *testing.T) {
	firstSignTx, err := MakeReimbursementTxSigner().SignTx(reimbursementTx, testPrivate)
	assert.NoError(t, err)
	hash, _ := MakeGasPayerSigner().Hash(firstSignTx)
	assert.NoError(t, err)
	assert.Equal(t, "0x91f21881c990ef7a14fc1d77fd0da95c96daa53573a85b6a2e208e3e9d75e7cd", hash.String())
}
func TestGasPayerSigner_GasPayer(t *testing.T) {
	firstSignTx, err := MakeReimbursementTxSigner().SignTx(reimbursementTx, testPrivate)
	assert.NoError(t, err)
	firstSignTx = GasPayerSignatureTx(firstSignTx, common.Big2, 2222)
	lastSignTx, err := MakeGasPayerSigner().SignTx(firstSignTx, gasPayerPrivate)
	assert.NoError(t, err)
	tx_gasPayer := lastSignTx.GasPayer()
	payer, err := MakeGasPayerSigner().GetSigners(lastSignTx)
	assert.NoError(t, err)
	assert.Equal(t, gasPayerAddr, tx_gasPayer, payer[0])
}
