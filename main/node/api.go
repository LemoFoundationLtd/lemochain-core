package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/consensus"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/miner"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/subscribe"
	"github.com/LemoFoundationLtd/lemochain-core/network"
	"github.com/LemoFoundationLtd/lemochain-core/network/p2p"
	"math/big"
	"runtime"
	"strconv"
	"time"
)

var (
	ErrToName         = errors.New("the length of toName field in transaction is out of max length limit")
	ErrTxMessage      = errors.New("the length of message field in transaction is out of max length limit")
	ErrCreateContract = errors.New("the data of create contract transaction can't be null")
	ErrSpecialTx      = errors.New("the data of special transaction can't be null")
	ErrTxType         = errors.New("the transaction type does not exist")
	ErrAssetId        = errors.New("assetid is incorrect")
	ErrTxExpiration   = errors.New("tx expiration time is out of date")
	ErrNegativeValue  = errors.New("negative value")
	ErrTxChainID      = errors.New("tx chainID is incorrect")
	ErrInputParams    = errors.New("input params incorrect")
	ErrTxTo           = errors.New("transaction to is incorrect")
)

// Private
type PrivateAccountAPI struct {
	manager *account.Manager
}

// NewPrivateAccountAPI
func NewPrivateAccountAPI(m *account.Manager) *PrivateAccountAPI {
	return &PrivateAccountAPI{m}
}

//go:generate gencodec -type LemoAccount -out gen_lemo_account_json.go
type LemoAccount struct {
	Private string         `json:"private"`
	Address common.Address `json:"address"`
}

// NewAccount get lemo address api
func (a *PrivateAccountAPI) NewKeyPair() (*LemoAccount, error) {
	accountKey, err := crypto.GenerateAddress()
	if err != nil {
		return nil, err
	}
	acc := &LemoAccount{
		Private: accountKey.Private,
		Address: accountKey.Address,
	}
	return acc, nil
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
func (a *PublicAccountAPI) GetBalance(lemoAddress string) (string, error) {
	lemoAccount, err := a.GetAccount(lemoAddress)
	if err != nil {
		return "", err
	}
	balance := lemoAccount.GetBalance().String()

	return balance, nil
}

// GetAccount return the struct of the &AccountData{}
func (a *PublicAccountAPI) GetAccount(lemoAddress string) (types.AccountAccessor, error) {
	address, err := common.StringToAddress(lemoAddress)
	if err != nil {
		log.Warnf("lemoAddress is incorrect. lemoAddress: %s", lemoAddress)
		return nil, err
	}
	accountData := a.manager.GetCanonicalAccount(address)
	// accountData := a.manager.GetAccount(address)
	return accountData, nil
}

// GetVoteFor
func (a *PublicAccountAPI) GetVoteFor(lemoAddress string) (string, error) {
	candiAccount, err := a.GetAccount(lemoAddress)
	if err != nil {
		return "", err
	}
	forAddress := candiAccount.GetVoteFor().String()
	return forAddress, nil
}

// GetAssetEquity returns asset equity
func (a *PublicAccountAPI) GetAssetEquityByAssetId(lemoAddress string, assetId common.Hash) (*types.AssetEquity, error) {
	acc, err := a.GetAccount(lemoAddress)
	if err != nil {
		return nil, err
	}
	return acc.GetEquityState(assetId)
}

//go:generate gencodec -type CandidateInfo -out gen_candidate_info_json.go
type CandidateInfo struct {
	CandidateAddress string            `json:"address" gencodec:"required"`
	Votes            string            `json:"votes" gencodec:"required"`
	Profile          map[string]string `json:"profile"  gencodec:"required"`
}

// ChainAPI
type PublicChainAPI struct {
	chain *chain.BlockChain
}

// NewChainAPI API for access to chain information
func NewPublicChainAPI(chain *chain.BlockChain) *PublicChainAPI {
	return &PublicChainAPI{chain}
}

//go:generate gencodec -type DeputyNodeInfo --field-override deputyNodeInfoMarshaling -out gen_deputyNode_info_json.go
type DeputyNodeInfo struct {
	MinerAddress  common.Address `json:"minerAddress"   gencodec:"required"` // candidate account address
	IncomeAddress common.Address `json:"incomeAddress" gencodec:"required"`
	NodeID        []byte         `json:"nodeID"         gencodec:"required"`
	Rank          uint32         `json:"rank"           gencodec:"required"` // start from 0
	Votes         *big.Int       `json:"votes"          gencodec:"required"`
	Host          string         `json:"host"          gencodec:"required"`
	Port          string         `json:"port"          gencodec:"required"`
	DepositAmount string         `json:"depositAmount"  gencodec:"required"` // 质押金额
	Introduction  string         `json:"introduction"  gencodec:"required"`  // 节点介绍
	P2pUri        string         `json:"p2pUri"  gencodec:"required"`        // p2p 连接用的定位符
}

type deputyNodeInfoMarshaling struct {
	NodeID hexutil.Bytes
	Rank   hexutil.Uint32
	Votes  *hexutil.Big10
}

// GetDeputyNodeList
func (c *PublicChainAPI) GetDeputyNodeList() []*DeputyNodeInfo {
	nodes := c.chain.DeputyManager().GetDeputiesByHeight(c.chain.CurrentBlock().Height())

	var result []*DeputyNodeInfo
	for _, n := range nodes {
		candidateAcc := c.chain.AccountManager().GetCanonicalAccount(n.MinerAddress)
		profile := candidateAcc.GetCandidate()
		incomeAddress, err := common.StringToAddress(profile[types.CandidateKeyIncomeAddress])
		if err != nil {
			log.Errorf("incomeAddress string to address type.incomeAddress: %s.error: %v", profile[types.CandidateKeyIncomeAddress], err)
			continue
		}
		host := profile[types.CandidateKeyHost]
		port := profile[types.CandidateKeyPort]
		nodeAddrString := fmt.Sprintf("%x@%s:%s", n.NodeID, host, port)
		deputyNodeInfo := &DeputyNodeInfo{
			MinerAddress:  n.MinerAddress,
			IncomeAddress: incomeAddress,
			NodeID:        n.NodeID,
			Rank:          n.Rank,
			Votes:         n.Votes,
			Host:          host,
			Port:          port,
			DepositAmount: profile[types.CandidateKeyDepositAmount],
			Introduction:  profile[types.CandidateKeyIntroduction],
			P2pUri:        nodeAddrString,
		}
		result = append(result, deputyNodeInfo)
	}
	return result
}

//go:generate gencodec -type TermRewardInfo --field-override termRewardInfoMarshaling -out gen_termReward_info_json.go
type TermRewardInfo struct {
	Term         uint32   `json:"term" gencodec:"required"`
	Value        *big.Int `json:"value" gencodec:"required"`
	RewardHeight uint32   `json:"rewardHeight" gencodec:"required"`
}
type termRewardInfoMarshaling struct {
	Term         hexutil.Uint32
	Value        *hexutil.Big10
	RewardHeight hexutil.Uint32
}

// GetAllRewardValue get the value for each bonus
func (a *PublicChainAPI) GetAllRewardValue() (params.RewardsMap, error) {
	address := params.TermRewardContract
	acc := a.chain.AccountManager().GetCanonicalAccount(address)
	key := address.Hash()
	value, err := acc.GetStorageState(key)
	if err != nil {
		return nil, err
	}
	rewardMap := make(params.RewardsMap)
	err = json.Unmarshal(value, &rewardMap)
	return rewardMap, err
}

// GetTermReward get term reward info by height
func (a *PublicChainAPI) GetTermReward(height uint32) (*TermRewardInfo, error) {
	term := deputynode.GetTermIndexByHeight(height)
	termValueMaplist, err := a.GetAllRewardValue()
	if err != nil {
		return nil, err
	}
	if reward, ok := termValueMaplist[term]; ok {
		return &TermRewardInfo{
			Term:         reward.Term,
			Value:        reward.Value,
			RewardHeight: (term+1)*params.TermDuration + params.InterimDuration + 1,
		}, nil
	} else {
		return nil, nil
	}
}

// GetCandidateTop30 get top 30 candidate node
func (c *PublicChainAPI) GetCandidateTop30() []*CandidateInfo {
	latestStableBlock := c.chain.StableBlock()
	stableBlockHash := latestStableBlock.Hash()
	storeInfos := c.chain.GetCandidatesTop(stableBlockHash)
	candidateList := make([]*CandidateInfo, 0, 30)
	for _, info := range storeInfos {
		candidateInfo := &CandidateInfo{
			Profile: make(map[string]string),
		}
		CandidateAddress := info.GetAddress()
		CandidateAccount := c.chain.AccountManager().GetCanonicalAccount(CandidateAddress)
		profile := CandidateAccount.GetCandidate()
		candidateInfo.Profile = profile
		candidateInfo.CandidateAddress = CandidateAddress.String()
		candidateInfo.Votes = info.GetTotal().String()
		candidateList = append(candidateList, candidateInfo)
	}
	return candidateList
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
	if len(common.FromHex(hash)) != common.HashLength {
		log.Warnf("Hash is incorrect, Hash: %s", hash)
		return nil
	}
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

// CurrentBlock get the latest block. It may not be confirmed by enough deputy nodes
func (c *PublicChainAPI) UnstableBlock(withBody bool) *types.Block {
	block := c.chain.CurrentBlock()
	if !withBody && block != nil {
		// copy only header
		block = &types.Block{
			Header: block.Header,
		}
	}
	return block
}

// CurrentBlock get the latest block.
func (c *PublicChainAPI) CurrentBlock(withBody bool) *types.Block {
	block := c.chain.StableBlock()
	if !withBody && block != nil {
		// copy only header
		block = &types.Block{
			Header: block.Header,
		}
	}
	return block
}

// UnstableHeight
func (c *PublicChainAPI) UnstableHeight() uint32 {
	return c.chain.CurrentBlock().Height()
}

// CurrentHeight
func (c *PublicChainAPI) CurrentHeight() uint32 {
	return c.chain.StableBlock().Height()
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

// SetLeastGasPrice
func (m *PrivateMineAPI) SetLeastGasPrice(price *big.Int) {
	params.MinGasPrice = price
}

// PublicMineAPI
type PublicMineAPI struct {
	miner *miner.Miner
}

// NewPublicMineAPI
func NewPublicMineAPI(miner *miner.Miner) *PublicMineAPI {
	return &PublicMineAPI{miner}
}

// GetLeastGasPrice
func (m *PublicMineAPI) GetLeastGasPrice() string {
	return params.MinGasPrice.String()
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

// Connect (node = nodeID@IP:Port)
func (n *PrivateNetAPI) Connect(node string) error {
	if err := network.VerifyNode(node); err != nil {
		log.Errorf("The node uri is incorrect: %s", node)
		return err
	}
	n.node.server.Connect(node)
	return nil
}

// Disconnect
func (n *PrivateNetAPI) Disconnect(node string) (bool, error) {
	if err := network.VerifyNode(node); err != nil {
		log.Errorf("The node uri is incorrect: %s", node)
		return false, err
	}
	return n.node.server.Disconnect(node), nil
}

// Connections
func (n *PrivateNetAPI) Connections() []p2p.PeerConnInfo {
	return n.node.server.Connections()
}

// BroadcastConfirm
func (n *PrivateNetAPI) BroadcastConfirm(hash string) (bool, error) {
	// load block
	var block *types.Block
	if len(hash) == 0 {
		block = n.node.chain.CurrentBlock()
	} else {
		block = n.node.chain.GetBlockByHash(common.HexToHash(hash))
	}
	if block == nil {
		return false, fmt.Errorf("block is not exist for hash: %s", hash)
	}

	// find my confirm in block
	sigBytes, err := consensus.SignBlock(block.Hash())
	if err != nil {
		return false, fmt.Errorf("the miner can't confirm block")
	}
	sig := types.BytesToSignData(sigBytes)
	if !block.IsConfirmExist(sig) {
		return false, fmt.Errorf("block has not been confirmed by the miner")
	}

	// broadcast
	pack := &network.BlockConfirmData{
		Hash:     block.Hash(),
		Height:   block.Height(),
		SignInfo: sig,
	}
	subscribe.Send(subscribe.NewConfirm, pack)
	return true, nil
}

// FetchConfirm
func (n *PrivateNetAPI) FetchConfirm(height uint32) error {
	return n.node.chain.FetchConfirm(height)
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

// NodeID
func (n *PublicNetAPI) NodeID() string {
	return common.ToHex(deputynode.GetSelfNodeID())
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
	if err := tx.VerifyTxBody(t.node.ChainID(), uint64(time.Now().Unix()), false); err != nil {
		log.Errorf("VerifyTxBody error: %s", err)
		return common.Hash{}, err
	}
	// 判断tx是否在当前分支已经存在了
	currentBlock := t.node.chain.CurrentBlock()
	guard := t.node.chain.TxGuard()
	isExist := guard.ExistTx(currentBlock.Hash(), tx)
	if !isExist {
		// 加入交易池
		err := t.node.txPool.AddTx(tx)
		if err != nil {
			log.Warnf("AddTx error: %s", err)
			return common.Hash{}, err
		}
		// 广播交易
		go subscribe.Send(subscribe.NewTx, tx)
	}
	return tx.Hash(), nil
}

// GetPendingTx
func (t *PublicTxAPI) GetPendingTx(size int) []*types.Transaction {
	return t.node.txPool.GetTxs(uint32(time.Now().Unix()), size)
}

// ReadContract read variables in a contract includes the return value of a function.
func (t *PublicTxAPI) ReadContract(to *common.Address, data hexutil.Bytes) (string, error) {
	if to == nil {
		return "", ErrInputParams
	}
	accM := account.NewReadOnlyManager(t.node.Db(), true)
	result, err := t.doCallTransaction(to, accM, data, 5*time.Second)
	return common.ToHex(result), err
}

// doCallTransaction
func (t *PublicTxAPI) doCallTransaction(to *common.Address, accM *account.ReadOnlyManager, data hexutil.Bytes, timeout time.Duration) ([]byte, error) {
	t.node.lock.Lock()
	defer t.node.lock.Unlock()

	defer func(start time.Time) {
		log.Debug("Executing EVM call finished", "cost time", time.Since(start))
	}(time.Now())
	// get current stableBlock
	currentBlock := t.node.chain.CurrentBlock()
	log.Infof("Current block height = %v", currentBlock.Height())
	currentHeader := currentBlock.Header
	p := t.node.chain.TxProcessor()
	ret, err := p.ReadContract(accM, currentHeader, *to, data, timeout)

	return ret, err
}
