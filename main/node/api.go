package node

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"strings"
)

// AccountAPI API for access to account information
type AccountAPI struct {
	// blockChain *chain.BlockChain
	accMan *account.Manager
}

//
func NewAccountAPI(accMan *account.Manager) *AccountAPI {
	return &AccountAPI{accMan}
}

// NewAccount get lemo address api
func (a *AccountAPI) NewKeyPair() *crypto.AddressKeyPair {
	account := crypto.GenerateAddress()
	return account
}

// GetBalance get balance api
func (a *AccountAPI) GetBalance(LemoAddress string) string {
	var address common.Address
	// Determine whether the input address is a Lemo address or a native address.
	if strings.HasPrefix(LemoAddress, "Lemo") {
		address = crypto.RestoreOriginalAddress(LemoAddress)
	} else {
		address = common.HexToAddress(LemoAddress)
	}
	account := a.accMan.GetCanonicalAccount(address)
	balance := account.GetBalance().String()
	lenth := len(balance)
	var toBytes = []byte(balance)
	if lenth <= 18 {
		var head = make([]byte, 18-lenth)
		for i := 0; i < 18-lenth; i++ {
			head[i] = '0'
		}
		// append head to make it 18 bytes
		fullbytes := append(head, toBytes...)
		Balance := "0." + string(fullbytes)
		return Balance
	} else {
		point := lenth % 18
		// Extended section length
		ToBytes := append(toBytes, '0')
		for i := lenth; i > point; i-- {
			ToBytes[i] = ToBytes[i-1]
		}
		ToBytes[point] = '.'

		return string(ToBytes)
	}
}
