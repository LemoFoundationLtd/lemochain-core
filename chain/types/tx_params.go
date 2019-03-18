package types

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"math/big"
)

// tx data marshal unmarshal struct
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
