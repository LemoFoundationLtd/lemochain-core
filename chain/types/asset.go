package types

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"math/big"
	"strconv"
	"strings"
)

const (
	Asset01               = uint32(1) // erc20
	Asset02               = uint32(2) // erc721
	Asset03               = uint32(3) // erc20+721
	MaxMarshalAssetLength = 680
	MaxMetaDataLength     = 256
)

//go:generate gencodec -type Asset --field-override assetMarshaling -out gen_asset_json.go
type Asset struct {
	Category        uint32         `json:"category" gencodec:"required"`
	IsDivisible     bool           `json:"isDivisible" gencodec:"required"`
	AssetCode       common.Hash    `json:"assetCode"`
	Decimal         uint32         `json:"decimal" gencodec:"required"`
	TotalSupply     *big.Int       `json:"totalSupply"`
	IsReplenishable bool           `json:"isReplenishable" gencodec:"required"`
	Issuer          common.Address `json:"issuer"`
	Profile         Profile        `json:"profile"`
}

type assetMarshaling struct {
	Category    hexutil.Uint32
	Decimal     hexutil.Uint32
	TotalSupply *hexutil.Big10
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
	result := &AssetEquity{
		AssetCode: equity.AssetCode,
		AssetId:   equity.AssetId,
	}

	if equity.Equity == nil {
		result.Equity = new(big.Int)
	} else {
		result.Equity = new(big.Int).Set(equity.Equity)
	}

	return result
}

func (equity *AssetEquity) String() string {
	set := []string{
		fmt.Sprintf("AssetCode: %s", equity.AssetCode.String()),
		fmt.Sprintf("AssetId: %s", equity.AssetId.String()),
	}
	if equity.Equity == nil {
		set = append(set, fmt.Sprintf("Equity: 0"))
	} else {
		set = append(set, fmt.Sprintf("Equity: %s", equity.Equity.String()))
	}

	return fmt.Sprintf("{%s}", strings.Join(set, ", "))
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
	}

	result := &Asset{
		Category:        asset.Category,
		IsDivisible:     asset.IsDivisible,
		AssetCode:       asset.AssetCode,
		Decimal:         asset.Decimal,
		IsReplenishable: asset.IsReplenishable,
		Issuer:          asset.Issuer,
		Profile:         clone(asset.Profile),
	}

	if asset.TotalSupply == nil {
		result.TotalSupply = new(big.Int)
	} else {
		result.TotalSupply = new(big.Int).Set(asset.TotalSupply)
	}

	return result
}

func (asset *Asset) String() string {
	set := []string{
		fmt.Sprintf("Category: %d", asset.Category),
		fmt.Sprintf("IsDivisible: %s", strconv.FormatBool(asset.IsDivisible)),
		fmt.Sprintf("AssetCode: %s", asset.AssetCode.String()),
		fmt.Sprintf("Decimal: %d", asset.Decimal),
		fmt.Sprintf("Issuer: %s", asset.Issuer.String()),
		fmt.Sprintf("IsReplenishable: %s", strconv.FormatBool(asset.IsReplenishable)),
	}

	if asset.TotalSupply == nil {
		set = append(set, fmt.Sprintf("TotalSupply: 0"))
	} else {
		set = append(set, fmt.Sprintf("TotalSupply: %s", asset.TotalSupply.String()))
	}

	if len(asset.Profile) <= 0 {
		set = append(set, fmt.Sprintf("Profile: []"))
	} else {
		records := make([]string, 0, len(asset.Profile))
		for k, v := range asset.Profile {
			records = append(records, fmt.Sprintf("%s => %s", k, v))
		}
		set = append(set, fmt.Sprintf("Profiles: {%s}", strings.Join(records, ", ")))
	}

	return fmt.Sprintf("{%s}", strings.Join(set, ", "))
}
