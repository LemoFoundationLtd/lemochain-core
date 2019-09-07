package deputynode

import (
	"crypto/ecdsa"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
)

var (
	selfNodeKey *ecdsa.PrivateKey
	selfNodeID  []byte
)

func GetSelfNodeKey() *ecdsa.PrivateKey {
	return selfNodeKey
}

func GetSelfNodeID() []byte {
	return selfNodeID
}

func SetSelfNodeKey(key *ecdsa.PrivateKey) {
	selfNodeKey = key
	selfNodeID = crypto.PrivateKeyToNodeID(selfNodeKey)
}
