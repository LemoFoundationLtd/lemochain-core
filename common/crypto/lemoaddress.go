package crypto

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/base26"
)

type AddressKeyPair struct {
	NativePubkey string
	LemoAddress  string
	PublicKey    string
	PrivateKey   string
}

// GenerateAddress generate Lemo address
func GenerateAddress() (*AddressKeyPair, error) {
	// Get privateKey
	privKey, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	// Get the public key through the private key
	pubKey := privKey.PublicKey
	// Get the address(Address) through the public key
	address := PubkeyToAddress(pubKey)
	// get lemoAddress
	lemoAddress := address.String()

	// PublicKey type is converted to bytes type
	publicToBytes := FromECDSAPub(&pubKey)
	// PrivateKey type is converted to bytes type
	privateToBytes := FromECDSA(privKey)
	return &AddressKeyPair{
		NativePubkey: common.ToHex(address.Bytes()),
		LemoAddress:  lemoAddress,
		PublicKey:    common.ToHex(publicToBytes[1:]),
		PrivateKey:   common.ToHex(privateToBytes),
	}, nil
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
