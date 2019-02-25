package types

import (
	"encoding/json"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
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
	assert.Equal(t, uint8(0), tx.Type())
	assert.Equal(t, TxVersion, tx.Version())
	assert.Equal(t, uint16(200), tx.ChainID())
	assert.Equal(t, big.NewInt(0x200c8), tx.data.V)
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
	assert.Equal(t, uint8(0), tx.Type())
	assert.Equal(t, TxVersion, tx.Version())
	assert.Equal(t, uint16(200), tx.ChainID())
	assert.Equal(t, big.NewInt(0x200c8), tx.data.V)
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
	txV, err := testTx.WithSignature(testSigner, sig)
	assert.NoError(t, err)
	from, err := txV.From()
	assert.NoError(t, err)
	assert.Equal(t, testAddr, from)
	V, R, S := txV.Raw()
	assert.NotEmpty(t, V)
	assert.NotEmpty(t, R)
	assert.NotEmpty(t, S)
}

func TestTransaction_WithSignature_From_Raw_bigTx(t *testing.T) {
	h := testSigner.Hash(testTxBig)
	sig, err := crypto.Sign(h[:], testPrivate)
	assert.NoError(t, err)
	txV, err := testTxBig.WithSignature(testSigner, sig)
	assert.NoError(t, err)
	from, err := txV.From()
	assert.NoError(t, err)
	assert.Equal(t, testAddr, from)
	V, R, S := txV.Raw()
	assert.NotEmpty(t, V)
	assert.NotEmpty(t, R)
	assert.NotEmpty(t, S)
}

func TestTransaction_EncodeRLP_DecodeRLP(t *testing.T) {
	txb, err := rlp.EncodeToBytes(testTx)
	assert.NoError(t, err)
	assert.Equal(t, "0xec9400000000000000000000000000000000000000018261610264010c845c107d9483616161830200c8808080", common.ToHex(txb))
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
	assert.Equal(t, testTx.data.V, result.data.V)
	assert.Equal(t, testTx.data.R, result.data.R)
	assert.Equal(t, testTx.data.S, result.data.S)
	assert.Equal(t, testTx.data.Hash, result.data.Hash)
	assert.Equal(t, testTx.GasPayerSig(), result.GasPayerSig())

	// with signature
	txV, err := testSigner.SignTx(testTx, testPrivate)
	assert.NoError(t, err)
	txb, err = rlp.EncodeToBytes(txV)
	assert.NoError(t, err)
	assert.Equal(t, "0xf86c9400000000000000000000000000000000000000018261610264010c845c107d9483616161830300c8a08c0499083cb3d27bead4f21994aeebf8e75fa11df6bfe01c71cad583fc9a3c70a0778a437607d072540719a866adb630001fabbfb6b032d1a8dfbffac7daed8f0280", common.ToHex(txb))
	result = Transaction{}
	err = rlp.DecodeBytes(txb, &result)
	assert.NoError(t, err)
	assert.Equal(t, txV.data.V, result.data.V)
	assert.Equal(t, txV.data.R, result.data.R)
	assert.Equal(t, txV.data.S, result.data.S)
	assert.Equal(t, txV.data.Hash, result.data.Hash)
	from, err := txV.From()
	assert.NoError(t, err)
	assert.Equal(t, testAddr, from)
	payer, err := txV.GasPayer()
	assert.NoError(t, err)
	assert.Equal(t, testAddr, payer)
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
	assert.Equal(t, reimbursementTx.data.V, result02.data.V)
	assert.Equal(t, reimbursementTx.data.R, result02.data.R)
	assert.Equal(t, reimbursementTx.data.S, result02.data.S)
	assert.Equal(t, reimbursementTx.data.Hash, result02.data.Hash)
	assert.Equal(t, reimbursementTx.GasPayerSig(), result02.GasPayerSig())

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
	assert.Equal(t, "0xf8e89400000000000000000000000000000000000000028003820d0502b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838845c107d9480830300c8a0470024fba0158a446082242da8e0b97e6898d0605e1bb62627c3c703dd4a90b9a03ad5767e3cea6084a2036f1580815ea9bca594b3dc7c8a14ac06b45e6740cc5cb841178516469e49899aeb2e97572d1ebd4576b45d32767c94bfdcedc0511ff89a58158e13736a7bffecf233c86e439e595081676e6ee1ab8696b5e28cbd34806c6901", common.ToHex(rlpdata))
	recovered := Transaction{}
	err = rlp.DecodeBytes(rlpdata, &recovered)
	assert.NoError(t, err)
	assert.Equal(t, lastSignTx.Data(), recovered.Data())
	assert.Equal(t, lastSignTx.data.S, recovered.data.S)
	assert.Equal(t, lastSignTx.data.R, recovered.data.R)
	assert.Equal(t, lastSignTx.data.V, recovered.data.V)
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
	assert.Equal(t, "0xf9011a941000000000000000000000000000000000000000b83c3838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838389e111111111111111111111111111111111111111111111111111111111111649e111111111111111111111111111111111111111111111111111111111111b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838845c107d94b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838830200c8808080", common.ToHex(txb))
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
	assert.Equal(t, testTxBig.data.V, result.data.V)
	assert.Equal(t, testTxBig.data.R, result.data.R)
	assert.Equal(t, testTxBig.data.S, result.data.S)
	assert.Equal(t, testTxBig.data.Hash, result.data.Hash)

	// with signature
	txV, err := testSigner.SignTx(testTxBig, testPrivate)
	assert.NoError(t, err)
	txb, err = rlp.EncodeToBytes(txV)
	assert.NoError(t, err)
	assert.Equal(t, "0xf9015a941000000000000000000000000000000000000000b83c3838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838389e111111111111111111111111111111111111111111111111111111111111649e111111111111111111111111111111111111111111111111111111111111b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838845c107d94b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838830300c8a0956da7af3519f957e1281e8f41735aa965e6a712e828856fe95bdbd31c787d76a00f235ae3ed6ff4ea8198d2528b4561b68544c5691de58aa974b7b6cc791f78d580", common.ToHex(txb))
	result = Transaction{}
	err = rlp.DecodeBytes(txb, &result)
	assert.NoError(t, err)
	assert.Equal(t, txV.data.V, result.data.V)
	assert.Equal(t, txV.data.R, result.data.R)
	assert.Equal(t, txV.data.S, result.data.S)
	assert.Equal(t, txV.data.Hash, result.data.Hash)
}

func TestTransaction_Hash(t *testing.T) {
	// hash without signature
	assert.Equal(t, common.HexToHash("0x1e5db63bf78744e4e8da72421954f327c2bc34af613a926007467c4fb6d92c4c"), testTx.Hash())

	// hash for sign
	h := testSigner.Hash(testTx)
	assert.Equal(t, common.HexToHash("0x81f8b8f725a9342a9ad85f31d2a6009afd52e43c5f86199a2089a32ea81913e6"), h)

	// hash with signature
	sig, err := crypto.Sign(h[:], testPrivate)
	assert.NoError(t, err)
	txV, err := testTx.WithSignature(testSigner, sig)
	assert.NoError(t, err)
	assert.Equal(t, common.HexToHash("0x4c1507aaa69df86e14fcc6be7de9c19ad95d3135c951c1038b99c31ece8fc24c"), txV.Hash())
}
func TestReimbursementTransaction(t *testing.T) {
	// without sign
	assert.Equal(t, "0x8a04fe4474f67d1f9c65bd62e7fbf09b3916385cc031b3fc59e80130f4881dc8", reimbursementTx.Hash().String())
	// two times sign
	h := MakeReimbursementTxSigner().Hash(reimbursementTx)
	// first sign
	sigData, err := crypto.Sign(h[:], testPrivate)
	assert.NoError(t, err)
	txV, err := reimbursementTx.WithSignature(MakeReimbursementTxSigner(), sigData)
	assert.NoError(t, err)
	// last sign
	txV = GasPayerSignatureTx(txV, common.Big3, 3333)
	txW, err := MakeGasPayerSigner().SignTx(txV, gasPayerPrivate)
	assert.NoError(t, err)
	assert.Equal(t, "0x824b5f5060fa19026d8002ba24f032a1dde5056b0e4dd2f0852a339b16456110", txW.Hash().String())
}

func TestTransaction_MarshalJSON_UnmarshalJSON(t *testing.T) {
	txV, err := testSigner.SignTx(testTx, testPrivate)
	assert.NoError(t, err)
	data, err := json.Marshal(txV)
	assert.NoError(t, err)
	assert.Equal(t, "0x7b22746f223a224c656d6f383838383838383838383838383838383838383838383838383838383838383838384257222c22746f4e616d65223a226161222c226761735072696365223a2232222c226761734c696d6974223a22313030222c22616d6f756e74223a2231222c2264617461223a2230783063222c2265787069726174696f6e54696d65223a2231353434353834353936222c226d657373616765223a22616161222c2276223a2230783330306338222c2272223a22307838633034393930383363623364323762656164346632313939346165656266386537356661313164663662666530316337316361643538336663396133633730222c2273223a22307837373861343337363037643037323534303731396138363661646236333030303166616262666236623033326431613864666266666163376461656438663032222c2268617368223a22307834633135303761616136396466383665313466636336626537646539633139616439356433313335633935316331303338623939633331656365386663323463222c226761735061796572536967223a223078227d", common.ToHex(data))
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
	assert.Equal(t, "0x7b22746f223a224c656d6f383838383838383838383838383838383838383838383838383838383838383838385152222c22746f4e616d65223a22222c226761735072696365223a2232222c226761734c696d6974223a2232323232222c22616d6f756e74223a2232222c2264617461223a223078333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338222c2265787069726174696f6e54696d65223a2231353434353834353936222c226d657373616765223a22222c2276223a2230783330306338222c2272223a22307834373030323466626130313538613434363038323234326461386530623937653638393864303630356531626236323632376333633730336464346139306239222c2273223a22307833616435373637653363656136303834613230333666313538303831356561396263613539346233646337633861313461633036623435653637343063633563222c2268617368223a22307838616563316435383564303936653261376437613936646261636232343261653663373334376430393438656233306439383766633961333962383761396639222c226761735061796572536967223a22307832633965626364666132356664373461353461623834323537646135363764353239343933646563313032633435643162363362386463663535633337336165326236646430383334646538343962666361363336383661383462633433306165613134613730393835326666346533613262303663666264326262623763323031227d", common.ToHex(data02))
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
	assert.Equal(t, "0x7b22746f223a224c656d6f385036593234535a324a5059374146514434484a5757525136444a365457325939434346222c22746f4e616d65223a22383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838222c226761735072696365223a22313137373839383034333138353538393535333035353533313636373136313934353637373231383332323539373931373037393330353431343430343133343139353037393835222c226761734c696d6974223a22313030222c22616d6f756e74223a22313137373839383034333138353538393535333035353533313636373136313934353637373231383332323539373931373037393330353431343430343133343139353037393835222c2264617461223a223078333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338222c2265787069726174696f6e54696d65223a2231353434353834353936222c226d657373616765223a22383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838222c2276223a2230783330306338222c2272223a22307839353664613761663335313966393537653132383165386634313733356161393635653661373132653832383835366665393562646264333163373837643736222c2273223a223078663233356165336564366666346561383139386432353238623435363162363835343463353639316465353861613937346237623663633739316637386435222c2268617368223a22307862363633396265333163306235323837613932666165653664613930393734333562346136656235336433636132613335313434393933343835366134656331222c226761735061796572536967223a223078227d", common.ToHex(data))
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
