package node

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/miner"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
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
		Balance := fmt.Sprintf("0.%018s", balance)
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

// GetBlockByNumber get block information by height
func (c *ChainAPI) GetBlockByNumber(height uint32) *types.Block {
	return c.chain.GetBlockByHeight(height)
}

// GetBlockByHash get block information by hash
func (c *ChainAPI) GetBlockByHash(hash string) *types.Block {
	return c.chain.GetBlockByHash(common.HexToHash(hash))
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

// GetLatestStableBlock get the latest currently agreed blocks
func (c *ChainAPI) GetLatestStableBlock() *types.Block {
	return c.chain.StableBlock()
}

// GetCurrentHeight
func (c *ChainAPI) GetCurrentHeight() uint32 {
	currentBlock := c.chain.CurrentBlock()
	height := currentBlock.Height()
	return height
}

// LatestStableHeight
func (c *ChainAPI) GetLatestStableHeight() uint32 {
	return c.chain.StableBlock().Height()
}

// GetGasPrice get suggest gas price
func (c *ChainAPI) GetSuggestGasPrice() *big.Int {
	// todo
	return big.NewInt(100000000)
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
func (t *TxAPI) SendTx(encodedTx hexutil.Bytes) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return common.Hash{}, err
	}
	txHash := tx.Hash()
	err := t.txpool.AddTx(tx)
	return txHash, err
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

// GetLemoBase
func (m *MineAPI) GetLemoBase() string {
	lemoBase := m.miner.GetLemoBase()
	return lemoBase.Hex()
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

// GetPeers
func (n *NetAPI) GetPeers() []p2p.PeerConnInfo {
	return n.server.Peers()
}

// GetNodeVersion
func (n *NetAPI) GetNodeVersion() string {
	// todo
	return "1.0"
}
