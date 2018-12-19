package network

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
)

type ProtocolHandshake struct {
	ChainID      uint16
	GenesisHash  common.Hash
	NodeVersion  uint32
	LatestStatus LatestStatus
}

func (phs *ProtocolHandshake) Bytes() []byte {
	buf, err := rlp.EncodeToBytes(phs)
	if err != nil {
		return nil
	}
	return buf
}
