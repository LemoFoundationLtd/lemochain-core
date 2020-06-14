package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/txpool"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var (
	testDpovpCfg = Config{
		LogForks:      true,
		RewardManager: common.HexToAddress("0x123"),
		ChainID:       testChainID,
		MineTimeout:   1234,
		MinerExtra:    []byte{0x12},
	}
)

func newTestDPoVP(attendedDeputyCount int) (*DPoVP, deputyTestDatas) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	deputyInfos := generateDeputies(attendedDeputyCount)
	deputies := deputyInfos.ToDeputyNodes()

	am := account.NewManager(common.Hash{}, db)
	stable := initGenesis(db, am, deputies)
	am.Reset(stable.Hash())
	// max deputy count is 5
	dm := deputynode.NewManager(5, db)

	dp := NewDPoVP(testDpovpCfg, db, dm, am, &parentLoader{db}, txpool.NewTxPool(), txpool.NewTxGuard(100))
	return dp, deputyInfos
}

func TestNewDPoVP(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()
	stable := &types.Block{Header: &types.Header{Height: 0, Time: 123}}
	err := db.SetBlock(stable.Hash(), stable)
	assert.NoError(t, err)
	_, err = db.SetStableBlock(stable.Hash())
	assert.NoError(t, err)

	dp := NewDPoVP(testDpovpCfg, db, nil, nil, nil, nil, nil)
	assert.Equal(t, testDpovpCfg.ChainID, dp.processor.ChainID)
	assert.Equal(t, testDpovpCfg.MinerExtra, dp.minerExtra)
	assert.Equal(t, testDpovpCfg.LogForks, dp.logForks)
	assert.Equal(t, testDpovpCfg.MineTimeout, dp.validator.timeoutTime)
}

func TestDPoVP_MineBlock(t *testing.T) {
	dp, deputyInfos := newTestDPoVP(3)
	defer dp.db.Close()

	// mine success
	parentBlock := dp.CurrentBlock()
	miner, err := GetCorrectMiner(parentBlock.Header, time.Now().Unix()*1000, int64(testDpovpCfg.MineTimeout), dp.dm)
	assert.NoError(t, err)
	deputynode.SetSelfNodeKey(deputyInfos.FindByMiner(miner).PrivateKey)
	block, err := dp.MineBlock(3000)
	assert.NoError(t, err)
	assert.Equal(t, parentBlock.Height()+1, block.Height())
	assert.Equal(t, parentBlock.Height()+1, dp.CurrentBlock().Height())
	assert.Equal(t, parentBlock.Height(), dp.StableBlock().Height())

	// mine success with tx
	tx1 := MakeTxFast(deputyInfos[0].PrivateKey)
	tx2 := MakeTxFast(deputyInfos[0].PrivateKey)
	dp.txPool.AddTx(tx1)
	dp.txPool.AddTx(tx2)
	parentBlock = dp.CurrentBlock()
	miner, err = GetCorrectMiner(parentBlock.Header, time.Now().Unix()*1000, int64(testDpovpCfg.MineTimeout), dp.dm)
	assert.NoError(t, err)
	deputynode.SetSelfNodeKey(deputyInfos.FindByMiner(miner).PrivateKey)
	block, err = dp.MineBlock(3000)
	assert.NoError(t, err)
	assert.Equal(t, parentBlock.Height()+1, block.Height())

	// same miner mine again: not in turn
	_, err = dp.MineBlock(3000)
	assert.Equal(t, ErrVerifyHeaderFailed, err)

	// prepareHeader error: I'm not miner
	randomPrivate, _ := crypto.GenerateKey()
	deputynode.SetSelfNodeKey(randomPrivate)
	_, err = dp.MineBlock(3000)
	assert.Equal(t, ErrNotDeputy, err)
}

func TestDPoVP_MineBlock_Stable(t *testing.T) {
	dp, deputyInfos := newTestDPoVP(1)
	defer dp.db.Close()

	// mine success and become stable block
	parentBlock := dp.CurrentBlock()
	miner, err := GetCorrectMiner(parentBlock.Header, time.Now().Unix()*1000, int64(testDpovpCfg.MineTimeout), dp.dm)
	assert.NoError(t, err)
	deputynode.SetSelfNodeKey(deputyInfos.FindByMiner(miner).PrivateKey)
	block, err := dp.MineBlock(3000)
	assert.NoError(t, err)
	assert.Equal(t, parentBlock.Height()+1, block.Height())
	assert.Equal(t, parentBlock.Height()+1, dp.CurrentBlock().Height())
	assert.Equal(t, parentBlock.Height()+1, dp.StableBlock().Height())
}

func newTestBlock(dp *DPoVP, parentHeader *types.Header, deputyInfos deputyTestDatas, txs types.Transactions) *types.Block {
	// find correct miner
	miner, err := GetCorrectMiner(parentHeader, time.Now().Unix()*1000, int64(testDpovpCfg.MineTimeout), dp.dm)
	if err != nil {
		panic(err)
	}

	// mine and seal
	header, err := dp.assembler.PrepareHeader(parentHeader, dp.minerExtra)
	if err != nil {
		panic(err)
	}
	block, _, err := dp.assembler.MineBlock(header, txs, 3000)
	if err != nil {
		panic(err)
	}

	// replace signification
	block.Header.MinerAddress = miner
	hash := block.Hash()
	// avoid to use the cache in block_signer.go
	signData, err := crypto.Sign(hash[:], deputyInfos.FindByMiner(miner).PrivateKey)
	if err != nil {
		panic(err)
	}
	block.Header.SignData = signData

	log.Infof("Create a test block %s by %s", block.ShortString(), block.MinerAddress().String())
	return block
}

func avoidMiner(miner common.Address, deputyInfos deputyTestDatas) {
	// let me be the other miner
	myPrivate := deputyInfos[0].PrivateKey
	if deputyInfos[0].MinerAddress == miner {
		myPrivate = deputyInfos[1].PrivateKey
	}
	deputynode.SetSelfNodeKey(myPrivate)
}

func TestDPoVP_InsertBlock(t *testing.T) {
	dp, deputyInfos := newTestDPoVP(3)
	defer dp.db.Close()

	// insert my block success
	block := newTestBlock(dp, dp.CurrentBlock().Header, deputyInfos, nil)
	// let me be the miner
	deputynode.SetSelfNodeKey(deputyInfos.FindByMiner(block.MinerAddress()).PrivateKey)
	newBlock, err := dp.InsertBlock(block)
	assert.NoError(t, err)
	assert.Equal(t, block.Hash(), newBlock.Hash())
	assert.Equal(t, block.Hash(), dp.CurrentBlock().Hash())
	assert.Equal(t, block.ParentHash(), dp.StableBlock().Hash())

	// insert same block again
	_, err = dp.InsertBlock(block)
	assert.Equal(t, ErrIgnoreBlock, err)

	// insert other's block success
	block = newTestBlock(dp, newBlock.Header, deputyInfos, nil)
	// let me be the other miner
	avoidMiner(block.MinerAddress(), deputyInfos)
	newBlock, err = dp.InsertBlock(block)
	assert.NoError(t, err)
	assert.Equal(t, block.Hash(), newBlock.Hash())
	assert.Equal(t, block.Hash(), dp.CurrentBlock().Hash())
	assert.Equal(t, block.Hash(), dp.StableBlock().Hash())

	// same height and same miner
	tx := MakeTxFast(deputyInfos[0].PrivateKey)
	block = newTestBlock(dp, newBlock.Header, deputyInfos, nil)
	sameHeightBlock := newTestBlock(dp, newBlock.Header, deputyInfos, types.Transactions{tx})
	// let me be the other miner
	avoidMiner(block.MinerAddress(), deputyInfos)
	newBlock, err = dp.InsertBlock(block)
	assert.NoError(t, err)
	assert.Equal(t, block.Hash(), newBlock.Hash())
	_, err = dp.InsertBlock(sameHeightBlock)
	assert.Equal(t, ErrIgnoreBlock, err)

	// verify error
	block = newTestBlock(dp, newBlock.Header, deputyInfos, nil)
	block.Header.SignData = []byte{0x12}
	_, err = dp.InsertBlock(block)
	assert.Equal(t, ErrVerifyBlockFailed, err)
}

// test forks
func TestDPoVP_InsertBlock2(t *testing.T) {
	dp, deputyInfos := newTestDPoVP(5)
	defer dp.db.Close()

	// insert height 1 blocks
	block1 := newTestBlock(dp, dp.CurrentBlock().Header, deputyInfos, nil)
	time.Sleep(time.Duration(testDpovpCfg.MineTimeout) * time.Millisecond)
	block1q := newTestBlock(dp, dp.CurrentBlock().Header, deputyInfos, nil)
	// 0───1 <-current
	newBlock1, err := dp.InsertBlock(block1) // insert my block
	assert.NoError(t, err)
	assert.Equal(t, newBlock1.Hash(), dp.CurrentBlock().Hash())
	// 0─┬─1 <-current
	//   └─1'
	_, err = dp.InsertBlock(block1q) // insert new fork block
	assert.NoError(t, err)
	assert.Equal(t, newBlock1.Hash(), dp.CurrentBlock().Hash())

	// insert height 2 blocks
	block2 := newTestBlock(dp, block1.Header, deputyInfos, nil)
	block2q := newTestBlock(dp, block1q.Header, deputyInfos, nil)
	// 0─┬─1 <-current
	//   └─1'──2'
	_, err = dp.InsertBlock(block2q) // insert new block on other fork
	assert.NoError(t, err)
	assert.Equal(t, block1.Hash(), dp.CurrentBlock().Hash())
	// 0─┬─1───2 <-current
	//   └─1'──2'
	newBlock2, err := dp.InsertBlock(block2) // insert new block on current fork
	assert.NoError(t, err)
	assert.Equal(t, newBlock2.Hash(), dp.CurrentBlock().Hash())
}
