package types

import (
	"encoding/json"
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

// GetIssueAsset
func GetIssueAsset(txData []byte) (*IssueAsset, error) {
	issue := &IssueAsset{}
	if err := json.Unmarshal(txData, issue); err != nil {
		return nil, err
	}
	return issue, nil
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

// GetReplenishAsset
func GetReplenishAsset(txData []byte) (*ReplenishAsset, error) {
	replenish := &ReplenishAsset{}
	if err := json.Unmarshal(txData, replenish); err != nil {
		return nil, err
	}
	return replenish, nil
}

// 修改资产profile
//go:generate gencodec -type ModifyAssetInfo -out gen_modifyAssetInfo_json.go
type ModifyAssetInfo struct {
	AssetCode     common.Hash `json:"assetCode" gencodec:"required"`
	UpdateProfile Profile     `json:"updateProfile" gencodec:"required"`
}

// GetModifyAssetInfo
func GetModifyAssetInfo(txData []byte) (*ModifyAssetInfo, error) {
	info := &ModifyAssetInfo{}
	if err := json.Unmarshal(txData, info); err != nil {
		return nil, err
	}
	return info, nil
}

// 交易资产
//go:generate gencodec -type TransferAsset --field-override transferAssetMarshaling -out gen_transferAsset_json.go
type TransferAsset struct {
	AssetId common.Hash `json:"assetId" gencodec:"required"`
	Amount  *big.Int    `json:"transferAmount" gencodec:"required"`
	Input   []byte      `json:"input"`
}

type transferAssetMarshaling struct {
	Amount *hexutil.Big10
}

// GetTransferAsset
func GetTransferAsset(txData []byte) (*TransferAsset, error) {
	transferAsset := &TransferAsset{}
	if err := json.Unmarshal(txData, transferAsset); err != nil {
		return nil, err
	}
	return transferAsset, nil
}

// 箱子交易
//go:generate gencodec -type Box -out gen_box_json.go
type Box struct {
	SubTxList Transactions `json:"subTxList"  gencodec:"required"`
}

// GetBox
func GetBox(txData []byte) (*Box, error) {
	box := &Box{}
	err := json.Unmarshal(txData, box)
	if err != nil {
		return nil, err
	}
	return box, nil
}

// MarshalBoxData 通过传入的子交易序列化出箱子data
func MarshalBoxData(txs Transactions) ([]byte, error) {
	box := &Box{
		SubTxList: txs,
	}
	return json.Marshal(box)
}
