package types

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"math/big"
)

type Asset struct {
	Category        int
	IsDivisible     bool
	AssetCode       common.Hash
	Decimals        int
	TotalSupply     *big.Int
	IsReplenishable bool
	Issuer          common.Address
	Profile         Profile
}

// type AssetExtend struct {
// 	MateData  map[common.Hash]string
// }

type AssetEquity struct {
	AssetCode common.Hash
	AssetId   common.Hash
	Equity    *big.Int
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
