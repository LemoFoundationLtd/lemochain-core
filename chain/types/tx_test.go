package types

import (
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

var (
	testSigner     = MakeSigner()
	testPrivate, _ = crypto.HexToECDSA("432a86ab8765d82415a803e29864dcfc1ed93dac949abf6f95a583179f27e4bb") // secp256k1.V = 1
	testAddr       = crypto.PubkeyToAddress(testPrivate.PublicKey)                                         // 0x0107134b9cdd7d89f83efa6175f9b3552f29094c

	testTx = NewTransaction(testAddr, common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 0, 200, 1544584596, "aa", "aaa")

	bigNum, _ = new(big.Int).SetString("111111111111111111111111111111111111111111111111111111111111", 16)
	bigString = "888888888888888888888888888888888888888888888888888888888888"
	testTxBig = NewTransaction(testAddr, common.HexToAddress("0x1000000000000000000000000000000000000000"), bigNum, 100, bigNum, []byte(bigString), 0, 200, 1544584596, bigString, bigString)
)

// reimbursed gas transaction test
var (
	gasPayerPrivate, _ = crypto.HexToECDSA("57a0b0be5616e74c4315882e3649ade12c775db3b5023dcaa168d01825612c9b")
	gasPayerAddr       = crypto.PubkeyToAddress(gasPayerPrivate.PublicKey)
	reimbursementTx    = NewReimbursementTransaction(testAddr, common.HexToAddress("0x2"), gasPayerAddr, common.Big2, []byte(bigString), 0, 200, 1544584596, "", "")
)

func TestNewReimbursementTransaction(t *testing.T) {
	rTx := reimbursementTx
	assert.Equal(t, uint64(0), rTx.GasLimit())
	assert.Equal(t, new(big.Int), rTx.GasPrice())
}

func ExpirationFromNow() uint64 {
	return uint64(time.Now().Unix()) + DefaultTTTL
}

func TestNewTransaction(t *testing.T) {
	expiration := ExpirationFromNow()
	tx := NewTransaction(testAddr, common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 0, 200, expiration, "aa", "")
	assert.Equal(t, params.OrdinaryTx, tx.Type())
	assert.Equal(t, TxVersion, tx.Version())
	assert.Equal(t, uint16(200), tx.ChainID())
	assert.Equal(t, common.HexToAddress("0x1"), *tx.To())
	assert.Equal(t, "aa", tx.ToName())
	assert.Equal(t, common.Big2, tx.GasPrice())
	assert.Equal(t, common.Big1, tx.Amount())
	assert.Equal(t, uint64(100), tx.GasLimit())
	assert.Equal(t, []byte{12}, tx.Data())
	assert.Equal(t, expiration, tx.Expiration())
	assert.Empty(t, tx.Message())
}

func TestNewContractCreation(t *testing.T) {
	expiration := ExpirationFromNow()
	tx := NewContractCreation(testAddr, common.Big1, 100, common.Big2, []byte{0x01, 0x02}, params.CreateContractTx, 200, expiration, "aa", "")
	assert.Equal(t, params.CreateContractTx, tx.Type())
	assert.Equal(t, TxVersion, tx.Version())
	assert.Equal(t, uint16(200), tx.ChainID())
	assert.Empty(t, tx.To())
	assert.Equal(t, "aa", tx.ToName())
	assert.Equal(t, common.Big2, tx.GasPrice())
	assert.Equal(t, common.Big1, tx.Amount())
	assert.Equal(t, uint64(100), tx.GasLimit())
	assert.Equal(t, []byte{0x01, 0x02}, tx.Data())
	assert.Equal(t, expiration, tx.Expiration())
	assert.Empty(t, tx.Message())
}

func TestTransaction_WithSignature_From_Raw(t *testing.T) {
	h := testSigner.Hash(testTx)
	sig, err := crypto.Sign(h[:], testPrivate)
	assert.NoError(t, err)
	txV := &Transaction{data: testTx.data}
	txV.data.Sigs = append(txV.data.Sigs, sig)
	from := txV.From()
	assert.NoError(t, err)
	assert.Equal(t, testAddr, from)
}

func TestTransaction_EncodeRLP_DecodeRLP(t *testing.T) {
	txb, err := rlp.EncodeToBytes(testTx)
	assert.NoError(t, err)
	assert.Equal(t, "0xeb800181c89400000000000000000000000000000000000000018261610264010c845c107d94836161618080", common.ToHex(txb))
	result := Transaction{}
	err = rlp.DecodeBytes(txb, &result)
	assert.NoError(t, err)
	assert.Equal(t, testTx.Type(), result.Type())
	assert.Equal(t, testTx.Version(), result.Version())
	assert.Equal(t, testTx.ChainID(), result.ChainID())
	assert.Equal(t, testTx.To(), result.To())
	assert.Equal(t, testTx.ToName(), result.ToName())
	assert.Equal(t, testTx.GasPrice(), result.GasPrice())
	assert.Equal(t, testTx.GasLimit(), result.GasLimit())
	assert.Equal(t, testTx.Data(), result.Data())
	assert.Equal(t, testTx.Expiration(), result.Expiration())
	assert.Equal(t, testTx.Message(), result.Message())
	assert.Equal(t, testTx.data.Hash, result.data.Hash)
	assert.Equal(t, testTx.GasPayerSigs(), result.GasPayerSigs())
	assert.Equal(t, testTx.Sigs(), result.Sigs())

	// with signature
	txV, err := testSigner.SignTx(testTx, testPrivate)
	assert.NoError(t, err)
	txb, err = rlp.EncodeToBytes(txV)
	assert.NoError(t, err)
	assert.Equal(t, "0xf86d800181c89400000000000000000000000000000000000000018261610264010c845c107d9483616161b8418c0499083cb3d27bead4f21994aeebf8e75fa11df6bfe01c71cad583fc9a3c70778a437607d072540719a866adb630001fabbfb6b032d1a8dfbffac7daed8f020180", common.ToHex(txb))
	result = Transaction{}
	err = rlp.DecodeBytes(txb, &result)
	assert.NoError(t, err)
	assert.Equal(t, txV.data.Hash, result.data.Hash)
	from := txV.From()
	assert.Equal(t, testAddr, from)
	payer := txV.GasPayer()
	assert.Equal(t, testAddr, payer)

	assert.Equal(t, txV.ChainID(), result.ChainID())
	assert.Equal(t, txV.Version(), result.Version())
	assert.Equal(t, txV.Type(), result.Type())
	assert.Equal(t, txV.Sigs(), result.Sigs())
}

func TestReimbursementTransaction_EncodeRLP_DecodeRLP(t *testing.T) {
	// reimbursement transaction
	rb, err := rlp.EncodeToBytes(reimbursementTx)
	assert.NoError(t, err)
	result02 := Transaction{}
	err = rlp.DecodeBytes(rb, &result02)
	assert.NoError(t, err)
	assert.Equal(t, reimbursementTx.Type(), result02.Type())
	assert.Equal(t, reimbursementTx.Version(), result02.Version())
	assert.Equal(t, reimbursementTx.ChainID(), result02.ChainID())
	assert.Equal(t, reimbursementTx.To(), result02.To())
	assert.Equal(t, reimbursementTx.ToName(), result02.ToName())
	assert.Equal(t, reimbursementTx.GasPrice(), result02.GasPrice())
	assert.Equal(t, reimbursementTx.GasLimit(), result02.GasLimit())
	assert.Equal(t, reimbursementTx.Data(), result02.Data())
	assert.Equal(t, reimbursementTx.Expiration(), result02.Expiration())
	assert.Equal(t, reimbursementTx.Message(), result02.Message())

	assert.Equal(t, reimbursementTx.data.Hash, result02.data.Hash)
	assert.Equal(t, reimbursementTx.GasPayerSigs(), result02.GasPayerSigs())
	assert.Equal(t, reimbursementTx.Sigs(), result02.Sigs())

	// 	two times sign
	firstSignTx, err := MakeReimbursementTxSigner().SignTx(reimbursementTx, testPrivate)
	assert.NoError(t, err)
	assert.Equal(t, new(big.Int), firstSignTx.GasPrice())
	assert.Equal(t, uint64(0), firstSignTx.GasLimit())
	assert.NoError(t, err)
	firstSignTx = GasPayerSignatureTx(firstSignTx, common.Big3, 3333)
	lastSignTx, err := MakeGasPayerSigner().SignTx(firstSignTx, gasPayerPrivate)
	assert.NoError(t, err)
	assert.Equal(t, common.Big3, lastSignTx.GasPrice())
	assert.Equal(t, uint64(3333), lastSignTx.GasLimit())
	actualPayer := lastSignTx.GasPayer()
	assert.Equal(t, gasPayerAddr, actualPayer)
	assert.Equal(t, "0x178516469e49899aeb2e97572d1ebd4576b45d32767c94bfdcedc0511ff89a58158e13736a7bffecf233c86e439e595081676e6ee1ab8696b5e28cbd34806c6901", common.ToHex(lastSignTx.GasPayerSigs()[0]))
	rlpdata, err := rlp.EncodeToBytes(lastSignTx)
	assert.NoError(t, err)
	assert.Equal(t, "0xf8e9800181c89400000000000000000000000000000000000000028003820d0502b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838845c107d9480b841470024fba0158a446082242da8e0b97e6898d0605e1bb62627c3c703dd4a90b93ad5767e3cea6084a2036f1580815ea9bca594b3dc7c8a14ac06b45e6740cc5c01b841178516469e49899aeb2e97572d1ebd4576b45d32767c94bfdcedc0511ff89a58158e13736a7bffecf233c86e439e595081676e6ee1ab8696b5e28cbd34806c6901", common.ToHex(rlpdata))
	recovered := Transaction{}
	err = rlp.DecodeBytes(rlpdata, &recovered)
	assert.NoError(t, err)
	assert.Equal(t, lastSignTx.Data(), recovered.Data())
	assert.Equal(t, lastSignTx.Sigs(), recovered.Sigs())
	assert.Equal(t, lastSignTx.GasPayerSigs(), recovered.GasPayerSigs())
	from02 := lastSignTx.From()
	assert.Equal(t, testAddr, from02)
	payer02 := lastSignTx.GasPayer()
	assert.Equal(t, gasPayerAddr, payer02)
}
func TestTransaction_EncodeRLP_DecodeRLP_bigTx(t *testing.T) {
	txb, err := rlp.EncodeToBytes(testTxBig)
	assert.NoError(t, err)
	assert.Equal(t, "0xf90119800181c8941000000000000000000000000000000000000000b83c3838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838389e111111111111111111111111111111111111111111111111111111111111649e111111111111111111111111111111111111111111111111111111111111b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838845c107d94b83c3838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838388080", common.ToHex(txb))
	result := Transaction{}
	err = rlp.DecodeBytes(txb, &result)
	assert.NoError(t, err)
	assert.Equal(t, testTxBig.Type(), result.Type())
	assert.Equal(t, testTxBig.Version(), result.Version())
	assert.Equal(t, testTxBig.ChainID(), result.ChainID())
	assert.Equal(t, testTxBig.To(), result.To())
	assert.Equal(t, testTxBig.ToName(), result.ToName())
	assert.Equal(t, testTxBig.GasPrice(), result.GasPrice())
	assert.Equal(t, testTxBig.GasLimit(), result.GasLimit())
	assert.Equal(t, testTxBig.Data(), result.Data())
	assert.Equal(t, testTxBig.Expiration(), result.Expiration())
	assert.Equal(t, testTxBig.Message(), result.Message())
	assert.Equal(t, testTxBig.Sigs(), result.Sigs())
	assert.Equal(t, testTxBig.data.Hash, result.data.Hash)

	// with signature
	txV, err := testSigner.SignTx(testTxBig, testPrivate)
	assert.NoError(t, err)
	txb, err = rlp.EncodeToBytes(txV)
	assert.NoError(t, err)
	assert.Equal(t, "0xf9015b800181c8941000000000000000000000000000000000000000b83c3838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838389e111111111111111111111111111111111111111111111111111111111111649e111111111111111111111111111111111111111111111111111111111111b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838845c107d94b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838b841956da7af3519f957e1281e8f41735aa965e6a712e828856fe95bdbd31c787d760f235ae3ed6ff4ea8198d2528b4561b68544c5691de58aa974b7b6cc791f78d50180", common.ToHex(txb))
	result = Transaction{}
	err = rlp.DecodeBytes(txb, &result)
	assert.NoError(t, err)
	assert.Equal(t, txV.Sigs(), result.Sigs())
	assert.Equal(t, txV.data.Hash, result.data.Hash)
}

func TestTransaction_Hash(t *testing.T) {
	// hash without signature
	assert.Equal(t, common.HexToHash("0xa2818bd2b84f64df106d67e92fac6103c1a1a5f5333d81761e36efb3e0f374f2"), testTx.Hash())

	// hash for sign
	h := testSigner.Hash(testTx)
	assert.Equal(t, common.HexToHash("0x81f8b8f725a9342a9ad85f31d2a6009afd52e43c5f86199a2089a32ea81913e6"), h)

	// hash with signature
	sig, err := crypto.Sign(h[:], testPrivate)
	assert.NoError(t, err)
	txV := testTx.Clone()
	txV.data.Sigs = append(txV.data.Sigs, sig)
	assert.Equal(t, common.HexToHash("0xa14ba16fec094ceae0d60fe6b7464e0d7fc3c26e85c9638f4f6928d7c5ac7f1e"), txV.Hash())
}
func TestReimbursementTransaction(t *testing.T) {
	// without sign
	assert.Equal(t, "0xce5f35b03b4f2269c2a0f765a1b08295076de9f727019f6782469a3b32a43244", reimbursementTx.Hash().String())
	// two times sign
	h := MakeReimbursementTxSigner().Hash(reimbursementTx)
	// first sign
	sigData, err := crypto.Sign(h[:], testPrivate)
	assert.NoError(t, err)
	txV := &Transaction{data: reimbursementTx.data}
	txV.data.Sigs = append(txV.data.Sigs, sigData)

	// last sign
	txV = GasPayerSignatureTx(txV, common.Big3, 3333)
	txW, err := MakeGasPayerSigner().SignTx(txV, gasPayerPrivate)
	assert.NoError(t, err)
	assert.Equal(t, "0x833ffd0cfc95dea9411c889ac3e6fcc18f732ef497aad95b61d21d7f8fc1014d", txW.Hash().String())
}

func TestTransaction_MarshalJSON_UnmarshalJSON(t *testing.T) {

	txV, err := testSigner.SignTx(testTx, testPrivate)
	assert.NoError(t, err)
	data, err := json.Marshal(txV)
	assert.NoError(t, err)
	assert.Equal(t, `{"type":"0","version":"1","chainID":"200","to":"Lemo8888888888888888888888888888888888BW","toName":"aa","gasPrice":"2","gasLimit":"100","amount":"1","data":"0x0c","expirationTime":"1544584596","message":"aaa","sig":"0x8c0499083cb3d27bead4f21994aeebf8e75fa11df6bfe01c71cad583fc9a3c70778a437607d072540719a866adb630001fabbfb6b032d1a8dfbffac7daed8f0201","hash":"0xa14ba16fec094ceae0d60fe6b7464e0d7fc3c26e85c9638f4f6928d7c5ac7f1e","gasPayerSig":"0x"}`, string(data))
	var parsedTx *Transaction
	err = json.Unmarshal(data, &parsedTx)
	assert.NoError(t, err)
	assert.Equal(t, txV.Hash(), parsedTx.Hash())

	//  reimbursement transaction
	firstSignTx, err := MakeReimbursementTxSigner().SignTx(reimbursementTx, testPrivate)
	assert.NoError(t, err)
	firstSignTx = GasPayerSignatureTx(firstSignTx, common.Big2, 2222)
	lastSignTx, err := MakeGasPayerSigner().SignTx(firstSignTx, gasPayerPrivate)
	data02, err := json.Marshal(lastSignTx)
	assert.NoError(t, err)
	assert.Equal(t, `{"type":"0","version":"1","chainID":"200","to":"Lemo8888888888888888888888888888888888QR","toName":"","gasPrice":"2","gasLimit":"2222","amount":"2","data":"0x383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838","expirationTime":"1544584596","message":"","sig":"0x470024fba0158a446082242da8e0b97e6898d0605e1bb62627c3c703dd4a90b93ad5767e3cea6084a2036f1580815ea9bca594b3dc7c8a14ac06b45e6740cc5c01","hash":"0x9aab86ad9ec5711c0a32a14fea895531523b324149535dd25a8e4dac674c501c","gasPayerSig":"0x2c9ebcdfa25fd74a54ab84257da567d529493dec102c45d1b63b8dcf55c373ae2b6dd0834de849bfca63686a84bc430aea14a709852ff4e3a2b06cfbd2bbb7c201"}`, string(data02))
	var txW *Transaction
	err = json.Unmarshal(data02, &txW)
	assert.NoError(t, err)
	assert.Equal(t, lastSignTx.Hash(), txW.Hash())
}

func TestTransaction_MarshalJSON_UnmarshalJSON_bigTx(t *testing.T) {
	txV, err := testSigner.SignTx(testTxBig, testPrivate)
	assert.NoError(t, err)
	data, err := json.Marshal(txV)
	assert.NoError(t, err)
	assert.Equal(t, `{"type":"0","version":"1","chainID":"200","to":"Lemo8P6Y24SZ2JPY7AFQD4HJWWRQ6DJ6TW2Y9CCF","toName":"888888888888888888888888888888888888888888888888888888888888","gasPrice":"117789804318558955305553166716194567721832259791707930541440413419507985","gasLimit":"100","amount":"117789804318558955305553166716194567721832259791707930541440413419507985","data":"0x383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838","expirationTime":"1544584596","message":"888888888888888888888888888888888888888888888888888888888888","sig":"0x956da7af3519f957e1281e8f41735aa965e6a712e828856fe95bdbd31c787d760f235ae3ed6ff4ea8198d2528b4561b68544c5691de58aa974b7b6cc791f78d501","hash":"0xcd74fe3f3e5bd900162a34bf756406fbf73dd1e164f98ba24404f43fa8e7f449","gasPayerSig":"0x"}`, string(data))
	var parsedTx *Transaction
	err = json.Unmarshal(data, &parsedTx)
	assert.NoError(t, err)
	assert.Equal(t, txV.Hash(), parsedTx.Hash())
}

func TestTransaction_Cost(t *testing.T) {
	assert.Equal(t, big.NewInt(201), testTx.Cost())
}

func TestGasPgnatureTx(t *testing.T) {
	type A struct {
		aa common.Hash
		bb []byte
	}
	h := &A{}
	if h.aa == (common.Hash{}) {
		t.Log("hash")
	}
	if h.bb == nil {
		t.Log("byte", h.bb)
	}
	var dd []byte
	if dd == nil {
		t.Log(dd)
	}
}

func TestTransaction_txlen(t *testing.T) {
	sigTx, err := MakeSigner().SignTx(testTx, testPrivate)
	assert.NoError(t, err)
	data, _ := sigTx.MarshalJSON()
	t.Log(len(data))
}
