package node

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"reflect"
	"strconv"
	"strings"
)

// AccountAPI API for access to account information
type AccountAPI struct {
	manager *account.Manager
}

// NewAccountAPI
func NewAccountAPI(m *account.Manager) *AccountAPI {
	return &AccountAPI{m}
}

// NewAccount get lemo address api
func (a *AccountAPI) NewKeyPair() (*crypto.AddressKeyPair, error) {
	accounts, err := crypto.GenerateAddress()
	if err != nil {
		return nil, err
	}
	return accounts, nil
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
	accounts := a.manager.GetCanonicalAccount(address)
	balance := accounts.GetBalance().String()
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

// GetVersion get version
func (a *AccountAPI) GetVersion(LemoAddress string, logType uint32) uint32 {
	var address common.Address
	// Determine whether the input address is a Lemo address or a native address.
	if strings.HasPrefix(LemoAddress, "Lemo") {
		address = crypto.RestoreOriginalAddress(LemoAddress)
	} else {
		address = common.HexToAddress(LemoAddress)
	}
	accountObj := a.manager.GetCanonicalAccount(address)
	return accountObj.GetVersion(types.ChangeLogType(logType))
}

// GetAccount return the struct of the &AccountData{}
func (a *AccountAPI) GetAccount(LemoAddress string) types.AccountAccessor {
	var address common.Address
	// Determine whether the input address is a Lemo address or a native address.
	if strings.HasPrefix(LemoAddress, "Lemo") {
		address = crypto.RestoreOriginalAddress(LemoAddress)
	} else {
		address = common.HexToAddress(LemoAddress)
	}
	accountData := a.manager.GetCanonicalAccount(address)

	return accountData
}

// ChainAPI
type ChainAPI struct {
	chain *chain.BlockChain
}

// NewChainAPI API for access to chain information
func NewChainAPI(chain *chain.BlockChain) *ChainAPI {
	return &ChainAPI{chain}
}

// GetBlock get block information by height or hash
func (c *ChainAPI) GetBlock(n interface{}) *types.Block {
	t := reflect.TypeOf(n).String()
	if f := "float64"; strings.EqualFold(t, f) {
		h := n.(float64)
		height := uint32(h)
		return c.chain.GetBlockByHeight(height)

	} else if f := "string"; strings.EqualFold(t, f) {
		h := n.(string)
		hash := common.HexToHash(h)
		return c.chain.GetBlockByHash(hash)
	} else {
		return nil
	}
}

// GetChainID get chain id
func (c *ChainAPI) GetChainID() string {
	return strconv.Itoa(int(c.chain.ChainID()))
}

// GetGenesis get the creation block
func (c *ChainAPI) GetGenesis() *types.Block {
	return c.chain.Genesis()
}

// GetCurrentBlock get the current latest block
func (c *ChainAPI) GetCurrentBlock() *types.Block {
	return c.chain.CurrentBlock()
}

// GetStableBlock get the latest currently agreed blocks
func (c *ChainAPI) GetStableBlock() *types.Block {
	return c.chain.StableBlock()
}

//
func (c *ChainAPI) GetCurrentHeight() uint32 {
	currentBlock := c.chain.CurrentBlock()
	height := currentBlock.Height()
	return height
}
