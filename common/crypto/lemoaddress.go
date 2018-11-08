package crypto

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
)

//go:generate gencodec -type AccountKey -out gen_account_keys_json.go

type AccountKey struct {
	Private string `json:"private"`
	// Public  string `json:"public"`
	Address string `json:"address"`
}

// GenerateAddress generate Lemo address
func GenerateAddress() (*AccountKey, error) {
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
	// publicToBytes := FromECDSAPub(&pubKey)
	// PrivateKey type is converted to bytes type
	privateToBytes := FromECDSA(privKey)
	return &AccountKey{
		Private: common.ToHex(privateToBytes),
		// Public:   common.ToHex(publicToBytes[1:]),
		Address: lemoAddress,
	}, nil
}
