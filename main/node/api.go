package node

import (
	"context"
	"fmt"
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
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"math/big"
	"runtime"
	"strconv"
	"time"
)

const (
	MaxTxToNameLength  = 100
	MaxTxMessageLength = 1024
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

// GetVoteFor
func (a *PublicAccountAPI) GetVoteFor(LemoAddress string) (string, error) {
	candiAccount, err := a.GetAccount(LemoAddress)
	if err != nil {
		return "", err
	}
	forAddress := candiAccount.GetVoteFor().String()
	return forAddress, nil
}

//go:generate gencodec -type CandiateInfo -out gen_candidate_info_json.go

type CandiateInfo struct {
	CandidateAddress string            `json:"candidate" gencodec:"required"`
	Votes            string            `json:"votes" gencodec:"required"`
	Profile          map[string]string `json:"profile"  gencodec:"required"`
}

// GetCandidateInfo get candidate node information
func (a *PublicAccountAPI) GetCandidateInfo(LemoAddress string) *CandiateInfo {
	candiAccount, err := a.GetAccount(LemoAddress)
	if err != nil {
		return nil
	}
	mapProfile := candiAccount.GetCandidateProfile()
	if _, ok := mapProfile[types.CandidateKeyIsCandidate]; !ok {
		return nil
	}

	candidateInfo := &CandiateInfo{
		Profile: make(map[string]string),
	}
	candidateInfo.Profile[types.CandidateKeyIsCandidate] = mapProfile[types.CandidateKeyIsCandidate]
	candidateInfo.Profile[types.CandidateKeyHost] = mapProfile[types.CandidateKeyHost]
	candidateInfo.Profile[types.CandidateKeyNodeID] = mapProfile[types.CandidateKeyNodeID]
	candidateInfo.Profile[types.CandidateKeyPort] = mapProfile[types.CandidateKeyPort]
	candidateInfo.Profile[types.CandidateKeyMinerAddress] = mapProfile[types.CandidateKeyMinerAddress]
	candidateInfo.Votes = candiAccount.GetVotes().String()
	candidateInfo.CandidateAddress = LemoAddress
	return candidateInfo
}

// ChainAPI
type PublicChainAPI struct {
	chain *chain.BlockChain
}

// NewChainAPI API for access to chain information
func NewPublicChainAPI(chain *chain.BlockChain) *PublicChainAPI {
	return &PublicChainAPI{chain}
}

// GetCandidateNodeList get all candidate node list information
func (c *PublicChainAPI) GetCandidateNodeList(no, size int) ([]*CandiateInfo, error) {
	addresses, err := c.chain.Db().GetCandidatesPage(no, size)
	if err != nil {
		return nil, err
	}
	candidateInfoes := make([]*CandiateInfo, 0)
	for i := 0; i < len(addresses); i++ {
		candidateAccount := c.chain.AccountManager().GetAccount(addresses[i])
		mapProfile := candidateAccount.GetCandidateProfile()
		if isCandidate, ok := mapProfile[types.CandidateKeyIsCandidate]; !ok || isCandidate == params.NotCandidateNode {
			err = fmt.Errorf("the node of %s is not candidate node", addresses[i].String())
			return nil, err
		}
		candidateInfo := &CandiateInfo{
			Profile: make(map[string]string),
		}
		candidateInfo.Profile[types.CandidateKeyIsCandidate] = mapProfile[types.CandidateKeyIsCandidate]
		candidateInfo.Profile[types.CandidateKeyHost] = mapProfile[types.CandidateKeyHost]
		candidateInfo.Profile[types.CandidateKeyNodeID] = mapProfile[types.CandidateKeyNodeID]
		candidateInfo.Profile[types.CandidateKeyPort] = mapProfile[types.CandidateKeyPort]
		candidateInfo.Profile[types.CandidateKeyMinerAddress] = mapProfile[types.CandidateKeyMinerAddress]
		candidateInfo.Votes = candidateAccount.GetVotes().String()
		candidateInfo.CandidateAddress = addresses[i].String()

		candidateInfoes = append(candidateInfoes, candidateInfo)
	}
	return candidateInfoes, nil
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

	toNameLength := len(tx.ToName())
	if toNameLength > MaxTxToNameLength {
		toNameErr := fmt.Errorf("the length of toName field in transaction is out of max length limit. toName length = %d. max length limit = %d. ", toNameLength, MaxTxToNameLength)
		return common.Hash{}, toNameErr
	}
	txMessageLength := len(tx.Message())
	if txMessageLength > MaxTxMessageLength {
		txMessageErr := fmt.Errorf("the length of message field in transaction is out of max length limit. message length = %d. max length limit = %d. ", txMessageLength, MaxTxMessageLength)
		return common.Hash{}, txMessageErr
	}
	err := t.node.txPool.AddTx(tx)
	return tx.Hash(), err
}

// PendingTx
func (t *PublicTxAPI) PendingTx(size int) []*types.Transaction {
	return t.node.txPool.Pending(size)
}

// GetTxByHash pull the specified transaction through a transaction hash
func (t *PublicTxAPI) GetTxByHash(hash string) (*store.VTransaction, error) {
	txHash := common.HexToHash(hash)
	bizDb := t.node.db.GetBizDatabase()
	vTx, err := bizDb.GetTx8Hash(txHash)
	return vTx, err
}

// GetTxListByAddress pull the list of transactions
func (t *PublicTxAPI) GetTxListByAddress(lemoAddress string, start int64, size int) ([]*store.VTransaction, int64, error) {
	src, err := common.StringToAddress(lemoAddress)
	if err != nil {
		return nil, start, err
	}
	bizDb := t.node.db.GetBizDatabase()
	vTxs, next, err := bizDb.GetTx8AddrNext(src, start, size)
	return vTxs, next, err
}

// // PullForwardTxListByAddress 向前拉取address所涉及的交易列表
// func (t *PublicTxAPI) PullForwardTxListByAddress(src common.Address, start int64, size int) ([]*store.VTransaction, int64, error) {
// 	// src, err := common.StringToAddress(lemoAddress)
// if err != nil {
// 	return nil, start, err
// }
// 	bizDb := t.node.db.GetBizDatabase()
// 	vTxs, previous, err := bizDb.GetTx8AddrPre(src, start, size)
// 	return vTxs, previous, err
// }

// ReadContract read variables in a contract includes the return value of a function.
func (t *PublicTxAPI) ReadContract(to *common.Address, data hexutil.Bytes) (string, error) {
	ctx := context.Background()
	result, _, err := t.doCall(ctx, to, data, 5*time.Second)
	return common.ToHex(result), err
}

// EstimateGas returns an estimate of the amount of gas needed to execute the given transaction.
func (t *PublicTxAPI) EstimateGas(to *common.Address, data hexutil.Bytes) (uint64, error) {
	ctx := context.Background()
	_, costGas, err := t.doCall(ctx, to, data, 5*time.Second)
	return costGas, err
}

// EstimateContractGas returns an estimate of the amount of gas needed to create a smart contract.
func (t *PublicTxAPI) EstimateCreateContractGas(data hexutil.Bytes) (uint64, error) {
	ctx := context.Background()
	_, costGas, err := t.doCall(ctx, nil, data, 5*time.Second)
	return costGas, err
}

// doCall
func (t *PublicTxAPI) doCall(ctx context.Context, to *common.Address, data hexutil.Bytes, timeout time.Duration) ([]byte, uint64, error) {
	t.node.lock.Lock()
	defer t.node.lock.Unlock()

	defer func(start time.Time) { log.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())
	// get latest stableBlock
	stableBlock := t.node.chain.StableBlock()
	log.Infof("stable block height = %v", stableBlock.Height())
	stableHeader := stableBlock.Header

	p := t.node.chain.TxProcessor()
	ret, costGas, err := p.CallTx(ctx, stableHeader, to, data, common.Hash{}, timeout)

	return ret, costGas, err
}
