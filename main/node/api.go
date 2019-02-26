package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
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
	for _, v := range rewardMap {
		result = append(result, v)
	}
	return result, nil
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

//go:generate gencodec -type CandidateListRes --field-override candidateListResMarshaling -out gen_candidate_list_res_json.go
type CandidateListRes struct {
	CandidateList []*CandidateInfo `json:"candidateList" gencodec:"required"`
	Total         uint32           `json:"total" gencodec:"required"`
}
type candidateListResMarshaling struct {
	Total hexutil.Uint32
}

// GetDeputyNodeList
func (c *PublicChainAPI) GetDeputyNodeList() []string {
	return deputynode.Instance().GetLatestDeputies()
}

// GetCandidateNodeList get all candidate node list information and return total candidate node
func (c *PublicChainAPI) GetCandidateList(index, size int) (*CandidateListRes, error) {
	addresses, total, err := c.chain.Db().GetCandidatesPage(index, size)
	if err != nil {
		return nil, err
	}
	candidateList := make([]*CandidateInfo, 0, len(addresses))
	for i := 0; i < len(addresses); i++ {
		candidateAccount := c.chain.AccountManager().GetAccount(addresses[i])
		mapProfile := candidateAccount.GetCandidateProfile()
		if isCandidate, ok := mapProfile[types.CandidateKeyIsCandidate]; !ok || isCandidate == params.NotCandidateNode {
			err = fmt.Errorf("the node of %s is not candidate node", addresses[i].String())
			return nil, err
		}

		candidateInfo := &CandidateInfo{
			Profile: make(map[string]string),
		}

		candidateInfo.Profile = mapProfile
		candidateInfo.Votes = candidateAccount.GetVotes().String()
		candidateInfo.CandidateAddress = addresses[i].String()

		candidateList = append(candidateList, candidateInfo)
	}
	result := &CandidateListRes{
		CandidateList: candidateList,
		Total:         total,
	}
	return result, nil
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
		profile := CandidateAccount.GetCandidateProfile()
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
	err := AvailableTx(tx)
	if err != nil {
		return common.Hash{}, err
	}
	err = t.node.txPool.AddTx(tx)
	return tx.Hash(), err
}

// SendReimbursedGasTx gas代付交易 todo 测试使用
func (t *PublicTxAPI) SendReimbursedGasTx(senderPrivate, gasPayerPrivate string, to, gasPayer common.Address, amount int64, data []byte, txType uint8, toName, message string) (common.Hash, error) {
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
	err = AvailableTx(lastSignTx)
	if err != nil {
		return common.Hash{}, err
	}
	err = t.node.txPool.AddTx(lastSignTx)
	return lastSignTx.Hash(), err
}

// IssueToken 发行token交易
func (t *PublicTxAPI) IssueToken(prv *ecdsa.PrivateKey, decimals uint8, amount *big.Int, mineable bool) (common.Hash, error) {
	data := []byte{1} // todo
	tx := types.NewContractCreation(nil, uint64(500000), big.NewInt(1), data, params.IssueTokenTx, t.node.chainID, uint64(time.Now().Unix()+30*60), "", "issue token tx")
	signTx, err := types.SignTx(tx, types.MakeSigner(), prv)
	if err != nil {
		return common.Hash{}, err
	}
	return t.SendTx(signTx)
}

// AdditionalToken 增发token交易
func (t *PublicTxAPI) AdditionalToken(prv *ecdsa.PrivateKey, code int32, amount *big.Int, receiver common.Address) (common.Address, error) {

}

// TradingToken 交易token,包含调用智能合约交易
func (t *PublicTxAPI) TradingToken(prv *ecdsa.PrivateKey, to common.Address, code int32, amount *big.Int, input []byte) (common.Address, error) {

}

// IssueAssert 发行资产交易
func (t *PublicTxAPI) IssueAssert(prv *ecdsa.PrivateKey, decimals uint8, metaDataHash []common.Hash, mineable bool) (common.Hash, error) {

}

// AdditionalAssert 增发资产交易
func (t *PublicTxAPI) AdditionalAssert(prv *ecdsa.PrivateKey, code int32, metaDataHash []common.Hash, receiver common.Address) (common.Hash, error) {

}

// TradingAssert 交易Assert
func (t *PublicTxAPI) TradingAssert(prv *ecdsa.PrivateKey, to common.Address, code int32, tokenIds []common.Hash, input []byte) (common.Hash, error) {

}

// ModifyTokenAssertInfo 修改token/资产info
func (t *PublicTxAPI) ModifyTokenAssertInfo(prv *ecdsa.PrivateKey, info map[string]interface{}, code uint32) (common.Hash, error) {

}

// AvailableTx transaction parameter verification
func AvailableTx(tx *types.Transaction) error {
	toNameLength := len(tx.ToName())
	if toNameLength > MaxTxToNameLength {
		toNameErr := fmt.Errorf("the length of toName field in transaction is out of max length limit. toName length = %d. max length limit = %d. ", toNameLength, MaxTxToNameLength)
		return toNameErr
	}
	txMessageLength := len(tx.Message())
	if txMessageLength > MaxTxMessageLength {
		txMessageErr := fmt.Errorf("the length of message field in transaction is out of max length limit. message length = %d. max length limit = %d. ", txMessageLength, MaxTxMessageLength)
		return txMessageErr
	}
	switch tx.Type() {
	case params.OrdinaryTx:
		if tx.To() == nil {
			if len(tx.Data()) == 0 {
				createContractErr := errors.New("The data of contract creation transaction can't be null ")
				return createContractErr
			}
		}
	case params.VoteTx:
	case params.RegisterTx:
		if len(tx.Data()) == 0 {
			registerTxErr := errors.New("The data of contract creation transaction can't be null ")

			return registerTxErr
		}
	default:
		txTypeErr := fmt.Errorf("transaction type error. txType = %v", tx.Type())
		return txTypeErr
	}
	return nil
}

// PendingTx
func (t *PublicTxAPI) PendingTx(size int) []*types.Transaction {
	return t.node.txPool.Pending(size)
}

// GetTxByHash pull the specified transaction through a transaction hash
func (t *PublicTxAPI) GetTxByHash(hash string) (*store.VTransactionDetail, error) {
	txHash := common.HexToHash(hash)
	bizDb := t.node.db.GetBizDatabase()
	vTxDetail, err := bizDb.GetTxByHash(txHash)
	return vTxDetail, err
}

//go:generate gencodec -type TxListRes --field-override txListResMarshaling -out gen_tx_list_res_json.go
type TxListRes struct {
	VTransactions []*store.VTransaction `json:"txList" gencodec:"required"`
	Total         uint32                `json:"total" gencodec:"required"`
}
type txListResMarshaling struct {
	Total hexutil.Uint32
}

// GetTxListByAddress pull the list of transactions
func (t *PublicTxAPI) GetTxListByAddress(lemoAddress string, index int, size int) (*TxListRes, error) {
	src, err := common.StringToAddress(lemoAddress)
	if err != nil {
		return nil, err
	}
	bizDb := t.node.db.GetBizDatabase()
	vTxs, total, err := bizDb.GetTxByAddr(src, index, size)
	if err != nil {
		return nil, err
	}
	txList := &TxListRes{
		VTransactions: vTxs,
		Total:         total,
	}

	return txList, nil
}

// ReadContract read variables in a contract includes the return value of a function.
func (t *PublicTxAPI) ReadContract(to *common.Address, data hexutil.Bytes) (string, error) {
	ctx := context.Background()
	result, _, err := t.doCall(ctx, to, params.OrdinaryTx, data, 5*time.Second)
	return common.ToHex(result), err
}

// EstimateGas returns an estimate of the amount of gas needed to execute the given transaction.
func (t *PublicTxAPI) EstimateGas(to *common.Address, txType uint8, data hexutil.Bytes) (string, error) {
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
func (t *PublicTxAPI) doCall(ctx context.Context, to *common.Address, txType uint8, data hexutil.Bytes, timeout time.Duration) ([]byte, uint64, error) {
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
