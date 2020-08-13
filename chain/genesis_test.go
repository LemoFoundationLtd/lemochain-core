package chain

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

func getTestGenesis() *Genesis {
	return &Genesis{
		Time:      123,
		ExtraData: []byte("abc"),
		GasLimit:  456,
		Founder:   common.HexToAddress("0x01"),
		DeputyNodesInfo: infos{
			&CandidateInfo{
				MinerAddress:  common.HexToAddress("0x02"),
				IncomeAddress: mustDecodeAddress("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG"),
				NodeID:        common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"),
				Host:          "10.0.22.23",
				Port:          "7001",
				Introduction:  "the first node",
			},
		},
	}
}

func TestGenesis_Verify(t *testing.T) {
	genesis := getTestGenesis()
	assert.NoError(t, genesis.Verify())
	expect := getTestGenesis()
	assert.Equal(t, expect.Time, genesis.Time)
	assert.Equal(t, expect.ExtraData, genesis.ExtraData)
	assert.Equal(t, expect.GasLimit, genesis.GasLimit)
	assert.Equal(t, expect.Founder, genesis.Founder)
	assert.Equal(t, expect.DeputyNodesInfo, genesis.DeputyNodesInfo)

	genesis = getTestGenesis()
	genesis.Time = 0
	assert.NoError(t, genesis.Verify())
	assert.Equal(t, DefaultGenesisConfig().Time, genesis.Time)

	genesis = getTestGenesis()
	genesis.Time = uint32(time.Now().Unix() + 1)
	assert.Equal(t, ErrGenesisTimeTooLarge, genesis.Verify())

	genesis = getTestGenesis()
	genesis.ExtraData = nil
	assert.NoError(t, genesis.Verify())

	genesis = getTestGenesis()
	var extra [257]byte
	genesis.ExtraData = (extra)[:]
	assert.Equal(t, ErrGenesisExtraTooLong, genesis.Verify())

	genesis = getTestGenesis()
	genesis.GasLimit = 0
	assert.NoError(t, genesis.Verify())
	assert.Equal(t, DefaultGenesisConfig().GasLimit, genesis.GasLimit)

	genesis = getTestGenesis()
	genesis.Founder = common.Address{}
	assert.NoError(t, genesis.Verify())

	genesis = getTestGenesis()
	genesis.DeputyNodesInfo = infos{}
	assert.Equal(t, ErrNoDeputyNodes, genesis.Verify())

	genesis = getTestGenesis()
	genesis.DeputyNodesInfo[0].NodeID = genesis.DeputyNodesInfo[0].NodeID[1:]
	assert.Equal(t, ErrInvalidDeputyNodes, genesis.Verify())
}

func TestSetupGenesisBlock(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()

	// customised genesis
	genesis := getTestGenesis()
	genesisBlock := SetupGenesisBlock(db, genesis)
	block, err := db.GetBlockByHeight(0)
	assert.NoError(t, err)
	assert.Equal(t, genesisBlock.Hash(), block.Hash())
	assert.Equal(t, genesis.Time, block.Time())
	founder, err := db.GetAccount(genesis.Founder)
	assert.NoError(t, err)
	assert.Equal(t, true, founder.Balance.Cmp(new(big.Int)) > 0)
}

func TestSetupGenesisBlock_Empty(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()

	// no genesis
	genesisBlock := SetupGenesisBlock(db, nil)
	block, err := db.GetBlockByHeight(0) // load first stable block
	assert.NoError(t, err)
	assert.Equal(t, genesisBlock.Hash(), block.Hash())
	assert.Equal(t, DefaultGenesisConfig().Time, block.Time())
	founder, err := db.GetAccount(DefaultGenesisConfig().Founder)
	assert.NoError(t, err)
	assert.Equal(t, true, founder.Balance.Cmp(new(big.Int)) > 0)
}

func TestSetupGenesisBlock_Exist(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()

	SetupGenesisBlock(db, nil)
	// setup when genesis exist
	assert.PanicsWithValue(t, ErrSaveGenesisFail, func() {
		SetupGenesisBlock(db, nil)
	})
}

func TestGenesis_ToBlock(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)

	genesis := getTestGenesis()
	block, err := genesis.ToBlock(am)
	assert.NoError(t, err)
	assert.Equal(t, common.Hash{}, block.ParentHash())
	assert.Equal(t, genesis.Time, block.Time())
	assert.Equal(t, genesis.ExtraData, block.Extra())
	assert.Equal(t, genesis.GasLimit, block.GasLimit())
	assert.Equal(t, genesis.Founder, block.MinerAddress())
	assert.Equal(t, len(genesis.DeputyNodesInfo), len(block.DeputyNodes))
	assert.Len(t, block.ChangeLogs, 3) // 初始化的16亿的balanceLog和初始化的候选节点的candidateLog
	assert.Len(t, block.Txs, 0)
	assert.Equal(t, common.Sha3Nil, block.TxRoot())
	assert.NotEqual(t, 0, block.Height())
	assert.NotEqual(t, common.Sha3Nil, block.DeputyRoot())
	assert.NotEqual(t, common.Sha3Nil, block.VersionRoot())
	assert.NotEqual(t, common.Sha3Nil, block.LogRoot())
}
