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
	assert.Equal(t, "0xf856800181c8940107134b9cdd7d89f83efa6175f9b3552f29094c940107134b9cdd7d89f83efa6175f9b3552f29094c940000000000000000000000000000000000000001826161026480010c845c107d9483616161c0c0", common.ToHex(txb))
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
	assert.Equal(t, uint64(0), result.GasUsed())

	// with signature
	txV, err := testSigner.SignTx(testTx, testPrivate)
	assert.NoError(t, err)
	txb, err = rlp.EncodeToBytes(txV)
	assert.NoError(t, err)
	assert.Equal(t, "0xf89a800181c8940107134b9cdd7d89f83efa6175f9b3552f29094c940107134b9cdd7d89f83efa6175f9b3552f29094c940000000000000000000000000000000000000001826161026480010c845c107d9483616161f843b841df998b9e98778c5b54e9c8c104b9841c34979f81ffa3c0bdac6c2d63a8b85e1001e6841ea0e9bbe43da601e81a5817a9b0a95b0e2a156d42f0863bb26aff2d0c00c0", common.ToHex(txb))
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
	assert.Equal(t, uint64(0), result02.GasUsed())

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
	assert.Equal(t, "0xca1d67ca4f9c6e1cef1e8b918beb18f0a34d1604b6a249bee79e213bdea16c533d84d1e461804da20fe8e66e548e2078b78adb49e4eb7eb15c75ab957a432da100", common.ToHex(lastSignTx.GasPayerSigs()[0]))
	rlpdata, err := rlp.EncodeToBytes(lastSignTx)
	assert.NoError(t, err)
	assert.Equal(t, "0xf90118800181c8940107134b9cdd7d89f83efa6175f9b3552f29094c9401989568a87e92e82a609891bd9de3d7f22e16289400000000000000000000000000000000000000028003820d058002b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838845c107d9480f843b8411f951f2f3d7c3fa558a928e0e546b2a1d2b2e687cedd35fd77d730f6c62c71c21899f23a3e07ea1647c0d9fa00d10f3cde622ce2ecd321dfbb029c6bc7d471de00f843b841ca1d67ca4f9c6e1cef1e8b918beb18f0a34d1604b6a249bee79e213bdea16c533d84d1e461804da20fe8e66e548e2078b78adb49e4eb7eb15c75ab957a432da100", common.ToHex(rlpdata))
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
	assert.Equal(t, "0xf90144800181c8940107134b9cdd7d89f83efa6175f9b3552f29094c940107134b9cdd7d89f83efa6175f9b3552f29094c941000000000000000000000000000000000000000b83c3838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838389e11111111111111111111111111111111111111111111111111111111111164809e111111111111111111111111111111111111111111111111111111111111b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838845c107d94b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838c0c0", common.ToHex(txb))
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
	assert.Equal(t, "0xf90188800181c8940107134b9cdd7d89f83efa6175f9b3552f29094c940107134b9cdd7d89f83efa6175f9b3552f29094c941000000000000000000000000000000000000000b83c3838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838389e11111111111111111111111111111111111111111111111111111111111164809e111111111111111111111111111111111111111111111111111111111111b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838845c107d94b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838f843b841ef73ebcc558f2e3ab791e9f81100abc3d3b17aa89d65d3915398ba090788b9680dfcb343be4519fd9a9199d6c0bbc6ebfef686722c2a63e507637a09ce4cbcd601c0", common.ToHex(txb))
	result = Transaction{}
	err = rlp.DecodeBytes(txb, &result)
	assert.NoError(t, err)
	assert.Equal(t, txV.Sigs(), result.Sigs())
	assert.Equal(t, txV.data.Hash, result.data.Hash)
}

func TestTransaction_Hash(t *testing.T) {
	// hash without signature
	assert.Equal(t, common.HexToHash("0xcb624bd921214763e1fe7fdbe7f573eff66575e4649916cac12f88775abfa0f7"), testTx.Hash())

	// hash for sign
	h := testSigner.Hash(testTx)
	assert.Equal(t, common.HexToHash("0x12c59cd1ba635a8a673e2276c870c912ce82a1157fae8dbed651a711682c260b"), h)

	// hash with signature
	sig, err := crypto.Sign(h[:], testPrivate)
	assert.NoError(t, err)
	txV := testTx.Clone()
	txV.data.Sigs = append(txV.data.Sigs, sig)
	assert.Equal(t, common.HexToHash("0x32b1f4bfe3193a04539c5f4850d503a4f7dcd484d29dc088b511ce19e2554ff4"), txV.Hash())
}
func TestReimbursementTransaction(t *testing.T) {
	// without sign
	assert.Equal(t, "0x2a39026fb2ca9e5bf5e9c966a7f481e2e17d485f9bb65c04d07e8e3d5a1fe579", reimbursementTx.Hash().String())
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
	assert.Equal(t, "0x83497d3a0cdb4b0b22d45f28f490ccb3b7e964cc7d6c2215b1ccd0173ffd324a", txW.Hash().String())
}

func TestTransaction_MarshalJSON_UnmarshalJSON(t *testing.T) {

	txV, err := testSigner.SignTx(testTx, testPrivate)
	assert.NoError(t, err)
	data, err := json.Marshal(txV)
	assert.NoError(t, err)
	assert.Equal(t, `{"type":"0","version":"1","chainID":"200","from":"Lemo836BQKCBZ8Z7B7N4G4N4SNGBT24ZZSJQD24D","gasPayer":"Lemo836BQKCBZ8Z7B7N4G4N4SNGBT24ZZSJQD24D","to":"Lemo8888888888888888888888888888888888BW","toName":"aa","gasPrice":"2","gasLimit":"100","gasUsed":"0","amount":"1","data":"0x0c","expirationTime":"1544584596","message":"aaa","sigs":["0xdf998b9e98778c5b54e9c8c104b9841c34979f81ffa3c0bdac6c2d63a8b85e1001e6841ea0e9bbe43da601e81a5817a9b0a95b0e2a156d42f0863bb26aff2d0c00"],"hash":"0x32b1f4bfe3193a04539c5f4850d503a4f7dcd484d29dc088b511ce19e2554ff4","gasPayerSigs":[]}`, string(data))
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
	assert.Equal(t, `{"type":"0","version":"1","chainID":"200","from":"Lemo836BQKCBZ8Z7B7N4G4N4SNGBT24ZZSJQD24D","gasPayer":"Lemo83S6HTWD6TW3KWFF465YSTJ7PJ7JNPBTYYGD","to":"Lemo8888888888888888888888888888888888QR","toName":"","gasPrice":"2","gasLimit":"2222","gasUsed":"0","amount":"2","data":"0x383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838","expirationTime":"1544584596","message":"","sigs":["0x1f951f2f3d7c3fa558a928e0e546b2a1d2b2e687cedd35fd77d730f6c62c71c21899f23a3e07ea1647c0d9fa00d10f3cde622ce2ecd321dfbb029c6bc7d471de00"],"hash":"0x619b93ac7694a5563441d3bc3cd8b2c22504db5b45ec81f7d256834ed44f7767","gasPayerSigs":["0xed9efa8a0977a5870417ca3b1a9c7f69a5886e55f3d03dd77ae05e08cbea5c9249aff8e8255b325a4508d686416aec65cc7380467f66ae85784a8c379ad9e55501"]}`, string(data02))
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
	assert.Equal(t, `{"type":"0","version":"1","chainID":"200","from":"Lemo836BQKCBZ8Z7B7N4G4N4SNGBT24ZZSJQD24D","gasPayer":"Lemo836BQKCBZ8Z7B7N4G4N4SNGBT24ZZSJQD24D","to":"Lemo8P6Y24SZ2JPY7AFQD4HJWWRQ6DJ6TW2Y9CCF","toName":"888888888888888888888888888888888888888888888888888888888888","gasPrice":"117789804318558955305553166716194567721832259791707930541440413419507985","gasLimit":"100","gasUsed":"0","amount":"117789804318558955305553166716194567721832259791707930541440413419507985","data":"0x383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838","expirationTime":"1544584596","message":"888888888888888888888888888888888888888888888888888888888888","sigs":["0xef73ebcc558f2e3ab791e9f81100abc3d3b17aa89d65d3915398ba090788b9680dfcb343be4519fd9a9199d6c0bbc6ebfef686722c2a63e507637a09ce4cbcd601"],"hash":"0x96f755393e975f932a613b88a47decf1eeba387ef377cd4b175e394e660053a6","gasPayerSigs":[]}`, string(data))
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
