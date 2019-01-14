// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package crypto

import (
	"encoding/json"
)

// MarshalJSON marshals as JSON.
func (a AccountKey) MarshalJSON() ([]byte, error) {
	type AccountKey struct {
		Private string `json:"private"`
		Public  string `json:"public"`
		Address string `json:"address"`
	}
	var enc AccountKey
	enc.Private = a.Private
	enc.Public = a.Public
	enc.Address = a.Address
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (a *AccountKey) UnmarshalJSON(input []byte) error {
	type AccountKey struct {
		Private *string `json:"private"`
		Public  *string `json:"public"`
		Address *string `json:"address"`
	}
	var dec AccountKey
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Private != nil {
		a.Private = *dec.Private
	}
	if dec.Public != nil {
		a.Public = *dec.Public
	}
	if dec.Address != nil {
		a.Address = *dec.Address
	}
	return nil
}