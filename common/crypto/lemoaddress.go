package crypto

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
)

type AddressKeyPair struct {
	// NativePubkey string
	LemoAddress string
	PublicKey   string
	PrivateKey  string
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
		// NativePubkey: common.ToHex(address.Bytes()),
		LemoAddress: lemoAddress,
		PublicKey:   common.ToHex(publicToBytes[1:]),
		PrivateKey:  common.ToHex(privateToBytes),
	}, nil
}
