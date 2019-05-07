package network

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/network/p2p"
)

// VerifyLemoAddress check lemoAddress
func VerifyLemoAddress(lemoAddress string) bool {
	return common.CheckLemoAddress(lemoAddress)
}

// VerifyNode check string node (node = nodeID@IP:Port)
func VerifyNode(node string) bool {
	nodeId, endpoint := p2p.ParseNodeString(node)
	if nodeId == nil || endpoint == "" {
		return false
	}
	return true
}
