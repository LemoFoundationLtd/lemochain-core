package node

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"math/big"
)

// AccountAPI API for access to account information
type AccountAPI struct {
	db   protocol.ChainDB
	data *types.AccountData
}

//
func NewAccountAPI(db protocol.ChainDB, data *types.AccountData) *AccountAPI {
	return &AccountAPI{db, data}
}

// NewAccount get lemo address api
func (a *AccountAPI) NewAccount() *crypto.AddressKeyPair {
	account := crypto.GenerateAddress()
	return account
}

// GetBalance get balance api
func (a *AccountAPI) GetBalance(lemoAddress string) *big.Int {
	accountPoint := account.NewAccount(a.db, crypto.RestoreOriginalAddress(lemoAddress), a.data)

	return accountPoint.GetBalance()
}
