package p2p

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"math/big"
	"net"
	"time"
)

const NodeIDBits = 512

// RNodeID 节点唯一编码
type NodeID [NodeIDBits / 8]byte

// Node 一个网络节点
type Node struct {
	IP       net.IP // len 4 for IPv4 or 16 for IPv6
	UDP, TCP uint16 // port numbers
	ID       NodeID // the node's public key

	// Time when the node was added to the table.
	addedAt time.Time
}

// NewNode 创建一个新的网络节点
func NewNode(id NodeID, ip net.IP, udpPort, tcpPort uint16) *Node {
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}
	return &Node{
		IP:  ip,
		UDP: udpPort,
		TCP: tcpPort,
		ID:  id,
	}
}

// PubkeyID returns a marshaled representation of the given public key.
func PubkeyID(pub *ecdsa.PublicKey) NodeID {
	var id NodeID
	pbytes := elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	if len(pbytes)-1 != len(id) {
		panic(fmt.Errorf("need %d bit pubkey, got %d bits", (len(id)+1)*8, len(pbytes)))
	}
	copy(id[:], pbytes[1:])
	return id
}

// PubKey 根据NodeID获取公钥
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

// String 获取nodeid的string形式
func (id NodeID) String() string {
	return fmt.Sprintf("%x", id[:])
}

func ToNodeID(input []byte) *NodeID {
	if len(input) != 64 {
		return nil
	}
	r := NodeID{}
	copy(r[:], input)
	return &r
}
