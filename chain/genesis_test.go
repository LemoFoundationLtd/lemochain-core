package chain

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"os"
	"testing"
	"time"
)

func GetStorePath() string {
	return "../testdata/genesis"
}

func ClearData() {
	_ = os.RemoveAll(GetStorePath())
}

func getTestGenesis() *Genesis {
	return &Genesis{
		Time:      123,
		ExtraData: []byte("abc"),
		GasLimit:  456,
		Founder:   common.HexToAddress("0x01"),
		DeputyNodes: types.DeputyNodes{
			&types.DeputyNode{
				MinerAddress: common.HexToAddress("0x02"),
				NodeID:       common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"),
				Rank:         0,
				Votes:        new(big.Int).SetInt64(5),
			},
		},
	}
}

func TestGenesis_Verify(t *testing.T) {
	genesis := getTestGenesis()
	var extra [257]byte
	genesis.ExtraData = (extra)[:]
	assert.Equal(t, ErrGenesisExtraTooLong, genesis.Verify())

	genesis = getTestGenesis()
	genesis.Time = uint32(time.Now().Unix() + 1)
	assert.Equal(t, ErrGenesisTimeTooLarge, genesis.Verify())

	genesis = getTestGenesis()
	genesis.DeputyNodes = types.DeputyNodes{}
	assert.Equal(t, ErrNoDeputyNodes, genesis.Verify())

	genesis = getTestGenesis()
	genesis.DeputyNodes[0].NodeID = genesis.DeputyNodes[0].NodeID[1:]
	assert.Equal(t, ErrInvalidDeputyNodes, genesis.Verify())
}

func TestSetupGenesisBlock(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), "", "")
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
	db := store.NewChainDataBase(GetStorePath(), "", "")
	defer db.Close()

	// no genesis
	genesisBlock := SetupGenesisBlock(db, nil)
	block, err := db.GetBlockByHeight(0) // load first stable block
	assert.NoError(t, err)
	assert.Equal(t, genesisBlock.Hash(), block.Hash())
	assert.Equal(t, DefaultGenesisBlock().Time, block.Time())
	founder, err := db.GetAccount(DefaultGenesisBlock().Founder)
	assert.NoError(t, err)
	assert.Equal(t, true, founder.Balance.Cmp(new(big.Int)) > 0)
}

func TestSetupGenesisBlock_Exist(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), "", "")
	defer db.Close()

	SetupGenesisBlock(db, nil)
	// setup when genesis exist
	assert.PanicsWithValue(t, ErrSaveGenesisFail, func() {
		SetupGenesisBlock(db, nil)
	})
}

func TestGenesis_ToBlock(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath(), "", "")
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
	assert.Equal(t, genesis.DeputyNodes, block.DeputyNodes)
	assert.Len(t, block.ChangeLogs, 1)
	assert.Len(t, block.Txs, 0)
	assert.Equal(t, common.Sha3Nil, block.TxRoot())
	assert.NotEqual(t, 0, block.Height())
	assert.NotEqual(t, common.Sha3Nil, block.DeputyRoot())
	assert.NotEqual(t, common.Sha3Nil, block.VersionRoot())
	assert.NotEqual(t, common.Sha3Nil, block.LogRoot())
}
