package types

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"math/big"
)

// tx data marshal unmarshal struct
// 发行资产
//go:generate gencodec -type IssueAsset --field-override issueAssetMarshaling -out gen_issueAsset_json.go
type IssueAsset struct {
	AssetCode common.Hash `json:"assetCode" gencodec:"required"`
	MetaData  string      `json:"metaData"` // 用户传进来的数据
	Amount    *big.Int    `json:"supplyAmount" gencodec:"required"`
}

type issueAssetMarshaling struct {
	Amount *hexutil.Big10
}

// 增发资产
//go:generate gencodec -type ReplenishAsset --field-override replenishAssetMarshaling -out gen_replenishAsset_json.go
type ReplenishAsset struct {
	AssetCode common.Hash `json:"assetCode" gencodec:"required"`
	AssetId   common.Hash `json:"assetId" gencodec:"required"`
	Amount    *big.Int    `json:"replenishAmount" gencodec:"required"`
}

type replenishAssetMarshaling struct {
	Amount *hexutil.Big10
}

// 修改资产profile
//go:generate gencodec -type ModifyAssetInfo -out gen_modifyAssetInfo_json.go
type ModifyAssetInfo struct {
	AssetCode     common.Hash `json:"assetCode" gencodec:"required"`
	UpdateProfile Profile     `json:"updateProfile" gencodec:"required"`
}

// 交易资产
//go:generate gencodec -type TradingAsset --field-override tradingAssetMarshaling -out gen_tradingAsset_json.go
type TradingAsset struct {
	AssetId common.Hash `json:"assetId" gencodec:"required"`
	Value   *big.Int    `json:"transferAmount" gencodec:"required"`
	Input   []byte      `json:"input"`
}

type tradingAssetMarshaling struct {
	Value *hexutil.Big10
}
