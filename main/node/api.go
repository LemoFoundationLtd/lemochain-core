package node

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/miner"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/network/p2p"
	"math/big"
	"runtime"
	"strconv"
	"time"
)

const (
	MaxTxToNameLength  = 100
	MaxTxMessageLength = 1024
)

var (
	toNameErr         = errors.New("the length of toName field in transaction is out of max length limit")
	txMessageErr      = errors.New("the length of message field in transaction is out of max length limit")
	createContractErr = errors.New("the data of create contract transaction can't be null")
	specialTxErr      = errors.New("the data of special transaction can't be null")
	txTypeErr         = errors.New("the transaction type does not exit")
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
	// accountData := a.manager.GetAccount(address)
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

// GetAllRewardValue get the value for each bonus
func (a *PublicAccountAPI) GetAllRewardValue() ([]*params.Reward, error) {
	address := params.TermRewardPrecompiledContractAddress
	acc, err := a.GetAccount(address.String())
	if err != nil {
		return nil, err
	}
	key := address.Hash()
	value, err := acc.GetStorageState(key)
	rewardMap := make(params.RewardsMap)
	json.Unmarshal(value, &rewardMap)
	var result = make([]*params.Reward, 0)
	// var maxTerm uint32 = 0
	// for _, v := range rewardMap {
	// 	if v.Term == maxTerm {
	// 		result = append(result, v)
	// 		maxTerm++
	// 	}
	// }
	var i uint32
	for i = 0; ; i++ {
		if v, ok := rewardMap[i]; ok {
			result = append(result, v)
		} else {
			break
		}
	}

	return result, nil
}

// GetAssetEquity returns asset equity
func (a *PublicAccountAPI) GetAssetEquityByAssetId(LemoAddress string, assetId common.Hash) (*types.AssetEquity, error) {
	acc, err := a.GetAccount(LemoAddress)
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

// GetDeputyNodeList
func (c *PublicChainAPI) GetDeputyNodeList() []string {
	nodes := c.chain.DeputyManager().GetDeputiesByHeight(c.chain.CurrentBlock().Height())

	var result []string
	for _, n := range nodes {
		result = append(result, n.NodeAddrString())
	}
	return result
}

// GetCandidateTop30 get top 30 candidate node
func (c *PublicChainAPI) GetCandidateTop30() []*CandidateInfo {
	latestStableBlock := c.chain.StableBlock()
	stableBlockHash := latestStableBlock.Hash()
	storeInfos := c.chain.Db().GetCandidatesTop(stableBlockHash)
	candidateList := make([]*CandidateInfo, 0, 30)
	for _, info := range storeInfos {
		candidateInfo := &CandidateInfo{
			Profile: make(map[string]string),
		}
		CandidateAddress := info.GetAddress()
		CandidateAccount := c.chain.AccountManager().GetAccount(CandidateAddress)
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
	err := VerifyTx(tx)
	if err != nil {
		return common.Hash{}, err
	}
	err = t.node.txPool.AddTx(tx)
	return tx.Hash(), err
}

// SendReimbursedGasTx gas代付交易 todo 测试使用
func (t *PublicTxAPI) SendReimbursedGasTx(senderPrivate, gasPayerPrivate string, to, gasPayer common.Address, amount int64, data []byte, txType uint16, toName, message string) (common.Hash, error) {
	tx := types.NewReimbursementTransaction(to, gasPayer, big.NewInt(amount), data, txType, t.node.chainID, uint64(time.Now().Unix()+1800), toName, message)
	senderPriv, _ := crypto.HexToECDSA(senderPrivate)
	gasPayerPriv, _ := crypto.HexToECDSA(gasPayerPrivate)
	firstSignTx, err := types.MakeReimbursementTxSigner().SignTx(tx, senderPriv)
	if err != nil {
		return common.Hash{}, err
	}
	signTx := types.GasPayerSignatureTx(firstSignTx, common.Big1, uint64(60000))
	lastSignTx, err := types.MakeGasPayerSigner().SignTx(signTx, gasPayerPriv)
	if err != nil {
		return common.Hash{}, err
	}
	err = VerifyTx(lastSignTx)
	if err != nil {
		return common.Hash{}, err
	}
	err = t.node.txPool.AddTx(lastSignTx)
	return lastSignTx.Hash(), err
}

// CreateAsset 创建资产
func (t *PublicTxAPI) CreateAsset(prv string, category, decimals uint32, isReplenishable, isDivisible bool) (common.Hash, error) {
	private, _ := crypto.HexToECDSA(prv)
	issuer := crypto.PubkeyToAddress(private.PublicKey)
	profile := make(types.Profile)
	profile[types.AssetName] = "Demo Token"
	profile[types.AssetSymbol] = "DT"
	profile[types.AssetDescription] = "test issue token"
	profile[types.AssetStop] = "false"
	profile[types.AssetSuggestedGasLimit] = "60000"
	asset := &types.Asset{
		Category:        category,
		IsDivisible:     isDivisible,
		AssetCode:       common.Hash{},
		Decimals:        decimals,
		TotalSupply:     big.NewInt(100000),
		IsReplenishable: isReplenishable,
		Issuer:          issuer,
		Profile:         profile,
	}
	data, err := json.Marshal(asset)
	if err != nil {
		return common.Hash{}, err
	}
	tx := types.NoReceiverTransaction(nil, uint64(500000), big.NewInt(1), data, params.CreateAssetTx, t.node.chainID, uint64(time.Now().Unix()+30*60), "", "create asset tx")
	signTx, err := types.MakeSigner().SignTx(tx, private)
	if err != nil {
		return common.Hash{}, err
	}
	return t.SendTx(signTx)
}

// 发行资产
func (t *PublicTxAPI) IssueAsset(prv string, receiver common.Address, assetCode common.Hash, amount *big.Int, metaData string) (common.Hash, error) {
	issue := &types.IssueAsset{
		AssetCode: assetCode,
		MetaData:  metaData,
		Amount:    amount,
	}
	data, err := json.Marshal(issue)
	if err != nil {
		return common.Hash{}, err
	}
	tx := types.NewTransaction(receiver, nil, uint64(500000), big.NewInt(1), data, params.IssueAssetTx, t.node.chainID, uint64(time.Now().Unix()+30*60), "", "issue asset tx")
	private, _ := crypto.HexToECDSA(prv)
	signTx, err := types.MakeSigner().SignTx(tx, private)
	if err != nil {
		return common.Hash{}, err
	}
	return t.SendTx(signTx)
}

// 增发资产
func (t *PublicTxAPI) ReplenishAsset(prv string, receiver common.Address, assetCode, assetId common.Hash, amount *big.Int) (common.Hash, error) {
	repl := &types.ReplenishAsset{
		AssetCode: assetCode,
		AssetId:   assetId,
		Amount:    amount,
	}
	data, err := json.Marshal(repl)
	if err != nil {
		return common.Hash{}, err
	}
	tx := types.NewTransaction(receiver, nil, uint64(500000), big.NewInt(1), data, params.ReplenishAssetTx, t.node.chainID, uint64(time.Now().Unix()+30*60), "", "replenish asset tx")
	private, _ := crypto.HexToECDSA(prv)
	signTx, err := types.MakeSigner().SignTx(tx, private)
	if err != nil {
		return common.Hash{}, err
	}
	return t.SendTx(signTx)
}

// ModifyAsset 修改资产信息
func (t *PublicTxAPI) ModifyAsset(prv string, assetCode common.Hash) (common.Hash, error) {
	info := make(types.Profile)
	info["name"] = "Modify"
	info["stop"] = "true"
	modify := &types.ModifyAssetInfo{
		AssetCode: assetCode,
		Info:      info,
	}
	data, err := json.Marshal(modify)
	if err != nil {
		return common.Hash{}, err
	}
	tx := types.NoReceiverTransaction(nil, uint64(500000), big.NewInt(1), data, params.ModifyAssetTx, t.node.chainID, uint64(time.Now().Unix()+30*60), "", "modify asset tx")
	private, _ := crypto.HexToECDSA(prv)
	signTx, err := types.MakeSigner().SignTx(tx, private)
	if err != nil {
		return common.Hash{}, err
	}
	return t.SendTx(signTx)
}

// 交易资产
func (t *PublicTxAPI) TradingAsset(prv string, to common.Address, assetCode, assetId common.Hash, amount *big.Int, input []byte) (common.Hash, error) {
	trading := &types.TradingAsset{
		AssetId: assetId,
		Value:   amount,
		Input:   input,
	}
	data, err := json.Marshal(trading)
	if err != nil {
		return common.Hash{}, err
	}
	tx := types.NewTransaction(to, amount, uint64(500000), big.NewInt(1), data, params.TransferAssetTx, t.node.chainID, uint64(time.Now().Unix()+30*60), "", "trading asset tx")
	private, _ := crypto.HexToECDSA(prv)
	signTx, err := types.MakeSigner().SignTx(tx, private)
	if err != nil {
		return common.Hash{}, err
	}
	return t.SendTx(signTx)
}

// VerifyTx transaction parameter verification
func VerifyTx(tx *types.Transaction) error {
	toNameLength := len(tx.ToName())
	if toNameLength > MaxTxToNameLength {

		log.Errorf("the length of toName field in transaction is out of max length limit. toName length = %d. max length limit = %d. ", toNameLength, MaxTxToNameLength)
		return toNameErr
	}
	txMessageLength := len(tx.Message())
	if txMessageLength > MaxTxMessageLength {
		log.Errorf("the length of message field in transaction is out of max length limit. message length = %d. max length limit = %d. ", txMessageLength, MaxTxMessageLength)
		return txMessageErr
	}
	switch tx.Type() {
	case params.OrdinaryTx:
		if tx.To() == nil {
			if len(tx.Data()) == 0 {
				return createContractErr
			}
		}
	case params.VoteTx:
	case params.RegisterTx, params.CreateAssetTx, params.IssueAssetTx, params.ReplenishAssetTx, params.ModifyAssetTx, params.TransferAssetTx:
		if len(tx.Data()) == 0 {
			return specialTxErr
		}
	default:
		log.Errorf("the transaction type does not exit . type = %v", tx.Type())
		return txTypeErr
	}
	return nil
}

// PendingTx
func (t *PublicTxAPI) PendingTx(size int) []*types.Transaction {
	return t.node.txPool.Pending(size)
}

// ReadContract read variables in a contract includes the return value of a function.
func (t *PublicTxAPI) ReadContract(to *common.Address, data hexutil.Bytes) (string, error) {
	ctx := context.Background()
	result, _, err := t.doCall(ctx, to, params.OrdinaryTx, data, 5*time.Second)
	return common.ToHex(result), err
}

// EstimateGas returns an estimate of the amount of gas needed to execute the given transaction.
func (t *PublicTxAPI) EstimateGas(to *common.Address, txType uint16, data hexutil.Bytes) (string, error) {
	var costGas uint64
	var err error
	ctx := context.Background()
	_, costGas, err = t.doCall(ctx, to, txType, data, 5*time.Second)
	strCostGas := strconv.FormatUint(costGas, 10)
	return strCostGas, err
}

// EstimateContractGas returns an estimate of the amount of gas needed to create a smart contract.
// todo will delete
func (t *PublicTxAPI) EstimateCreateContractGas(data hexutil.Bytes) (uint64, error) {
	ctx := context.Background()
	_, costGas, err := t.doCall(ctx, nil, params.OrdinaryTx, data, 5*time.Second)
	return costGas, err
}

// doCall
func (t *PublicTxAPI) doCall(ctx context.Context, to *common.Address, txType uint16, data hexutil.Bytes, timeout time.Duration) ([]byte, uint64, error) {
	t.node.lock.Lock()
	defer t.node.lock.Unlock()

	defer func(start time.Time) { log.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())
	// get latest stableBlock
	stableBlock := t.node.chain.StableBlock()
	log.Infof("stable block height = %v", stableBlock.Height())
	stableHeader := stableBlock.Header

	p := t.node.chain.TxProcessor()
	ret, costGas, err := p.CallTx(ctx, stableHeader, to, txType, data, common.Hash{}, timeout)

	return ret, costGas, err
}
