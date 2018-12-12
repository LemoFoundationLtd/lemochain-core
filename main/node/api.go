package node

import (
	"context"
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/miner"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"math/big"
	"runtime"
	"strconv"
	"time"
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
func (a *PrivateAccountAPI) NewKeyPair() (*crypto.AccountKey, error) {
	accountKey, err := crypto.GenerateAddress()
	if err != nil {
		return nil, err
	}
	return accountKey, nil
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
	// txpool *chain.TxPool
	node *Node
}

// NewTxAPI API for send a transaction
func NewPublicTxAPI(node *Node) *PublicTxAPI {
	return &PublicTxAPI{node}
}

// Send send a transaction
func (t *PublicTxAPI) SendTx(tx *types.Transaction) (common.Hash, error) {
	err := t.node.txPool.AddTx(tx)
	return tx.Hash(), err
}

// PendingTx
func (t *PublicTxAPI) PendingTx(size int) []*types.Transaction {
	return t.node.txPool.Pending(size)
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

// Miner
func (m *PublicMineAPI) Miner() string {
	address := m.miner.GetMinerAddress()
	return address.String()
}

// PrivateNetAPI
type PrivateNetAPI struct {
	node *Node
}

// NewPrivateNetAPI
func NewPrivateNetAPI(node *Node) *PrivateNetAPI {
	return &PrivateNetAPI{node}
}

// Connect
func (n *PrivateNetAPI) Connect(node string) {
	n.node.server.Connect(node)
}

// Disconnect
func (n *PrivateNetAPI) Disconnect(node string) bool {
	return n.node.server.Disconnect(node)
}

// Connections
func (n *PrivateNetAPI) Connections() []p2p.PeerConnInfo {
	return n.node.server.Connections()
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
	count := strconv.Itoa(len(n.node.server.Connections()))
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

// // PublicContractAPI
// type PublicContractAPI struct {
// 	node *Node
// }
//
// // NewPublicContractAPI
// func NewPublicContractAPI(node *Node) *PublicContractAPI {
// 	return &PublicContractAPI{node}
// }

// Call
func (t *PublicTxAPI) Call(ctx context.Context, tx *types.Transaction) (hexutil.Bytes, error) {
	result, _, _, err := t.doCall(ctx, tx, 5*time.Second)
	return (hexutil.Bytes)(result), err
}

// doCall
func (t *PublicTxAPI) doCall(ctx context.Context, tx *types.Transaction, timeout time.Duration) ([]byte, uint64, bool, error) {
	t.node.lock.Lock()
	defer t.node.lock.Unlock()

	defer func(start time.Time) { log.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())
	// 得到最新块
	latestBlock, err := t.node.db.LoadLatestBlock()
	if err != nil || latestBlock.Height() == 0 {
		return nil, 0, false, err
	}
	latestHeader := latestBlock.Header
	gp := new(types.GasPool).AddGas(latestHeader.GasLimit)
	// gasUsed := uint64(0)
	t.node.AccountManager().Reset(latestHeader.ParentHash)

	if gp.Gas() < params.TxGas {
		log.Info("Not enough gas for further transactions", "gp", gp)
		return nil, 0, false, err
	}
	p := t.node.chain.TxProcessor()
	ret, restGas, failed, err := p.CallContract(gp, latestHeader, tx, common.Hash{})

	return ret, restGas, failed, err
}
