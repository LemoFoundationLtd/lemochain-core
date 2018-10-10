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
	testAddr       = crypto.PubkeyToAddress(testPrivate.PublicKey)

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
	assert.Equal(t, big.NewInt(0x200c8), tx.data.V)
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
	assert.Equal(t, big.NewInt(0x200c8), tx.data.V)
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
	assert.Equal(t, "f8689400000000000000000000000000000000000000018261610201640c845c107d9422830300c8a03de7cbaaff085cfc1db7d1f31bea6819413d2391d9c5f81684faaeb9835df877a04727d43924a0eb18621076607211edd7062c413d1663f29eadda0b0ee3c467fe", common.Bytes2Hex(txb))
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
	assert.Equal(t, "7b22746f223a22307830303030303030303030303030303030303030303030303030303030303030303030303030303031222c22746f4e616d65223a22307836313631222c226761735072696365223a22307832222c2276616c7565223a22307831222c22676173223a2230783634222c2264617461223a2230783063222c2265787069726174696f6e54696d65223a2230783563313037643934222c226d657373616765223a2230783232222c2276223a2230783330306338222c2272223a22307833646537636261616666303835636663316462376431663331626561363831393431336432333931643963356638313638346661616562393833356466383737222c2273223a22307834373237643433393234613065623138363231303736363037323131656464373036326334313364313636336632396561646461306230656533633436376665222c2268617368223a22307863313937383334613433363638643461646238363937363535346563646234306431633565353264373037316263336364323739346632353633323835386631227d", common.Bytes2Hex(data))
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
