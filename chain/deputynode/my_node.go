package deputynode

import (
	"crypto/ecdsa"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
)

var (
	selfNodeKey *ecdsa.PrivateKey
)

func GetSelfNodeKey() *ecdsa.PrivateKey {
	return selfNodeKey
}

func GetSelfNodeID() []byte {
	// TODO cache it
	return (crypto.FromECDSAPub(&selfNodeKey.PublicKey))[1:]
}

func SetSelfNodeKey(key *ecdsa.PrivateKey) {
	selfNodeKey = key
}
