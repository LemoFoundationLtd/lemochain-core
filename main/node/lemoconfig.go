package node

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var DefaultConfig = LemoConfig{
	NetworkId: 1,
	MaxPeers:  1000,
	Port:      60001,
}

//go:generate gencodec -type LemoConfig -field-override configMarshaling -formats toml -out gen_config.go

type LemoConfig struct {
	Genesis   *chain.Genesis `toml:",omitempty"`
	NetworkId uint64
	MaxPeers  int
	Port      int
	NodeKey   string
	ExtraData []byte `toml:",omitempty"`
}

type configMarshaling struct {
	ExtraData hexutil.Bytes
}
