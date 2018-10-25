package chain

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/math"
	"github.com/LemoFoundationLtd/lemochain-go/common/merkle"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"math/big"
	"net"
	"time"
)

//go:generate gencodec -type Genesis -field-override genesisSpecMarshaling -out gen_genesis.go

type Genesis struct {
	Time        uint64                 `json:"timestamp"   gencodec:"required"`
	ExtraData   []byte                 `json:"extraData"`
	GasLimit    uint64                 `json:"gasLimit"    gencodec:"required"`
	LemoBase    common.Address         `json:"lemoBase"    gencodec:"required"`
	DeputyNodes deputynode.DeputyNodes `json:"deputyNodes" gencodec:"required"`
}

type genesisSpecMarshaling struct {
	Time        math.HexOrDecimal64
	ExtraData   hexutil.Bytes
	GasLimit    math.HexOrDecimal64
	DeputyNodes []*deputynode.DeputyNode
}

// DefaultGenesisBlock default genesis block
func DefaultGenesisBlock() *Genesis {
	timeSpan, _ := time.ParseInLocation("2006-01-02 15:04:05", "2018-08-30 12:00:00", time.UTC)
	return &Genesis{
		Time:      uint64(timeSpan.Unix()),
		ExtraData: []byte(""),
		GasLimit:  105000000,
		LemoBase:  common.HexToAddress("0x015780F8456F9c1532645087a19DcF9a7e0c7F97"),
		DeputyNodes: deputynode.DeputyNodes{
			&deputynode.DeputyNode{
				LemoBase: common.HexToAddress("0x015780F8456F9c1532645087a19DcF9a7e0c7F97"),
				NodeID:   common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"),
				IP:       net.ParseIP("127.0.0.1"),
				Port:     7001,
				Rank:     0,
				Votes:    50000,
			},
			&deputynode.DeputyNode{
				LemoBase: common.HexToAddress("0x016ad4Fc7e1608685Bf5fe5573973BF2B1Ef9B8A"),
				NodeID:   common.FromHex("0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0"),
				IP:       net.ParseIP("127.0.0.1"),
				Port:     7002,
				Rank:     1,
				Votes:    40000,
			},
			&deputynode.DeputyNode{
				LemoBase: common.HexToAddress("0x01f98855Be9ecc5c23A28Ce345D2Cc04686f2c61"),
				NodeID:   common.FromHex("0x7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2d0347692f36ef736f342fb5ce4710f7e337f062cc2110d134b63a9575f78cb167bfae2f43"),
				IP:       net.ParseIP("127.0.0.1"),
				Port:     7003,
				Rank:     2,
				Votes:    30000,
			},
			&deputynode.DeputyNode{
				LemoBase: common.HexToAddress("0x0112fDDcF0C08132A5dcd9ED77e1a3348ff378D2"),
				NodeID:   common.FromHex("0x34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"),
				IP:       net.ParseIP("127.0.0.1"),
				Port:     7004,
				Rank:     3,
				Votes:    20000,
			},
			&deputynode.DeputyNode{
				LemoBase: common.HexToAddress("0x016017aF50F4bB67101CE79298ACBdA1A3c12C15"),
				NodeID:   common.FromHex("0x5b980ffb1b463fce4773a22ebf376c07c6207023b016b36ccfaba7be1cd1ab4a91737741cd43b7fcb10879e0fcf314d69fa953daec0f02be0f8f9cedb0cb3797"),
				IP:       net.ParseIP("127.0.0.1"),
				Port:     7005,
				Rank:     4,
				Votes:    10000,
			},
		},
	}
}

// SetupGenesisBlock setup genesis block
func SetupGenesisBlock(db protocol.ChainDB, genesis *Genesis) (common.Hash, error) {
	if genesis == nil {
		log.Info("Writing default genesis block.")
		genesis = DefaultGenesisBlock()
		if len(genesis.DeputyNodes) == 0 {
			panic("default deputy nodes can't be empty")
		}
	}

	// check genesis block's time
	if genesis.Time > uint64(time.Now().Unix()) {
		panic("Genesis block's time can't be larger than current time.")
	}

	am := account.NewManager(common.Hash{}, db)
	block := genesis.ToBlock()
	genesis.setBalance(am)
	if err := am.Finalise(); err != nil {
		return common.Hash{}, fmt.Errorf("setup genesis block failed: %v", err)
	}
	block.Header.VersionRoot = am.GetVersionRoot()
	logs := am.GetChangeLogs()
	block.SetChangeLogs(logs)
	block.Header.LogsRoot = types.DeriveChangeLogsSha(logs)
	hash := block.Hash()
	if err := db.SetBlock(hash, block); err != nil {
		return common.Hash{}, fmt.Errorf("setup genesis block failed: %v", err)
	}
	if err := am.Save(hash); err != nil {
		return common.Hash{}, fmt.Errorf("setup genesis block failed: %v", err)
	}
	if err := db.SetStableBlock(hash); err != nil {
		return common.Hash{}, fmt.Errorf("setup genesis block failed: %v", err)
	}
	return block.Hash(), nil
}

// ToBlock
func (g *Genesis) ToBlock() *types.Block {
	leafHashes := make([]common.Hash, len(g.DeputyNodes))
	for i, n := range g.DeputyNodes {
		leafHashes[i] = n.Hash()
	}
	deputyTree := merkle.New(leafHashes)

	head := &types.Header{
		ParentHash: common.Hash{},
		LemoBase:   g.LemoBase,
		TxRoot:     common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"), // empty merkle
		EventRoot:  common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"), // empty merkle
		Height:     0,
		GasLimit:   g.GasLimit,
		Extra:      g.ExtraData,
		Time:       new(big.Int).SetUint64(g.Time),
		DeputyRoot: deputyTree.Root().Bytes(),
	}
	return types.NewBlock(head, nil, nil, nil, nil, g.DeputyNodes)
}

func (g *Genesis) setBalance(am *account.Manager) {
	lemoBase := am.GetAccount(g.LemoBase)
	oneLemo := new(big.Int).SetUint64(1000000000000000000) // 1 lemo
	total := new(big.Int).SetUint64(1600000000)
	total = total.Mul(total, oneLemo)
	lemoBase.SetBalance(total)
}
