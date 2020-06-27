package network

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/network/p2p"
)

var (
	ErrInvalidNodeUri = errors.New("invalid node uri")
)

// VerifyLemoAddress check lemoAddress
func VerifyLemoAddress(lemoAddress string) bool {
	return common.CheckLemoAddress(lemoAddress)
}

// VerifyNode check string node (node = nodeID@IP:Port)
func VerifyNode(node string) error {
	nodeId, endpoint := p2p.ParseNodeString(node)
	if nodeId == nil || endpoint == "" {
		return ErrInvalidNodeUri
	}
	return nil
}
