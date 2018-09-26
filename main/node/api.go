package node

import (
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"math/big"

	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/common"
)

// AccountAPI API for access to account information
type AccountAPI struct {
	blockChain *chain.BlockChain
}

//
func NewAccountAPI(blockChain *chain.BlockChain) *AccountAPI {
	return &AccountAPI{blockChain}
}

// NewAccount get lemo address api
func (a *AccountAPI) NewAccount() *crypto.AddressKeyPair {
	account := crypto.GenerateAddress()
	return account
}

// GetBalance get balance api
func (a *AccountAPI) GetBalance(address common.Address) *big.Int {
	// address := crypto.RestoreOriginalAddress(LemoAddress)
	account, err := a.blockChain.AccountManager().GetAccount(address)
	if err != nil {
		return nil
	}
	return account.GetBalance()
}
