package types

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"math/big"
)

const (
	Asset01                = 1    // erc20
	Asset02                = 2    // erc721
	Asset03                = 3    // erc20+721
	MaxProfileStringLength = 1024 // max string length
	MaxMarshalAssetLength  = 2048
	MaxMetaDataLength      = 256
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

// 发行资产
type IssueAsset struct {
	AssetCode common.Hash
	MetaData  string // 用户传进来的数据
	Amount    *big.Int
}

// 增发资产
type ReplenishAsset struct {
	AssetCode common.Hash
	AssetId   common.Hash
	Amount    *big.Int
}

// 修改资产profile
type ModifyAssetInfo struct {
	AssetCode common.Hash
	Info      Profile
}

// 交易资产
type TradingAsset struct {
	AssetId common.Hash
	Value   *big.Int
	Input   []byte
}
