package types

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"math/big"
)

var (
	DigAssetTypeToken = 1
	DigAssetTypeAsset = 2
)

type DigAsset struct {
	Type        int
	Token       common.Token
	Decimals    int
	TotalSupply *big.Int
	Mineable    bool
	Issuer      common.Address
	Profile     Profile
}

func (asset *DigAsset) Clone() *DigAsset {
	if asset == nil {
		return nil
	}

	result := &DigAsset{
		Type:        asset.Type,
		Token:       asset.Token,
		Decimals:    asset.Decimals,
		TotalSupply: new(big.Int).Set(asset.TotalSupply),
		Mineable:    asset.Mineable,
		Issuer:      asset.Issuer,
	}

	result.Profile = make(Profile)
	if len(asset.Profile) <= 0 {
		return result
	} else {
		for k, v := range asset.Profile {
			result.Profile[k] = v
		}
		return result
	}
}
