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
	TotalLEMO              = common.Lemo2Mo("1600000000")                                  // 1.6 billion
	DefaultFounder         = mustDecodeAddress("Lemo83RJZ3ATGP2CDQYSRN9AWZQZQ8P24694G5Z3") // Initial LEMO holder
	DefaultDeputyNodesInfo = infos{
		&CandidateInfo{
			MinerAddress:  mustDecodeAddress("Lemo83PT4CRWS4QQ3KQN5HSQ9KY9C2ZAYWGANZY8"),
			IncomeAddress: mustDecodeAddress("Lemo83R9DWRRD9HDW8HATA9R558ANQ93R4AQBFHQ"),
			NodeID:        common.FromHex("0xa496266523fbdc3541d238fbc161ac4aad2b9939c01553e5ec6969581dfe33946703c720cf09870b0f90b13894153a363a17634d78303d28639f63d8c0182e92"),
			Host:          "127.0.0.1",
			Port:          "7001",
			Introduction:  "LemoChain Node 1",
		},
		&CandidateInfo{
			MinerAddress:  mustDecodeAddress("Lemo848R4PKJ8CZ2TBQC58K92Z2S3N23SNSNAQ8K"),
			IncomeAddress: mustDecodeAddress("Lemo835SBHCBGGTKR6GY67GNZ6KG9BB85QNTZ5CP"),
			NodeID:        common.FromHex("0xfc5bd09388220b8c9887edb58f30aee2dfc1cbfddb2842e20b3417332518862f03a37e4d93378c2171f263d007038b9c550dbe06f5b5557a47757ee96cc0e7da"),
			Host:          "127.0.0.1",
			Port:          "7002",
			Introduction:  "LemoChain Node 2",
		},
		&CandidateInfo{
			MinerAddress:  mustDecodeAddress("Lemo848RTFC7D55284Z6RYSBRYNQ7968SSZ74269"),
			IncomeAddress: mustDecodeAddress("Lemo837F67QG2AC472SRW4PD8RWDB5SQ4H6K9HQP"),
			NodeID:        common.FromHex("0xc2061403840154c5adfa3aba18a51b246964078ff055dc745bdb77f97fa0a97ee453a13e313ca20ec9bec37bde590e96d1fd29dd49b65e159ddcfcc5771d3077"),
			Host:          "127.0.0.1",
			Port:          "7003",
			Introduction:  "LemoChain Node 3",
		},
		&CandidateInfo{
			MinerAddress:  mustDecodeAddress("Lemo832363YDBAAKZKTKKA24HTZ99288487RF3QG"),
			IncomeAddress: mustDecodeAddress("Lemo83FW5J9DJ9PTNTFGS9F5FW554QKFFBRNP573"),
			NodeID:        common.FromHex("0xe9e437bd288d0984197cb8b7c9a6a7e28467613d182c5bad734953516c8e8775119a3eb86eb6bc3a25f0c5c93723553cbdd7909de0108952f17e4fe72af7b05b"),
			Host:          "127.0.0.1",
			Port:          "7004",
			Introduction:  "LemoChain Node 4",
		},
		&CandidateInfo{
			MinerAddress:  mustDecodeAddress("Lemo837PN8C5JB9GG4CYZR3KHDPCDWNY2FWT7KKD"),
			IncomeAddress: mustDecodeAddress("Lemo83QJCRRWR4J75ANNWNSGZFJBS5DW3AGG4NSG"),
			NodeID:        common.FromHex("0xa974b10a4ca92e50db6ba2578abe3979787315f081ae3e2b4fbbd93cca45efae75b33ad079d4dea20dd4557b3d746a7f04bfcb362f0347ece7c9677d01e220fe"),
			Host:          "127.0.0.1",
			Port:          "7005",
			Introduction:  "LemoChain Node 5",
		},
		&CandidateInfo{
			MinerAddress:  mustDecodeAddress("Lemo83QP4RJWR7QCTPRAQ824897A8BJ5DJDGZD2T"),
			IncomeAddress: mustDecodeAddress("Lemo83QQGTDAYN4SW462WTBKTKBHSDKW96D2W848"),
			NodeID:        common.FromHex("0xeebeeb9528bc8d172dcf47076eae31da5989ec81eb528cafc566fed593d4a09c20198dc51eecc3b2634ba374d98e0e4bf4d9d2b3c11d5e99b6641745e5e23a9a"),
			Host:          "127.0.0.1",
			Port:          "7006",
			Introduction:  "LemoChain Node 6",
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

func mustDecodeAddress(input string) common.Address {
	if address, err := common.StringToAddress(input); err == nil {
		return address
	}
	panic(fmt.Sprintf("deputy nodes have invalid miner address: %s", input))
}

//go:generate gencodec -type Genesis -field-override genesisSpecMarshaling -out gen_genesis_json.go
type Genesis struct {
	Time            uint32           `json:"timestamp"`
	ExtraData       string           `json:"extraData"`
	GasLimit        uint64           `json:"gasLimit"`
	Founder         common.Address   `json:"founder"       gencodec:"required"`
	DeputyNodesInfo []*CandidateInfo `json:"deputyNodesInfo"   gencodec:"required"`
}

type genesisSpecMarshaling struct {
	Time            hexutil.Uint32
	GasLimit        hexutil.Uint64
	DeputyNodesInfo []*CandidateInfo
}

// DefaultGenesisConfig default genesis block config
func DefaultGenesisConfig() *Genesis {
	timeSpan, _ := time.ParseInLocation("2006-01-02 15:04:05", "2018-08-30 12:00:00", time.UTC)
	return &Genesis{
		Time:            uint32(timeSpan.Unix()),
		ExtraData:       "",
		GasLimit:        params.GenesisGasLimit,
		Founder:         DefaultFounder,
		DeputyNodesInfo: DefaultDeputyNodesInfo,
	}
}

func (g *Genesis) Verify() error {
	defaultGenesis := DefaultGenesisConfig()
	if g.Time == 0 {
		g.Time = defaultGenesis.Time
	}
	if g.GasLimit == 0 {
		g.GasLimit = defaultGenesis.GasLimit
	}

	if int64(g.Time) > time.Now().Unix() {
		return ErrGenesisTimeTooLarge
	}
	if len(g.ExtraData) > 256 {
		return ErrGenesisExtraTooLong
	}
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
