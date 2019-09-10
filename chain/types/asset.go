package types

import (
	"encoding/json"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"

	"errors"
	"math/big"
	"strconv"
	"strings"
)

const (
	TokenAsset            = uint32(1)  // erc20
	NonFungibleAsset      = uint32(2)  // erc721
	CommonAsset           = uint32(3)  // erc20+721
	MaxAssetDecimal       = uint32(18) // 资产小数位最大值
	MaxMarshalAssetLength = 680
	MaxMetaDataLength     = 256
)

var (
	ErrAssetKind                     = errors.New("this type of asset does not exist")
	ErrTokenAssetDivisible           = errors.New("an asset of type 1 must be divisible")
	ErrNonFungibleAssetDivisible     = errors.New("an asset of type 2 must be indivisible")
	ErrNonFungibleAssetReplenishable = errors.New("an asset of type 2 must be non-replenishable")
	ErrCommonAssetDivisible          = errors.New("an asset of type 3 must be divisible")
	ErrAssetDecimal                  = errors.New("asset decimal must be less than 18")
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

// VerifyAsset 验证资产设置
func (asset *Asset) VerifyAsset() error {
	category := asset.Category
	isDivisible := asset.IsDivisible
	isReplenishable := asset.IsReplenishable
	decimal := asset.Decimal
	// 资产设置的小数位不能大于18位
	if decimal > MaxAssetDecimal {
		return ErrAssetDecimal
	}

	// 验证资产设置的分割和增发是否符合此类资产
	switch category {
	case TokenAsset:
		if !isDivisible {
			return ErrTokenAssetDivisible
		}
	case NonFungibleAsset:
		if isDivisible {
			return ErrNonFungibleAssetDivisible
		}
		if isReplenishable {
			return ErrNonFungibleAssetReplenishable
		}
	case CommonAsset:
		if !isDivisible {
			return ErrCommonAssetDivisible
		}
	default:
		return ErrAssetKind
	}
	return nil
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

// GetAsset
func GetAsset(txData []byte) (*Asset, error) {
	asset := &Asset{}
	if err := json.Unmarshal(txData, asset); err != nil {
		return nil, err
	}
	return asset, nil
}
