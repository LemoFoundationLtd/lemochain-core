// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package types

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
)

var _ = (*assetMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (a Asset) MarshalJSON() ([]byte, error) {
	type Asset struct {
		Category        hexutil.Uint32 `json:"category" gencodec:"required"`
		IsDivisible     bool           `json:"isDivisible" gencodec:"required"`
		AssetCode       common.Hash    `json:"assetCode"`
		Decimal         hexutil.Uint32 `json:"decimal" gencodec:"required"`
		TotalSupply     *hexutil.Big10 `json:"totalSupply"`
		IsReplenishable bool           `json:"isReplenishable" gencodec:"required"`
		Issuer          common.Address `json:"issuer"`
		Profile         Profile        `json:"profile"`
	}
	var enc Asset
	enc.Category = hexutil.Uint32(a.Category)
	enc.IsDivisible = a.IsDivisible
	enc.AssetCode = a.AssetCode
	enc.Decimal = hexutil.Uint32(a.Decimal)
	enc.TotalSupply = (*hexutil.Big10)(a.TotalSupply)
	enc.IsReplenishable = a.IsReplenishable
	enc.Issuer = a.Issuer
	enc.Profile = a.Profile
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (a *Asset) UnmarshalJSON(input []byte) error {
	type Asset struct {
		Category        *hexutil.Uint32 `json:"category" gencodec:"required"`
		IsDivisible     *bool           `json:"isDivisible" gencodec:"required"`
		AssetCode       *common.Hash    `json:"assetCode"`
		Decimal         *hexutil.Uint32 `json:"decimal" gencodec:"required"`
		TotalSupply     *hexutil.Big10  `json:"totalSupply"`
		IsReplenishable *bool           `json:"isReplenishable" gencodec:"required"`
		Issuer          *common.Address `json:"issuer"`
		Profile         *Profile        `json:"profile"`
	}
	var dec Asset
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Category == nil {
		return errors.New("missing required field 'category' for Asset")
	}
	a.Category = uint32(*dec.Category)
	if dec.IsDivisible == nil {
		return errors.New("missing required field 'isDivisible' for Asset")
	}
	a.IsDivisible = *dec.IsDivisible
	if dec.AssetCode != nil {
		a.AssetCode = *dec.AssetCode
	}
	if dec.Decimal == nil {
		return errors.New("missing required field 'decimal' for Asset")
	}
	a.Decimal = uint32(*dec.Decimal)
	if dec.TotalSupply != nil {
		a.TotalSupply = (*big.Int)(dec.TotalSupply)
	}
	if dec.IsReplenishable == nil {
		return errors.New("missing required field 'isReplenishable' for Asset")
	}
	a.IsReplenishable = *dec.IsReplenishable
	if dec.Issuer != nil {
		a.Issuer = *dec.Issuer
	}
	if dec.Profile != nil {
		a.Profile = *dec.Profile
	}
	return nil
}