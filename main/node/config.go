package node

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/common/math"
	"math/big"
)

//go:generate gencodec -type ChainConfigFile -field-override ChainConfigFileMarshaling -out gen_config_json.go

type ChainConfigFile struct {
	ChainID     *big.Int                `json:"chainID"     gencodec:"required"`
	SleepTime   uint64                  `json:"sleepTime"   gencodec:"required"`
	Timeout     uint64                  `json:"timeout"     gencodec:"required"`
	DeputyNodes []deputynode.DeputyNode `json:"deputyNodes" gencodec:"required"`
}

type ChainConfigFileMarshaling struct {
	ChainID   *math.HexOrDecimal256
	SleepTime math.HexOrDecimal64
	Timeout   math.HexOrDecimal64
}
