package types

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"

	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
)

var (
	testSigner     = MakeSigner()
	testPrivate, _ = crypto.HexToECDSA("432a86ab8765d82415a803e29864dcfc1ed93dac949abf6f95a583179f27e4bb") // secp256k1.V = 1
	testAddr       = crypto.PubkeyToAddress(testPrivate.PublicKey)                                         // 0x0107134b9cdd7d89f83efa6175f9b3552f29094c

	testTx = NewTransaction(common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 200, 1544584596, "aa", "aaa")

	bigNum, _ = new(big.Int).SetString("111111111111111111111111111111111111111111111111111111111111", 16)
	bigString = "888888888888888888888888888888888888888888888888888888888888"
	testTxBig = NewTransaction(common.HexToAddress("0x1000000000000000000000000000000000000000"), bigNum, 100, bigNum, []byte(bigString), 200, 1544584596, bigString, bigString)
)

func ExpirationFromNow() uint64 {
	return uint64(time.Now().Unix()) + DefaultTTTL
}

func TestNewTransaction(t *testing.T) {
	expiration := ExpirationFromNow()
	tx := NewTransaction(common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 200, expiration, "aa", "")
	assert.Equal(t, uint8(0), tx.Type())
	assert.Equal(t, TxVersion, tx.Version())
	assert.Equal(t, uint16(200), tx.ChainId())
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
	tx := NewContractCreation(common.Big1, 100, common.Big2, []byte{12}, 200, expiration, "aa", "")
	assert.Equal(t, uint8(0), tx.Type())
	assert.Equal(t, TxVersion, tx.Version())
	assert.Equal(t, uint16(200), tx.ChainId())
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
	assert.Equal(t, "0xeb9400000000000000000000000000000000000000018261610264010c845c107d9483616161830200c88080", common.ToHex(txb))
	result := Transaction{}
	err = rlp.DecodeBytes(txb, &result)
	assert.NoError(t, err)
	assert.Equal(t, testTx.Type(), result.Type())
	assert.Equal(t, testTx.Version(), result.Version())
	assert.Equal(t, testTx.ChainId(), result.ChainId())
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

	// with signature
	txV, err := SignTx(testTx, testSigner, testPrivate)
	assert.NoError(t, err)
	txb, err = rlp.EncodeToBytes(txV)
	assert.NoError(t, err)
	assert.Equal(t, "0xf86b9400000000000000000000000000000000000000018261610264010c845c107d9483616161830300c8a08c0499083cb3d27bead4f21994aeebf8e75fa11df6bfe01c71cad583fc9a3c70a0778a437607d072540719a866adb630001fabbfb6b032d1a8dfbffac7daed8f02", common.ToHex(txb))
	result = Transaction{}
	err = rlp.DecodeBytes(txb, &result)
	assert.NoError(t, err)
	assert.Equal(t, txV.data.V, result.data.V)
	assert.Equal(t, txV.data.R, result.data.R)
	assert.Equal(t, txV.data.S, result.data.S)
	assert.Equal(t, txV.data.Hash, result.data.Hash)
}

func TestTransaction_EncodeRLP_DecodeRLP_bigTx(t *testing.T) {
	txb, err := rlp.EncodeToBytes(testTxBig)
	assert.NoError(t, err)
	assert.Equal(t, "0xf90119941000000000000000000000000000000000000000b83c3838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838389e111111111111111111111111111111111111111111111111111111111111649e111111111111111111111111111111111111111111111111111111111111b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838845c107d94b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838830200c88080", common.ToHex(txb))
	result := Transaction{}
	err = rlp.DecodeBytes(txb, &result)
	assert.NoError(t, err)
	assert.Equal(t, testTxBig.Type(), result.Type())
	assert.Equal(t, testTxBig.Version(), result.Version())
	assert.Equal(t, testTxBig.ChainId(), result.ChainId())
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
	txV, err := SignTx(testTxBig, testSigner, testPrivate)
	assert.NoError(t, err)
	txb, err = rlp.EncodeToBytes(txV)
	assert.NoError(t, err)
	assert.Equal(t, "0xf90159941000000000000000000000000000000000000000b83c3838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838389e111111111111111111111111111111111111111111111111111111111111649e111111111111111111111111111111111111111111111111111111111111b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838845c107d94b83c383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838830300c8a0956da7af3519f957e1281e8f41735aa965e6a712e828856fe95bdbd31c787d76a00f235ae3ed6ff4ea8198d2528b4561b68544c5691de58aa974b7b6cc791f78d5", common.ToHex(txb))
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
	assert.Equal(t, common.HexToHash("0xa3cddd511dd5ca88b3f724e82e469d828cdc6dc6bb436e9e40a7ceb8cac94bdd"), testTx.Hash())

	// hash for sign
	h := testSigner.Hash(testTx)
	assert.Equal(t, common.HexToHash("0x81f8b8f725a9342a9ad85f31d2a6009afd52e43c5f86199a2089a32ea81913e6"), h)

	// hash with signature
	sig, err := crypto.Sign(h[:], testPrivate)
	assert.NoError(t, err)
	txV, err := testTx.WithSignature(testSigner, sig)
	assert.NoError(t, err)
	assert.Equal(t, common.HexToHash("0x2b6d49ea4bd64f4c94f7c0677566c867dbae50eb9d2142448a97b145baf4a277"), txV.Hash())
}

func TestTransaction_MarshalJSON_UnmarshalJSON(t *testing.T) {
	txV, err := SignTx(testTx, testSigner, testPrivate)
	assert.NoError(t, err)
	data, err := json.Marshal(txV)
	assert.NoError(t, err)
	assert.Equal(t, "0x7b22746f223a224c656d6f383838383838383838383838383838383838383838383838383838383838383838384257222c22746f4e616d65223a226161222c226761735072696365223a2232222c226761734c696d6974223a22313030222c22616d6f756e74223a2231222c2264617461223a2230783063222c2265787069726174696f6e54696d65223a2231353434353834353936222c226d657373616765223a22616161222c2276223a2230783330306338222c2272223a22307838633034393930383363623364323762656164346632313939346165656266386537356661313164663662666530316337316361643538336663396133633730222c2273223a22307837373861343337363037643037323534303731396138363661646236333030303166616262666236623033326431613864666266666163376461656438663032222c2268617368223a22307832623664343965613462643634663463393466376330363737353636633836376462616535306562396432313432343438613937623134356261663461323737227d", common.ToHex(data))
	var parsedTx *Transaction
	err = json.Unmarshal(data, &parsedTx)
	assert.NoError(t, err)
	assert.Equal(t, txV.Hash(), parsedTx.Hash())
}

func TestTransaction_MarshalJSON_UnmarshalJSON_bigTx(t *testing.T) {
	txV, err := SignTx(testTxBig, testSigner, testPrivate)
	assert.NoError(t, err)
	data, err := json.Marshal(txV)
	assert.NoError(t, err)
	assert.Equal(t, "0x7b22746f223a224c656d6f385036593234535a324a5059374146514434484a5757525136444a365457325939434346222c22746f4e616d65223a22383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838222c226761735072696365223a22313137373839383034333138353538393535333035353533313636373136313934353637373231383332323539373931373037393330353431343430343133343139353037393835222c226761734c696d6974223a22313030222c22616d6f756e74223a22313137373839383034333138353538393535333035353533313636373136313934353637373231383332323539373931373037393330353431343430343133343139353037393835222c2264617461223a223078333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338333833383338222c2265787069726174696f6e54696d65223a2231353434353834353936222c226d657373616765223a22383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838383838222c2276223a2230783330306338222c2272223a22307839353664613761663335313966393537653132383165386634313733356161393635653661373132653832383835366665393562646264333163373837643736222c2273223a223078663233356165336564366666346561383139386432353238623435363162363835343463353639316465353861613937346237623663633739316637386435222c2268617368223a22307862653931653962613837636239333231343463643362656530343536323063343261626632623639633062623565663564633732313764666231613831636331227d", common.ToHex(data))
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
	txType, version, secp256k1V, chainId := ParseV(big.NewInt(0x103ffff))
	assert.Equal(t, uint8(1), txType)
	assert.Equal(t, uint8(1), version)
	assert.Equal(t, uint8(1), secp256k1V)
	assert.Equal(t, uint16(0xffff), chainId)

	txType, version, secp256k1V, chainId = ParseV(big.NewInt(0))
	assert.Equal(t, uint8(0), txType)
	assert.Equal(t, uint8(0), version)
	assert.Equal(t, uint8(0), secp256k1V)
	assert.Equal(t, uint16(0), chainId)
}
