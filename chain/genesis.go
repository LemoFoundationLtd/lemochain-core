package chain

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/math"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"math/big"
	"time"
)

//go:generate gencodec -type Genesis -field-override genesisSpecMarshaling -out gen_genesis.go

type Genesis struct {
	Time      uint64         `json:"timestamp"   gencodec:"required"`
	ExtraData []byte         `json:"extraData"`
	GasLimit  uint64         `json:"gasLimit"    gencodec:"required"`
	LemoBase  common.Address `json:"lemoBase"    gencodec:"required"`
}

type genesisSpecMarshaling struct {
	Time      math.HexOrDecimal64
	ExtraData hexutil.Bytes
	GasLimit  math.HexOrDecimal64
}

// DefaultGenesisBlock default genesis block
func DefaultGenesisBlock() *Genesis {
	timeSpan, _ := time.ParseInLocation("2006-01-02 15:04:05", "2018-08-30 12:00:00", time.UTC)
	return &Genesis{
		Time:      uint64(timeSpan.Unix()),
		ExtraData: []byte(""),
		GasLimit:  105000000,
		LemoBase:  common.HexToAddress("0x015780F8456F9c1532645087a19DcF9a7e0c7F97"),
	}
}

// SetupGenesisBlock setup genesis block
func SetupGenesisBlock(db protocol.ChainDB, genesis *Genesis) (common.Hash, error) {
	if genesis == nil {
		log.Info("Writing default genesis block.")
		genesis = DefaultGenesisBlock()
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
	head := &types.Header{
		ParentHash: common.Hash{},
		LemoBase:   g.LemoBase,
		TxRoot:     common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"), // empty merkle
		EventRoot:  common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"), // empty merkle
		Height:     0,
		GasLimit:   g.GasLimit,
		Extra:      g.ExtraData,
		Time:       new(big.Int).SetUint64(g.Time),
	}
	block := types.NewBlock(head, nil, nil, nil, nil)
	return block
}

func (g *Genesis) setBalance(am *account.Manager) {
	lemoBase := am.GetAccount(g.LemoBase)
	oneLemo := new(big.Int).SetUint64(1000000000000000000) // 1 lemo
	total := new(big.Int).SetUint64(1600000000)
	total = total.Mul(total, oneLemo)
	lemoBase.SetBalance(total)
}
