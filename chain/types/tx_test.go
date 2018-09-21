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
	chainId        uint16 = 1
	testSigner            = MakeSigner(TxVersion, chainId)
	testPrivate, _        = crypto.HexToECDSA("ddda62423014e1811a6bb3bae3791baf5dbbdd30a2b38938ecfe2aaaf9da6f21")
	testAddr              = crypto.PubkeyToAddress(testPrivate.PublicKey)

	testTx = NewTransaction(common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 200, big.NewInt(1544584596), "aa", []byte{34})
)

func ExpirationFromNow() *big.Int {
	return big.NewInt(time.Now().Unix() + DefaultTTTL)
}

func TestNewTransaction(t *testing.T) {
	expiration := ExpirationFromNow()
	tx := NewTransaction(common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 200, expiration, "aa", nil)
	assert.Equal(t, uint8(0), tx.Type())
	assert.Equal(t, TxVersion, tx.Version())
	assert.Equal(t, uint16(200), tx.ChainId())
	assert.Equal(t, big.NewInt(0x100c8), tx.data.V)
	assert.Equal(t, common.HexToAddress("0x1"), *tx.To())
	assert.Equal(t, "aa", tx.ToName())
	assert.Equal(t, common.Big2, tx.GasPrice())
	assert.Equal(t, common.Big1, tx.Value())
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
	assert.Equal(t, big.NewInt(0x100c8), tx.data.V)
	assert.Empty(t, tx.To())
	assert.Equal(t, "aa", tx.ToName())
	assert.Equal(t, common.Big2, tx.GasPrice())
	assert.Equal(t, common.Big1, tx.Value())
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

func TestTransaction_EncodeRLP_DecodeRLP(t *testing.T) {
	txb, err := rlp.EncodeToBytes(testTx)
	assert.NoError(t, err)
	assert.Equal(t, common.FromHex("0xe89400000000000000000000000000000000000000018261610201640c845c107d9422830200c88080"), txb)
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
	assert.Equal(t, common.FromHex("0xf8689400000000000000000000000000000000000000018261610201640c845c107d9422830300c8a050b5d3bd90aafc3c6a2a4c9432645ce03fa1e0e567d17078f60615987fd0a5aca0225b7de28bb426617c2d461cd6c2300ec3c6bfc8cd23a16693bdc1dba22ae56f"), txb)
	result = Transaction{}
	err = rlp.DecodeBytes(txb, &result)
	assert.NoError(t, err)
	assert.Equal(t, txV.data.V, result.data.V)
	assert.Equal(t, txV.data.R, result.data.R)
	assert.Equal(t, txV.data.S, result.data.S)
	assert.Equal(t, txV.data.Hash, result.data.Hash)
}

func TestTransaction_Hash(t *testing.T) {
	assert.Equal(t, common.HexToHash("0x63769cf42a8ba67c452ec54cd805258cafb4bc502a166aff12654132c7d47e9d"), testTx.Hash())
}

func TestTransaction_MarshalJSON_UnmarshalJSON(t *testing.T) {
	txV, err := SignTx(testTx, testSigner, testPrivate)
	assert.NoError(t, err)
	data, err := json.Marshal(txV)
	assert.NoError(t, err)
	assert.Equal(t, "7b22746f223a22307830303030303030303030303030303030303030303030303030303030303030303030303030303031222c22746f4e616d65223a22307836313631222c226761735072696365223a22307832222c2276616c7565223a22307831222c22676173223a2230783634222c2264617461223a2230783063222c2265787069726174696f6e54696d65223a2230783563313037643934222c226d657373616765223a2230783232222c2276223a2230783330306338222c2272223a22307835306235643362643930616166633363366132613463393433323634356365303366613165306535363764313730373866363036313539383766643061356163222c2273223a22307832323562376465323862623432363631376332643436316364366332333030656333633662666338636432336131363639336264633164626132326165353666222c2268617368223a22307864303638346663646434663634363164306433373935346135663466383932633931663738313835313135393766353262663862363437636137636436616136227d", common.Bytes2Hex(data))
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
