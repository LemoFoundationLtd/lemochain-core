package types

import (
	"encoding/json"
	"fmt"
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

	testTx = NewTransaction(common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 0, 200, 1544584596, "aa", "aaa")

	bigNum, _ = new(big.Int).SetString("111111111111111111111111111111111111111111111111111111111111", 16)
	bigString = "888888888888888888888888888888888888888888888888888888888888"
	testTxBig = NewTransaction(common.HexToAddress("0x1000000000000000000000000000000000000000"), bigNum, 100, bigNum, []byte(bigString), 0, 200, 1544584596, bigString, bigString)
)

// reimbursed gas transaction test
var (
	gasPayerPrivate, _ = crypto.HexToECDSA("57a0b0be5616e74c4315882e3649ade12c775db3b5023dcaa168d01825612c9b")
	gasPayerAddr       = crypto.PubkeyToAddress(gasPayerPrivate.PublicKey)
	reimbursementTx    = NewReimbursementTransaction(common.HexToAddress("0x2"), gasPayerAddr, common.Big2, []byte(bigString), 0, 200, 1544584596, "", "")
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
	tx := NewTransaction(common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 0, 200, expiration, "aa", "")
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
	tx := NewContractCreation(common.Big1, 100, common.Big2, []byte{12}, 0, 200, expiration, "aa", "")
	assert.Equal(t, params.OrdinaryTx, tx.Type())
	assert.Equal(t, TxVersion, tx.Version())
	assert.Equal(t, uint16(200), tx.ChainID())
	assert.Empty(t, tx.To())
	assert.Equal(t, "aa", tx.ToName())
	assert.Equal(t, common.Big2, tx.GasPrice())
	assert.Equal(t, common.Big1, tx.Amount())
	assert.Equal(t, uint64(100), tx.GasLimit())
	assert.Equal(t, []byte{12}, tx.Data())
	assert.Equal(t, expiration, tx.Expiration())
	assert.Empty(t, tx.Message())
}

func TestTransaction_WithSignature_From_Raw(t *testing.T) {
	h := testSigner.Hash(testTx)
	sig, err := crypto.Sign(h[:], testPrivate)
	assert.NoError(t, err)
	txV := &Transaction{data: testTx.data}
	txV.data.Sig = sig
	from, err := txV.From()
	assert.NoError(t, err)
	assert.Equal(t, testAddr, from)
}

func TestTransaction_EncodeRLP_DecodeRLP(t *testing.T) {
	txb, err := rlp.EncodeToBytes(testTx)
	assert.NoError(t, err)
	assert.Equal(t, "0xeb9400000000000000000000000000000000000000018261610264010c845c107d9483616161800181c88080", common.ToHex(txb))
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
	assert.Equal(t, testTx.GasPayerSig(), result.GasPayerSig())
	assert.Equal(t, testTx.Sig(), result.Sig())

	// with signature
	txV, err := testSigner.SignTx(testTx, testPrivate)
	assert.NoError(t, err)
	txb, err = rlp.EncodeToBytes(txV)
	assert.NoError(t, err)
	assert.Equal(t, "0xf86d9400000000000000000000000000000000000000018261610264010c845c107d9483616161800181c8b8418c0499083cb3d27bead4f21994aeebf8e75fa11df6bfe01c71cad583fc9a3c70778a437607d072540719a866adb630001fabbfb6b032d1a8dfbffac7daed8f020180", common.ToHex(txb))
	result = Transaction{}
	err = rlp.DecodeBytes(txb, &result)
	assert.NoError(t, err)
	assert.Equal(t, txV.data.Hash, result.data.Hash)
	from, err := txV.From()
	assert.NoError(t, err)
	assert.Equal(t, testAddr, from)
	payer, err := txV.GasPayer()
	assert.NoError(t, err)
	assert.Equal(t, testAddr, payer)

	assert.Equal(t, txV.ChainID(), result.ChainID())
	assert.Equal(t, txV.Version(), result.Version())
	assert.Equal(t, txV.Type(), result.Type())
	assert.Equal(t, txV.Sig(), result.Sig())
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
	assert.Equal(t, reimbursementTx.GasPayerSig(), result02.GasPayerSig())
	assert.Equal(t, reimbursementTx.Sig(), result02.Sig())

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
	actualPayer, err := lastSignTx.GasPayer()
	assert.NoError(t, err)
	assert.Equal(t, gasPayerAddr, actualPayer)
	assert.Equal(t, "0x178516469e49899aeb2e97572d1ebd4576b45d32767c94bfdcedc0511ff89a58158e13736a7bffecf233c86e439e595081676e6ee1ab8696b5e28cbd34806c6901", common.ToHex(lastSignTx.GasPayerSig()))
	rlpdata, err := rlp.EncodeToBytes(lastSignTx)
	assert.NoError(t, err)
	assert.Equal(t, "0xf8e99400000000000000000000000000000000000000028003820d0502b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838845c107d9480800181c8b841470024fba0158a446082242da8e0b97e6898d0605e1bb62627c3c703dd4a90b93ad5767e3cea6084a2036f1580815ea9bca594b3dc7c8a14ac06b45e6740cc5c01b841178516469e49899aeb2e97572d1ebd4576b45d32767c94bfdcedc0511ff89a58158e13736a7bffecf233c86e439e595081676e6ee1ab8696b5e28cbd34806c6901", common.ToHex(rlpdata))
	recovered := Transaction{}
	err = rlp.DecodeBytes(rlpdata, &recovered)
	assert.NoError(t, err)
	assert.Equal(t, lastSignTx.Data(), recovered.Data())
	assert.Equal(t, lastSignTx.Sig(), recovered.Sig())
	assert.Equal(t, lastSignTx.GasPayerSig(), recovered.GasPayerSig())
	from02, err := lastSignTx.From()
	assert.NoError(t, err)
	assert.Equal(t, testAddr, from02)
	payer02, err := lastSignTx.GasPayer()
	assert.NoError(t, err)
	assert.Equal(t, gasPayerAddr, payer02)
}
func TestTransaction_EncodeRLP_DecodeRLP_bigTx(t *testing.T) {
	txb, err := rlp.EncodeToBytes(testTxBig)
	assert.NoError(t, err)
	assert.Equal(t, "0xf90119941000000000000000000000000000000000000000b83c3838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838389e111111111111111111111111111111111111111111111111111111111111649e111111111111111111111111111111111111111111111111111111111111b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838845c107d94b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838800181c88080", common.ToHex(txb))
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
	assert.Equal(t, testTxBig.Sig(), result.Sig())
	assert.Equal(t, testTxBig.data.Hash, result.data.Hash)

	// with signature
	txV, err := testSigner.SignTx(testTxBig, testPrivate)
	assert.NoError(t, err)
	txb, err = rlp.EncodeToBytes(txV)
	assert.NoError(t, err)
	assert.Equal(t, "0xf9015b941000000000000000000000000000000000000000b83c3838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838389e111111111111111111111111111111111111111111111111111111111111649e111111111111111111111111111111111111111111111111111111111111b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838845c107d94b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838800181c8b841956da7af3519f957e1281e8f41735aa965e6a712e828856fe95bdbd31c787d760f235ae3ed6ff4ea8198d2528b4561b68544c5691de58aa974b7b6cc791f78d50180", common.ToHex(txb))
	result = Transaction{}
	err = rlp.DecodeBytes(txb, &result)
	assert.NoError(t, err)
	assert.Equal(t, txV.Sig(), result.Sig())
	assert.Equal(t, txV.data.Hash, result.data.Hash)
}

func TestTransaction_Hash(t *testing.T) {
	// hash without signature
	assert.Equal(t, common.HexToHash("0xa276a21f02f6fb42bc1efeb12123936685591ae1cd273b1b4bedc0619a549971"), testTx.Hash())

	// hash for sign
	h := testSigner.Hash(testTx)
	assert.Equal(t, common.HexToHash("0x81f8b8f725a9342a9ad85f31d2a6009afd52e43c5f86199a2089a32ea81913e6"), h)

	// hash with signature
	sig, err := crypto.Sign(h[:], testPrivate)
	assert.NoError(t, err)
	txV := testTx.Clone()
	txV.data.Sig = sig
	assert.Equal(t, common.HexToHash("0xc926cd0517ff2b272b431ce2def134c63af32645b9e47cb951ce4dc07a7fd754"), txV.Hash())
}
func TestReimbursementTransaction(t *testing.T) {
	// without sign
	assert.Equal(t, "0x551919cd818648e074fb83b7f982ef8ae33f9811f707d4c47f4fb30e46b85336", reimbursementTx.Hash().String())
	// two times sign
	h := MakeReimbursementTxSigner().Hash(reimbursementTx)
	// first sign
	sigData, err := crypto.Sign(h[:], testPrivate)
	assert.NoError(t, err)
	txV := &Transaction{data: reimbursementTx.data, gasPayer: reimbursementTx.gasPayer}
	txV.data.Sig = sigData

	// last sign
	txV = GasPayerSignatureTx(txV, common.Big3, 3333)
	txW, err := MakeGasPayerSigner().SignTx(txV, gasPayerPrivate)
	assert.NoError(t, err)
	assert.Equal(t, "0x92215a03721b15c36b08ee9c749e62bcef345524e031b6455267c85827e431ff", txW.Hash().String())
}

func TestTransaction_MarshalJSON_UnmarshalJSON(t *testing.T) {
	txV, err := testSigner.SignTx(testTx, testPrivate)
	assert.NoError(t, err)
	data, err := json.Marshal(txV)
	assert.NoError(t, err)
	assert.Equal(t, `{"to":"Lemo8888888888888888888888888888888888BW","toName":"aa","gasPrice":"2","gasLimit":"100","amount":"1","data":"0x0c","expirationTime":"1544584596","message":"aaa","type":"0","version":"1","chainID":"200","sig":"0x8c0499083cb3d27bead4f21994aeebf8e75fa11df6bfe01c71cad583fc9a3c70778a437607d072540719a866adb630001fabbfb6b032d1a8dfbffac7daed8f0201","hash":"0xc926cd0517ff2b272b431ce2def134c63af32645b9e47cb951ce4dc07a7fd754","gasPayerSig":"0x"}`, string(data))
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
	assert.Equal(t, `{"to":"Lemo8888888888888888888888888888888888QR","toName":"","gasPrice":"2","gasLimit":"2222","amount":"2","data":"0x383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838","expirationTime":"1544584596","message":"","type":"0","version":"1","chainID":"200","sig":"0x470024fba0158a446082242da8e0b97e6898d0605e1bb62627c3c703dd4a90b93ad5767e3cea6084a2036f1580815ea9bca594b3dc7c8a14ac06b45e6740cc5c01","hash":"0x3579e799b03d142a18879f81eadd33483a5ea28928f7eaf7aeecdd1302db454e","gasPayerSig":"0x2c9ebcdfa25fd74a54ab84257da567d529493dec102c45d1b63b8dcf55c373ae2b6dd0834de849bfca63686a84bc430aea14a709852ff4e3a2b06cfbd2bbb7c201"}`, string(data02))
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
	assert.Equal(t, `{"to":"Lemo8P6Y24SZ2JPY7AFQD4HJWWRQ6DJ6TW2Y9CCF","toName":"888888888888888888888888888888888888888888888888888888888888","gasPrice":"117789804318558955305553166716194567721832259791707930541440413419507985","gasLimit":"100","amount":"117789804318558955305553166716194567721832259791707930541440413419507985","data":"0x383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838","expirationTime":"1544584596","message":"888888888888888888888888888888888888888888888888888888888888","type":"0","version":"1","chainID":"200","sig":"0x956da7af3519f957e1281e8f41735aa965e6a712e828856fe95bdbd31c787d760f235ae3ed6ff4ea8198d2528b4561b68544c5691de58aa974b7b6cc791f78d501","hash":"0x67d885897bf4c6fc0fa5259fda45340d402dcf3f9b749228d35ff7fff60f577c","gasPayerSig":"0x"}`, string(data))
	var parsedTx *Transaction
	err = json.Unmarshal(data, &parsedTx)
	assert.NoError(t, err)
	assert.Equal(t, txV.Hash(), parsedTx.Hash())
}

func TestTransaction_Cost(t *testing.T) {
	assert.Equal(t, big.NewInt(201), testTx.Cost())
}

func TestSetSecp256k1V(t *testing.T) {
	V := big.NewInt(0)
	assert.Equal(t, big.NewInt(0x10000), SetSecp256k1V(V, 1))
	assert.Equal(t, big.NewInt(0x00000), SetSecp256k1V(V, 0))
	V.SetUint64(0xffffffff)
	assert.Equal(t, big.NewInt(0xffffffff), SetSecp256k1V(V, 1))
	assert.Equal(t, big.NewInt(0xfffeffff), SetSecp256k1V(V, 0))
}

func TestCombineV(t *testing.T) {
	assert.Equal(t, big.NewInt(0x102ffff), CombineV(1, 1, 0xffff))
	assert.Equal(t, big.NewInt(0), CombineV(0, 0, 0))
}

func TestParseV(t *testing.T) {
	txType, version, secp256k1V, chainID := ParseV(big.NewInt(0x103ffff))
	assert.Equal(t, uint8(1), txType)
	assert.Equal(t, uint8(1), version)
	assert.Equal(t, uint8(1), secp256k1V)
	assert.Equal(t, uint16(0xffff), chainID)

	txType, version, secp256k1V, chainID = ParseV(big.NewInt(0))
	assert.Equal(t, uint8(0), txType)
	assert.Equal(t, uint8(0), version)
	assert.Equal(t, uint8(0), secp256k1V)
	assert.Equal(t, uint16(0), chainID)

	fmt.Println(crypto.GenerateAddress())
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
