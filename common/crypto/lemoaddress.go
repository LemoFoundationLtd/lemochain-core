package crypto

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/base26"
)

// Lemo logo
const logo = "Lemo"

type AddressKeyPair struct {
	NativePubkey string
	LemoAddress  string
	PublicKey    string
	PrivateKet   string
}

// getCheckSum get the check digit by doing an exclusive OR operation
func getCheckSum(address common.Address) byte {
	// Conversion Address type
	addressToBytes := address.Bytes()
	var temp = addressToBytes[0]
	for _, c := range addressToBytes {
		temp ^= c
	}
	return temp
}

// GenerateAddress generate Lemo address
func GenerateAddress() *AddressKeyPair {
	// Get privateKey
	privKey, err := GenerateKey()
	if err != nil {
		return nil
	}
	// Get the public key through the private key
	pubKey := privKey.PublicKey
	// Get the address(Address) through the public key
	address := PubkeyToAddress(pubKey)
	// Get check digit
	checkSum := getCheckSum(address)
	// Stitching the check digit at the end
	fullPayload := append(address.Bytes(), checkSum)
	// base26 encoding
	bytesAddress := base26.Encode(fullPayload)
	// Add logo at the top
	lemoAddress := append([]byte(logo), bytesAddress...)
	// Get PublicKey through the PrivateKey
	public := privKey.PublicKey
	// PublicKey type is converted to bytes type
	publicToBytes := FromECDSAPub(&public)
	// PrivateKey type is converted to bytes type
	privateToBytes := FromECDSA(privKey)
	return &AddressKeyPair{
		NativePubkey: common.ToHex(address.Bytes()),
		LemoAddress:  string(lemoAddress),
		PublicKey:    common.ToHex(publicToBytes[1:]),
		PrivateKet:   common.ToHex(privateToBytes),
	}
}

// RestoreOriginalAddress Restore original address the LemoAddress and return the Address type.
func RestoreOriginalAddress(LemoAddress string) common.Address {
	// Remove logo
	address := []byte(LemoAddress)[4:]
	// Base26 decoding
	fullPayload := base26.Decode(address)
	// Get the native address
	BytesAddress := fullPayload[:len(fullPayload)-1]
	nativeAddress := common.BytesToAddress(BytesAddress)

	return nativeAddress
}
