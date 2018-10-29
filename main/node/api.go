package node

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/miner"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"math/big"
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

// addPoint digital processing
func addPoint(balance string) string {
	length := len(balance)
	var toBytes = []byte(balance)
	if length <= 18 {
		Balance := fmt.Sprintf("0.%018s", balance)
		return Balance
	} else {
		point := length - 18
		// Extended section length
		Balance := string(toBytes[:point]) + "." + string(toBytes[point:])

		return Balance
	}
}

// GetBalance get balance api
func (a *AccountAPI) GetBalance(LemoAddress string) (string, error) {
	accounts, err := a.GetAccount(LemoAddress)
	if err != nil {
		return "", err
	}
	balance := accounts.GetBalance().String()
	lastBalance := addPoint(balance)

	return lastBalance, nil
}

// GetAccount return the struct of the &AccountData{}
func (a *AccountAPI) GetAccount(LemoAddress string) (types.AccountAccessor, error) {
	// Determine if the input address length is valid
	addressLen := len(LemoAddress)
	if addressLen != 39 && addressLen != 42 {
		return nil, errors.New("address length is incorrect")
	}
	var address common.Address
	// Determine whether the input address is a Lemo address or a native address.
	if strings.HasPrefix(LemoAddress, "Lemo") {
		var err error
		address, err = common.RestoreOriginalAddress(LemoAddress)
		if err != nil {
			return nil, err
		}
	} else {
		address = common.HexToAddress(LemoAddress)
	}

	accountData := a.manager.GetCanonicalAccount(address)

	return accountData, nil
}

// ChainAPI
type ChainAPI struct {
	chain *chain.BlockChain
}

// NewChainAPI API for access to chain information
func NewChainAPI(chain *chain.BlockChain) *ChainAPI {
	return &ChainAPI{chain}
}

// GetBlockByNumber get block information by height
func (c *ChainAPI) GetBlockByHeight(height uint32, withTxs bool) *types.Block {
	if withTxs {
		return c.chain.GetBlockByHeight(height)
	} else {
		block := c.chain.GetBlockByHeight(height)
		if block == nil {
			return nil
		}
		// set the Txs field to null
		block.SetTxs([]*types.Transaction{})
		return block
	}
}

// GetBlockByHash get block information by hash
func (c *ChainAPI) GetBlockByHash(hash string, withTxs bool) *types.Block {
	if withTxs {
		return c.chain.GetBlockByHash(common.HexToHash(hash))
	} else {
		block := c.chain.GetBlockByHash(common.HexToHash(hash))
		if block == nil {
			return nil
		}
		// set the Txs field to null
		block.SetTxs([]*types.Transaction{})
		return block
	}

}

// ChainID get chain id
func (c *ChainAPI) ChainID() string {
	return strconv.Itoa(int(c.chain.ChainID()))
}

// Genesis get the creation block
func (c *ChainAPI) Genesis() *types.Block {
	return c.chain.Genesis()
}

// CurrentBlock get the current latest block
func (c *ChainAPI) CurrentBlock(withTxs bool) *types.Block {
	if withTxs {
		return c.chain.CurrentBlock()
	} else {
		currentBlock := c.chain.CurrentBlock()
		if currentBlock == nil {
			return nil
		}
		// set the Txs field to null
		currentBlock.SetTxs([]*types.Transaction{})
		return currentBlock
	}

}

// LatestStableBlock get the latest currently agreed blocks
func (c *ChainAPI) LatestStableBlock(withTxs bool) *types.Block {
	if withTxs == true {
		return c.chain.StableBlock()
	} else {
		stableBlock := c.chain.StableBlock()
		if stableBlock == nil {
			return nil
		}
		// set the Txs field to null
		stableBlock.SetTxs([]*types.Transaction{})
		return stableBlock
	}

}

// CurrentHeight
func (c *ChainAPI) CurrentHeight() uint32 {
	currentBlock := c.chain.CurrentBlock()
	height := currentBlock.Height()
	return height
}

// LatestStableHeight
func (c *ChainAPI) LatestStableHeight() uint32 {
	return c.chain.StableBlock().Height()
}

// GasPriceAdvice get suggest gas price
func (c *ChainAPI) GasPriceAdvice() *big.Int {
	// todo
	return big.NewInt(100000000)
}

// NodeVersion
func (n *ChainAPI) NodeVersion() string {
	// todo
	return "1.0"
}

// TXAPI
type TxAPI struct {
	txpool *chain.TxPool
}

// NewTxAPI API for send a transaction
func NewTxAPI(txpool *chain.TxPool) *TxAPI {
	return &TxAPI{txpool}
}

// Send send a transaction
func (t *TxAPI) SendTx(tx *types.Transaction) (common.Hash, error) {
	err := t.txpool.AddTx(tx)
	return tx.Hash(), err
}

// MineAPI
type MineAPI struct {
	miner *miner.Miner
}

// NewMineAPI
func NewMineAPI(miner *miner.Miner) *MineAPI {
	return &MineAPI{miner}
}

// MineStart
func (m *MineAPI) MineStart() {
	m.miner.Start()
}

// MineStop
func (m *MineAPI) MineStop() {
	m.miner.Stop()
}

// IsMining
func (m *MineAPI) IsMining() bool {
	return m.miner.IsMining()
}

// LemoBase
func (m *MineAPI) LemoBase() string {
	lemoBase := m.miner.GetLemoBase()
	return lemoBase.String()
}

// NetAPI
type NetAPI struct {
	server *p2p.Server
}

// NewNetAPI
func NewNetAPI(server *p2p.Server) *NetAPI {
	return &NetAPI{server}
}

// AddStaticPeer
func (n *NetAPI) AddStaticPeer(node string) {
	n.server.AddStaticPeer(node)
}

func (n *NetAPI) DropPeer(node string) {
	n.server.DropPeer(node)
}

// Peers
func (n *NetAPI) Peers() []p2p.PeerConnInfo {
	return n.server.Peers()
}

// NetInfo
func (n *NetAPI) Info() string {
	return n.server.ListenAddr()
}
