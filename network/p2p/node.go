package p2p

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"math/big"
)

const NodeIDBits = 512

// NodeID
type NodeID [NodeIDBits / 8]byte

// PubKey convert to public key
func (id NodeID) PubKey() (*ecdsa.PublicKey, error) {
	p := &ecdsa.PublicKey{Curve: crypto.S256(), X: new(big.Int), Y: new(big.Int)}
	half := len(id) / 2
	p.X.SetBytes(id[:half])
	p.Y.SetBytes(id[half:])
	if !p.Curve.IsOnCurve(p.X, p.Y) {
		return nil, errors.New("invalid secp256k1 curve point")
	}
	return p, nil
}

// String
func (id NodeID) String() string {
	return fmt.Sprintf("%x", id[:])
}

// PubKeyToNodeID returns a marshaled representation of the given public key.
func PubKeyToNodeID(pub *ecdsa.PublicKey) NodeID {
	var id NodeID
	pBytes := elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	if len(pBytes)-1 != len(id) {
		panic(fmt.Errorf("need %d bit pubkey, got %d bits", (len(id)+1)*8, len(pBytes)))
	}
	copy(id[:], pBytes[1:])
	return id
}

// BytesToNodeID convert bytes to NodeID
func BytesToNodeID(input []byte) *NodeID {
	if len(input) != 64 {
		return nil
	}
	r := NodeID{}
	copy(r[:], input)
	return &r
}
