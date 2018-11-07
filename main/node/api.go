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
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"math/big"
	"runtime"
	"strconv"
)

// Private
type PrivateAccountAPI struct {
	manager *account.Manager
}

// NewPrivateAccountAPI
func NewPrivateAccountAPI(m *account.Manager) *PrivateAccountAPI {
	return &PrivateAccountAPI{m}
}

// NewAccount get lemo address api
func (a *PrivateAccountAPI) NewKeyPair() (*crypto.AddressKeyPair, error) {
	accounts, err := crypto.GenerateAddress()
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// PublicAccountAPI API for access to account information
type PublicAccountAPI struct {
	manager *account.Manager
}

// NewPublicAccountAPI
func NewPublicAccountAPI(m *account.Manager) *PublicAccountAPI {
	return &PublicAccountAPI{m}
}

// GetBalance get balance in mo
func (a *PublicAccountAPI) GetBalance(LemoAddress string) (string, error) {
	accounts, err := a.GetAccount(LemoAddress)
	if err != nil {
		return "", err
	}
	balance := accounts.GetBalance().String()

	return balance, nil
}

// GetAccount return the struct of the &AccountData{}
func (a *PublicAccountAPI) GetAccount(LemoAddress string) (types.AccountAccessor, error) {
	address, err := common.StringToAddress(LemoAddress)
	if err != nil {
		return nil, err
	}

	accountData := a.manager.GetCanonicalAccount(address)
	return accountData, nil
}

// ChainAPI
type PublicChainAPI struct {
	chain *chain.BlockChain
}

// NewChainAPI API for access to chain information
func NewPublicChainAPI(chain *chain.BlockChain) *PublicChainAPI {
	return &PublicChainAPI{chain}
}

// GetBlockByNumber get block information by height
func (c *PublicChainAPI) GetBlockByHeight(height uint32, withBody bool) *types.Block {
	if withBody {
		return c.chain.GetBlockByHeight(height)
	} else {
		block := c.chain.GetBlockByHeight(height)
		if block == nil {
			return nil
		}
		// copy only header
		onlyHeaderBlock := &types.Block{
			Header: block.Header,
		}
		return onlyHeaderBlock
	}
}

// GetBlockByHash get block information by hash
func (c *PublicChainAPI) GetBlockByHash(hash string, withBody bool) *types.Block {
	if withBody {
		return c.chain.GetBlockByHash(common.HexToHash(hash))
	} else {
		block := c.chain.GetBlockByHash(common.HexToHash(hash))
		if block == nil {
			return nil
		}
		// copy only header
		onlyHeaderBlock := &types.Block{
			Header: block.Header,
		}
		return onlyHeaderBlock
	}

}

// ChainID get chain id
func (c *PublicChainAPI) ChainID() uint16 {
	return c.chain.ChainID()
}

// Genesis get the creation block
func (c *PublicChainAPI) Genesis() *types.Block {
	return c.chain.Genesis()
}

// CurrentBlock get the current latest block
func (c *PublicChainAPI) CurrentBlock(withBody bool) *types.Block {
	if withBody {
		return c.chain.CurrentBlock()
	} else {
		currentBlock := c.chain.CurrentBlock()
		if currentBlock == nil {
			return nil
		}
		// copy only header
		onlyHeaderBlock := &types.Block{
			Header: currentBlock.Header,
		}
		return onlyHeaderBlock
	}
}

// LatestStableBlock get the latest currently agreed blocks
func (c *PublicChainAPI) LatestStableBlock(withBody bool) *types.Block {
	if withBody {
		return c.chain.StableBlock()
	} else {
		stableBlock := c.chain.StableBlock()
		if stableBlock == nil {
			return nil
		}
		// copy only header
		onlyHeaderBlock := &types.Block{
			Header: stableBlock.Header,
		}
		return onlyHeaderBlock
	}
}

// CurrentHeight
func (c *PublicChainAPI) CurrentHeight() uint32 {
	currentBlock := c.chain.CurrentBlock()
	height := currentBlock.Height()
	return height
}

// LatestStableHeight
func (c *PublicChainAPI) LatestStableHeight() uint32 {
	return c.chain.StableBlock().Height()
}

// GasPriceAdvice get suggest gas price
func (c *PublicChainAPI) GasPriceAdvice() *big.Int {
	// todo
	return big.NewInt(100000000)
}

// NodeVersion
func (n *PublicChainAPI) NodeVersion() string {
	return params.Version
}

// TXAPI
type PublicTxAPI struct {
	txpool *chain.TxPool
}

// NewTxAPI API for send a transaction
func NewPublicTxAPI(txpool *chain.TxPool) *PublicTxAPI {
	return &PublicTxAPI{txpool}
}

// Send send a transaction
func (t *PublicTxAPI) SendTx(tx *types.Transaction) (common.Hash, error) {
	err := t.txpool.AddTx(tx)
	return tx.Hash(), err
}

// PendingTx
func (t *PublicTxAPI) PendingTx(size int) []*types.Transaction {
	return t.txpool.Pending(size)
}

// PrivateMineAPI
type PrivateMineAPI struct {
	miner *miner.Miner
}

// NewPrivateMinerAPI
func NewPrivateMinerAPI(miner *miner.Miner) *PrivateMineAPI {
	return &PrivateMineAPI{miner}
}

// MineStart
func (m *PrivateMineAPI) MineStart() {
	m.miner.Start()
}

// MineStop
func (m *PrivateMineAPI) MineStop() {
	m.miner.Stop()
}

// PublicMineAPI
type PublicMineAPI struct {
	miner *miner.Miner
}

// NewPublicMineAPI
func NewPublicMineAPI(miner *miner.Miner) *PublicMineAPI {
	return &PublicMineAPI{miner}
}

// IsMining
func (m *PublicMineAPI) IsMining() bool {
	return m.miner.IsMining()
}

// LemoBase
func (m *PublicMineAPI) LemoBase() string {
	lemoBase := m.miner.GetLemoBase()
	return lemoBase.String()
}

// PrivateNetAPI
type PrivateNetAPI struct {
	node *Node
}

// NewPrivateNetAPI
func NewPrivateNetAPI(node *Node) *PrivateNetAPI {
	return &PrivateNetAPI{node}
}

// AddStaticPeer
func (n *PrivateNetAPI) AddStaticPeer(node string) {
	n.node.server.AddStaticPeer(node)
}

// DropPeer
func (n *PrivateNetAPI) DropPeer(node string) string {
	if n.node.server.DropPeer(node) {
		return fmt.Sprintf("drop a peer success. id %v", node)
	} else {
		return fmt.Sprintf("drop a peer fail. id %v", node)
	}

}

// Peers
func (n *PrivateNetAPI) Peers() []p2p.PeerConnInfo {
	return n.node.server.Peers()
}

// PublicNetAPI
type PublicNetAPI struct {
	node *Node
}

// NewPublicNetAPI
func NewPublicNetAPI(node *Node) *PublicNetAPI {
	return &PublicNetAPI{node}
}

// PeersCount return peers number
func (n *PublicNetAPI) PeersCount() string {
	count := strconv.Itoa(len(n.node.server.Peers()))
	return count
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
	Port hexutil.Uint32
}

// Info
func (n *PublicNetAPI) Info() *NetInfo {
	return &NetInfo{
		Port:     uint32(n.node.server.Port),
		NodeName: n.node.config.NodeName(),
		Version:  n.node.config.Version,
		OS:       runtime.GOOS + "-" + runtime.GOARCH,
		Go:       runtime.Version(),
	}
}
