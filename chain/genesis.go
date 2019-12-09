package chain

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/transaction"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"math/big"
	"time"
)

var (
	ErrBuildGenesisFail    = errors.New("build genesis block failed")
	ErrSaveGenesisFail     = errors.New("save genesis block failed")
	ErrGenesisExtraTooLong = errors.New("genesis config's extraData is longer than 256")
	ErrGenesisTimeTooLarge = errors.New("genesis config's time is larger than current time")
	ErrNoDeputyNodes       = errors.New("no deputy nodes in genesis")
	ErrInvalidDeputyNodes  = errors.New("genesis config's deputy nodes are invalid")
)

type infos []*CandidateInfo

//go:generate gencodec -type CandidateInfo -field-override candidateInfoMarshaling -out gen_candidateInfo_json.go
type CandidateInfo struct {
	MinerAddress  common.Address `json:"minerAddress" gencodec:"required"`
	IncomeAddress common.Address `json:"incomeAddress" gencodec:"required"`
	NodeID        []byte         `json:"nodeID" gencodec:"required"`
	Host          string         `json:"host" gencodec:"required"`
	Port          string         `json:"port" gencodec:"required"`
	Introduction  string         `json:"introduction"`
}

type candidateInfoMarshaling struct {
	NodeID hexutil.Bytes
}

func (info *CandidateInfo) check() error {
	profile := make(types.Profile)
	profile[types.CandidateKeyIncomeAddress] = info.IncomeAddress.String()
	profile[types.CandidateKeyNodeID] = common.ToHex(info.NodeID)
	profile[types.CandidateKeyHost] = info.Host
	profile[types.CandidateKeyPort] = info.Port
	profile[types.CandidateKeyIntroduction] = info.Introduction
	// check
	return transaction.CheckRegisterTxProfile(profile)
}

var (
	TotalLEMO              = common.Lemo2Mo("1600000000")                                   // 1.6 billion
	DefaultFounder         = decodeMinerAddress("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG") // Initial LEMO holder
	DefaultDeputyNodesInfo = infos{
		&CandidateInfo{
			MinerAddress:  DefaultFounder,
			IncomeAddress: decodeMinerAddress("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG"),
			NodeID:        common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"),
			Host:          "10.0.22.23",
			Port:          "7001",
			Introduction:  "the first node",
		},
		&CandidateInfo{
			MinerAddress:  decodeMinerAddress("Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY"),
			IncomeAddress: decodeMinerAddress("Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY"),
			NodeID:        common.FromHex("0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0"),
			Host:          "10.0.22.23",
			Port:          "7002",
			Introduction:  "the second node",
		},
		&CandidateInfo{
			MinerAddress:  decodeMinerAddress("Lemo842BJZ4DKCC764C63Y6A943775JH6NQ3Z33Y"),
			IncomeAddress: decodeMinerAddress("Lemo842BJZ4DKCC764C63Y6A943775JH6NQ3Z33Y"),
			NodeID:        common.FromHex("0x7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2d0347692f36ef736f342fb5ce4710f7e337f062cc2110d134b63a9575f78cb167bfae2f43"),
			Host:          "10.0.22.23",
			Port:          "7003",
			Introduction:  "the third node",
		},
		&CandidateInfo{
			MinerAddress:  decodeMinerAddress("Lemo837QGPS3YNTYNF53CD88WA5DR3ABNA95W2DG"),
			IncomeAddress: decodeMinerAddress("Lemo837QGPS3YNTYNF53CD88WA5DR3ABNA95W2DG"),
			NodeID:        common.FromHex("0x34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"),
			Host:          "10.0.22.23",
			Port:          "7004",
			Introduction:  "the fourth node",
		},
		&CandidateInfo{
			MinerAddress:  decodeMinerAddress("Lemo83HKZK68JQZDRGS5PWT2ZBSKR5CRADCSJB9B"),
			IncomeAddress: decodeMinerAddress("Lemo83HKZK68JQZDRGS5PWT2ZBSKR5CRADCSJB9B"),
			NodeID:        common.FromHex("0x5b980ffb1b463fce4773a22ebf376c07c6207023b016b36ccfaba7be1cd1ab4a91737741cd43b7fcb10879e0fcf314d69fa953daec0f02be0f8f9cedb0cb3797"),
			Host:          "10.0.22.24",
			Port:          "7005",
			Introduction:  "the fifth node",
		},
		&CandidateInfo{
			MinerAddress:  decodeMinerAddress("Lemo83W3DBN8QASNAR2D5386QSNGC8DAN8TSRK53"),
			IncomeAddress: decodeMinerAddress("Lemo83W3DBN8QASNAR2D5386QSNGC8DAN8TSRK53"),
			NodeID:        common.FromHex("0x0e53292ab5a51286d64422344c6b0751dc1429497fe72820a0a273c70e35bbbe8196af0c5526588fee62f1b68558773501d32e5d552fd9863d740f30ed41f4b0"),
			Host:          "10.0.22.25",
			Port:          "7006",
			Introduction:  "the sixth node",
		},
	}
)

// buildDeputyNodes 通过candidate info 来构建出deputy node
func buildDeputyNodes(DeputyNodesInfo []*CandidateInfo) types.DeputyNodes {
	deputyNodes := make(types.DeputyNodes, 0)
	for i, info := range DeputyNodesInfo {
		node := &types.DeputyNode{
			MinerAddress: info.MinerAddress,
			NodeID:       info.NodeID,
			Rank:         uint32(i),
			Votes:        new(big.Int).SetInt64(0), // 初始的代理节点列表中的votes都为0，因为初始的时候没有一个账户中有lemo.
		}
		deputyNodes = append(deputyNodes, node)
	}
	return deputyNodes
}

func decodeMinerAddress(input string) common.Address {
	if address, err := common.StringToAddress(input); err == nil {
		return address
	}
	panic(fmt.Sprintf("deputy nodes have invalid miner address: %s", input))
}

//go:generate gencodec -type Genesis -field-override genesisSpecMarshaling -out gen_genesis_json.go
type Genesis struct {
	Time            uint32           `json:"timestamp"     gencodec:"required"`
	ExtraData       []byte           `json:"extraData"`
	GasLimit        uint64           `json:"gasLimit"      gencodec:"required"`
	Founder         common.Address   `json:"founder"       gencodec:"required"`
	DeputyNodesInfo []*CandidateInfo `json:"deputyNodesInfo"   gencodec:"required"`
}

type genesisSpecMarshaling struct {
	Time            hexutil.Uint32
	ExtraData       hexutil.Bytes
	GasLimit        hexutil.Uint64
	DeputyNodesInfo []*CandidateInfo
}

// DefaultGenesisConfig default genesis block config
func DefaultGenesisConfig() *Genesis {
	timeSpan, _ := time.ParseInLocation("2006-01-02 15:04:05", "2018-08-30 12:00:00", time.UTC)
	return &Genesis{
		Time:            uint32(timeSpan.Unix()),
		ExtraData:       []byte(""),
		GasLimit:        params.GenesisGasLimit,
		Founder:         DefaultFounder,
		DeputyNodesInfo: DefaultDeputyNodesInfo,
	}
}

func (g *Genesis) Verify() error {
	if len(g.ExtraData) > 256 {
		return ErrGenesisExtraTooLong
	}

	// check genesis block's time
	if int64(g.Time) > time.Now().Unix() {
		return ErrGenesisTimeTooLarge
	}
	// check deputy nodes
	if len(g.DeputyNodesInfo) == 0 {
		return ErrNoDeputyNodes
	}
	for _, info := range g.DeputyNodesInfo {
		if err := info.check(); err != nil {
			return ErrInvalidDeputyNodes
		}
	}
	return nil
}

// SetupGenesisBlock setup genesis block
func SetupGenesisBlock(db protocol.ChainDB, genesis *Genesis) *types.Block {
	if genesis == nil {
		log.Info("Writing default genesis block.")
		genesis = DefaultGenesisConfig()
	}
	if err := genesis.Verify(); err != nil {
		panic(err)
	}

	am := account.NewManager(common.Hash{}, db)
	block, err := genesis.ToBlock(am)
	if err != nil {
		log.Errorf("build genesis block failed: %v", err)
		panic(ErrBuildGenesisFail)
	}
	hash := block.Hash()
	if err := db.SetBlock(hash, block); err != nil {
		log.Errorf("setup genesis block failed: %v", err)
		panic(ErrSaveGenesisFail)
	}
	if err := am.Save(hash); err != nil {
		log.Errorf("setup genesis block failed: %v", err)
		panic(ErrSaveGenesisFail)
	}
	if _, err := db.SetStableBlock(hash); err != nil {
		log.Errorf("setup genesis block failed: %v", err)
		panic(ErrSaveGenesisFail)
	}
	return block
}

// ToBlock
func (g *Genesis) ToBlock(am *account.Manager) (*types.Block, error) {
	// set balance for some account
	am.GetAccount(g.Founder).SetBalance(TotalLEMO)
	// 注册第一届候选节点info
	g.initCandidateListInfo(am)
	// register candidate node for first term deputy nodes
	deputyNodes := buildDeputyNodes(g.DeputyNodesInfo)
	err := am.Finalise()
	if err != nil {
		return nil, err
	}
	logs := am.GetChangeLogs()

	header := &types.Header{
		ParentHash:   common.Hash{},
		MinerAddress: g.Founder,
		TxRoot:       (types.Transactions{}).MerkleRootSha(),
		Height:       0,
		GasLimit:     g.GasLimit,
		Extra:        g.ExtraData,
		Time:         g.Time,
		DeputyRoot:   deputyNodes.MerkleRootSha().Bytes(),
		VersionRoot:  am.GetVersionRoot(),
		LogRoot:      logs.MerkleRootSha(),
	}
	block := types.NewBlock(header, nil, logs)
	block.SetDeputyNodes(deputyNodes)
	return block, nil
}

// initCandidateListInfo 设置初始的候选节点列表的info
func (g *Genesis) initCandidateListInfo(am *account.Manager) {
	for _, v := range g.DeputyNodesInfo {
		acc := am.GetAccount(v.MinerAddress)
		newProfile := make(map[string]string, 7)
		newProfile[types.CandidateKeyIsCandidate] = types.IsCandidateNode
		newProfile[types.CandidateKeyIncomeAddress] = v.IncomeAddress.String()
		newProfile[types.CandidateKeyNodeID] = common.ToHex(v.NodeID)
		newProfile[types.CandidateKeyHost] = v.Host
		newProfile[types.CandidateKeyPort] = v.Port
		newProfile[types.CandidateKeyIntroduction] = v.Introduction
		newProfile[types.CandidateKeyDepositAmount] = big.NewInt(0).String()
		transaction.InitCandidateProfile(acc, newProfile)
		// make a changelog so the vote logic can be init correctly
		acc.SetVotes(big.NewInt(0))
	}
}
