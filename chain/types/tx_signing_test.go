package types

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto/secp256k1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSignTx(t *testing.T) {

	assert.Empty(t, testTx.Sigs())
	assert.Empty(t, testTx.GasPayerSig())
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

	addr, err := testSigner.GetSigner(txV)
	assert.NoError(t, err)
	assert.Equal(t, testAddr, addr)

	// longer sig
	tx := &Transaction{data: txV.data}
	tx.data.Sig = common.FromHex("3de7cbaaff085cfc1db7d1f31bea6819413d2391d9c5f81684faaeb9835df8774727d43924a0eb18621076607211edd7062c413d1663f29eadda0b0ee3c467fe0100")

	addr, err = testSigner.GetSigner(tx)
	assert.Equal(t, secp256k1.ErrInvalidSignatureLen, err)

	// empty sig
	addr, err = testSigner.GetSigner(testTx)
	assert.Equal(t, secp256k1.ErrInvalidSignatureLen, err)
}

func TestDefaultSigner_Hash(t *testing.T) {

	assert.Equal(t, "0x81f8b8f725a9342a9ad85f31d2a6009afd52e43c5f86199a2089a32ea81913e6", testSigner.Hash(testTx).Hex())
}
func TestReimbursementTxSigner_GetSender(t *testing.T) {
	txW, err := MakeReimbursementTxSigner().SignTx(reimbursementTx, testPrivate)
	assert.NoError(t, err)

	addr, err := MakeReimbursementTxSigner().GetSigner(txW)
	assert.NoError(t, err)
	assert.Equal(t, testAddr, addr)
}
func TestReimbursementTxSigner_Hash(t *testing.T) {
	assert.Equal(t, "0x92771cb37ca07d610e88d8d3beb2cdc86c4ba9e45b249ed7af39165c1e755644", MakeReimbursementTxSigner().Hash(reimbursementTx).Hex())
}

func TestGasPayerSigner_GasPayerSignTx(t *testing.T) {
	firstSignTx, err := MakeReimbursementTxSigner().SignTx(reimbursementTx, testPrivate)
	assert.NoError(t, err)
	assert.Equal(t, []byte{}, firstSignTx.GasPayerSig())
	assert.Empty(t, firstSignTx.GasPrice())
	assert.Empty(t, firstSignTx.GasLimit())
	firstSignTx = GasPayerSignatureTx(firstSignTx, common.Big2, 2222)
	lastSignTx, err := MakeGasPayerSigner().SignTx(firstSignTx, gasPayerPrivate)
	assert.NoError(t, err)
	assert.NotEmpty(t, lastSignTx.GasPayerSig())
	assert.Equal(t, 65, len(lastSignTx.GasPayerSig()))
	assert.Equal(t, common.Big2, lastSignTx.GasPrice())
	assert.Equal(t, uint64(2222), lastSignTx.GasLimit())

}
func TestGasPayerSigner_SignHash(t *testing.T) {
	firstSignTx, err := MakeReimbursementTxSigner().SignTx(reimbursementTx, testPrivate)
	assert.NoError(t, err)
	hash := MakeGasPayerSigner().Hash(firstSignTx)
	assert.NoError(t, err)
	assert.Equal(t, "0x42dec4096939ffd5f033d4c9ba0b082341aa99fb51e50d6ec4f34810f75fc34f", hash.String())
}
func TestGasPayerSigner_GasPayer(t *testing.T) {
	firstSignTx, err := MakeReimbursementTxSigner().SignTx(reimbursementTx, testPrivate)
	assert.NoError(t, err)
	firstSignTx = GasPayerSignatureTx(firstSignTx, common.Big2, 2222)
	lastSignTx, err := MakeGasPayerSigner().SignTx(firstSignTx, gasPayerPrivate)
	assert.NoError(t, err)
	tx_gasPayer, err := lastSignTx.GasPayer()
	assert.NoError(t, err)
	payer, err := MakeGasPayerSigner().GetSigner(lastSignTx)
	assert.NoError(t, err)
	assert.Equal(t, gasPayerAddr, tx_gasPayer, payer)
}
