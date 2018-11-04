package node

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/miner"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/math"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"math/big"
	"runtime"
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

// GetBalance get balance in mo
func (a *AccountAPI) GetBalance(LemoAddress string) (string, error) {
	accounts, err := a.GetAccount(LemoAddress)
	if err != nil {
		return "", err
	}
	balance := accounts.GetBalance().String()

	return balance, nil
}

// GetAccount return the struct of the &AccountData{}
func (a *AccountAPI) GetAccount(LemoAddress string) (types.AccountAccessor, error) {
	address, err := common.StringToAddress(LemoAddress)
	if err != nil {
		return nil, err
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
func (c *ChainAPI) ChainID() uint16 {
	return c.chain.ChainID()
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
	return params.Version
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
	node *Node
	// server *p2p.Server
}

// NewNetAPI
func NewNetAPI(node *Node) *NetAPI {
	return &NetAPI{node}
}

// AddStaticPeer
func (n *NetAPI) AddStaticPeer(node string) {
	n.node.server.AddStaticPeer(node)
}

// DropPeer
func (n *NetAPI) DropPeer(node string) string {
	if n.node.server.DropPeer(node) {
		return fmt.Sprintf("drop a peer success. id %v", node)
	} else {
		return fmt.Sprintf("drop a peer fail. id %v", node)
	}

}

// Peers
func (n *NetAPI) Peers() []p2p.PeerConnInfo {
	return n.node.server.Peers()
}

// PeersCount return peers number
func (n *NetAPI) PeersCount() int {
	return len(n.node.server.Peers())
}

//go:generate gencodec -type NetInfo --field-override netInfoMarshaling -out gen_net_info_json.go

type NetInfo struct {
	Port     uint32 `json:"port"        gencodec:"required"`
	NodeName string `json:"nodeName"    gencodec:"required"`
	Version  string `json:"nodeVersion" gencodec:"required"`
	OS       string `json:"os"          gencodec:"required"`
	Go       string `json:"runtime"     gencodec:"required"`
}

type netInfoMarshaling struct {
	Port math.Decimal32
}

// Info
func (n *NetAPI) Info() *NetInfo {
	return &NetInfo{
		Port:     uint32(n.node.server.Port),
		NodeName: n.node.config.NodeName(),
		Version:  n.node.config.Version,
		OS:       runtime.GOOS + "-" + runtime.GOARCH,
		Go:       runtime.Version(),
	}
}
