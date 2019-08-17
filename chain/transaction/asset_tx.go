package transaction

import (
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"math/big"
	"sort"
	"strings"
)

var (
	ErrIssueAssetAmount     = errors.New("issue asset amount can't be 0 or nil")
	ErrIssueAssetMetaData   = errors.New("the length of metaData more than limit")
	ErrReplenishAssetAmount = errors.New("replenish asset amount can't be 0 or nil")
	ErrAssetIssuer          = errors.New("issue asset transaction's sender must the asset issuer")
	ErrFrozenAsset          = errors.New("can't replenish the frozen assets")
	ErrIsReplenishable      = errors.New("asset's \"IsReplenishable\" is false")
	ErrIsDivisible          = errors.New("this \"isDivisible == false\" kind of asset can't be replenished")
	ErrNotEqualAssetCode    = errors.New("assetCode not equal")
	ErrModifyAssetInfo      = errors.New("missing required field 'info' for ModifyAssetInfo")
	ErrMarshalAssetLength   = errors.New("the length of data by marshal asset more than max length")
	ErrAssetCategory        = errors.New("assert's Category not exist")
	ErrModifyAssetTxSender  = errors.New("the sender does not have permission to modify this asset")
)

type RunAssetEnv struct {
	am *account.Manager
}

func NewRunAssetEnv(am *account.Manager) *RunAssetEnv {
	return &RunAssetEnv{
		am: am,
	}
}

// CreateAssetTx
func (r *RunAssetEnv) CreateAssetTx(sender common.Address, data []byte, txHash common.Hash) error {
	var err error
	issuerAcc := r.am.GetAccount(sender)
	asset := &types.Asset{}
	err = json.Unmarshal(data, asset)
	if err != nil {
		return err
	}
	// verify
	err = asset.VerifyAsset()
	if err != nil {
		return err
	}

	asset.Issuer = sender
	asset.AssetCode = txHash
	asset.TotalSupply = big.NewInt(0) // init 0
	newData, err := json.Marshal(asset)
	if err != nil {
		return err
	}
	// judge data's length
	if len(newData) > types.MaxMarshalAssetLength {
		log.Errorf("The length of data by marshal asset more than max length,len(data) = %d ", len(newData))
		return ErrMarshalAssetLength
	}
	var snapshot = r.am.Snapshot()
	err = issuerAcc.SetAssetCode(asset.AssetCode, asset)
	if err != nil {
		r.am.RevertToSnapshot(snapshot)
	}
	return err
}

// unmarshalIssueAssetData
func unmarshalIssueAssetData(data []byte) (*types.IssueAsset, error) {
	issueAsset, err := types.GetIssueAsset(data)
	if err != nil {
		return nil, err
	}
	// metaData length limit
	if len(issueAsset.MetaData) > types.MaxMetaDataLength {
		log.Errorf("The length of metaData more than limit, len(metaData) = %d ", len(issueAsset.MetaData))
		return nil, ErrIssueAssetMetaData
	}
	// amount != nil && amount > 0
	if issueAsset.Amount == nil || issueAsset.Amount.Cmp(big.NewInt(0)) <= 0 {
		log.Errorf("Issue asset amount must > 0 , currentAmount = %s", issueAsset.Amount.String())
		return nil, ErrIssueAssetAmount
	}
	return issueAsset, nil
}

// IssueAssetTx
func (r *RunAssetEnv) IssueAssetTx(sender, receiver common.Address, txHash common.Hash, data []byte) error {
	issueAsset, err := unmarshalIssueAssetData(data)
	if err != nil {
		return err
	}
	assetCode := issueAsset.AssetCode
	issuerAcc := r.am.GetAccount(sender)
	asset, err := issuerAcc.GetAssetCode(assetCode)
	if err != nil {
		return err
	}
	// judge sender is issuer
	if asset.Issuer != sender {
		log.Errorf("Transaction sender is not the asset issuer")
		return ErrAssetIssuer
	}
	// Determine whether asset is frozen
	freeze, err := issuerAcc.GetAssetCodeState(assetCode, types.AssetFreeze)
	if err == nil && freeze == "true" {
		log.Errorf("Can't issue the frozen assets.")
		return ErrFrozenAsset
	}
	recAcc := r.am.GetAccount(receiver)
	equity := &types.AssetEquity{}
	equity.AssetCode = assetCode
	// equity.Equity = issueAsset.Amount

	// judge assert type
	AssType := asset.Category
	if AssType == types.Asset01 { // ERC20
		equity.AssetId = asset.AssetCode // assetId == assetCode
		// 第一类资产由于assetCode和assetId相等，所以要判断receiver是否已经拥有了此类资产，如果已经拥有了，我们对equity采取增加而不是错误的覆盖
		oldAssetEquity, err := recAcc.GetEquityState(asset.AssetCode)
		if err != nil && err != store.ErrNotExist {
			return err
		}
		if err == store.ErrNotExist { // 未拥有过此类资产
			equity.Equity = issueAsset.Amount
		} else { // 已经拥有过此资产,资产余额则相加
			equity.Equity = new(big.Int).Add(issueAsset.Amount, oldAssetEquity.Equity)
		}

	} else if AssType == types.Asset02 || AssType == types.Asset03 { // ERC721 or ERC721+20
		equity.AssetId = txHash           // assetId == 当前交易的hash
		equity.Equity = issueAsset.Amount // asset 余额为发行金额
	} else {
		log.Errorf("Assert's Category not exist ,Category = %d ", AssType)
		return ErrAssetCategory
	}
	var snapshot = r.am.Snapshot()
	newAsset := asset.Clone()
	// add totalSupply
	var oldTotalSupply *big.Int
	var newTotalSupply *big.Int
	oldTotalSupply, err = issuerAcc.GetAssetCodeTotalSupply(assetCode)
	if err != nil {
		return err
	}
	if !newAsset.IsDivisible {
		newTotalSupply = new(big.Int).Add(oldTotalSupply, big.NewInt(1))
	} else {
		newTotalSupply = new(big.Int).Add(oldTotalSupply, issueAsset.Amount)
	}
	// set new totalSupply
	err = issuerAcc.SetAssetCodeTotalSupply(assetCode, newTotalSupply)
	if err != nil {
		r.am.RevertToSnapshot(snapshot)
		return err
	}
	// set new asset equity for receiver
	err = recAcc.SetEquityState(equity.AssetId, equity)
	if err != nil {
		r.am.RevertToSnapshot(snapshot)
		return err
	}
	err = recAcc.SetAssetIdState(equity.AssetId, issueAsset.MetaData)
	if err != nil {
		r.am.RevertToSnapshot(snapshot)
		return err
	}
	return nil
}

// unmarshalReplenishAssetData
func unmarshalReplenishAssetData(data []byte) (*types.ReplenishAsset, error) {
	repl, err := types.GetReplenishAsset(data)
	if err != nil {
		return nil, err
	}
	addAmount := repl.Amount
	// amount > 0
	if addAmount == nil || addAmount.Cmp(big.NewInt(0)) <= 0 {
		log.Errorf("Replenish asset amount must > 0,currentAmount = %s", addAmount.String())
		return nil, ErrReplenishAssetAmount
	}
	return repl, nil
}

// judgeReplenish 判断满足可增发的条件
func judgeReplenish(issuerAcc types.AccountAccessor, newAssetCode common.Hash) error {
	// assert issuer account
	asset, err := issuerAcc.GetAssetCode(newAssetCode)
	if err != nil {
		return err
	}
	// Determine whether asset is frozen
	freeze, err := issuerAcc.GetAssetCodeState(newAssetCode, types.AssetFreeze)
	if err == nil && freeze == "true" {
		return ErrFrozenAsset
	}
	// judge IsReplenishable
	if !asset.IsReplenishable {
		return ErrIsReplenishable
	}
	// erc721 asset can't be replenished
	if !asset.IsDivisible {
		return ErrIsDivisible
	}
	return nil
}

// ReplenishAssetTx
func (r *RunAssetEnv) ReplenishAssetTx(sender, receiver common.Address, data []byte) error {
	repl, err := unmarshalReplenishAssetData(data)
	if err != nil {
		return err
	}
	newAssetCode := repl.AssetCode
	newAssetId := repl.AssetId
	addAmount := repl.Amount

	// assert issuer account
	issuerAcc := r.am.GetAccount(sender)
	err = judgeReplenish(issuerAcc, newAssetCode)
	if err != nil {
		return err
	}
	// receiver account
	recAcc := r.am.GetAccount(receiver)
	equity, err := recAcc.GetEquityState(newAssetId)
	if err != nil && err != store.ErrNotExist {
		return err
	}
	if err == store.ErrNotExist {
		equity = &types.AssetEquity{
			AssetCode: newAssetCode,
			AssetId:   newAssetId,
			Equity:    big.NewInt(0),
		}
	}

	if newAssetCode != equity.AssetCode {
		log.Errorf("AssetCode not equal: newAssetCode = %s,originalAssetCode = %s. ", newAssetCode.String(), equity.AssetCode.String())
		return ErrNotEqualAssetCode
	}
	var snapshot = r.am.Snapshot()
	// add amount
	newEquity := equity.Clone()
	newEquity.Equity = new(big.Int).Add(newEquity.Equity, addAmount)
	err = recAcc.SetEquityState(newEquity.AssetId, newEquity)
	if err != nil {
		r.am.RevertToSnapshot(snapshot)
		return err
	}
	// add asset totalSupply
	var oldTotalSupply *big.Int
	var newTotalSupply *big.Int
	oldTotalSupply, err = issuerAcc.GetAssetCodeTotalSupply(newAssetCode)
	if err != nil {
		return err
	}
	newTotalSupply = new(big.Int).Add(oldTotalSupply, addAmount)
	err = issuerAcc.SetAssetCodeTotalSupply(newAssetCode, newTotalSupply)
	if err != nil {
		r.am.RevertToSnapshot(snapshot)
		return err
	}
	return nil
}

func unmarshalModifyAssetData(data []byte) (*types.ModifyAssetInfo, error) {
	modifyInfo, err := types.GetModifyAssetInfo(data)
	if err != nil {
		return nil, err
	}
	info := modifyInfo.UpdateProfile
	if info == nil || len(info) == 0 {
		return nil, ErrModifyAssetInfo
	}
	return modifyInfo, nil
}

// ModifyAssetProfileTx
func (r *RunAssetEnv) ModifyAssetProfileTx(sender common.Address, data []byte) error {
	modifyInfo, err := unmarshalModifyAssetData(data)
	if err != nil {
		return err
	}
	info := modifyInfo.UpdateProfile

	acc := r.am.GetAccount(sender)

	// determine whether there is modification permission
	oldAsset, err := acc.GetAssetCode(modifyInfo.AssetCode)
	if err != nil {
		return err
	}
	if oldAsset.Issuer != sender {
		log.Errorf("SenderAddress:%s,assetIssuer:%s", sender.String(), oldAsset.Issuer.String())
		return ErrModifyAssetTxSender
	}

	var snapshot = r.am.Snapshot()
	infoSlice := make([]string, 0, len(info))
	for k, _ := range info {
		infoSlice = append(infoSlice, strings.ToLower(k))
	}
	sort.Strings(infoSlice)
	for i := 0; i < len(infoSlice); i++ {
		err = acc.SetAssetCodeState(modifyInfo.AssetCode, infoSlice[i], info[infoSlice[i]])
		if err != nil {
			r.am.RevertToSnapshot(snapshot)
			return err
		}
	}
	// 	judge profile size
	newAsset, err := acc.GetAssetCode(modifyInfo.AssetCode)
	if err != nil {
		r.am.RevertToSnapshot(snapshot)
		return err
	}
	newData, err := json.Marshal(newAsset)
	if err != nil {
		r.am.RevertToSnapshot(snapshot)
		return err
	}
	// judge data's length
	if len(newData) > types.MaxMarshalAssetLength {
		log.Errorf("The length of marshaling asset data exceed limit, len(data) = %d max = %d", len(newData), types.MaxMarshalAssetLength)
		r.am.RevertToSnapshot(snapshot)
		return ErrMarshalAssetLength
	}
	return nil
}
