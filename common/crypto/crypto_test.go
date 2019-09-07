package crypto

import (
	"crypto/ecdsa"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

var testAddrHex = "015ef7c6d6a3077b2531e72b08e55516265d451a"
var testPrivHex = "356e2257d4538e37fb9637507e474309d61335395f21a8b9ab6df88fd5232865"

// These tests are sanity checks.
// They should ensure that we don't e.g. use Sha3-224 instead of Sha3-256
// and that the sha3 library uses keccak-f permutation.
func TestKeccak256Hash(t *testing.T) {
	msg := []byte("abc")
	exp := "0x4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45"
	assert.Equal(t, exp, Keccak256Hash(msg).Hex())
}

func TestToECDSAErrors(t *testing.T) {
	_, err := HexToECDSA("0000000000000000000000000000000000000000000000000000000000000000")
	assert.Equal(t, fmt.Errorf("invalid private key, zero or negative"), err)
	_, err = HexToECDSA("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	assert.Equal(t, fmt.Errorf("invalid private key, >=N"), err)
}

func BenchmarkSha3(b *testing.B) {
	a := []byte("hello world")
	for i := 0; i < b.N; i++ {
		Keccak256(a)
	}
}

func TestSign(t *testing.T) {
	key, _ := HexToECDSA(testPrivHex)
	addr := common.HexToAddress(testAddrHex)

	msg := Keccak256([]byte("foo"))
	sig, err := Sign(msg, key)
	assert.NoError(t, err)
	recoveredPub, err := Ecrecover(msg, sig)
	assert.NoError(t, err)
	pubKey := ToECDSAPub(recoveredPub)
	recoveredAddr := PubkeyToAddress(*pubKey)
	assert.Equal(t, addr, recoveredAddr)

	// should be equal to SigToPub
	recoveredPub2, err := SigToPub(msg, sig)
	assert.NoError(t, err)
	recoveredAddr2 := PubkeyToAddress(*recoveredPub2)
	assert.Equal(t, addr, recoveredAddr2)
}

func TestInvalidSign(t *testing.T) {
	_, err := Sign(make([]byte, 1), nil)
	assert.Equal(t, fmt.Errorf("hash is required to be exactly 32 bytes (1)"), err)
	_, err = Sign(make([]byte, 33), nil)
	assert.Equal(t, fmt.Errorf("hash is required to be exactly 32 bytes (33)"), err)
}

func TestNewContractAddress(t *testing.T) {
	key, _ := HexToECDSA(testPrivHex)
	addr := common.HexToAddress(testAddrHex)
	genAddr := PubkeyToAddress(key.PublicKey)
	// sanity check before using addr to create contract address
	assert.Equal(t, genAddr, addr)

	caddr0 := CreateContractAddress(addr, common.HexToHash("0"))
	caddr1 := CreateContractAddress(addr, common.HexToHash("1"))
	caddr2 := CreateContractAddress(addr, common.HexToHash("2"))

	assert.Equal(t, common.HexToAddress("0x02208Cc79767dD9559EE43A673daA13f0eaE2737"), caddr0)
	assert.Equal(t, common.HexToAddress("0x022A6a88F9cfa542A805d2F6941a4c9faf793957"), caddr1)
	assert.Equal(t, common.HexToAddress("0x028CadC3967854c060268084256A9c8A7C1B7c20"), caddr2)
}

func TestLoadECDSAFile(t *testing.T) {
	keyBytes := common.FromHex(testPrivHex)
	fileName0 := "test_key0"
	fileName1 := "test_key1"
	checkKey := func(k *ecdsa.PrivateKey) {
		assert.Equal(t, PubkeyToAddress(k.PublicKey), common.HexToAddress(testAddrHex))
		loadedKeyBytes := FromECDSA(k)
		assert.Equal(t, keyBytes, loadedKeyBytes)
	}

	ioutil.WriteFile(fileName0, []byte(testPrivHex), 0600)
	defer os.Remove(fileName0)

	key0, err := LoadECDSA(fileName0)
	assert.NoError(t, err)
	checkKey(key0)

	// again, this time with SaveECDSA instead of manual save:
	err = SaveECDSA(fileName1, key0)
	assert.NoError(t, err)
	defer os.Remove(fileName1)

	key1, err := LoadECDSA(fileName1)
	assert.NoError(t, err)
	checkKey(key1)
}

func TestValidateSignatureValues(t *testing.T) {
	check := func(expected bool, v byte, r, s *big.Int) {
		if ValidateSignatureValues(v, r, s) != expected {
			assert.Equal(t, expected, ValidateSignatureValues(v, r, s), "mismatch for v: %d r: %d s: %d want: %v", v, r, s, expected)
		}
	}
	minusOne := big.NewInt(-1)
	one := common.Big1
	zero := common.Big0
	secp256k1nMinus1 := new(big.Int).Sub(secp256k1_N, common.Big1)

	// correct v,r,s
	check(true, 0, one, one)
	check(true, 1, one, one)
	// incorrect v, correct r,s,
	check(false, 2, one, one)
	check(false, 3, one, one)

	// incorrect v, combinations of incorrect/correct r,s at lower limit
	check(false, 2, zero, zero)
	check(false, 2, zero, one)
	check(false, 2, one, zero)
	check(false, 2, one, one)

	// correct v for any combination of incorrect r,s
	check(false, 0, zero, zero)
	check(false, 0, zero, one)
	check(false, 0, one, zero)

	check(false, 1, zero, zero)
	check(false, 1, zero, one)
	check(false, 1, one, zero)

	// correct sig with max r,s
	check(false, 0, secp256k1nMinus1, secp256k1nMinus1)
	// correct v, combinations of incorrect r,s at upper limit
	check(false, 0, secp256k1_N, secp256k1nMinus1)
	check(false, 0, secp256k1nMinus1, secp256k1_N)
	check(false, 0, secp256k1_N, secp256k1_N)

	// current callers ensures r,s cannot be negative, but let's test for that too
	// as crypto package could be used stand-alone
	check(false, 0, minusOne, one)
	check(false, 0, one, minusOne)
}

func TestCreateAddress(t *testing.T) {
	// keypair1
	priKey, err := GenerateKey()
	assert.NoError(t, err)
	pubKey := priKey.PublicKey
	addr1 := PubkeyToAddress(pubKey)
	pribytes1 := FromECDSA(priKey)
	pubbytes1 := FromECDSAPub(&pubKey)

	// sign and recover test
	msg := Keccak256([]byte("foo"))
	sig, err := Sign(msg, priKey)
	assert.NoError(t, err)
	recoveredPub, err := Ecrecover(msg, sig)
	assert.NoError(t, err)
	assert.Equal(t, pubKey, *ToECDSAPub(recoveredPub))

	// keypair2
	priKey, err = GenerateKey()
	assert.NoError(t, err)
	pubKey = priKey.PublicKey
	addr2 := PubkeyToAddress(pubKey)
	pribytes2 := FromECDSA(priKey)
	pubbytes2 := FromECDSAPub(&pubKey)
	// they are different
	assert.NotEqual(t, addr1, addr2)
	assert.NotEqual(t, pribytes1, pribytes2)
	assert.NotEqual(t, pubbytes1, pubbytes2)
}

func TestCreateTempAddress(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i <= 20; i++ {
		// 随机address
		var hex = "0x"
		for len(hex) <= 42 {
			hex = hex + strconv.Itoa(rand.Intn(10))
		}
		creator := common.HexToAddress(hex)
		// 随机user id
		userId := [10]byte{}
		for i := 0; i < len(userId); i++ {
			userId[i] = uint8(rand.Int())
		}
		tempAddr := CreateTempAddress(creator, userId)
		assert.Equal(t, common.TempAddressType, uint8(tempAddr[0]))
		assert.Equal(t, userId[:], tempAddr[10:])
		assert.Equal(t, creator[11:], tempAddr[1:10])
	}
}
