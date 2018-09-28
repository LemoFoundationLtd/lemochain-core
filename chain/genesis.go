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
// //go:generate gencodec -type GenesisAccount -field-override genesisAccountMarshaling -out gen_genesis_account.go

type Genesis struct {
	// Config    *params.ChainConfig `json:"config"`
	Time      uint64         `json:"timestamp"`
	ExtraData []byte         `json:"extraData"`
	GasLimit  uint64         `json:"gasLimit"    gencodec:"required"`
	LemoBase  common.Address `json:"lemoBase"    gencodec:"required"`
	// Alloc     GenesisAlloc        `json:"alloc"       gencodec:"required"`
}

type genesisSpecMarshaling struct {
	Time      math.HexOrDecimal64
	ExtraData hexutil.Bytes
	GasLimit  math.HexOrDecimal64
	// Alloc     map[common.UnprefixedAddress]GenesisAccount
}

// type GenesisAccount struct {
// 	Balance *big.Int
// }

// type genesisAccountMarshaling struct {
// 	Balance *math.HexOrDecimal256
// }

// DefaultGenesisBlock 默认创始区块配置
func DefaultGenesisBlock() *Genesis {
	timeSpan, err := time.ParseInLocation("2006-01-02 15:04:05", "2018-08-30 12:00:00", time.Local)
	if err != nil {
		timeSpan = time.Now()
	}
	return &Genesis{
		// Config:    params.DefaultChainConfig(),
		Time:      uint64(timeSpan.Unix()),
		ExtraData: []byte(""),
		GasLimit:  0x6422c40,
		LemoBase:  common.HexToAddress("0x015780F8456F9c1532645087a19DcF9a7e0c7F97"),
	}
}

// decodePrealloc 解析初始化账户余额信息
// func decodePrealloc(input string) GenesisAlloc {
// 	var p []struct{ Addr, Balance *big.Int }
// 	if err := rlp.NewStream(strings.NewReader(input), 0).Decode(&p); err != nil {
// 		panic(err)
// 	}
// 	ga := make(GenesisAlloc, len(p))
// 	for _, acc := range p {
// 		account := GenesisAccount{Balance: acc.Balance}
// 		ga[common.BigToAddress(acc.Addr)] = account
// 	}
// 	return ga
// }

// GenesisAlloc 创始块初始化状态
// type GenesisAlloc map[common.Address]GenesisAccount

// SetupGenesisBlock 设置创始区块
func SetupGenesisBlock(db protocol.ChainDB, genesis *Genesis) (common.Hash, error) {
	// if genesis != nil && genesis.Config == nil {
	// 	return nil, common.Hash{}, fmt.Errorf("setup genesis block failed. not set config")
	// }
	if genesis == nil {
		log.Info("Writing default genesis block.")
		genesis = DefaultGenesisBlock()
	}
	am := account.NewManager(common.Hash{}, db)
	block := genesis.ToBlock(am)
	if err := am.Finalise(); err != nil {
		return common.Hash{}, fmt.Errorf("setup genesis block failed: %v", err)
	}
	block.Header.VersionRoot = am.GetVersionRoot()
	logs := am.GetChangeLogs()
	block.SetChangeLog(logs)
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

// ToBlock 生成创始区块
func (g *Genesis) ToBlock(am *account.Manager) *types.Block {
	head := &types.Header{
		ParentHash: common.Hash{},
		LemoBase:   g.LemoBase,
		TxRoot:     common.Hash{},
		EventRoot:  common.Hash{},
		Height:     0,
		GasLimit:   g.GasLimit,
		Extra:      g.ExtraData,
		Time:       new(big.Int).SetUint64(g.Time),
	}
	lemoBase := am.GetAccount(g.LemoBase)
	log.Infof("%d %d", lemoBase.GetVersion(), lemoBase.GetBalance().Uint64())
	oneLemo := new(big.Int).SetUint64(1000000000000000000) // 1 lemo
	total := new(big.Int).SetUint64(1600000000)
	total = total.Mul(total, oneLemo)
	lemoBase.SetBalance(total)
	block := types.NewBlock(head, nil, nil, nil, nil)
	return block
}
