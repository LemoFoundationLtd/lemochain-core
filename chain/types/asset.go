package types

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"math/big"
)

const (
	Asset01                = uint32(1) // erc20
	Asset02                = uint32(2) // erc721
	Asset03                = uint32(3) // erc20+721
	MaxProfileStringLength = 1024      // max string length
	MaxMarshalAssetLength  = 2048
	MaxMetaDataLength      = 256
)

type Asset struct {
	Category        uint32
	IsDivisible     bool
	AssetCode       common.Hash
	Decimals        uint32
	TotalSupply     *big.Int
	IsReplenishable bool
	Issuer          common.Address
	Profile         Profile
}

// type AssetExtend struct {
// 	MateData  map[common.Hash]string
// }
//go:generate gencodec -type AssetEquity --field-override assetEquityMarshaling -out gen_assetEquity_json.go
type AssetEquity struct {
	AssetCode common.Hash `json:"assetCode" gencodec:"required"`
	AssetId   common.Hash `json:"assetId" gencodec:"required"`
	Equity    *big.Int    `json:"equity" gencodec:"required"`
}
type assetEquityMarshaling struct {
	Equity *hexutil.Big10
}

func (equity *AssetEquity) Clone() *AssetEquity {
	return &AssetEquity{
		AssetCode: equity.AssetCode,
		AssetId:   equity.AssetId,
		Equity:    new(big.Int).Set(equity.Equity),
	}
}

func (asset *Asset) Clone() *Asset {
	clone := func(profile Profile) Profile {
		result := make(Profile)
		if len(profile) <= 0 {
			return result
		} else {
			for k, v := range profile {
				result[k] = v
			}

			return result
		}
	}

	if asset == nil {
		return nil
	} else {
		return &Asset{
			Category:        asset.Category,
			IsDivisible:     asset.IsDivisible,
			AssetCode:       asset.AssetCode,
			Decimals:        asset.Decimals,
			TotalSupply:     new(big.Int).Set(asset.TotalSupply),
			IsReplenishable: asset.IsReplenishable,
			Issuer:          asset.Issuer,
			Profile:         clone(asset.Profile),
		}
	}
}
