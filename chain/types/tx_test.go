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

	testTx = NewTransaction(common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 200, 1544584596, "aa", []byte{34})

	bigNum, e = new(big.Int).SetString("111111111111111111111111111111111111111111111111111111111111", 16)
	bigBytes  = []byte("222222222222222222222222222222222222222222222222222222222222")
	testTxBig = NewTransaction(common.HexToAddress("0x1000000000000000000000000000000000000000"), bigNum, 100, bigNum, bigBytes, 200, 1544584596, string(bigBytes), bigBytes)
)

func ExpirationFromNow() uint64 {
	return uint64(time.Now().Unix()) + DefaultTTTL
}

func TestNewTransaction(t *testing.T) {
	expiration := ExpirationFromNow()
	tx := NewTransaction(common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 200, expiration, "aa", nil)
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
	tx := NewContractCreation(common.Big1, 100, common.Big2, []byte{12}, 200, expiration, "aa", nil)
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
	assert.Equal(t, common.FromHex("0xe89400000000000000000000000000000000000000018261610264010c845c107d9422830200c88080"), txb)
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
	assert.Equal(t, "0xf8689400000000000000000000000000000000000000018261610264010c845c107d9422830300c8a0158b80d695e7d543ddb3ae09ed89b0fdd0c9f72b95a96e5f2b5e67a4d6d71a88a02b893b663e36f997df1e3f489b98d001cf615ee1e32b3c28ce6364f5cc681d5c", common.ToHex(txb))
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
	assert.Equal(t, "0xf90119941000000000000000000000000000000000000000b83c3232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232329e111111111111111111111111111111111111111111111111111111111111649e111111111111111111111111111111111111111111111111111111111111b83c323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232845c107d94b83c323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232830200c88080", common.ToHex(txb))
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
	assert.Equal(t, "0xf90159941000000000000000000000000000000000000000b83c3232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232329e111111111111111111111111111111111111111111111111111111111111649e111111111111111111111111111111111111111111111111111111111111b83c323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232845c107d94b83c323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232323232830300c8a05742441ea798525c70282af6b42bd403d8195ae81c607dfa9ae1fd065b0d7353a0430ece3407d98f5b76328b3f6c537c6f588fef58fc5d11ba8433cdc59a5f080e", common.ToHex(txb))
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
	assert.Equal(t, common.HexToHash("0x0e1ed8d9733a08f1fcd859827f418ad78e1b402eb28813a55d31aba7d71aeea3"), testTx.Hash())

	// hash for sign
	h := testSigner.Hash(testTx)
	assert.Equal(t, common.HexToHash("0x9f79748da47a0c32d2d268a5cfbe3a2a7d6c29d1a2f0534f416f3d2157933808"), h)

	// hash with signature
	sig, err := crypto.Sign(h[:], testPrivate)
	assert.NoError(t, err)
	txV, err := testTx.WithSignature(testSigner, sig)
	assert.NoError(t, err)
	assert.Equal(t, common.HexToHash("0xd77585b7e14ed8b1133ccb80f12ff3f89e74c1dc646f0a3fce75ac528f3d1f88"), txV.Hash())
}

func TestTransaction_MarshalJSON_UnmarshalJSON(t *testing.T) {
	txV, err := SignTx(testTx, testSigner, testPrivate)
	assert.NoError(t, err)
	data, err := json.Marshal(txV)
	assert.NoError(t, err)
	assert.Equal(t, "0x7b22746f223a22307830303030303030303030303030303030303030303030303030303030303030303030303030303031222c22746f4e616d65223a22307836313631222c226761735072696365223a22307832222c226761734c696d6974223a2230783634222c22616d6f756e74223a22307831222c2264617461223a2230783063222c2265787069726174696f6e54696d65223a2230783563313037643934222c226d657373616765223a2230783232222c2276223a2230783330306338222c2272223a22307831353862383064363935653764353433646462336165303965643839623066646430633966373262393561393665356632623565363761346436643731613838222c2273223a22307832623839336236363365333666393937646631653366343839623938643030316366363135656531653332623363323863653633363466356363363831643563222c2268617368223a22307864373735383562376531346564386231313333636362383066313266663366383965373463316463363436663061336663653735616335323866336431663838227d", common.ToHex(data))
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
	assert.Equal(t, "0x7b22746f223a22307831303030303030303030303030303030303030303030303030303030303030303030303030303030222c22746f4e616d65223a223078333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332222c226761735072696365223a223078313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131222c226761734c696d6974223a2230783634222c22616d6f756e74223a223078313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131222c2264617461223a223078333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332222c2265787069726174696f6e54696d65223a2230783563313037643934222c226d657373616765223a223078333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332333233323332222c2276223a2230783330306338222c2272223a22307835373432343431656137393835323563373032383261663662343262643430336438313935616538316336303764666139616531666430363562306437333533222c2273223a22307834333065636533343037643938663562373633323862336636633533376336663538386665663538666335643131626138343333636463353961356630383065222c2268617368223a22307861363138373635303437666537386665303435326333306431376130633533653135333366393666333230353732356439333538386530616630393633363136227d", common.ToHex(data))
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
