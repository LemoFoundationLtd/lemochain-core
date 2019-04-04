package chain

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"math/big"
	"net"
	"time"
)

var (
	ErrGenesisExtraTooLong = errors.New("genesis config's extraData is longer than 256")
	ErrGenesisTimeTooLarge = errors.New("genesis config's time is larger than current time")
	ErrNoDeputyNodes       = errors.New("no deputy nodes in genesis")
	ErrInvalidDeputyNodes  = errors.New("genesis config's deputy nodes are invalid")
)

var (
	DefaultFounder     = decodeMinerAddress("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG")
	DefaultDeputyNodes = deputynode.DeputyNodes{
		&deputynode.DeputyNode{
			MinerAddress: DefaultFounder,
			NodeID:       common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"),
			IP:           net.ParseIP("120.78.132.151"),
			Port:         7003,
			Rank:         0,
			Votes:        new(big.Int).SetInt64(5),
		},
		&deputynode.DeputyNode{
			MinerAddress: decodeMinerAddress("Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY"),
			NodeID:       common.FromHex("0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0"),
			IP:           net.ParseIP("47.99.139.137"),
			Port:         7003,
			Rank:         1,
			Votes:        new(big.Int).SetInt64(4),
		},
		&deputynode.DeputyNode{
			MinerAddress: decodeMinerAddress("Lemo842BJZ4DKCC764C63Y6A943775JH6NQ3Z33Y"),
			NodeID:       common.FromHex("0x7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2d0347692f36ef736f342fb5ce4710f7e337f062cc2110d134b63a9575f78cb167bfae2f43"),
			IP:           net.ParseIP("39.106.51.181"),
			Port:         7003,
			Rank:         2,
			Votes:        new(big.Int).SetInt64(3),
		},
		&deputynode.DeputyNode{
			MinerAddress: decodeMinerAddress("Lemo837QGPS3YNTYNF53CD88WA5DR3ABNA95W2DG"),
			NodeID:       common.FromHex("0x34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"),
			IP:           net.ParseIP("47.102.138.174"),
			Port:         7003,
			Rank:         3,
			Votes:        new(big.Int).SetInt64(2),
		},
		&deputynode.DeputyNode{
			MinerAddress: decodeMinerAddress("Lemo83HKZK68JQZDRGS5PWT2ZBSKR5CRADCSJB9B"),
			NodeID:       common.FromHex("0x5b980ffb1b463fce4773a22ebf376c07c6207023b016b36ccfaba7be1cd1ab4a91737741cd43b7fcb10879e0fcf314d69fa953daec0f02be0f8f9cedb0cb3797"),
			IP:           net.ParseIP("63.211.111.245"),
			Port:         7003,
			Rank:         4,
			Votes:        new(big.Int).SetInt64(1),
		},
	}
)

func decodeMinerAddress(input string) common.Address {
	if address, err := common.StringToAddress(input); err == nil {
		return address
	}
	panic(fmt.Sprintf("deputy nodes have invalid miner address: %s", input))
}

//go:generate gencodec -type Genesis -field-override genesisSpecMarshaling -out gen_genesis_json.go

type Genesis struct {
	Time        uint32                 `json:"timestamp"     gencodec:"required"`
	ExtraData   []byte                 `json:"extraData"`
	GasLimit    uint64                 `json:"gasLimit"      gencodec:"required"`
	Founder     common.Address         `json:"founder"       gencodec:"required"`
	DeputyNodes deputynode.DeputyNodes `json:"deputyNodes"   gencodec:"required"`
}

type genesisSpecMarshaling struct {
	Time        hexutil.Uint32
	ExtraData   hexutil.Bytes
	GasLimit    hexutil.Uint64
	DeputyNodes []*deputynode.DeputyNode
}

// DefaultGenesisBlock default genesis block
func DefaultGenesisBlock() *Genesis {
	timeSpan, _ := time.ParseInLocation("2006-01-02 15:04:05", "2018-08-30 12:00:00", time.UTC)
	return &Genesis{
		Time:        uint32(timeSpan.Unix()),
		ExtraData:   []byte(""),
		GasLimit:    105000000,
		Founder:     DefaultFounder,
		DeputyNodes: DefaultDeputyNodes,
	}
}

func checkGenesisConfig(genesis *Genesis) error {
	if len(genesis.ExtraData) > 256 {
		return ErrGenesisExtraTooLong
	}

	// check genesis block's time
	if int64(genesis.Time) > time.Now().Unix() {
		return ErrGenesisTimeTooLarge
	}
	// check deputy nodes
	if len(genesis.DeputyNodes) == 0 {
		return ErrNoDeputyNodes
	}
	for _, deputy := range genesis.DeputyNodes {
		if err := deputy.Check(); err != nil {
			return ErrInvalidDeputyNodes
		}
	}
	return nil
}

// SetupGenesisBlock setup genesis block
func SetupGenesisBlock(db protocol.ChainDB, genesis *Genesis) *types.Block {
	if genesis == nil {
		log.Info("Writing default genesis block.")
		genesis = DefaultGenesisBlock()
	}
	if err := checkGenesisConfig(genesis); err != nil {
		panic(err)
	}

	am := account.NewManager(common.Hash{}, db)
	block, err := genesis.ToBlock(am)
	if err != nil {
		panic(fmt.Errorf("build genesis block failed: %v", err))
	}
	hash := block.Hash()
	if err := db.SetBlock(hash, block); err != nil {
		panic(fmt.Errorf("setup genesis block failed: %v", err))
	}
	if err := am.Save(hash); err != nil {
		panic(fmt.Errorf("setup genesis block failed: %v", err))
	}
	if err := db.SetStableBlock(hash); err != nil {
		panic(fmt.Errorf("setup genesis block failed: %v", err))
	}
	return block
}

// ToBlock
func (g *Genesis) ToBlock(am *account.Manager) (*types.Block, error) {
	g.setBalance(am)
	err := am.Finalise()
	if err != nil {
		return nil, err
	}
	logs := am.GetChangeLogs()

	header := &types.Header{
		ParentHash:   common.Hash{},
		MinerAddress: g.Founder,
		TxRoot:       types.DeriveTxsSha(nil),
		Height:       0,
		GasLimit:     g.GasLimit,
		Extra:        g.ExtraData,
		Time:         g.Time,
		DeputyRoot:   types.DeriveDeputyRootSha(g.DeputyNodes).Bytes(),
		VersionRoot:  am.GetVersionRoot(),
		LogRoot:      types.DeriveChangeLogsSha(logs),
	}
	block := types.NewBlock(header, nil, logs)
	block.SetDeputyNodes(g.DeputyNodes)
	return block, nil
}

func (g *Genesis) setBalance(am *account.Manager) {
	total, _ := new(big.Int).SetString("1600000000000000000000000000", 10) // 1.6 billion
	am.GetAccount(g.Founder).SetBalance(total)
}
