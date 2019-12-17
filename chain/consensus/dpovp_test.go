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
	dp.txPool.PushTx(tx1)
	dp.txPool.PushTx(tx2)
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

func TestDPoVP_handleBlockTxs(t *testing.T) {
	dp, deputyInfos := newTestDPoVP(5)
	defer dp.db.Close()

	tx01 := MakeTxFast(deputyInfos[0].PrivateKey)
	tx02 := MakeTxFast(deputyInfos[1].PrivateKey)
	tx03 := MakeTxFast(deputyInfos[2].PrivateKey)
	tx04 := MakeTxFast(deputyInfos[3].PrivateKey)
	// 0. 测试block中没有交易的情况
	block00 := newBlock(dp.CurrentBlock().Hash(), 1, nil, 3600)
	dp.handleBlockTxs(block00)
	assert.Equal(t, 0, len(dp.txPool.PendingTxs.TxsQueue))
	assert.Equal(t, 0, len(dp.txPool.RecentTxs.TraceMap))

	block00 = newBlock(common.HexToHash("0x111"), 1, nil, 3600)
	dp.handleBlockTxs(block00)
	assert.Equal(t, 0, len(dp.txPool.PendingTxs.TxsQueue))
	assert.Equal(t, 0, len(dp.txPool.RecentTxs.TraceMap))

	// 1. 测试block为当前分支的情况
	txs := types.Transactions{tx01, tx02}
	block01 := newBlock(dp.CurrentBlock().Hash(), 1, txs, 3600)
	dp.handleBlockTxs(block01)
	// 验证区块中的交易状态已经被删除
	assert.False(t, dp.txPool.PendingTxs.TxsStatus[tx01.Hash()])
	assert.False(t, dp.txPool.PendingTxs.TxsStatus[tx02.Hash()])

	// 2. 测试block不在当前分支的情况
	txs = types.Transactions{tx03, tx04}
	block02 := newBlock(common.HexToHash("0x111"), 1, txs, 3600)
	dp.handleBlockTxs(block02)
	assert.True(t, dp.txPool.PendingTxs.TxsStatus[tx03.Hash()])
	assert.True(t, dp.txPool.PendingTxs.TxsStatus[tx04.Hash()])
}

func newBlock(parentHash common.Hash, height uint32, txs types.Transactions, timestamp uint32) *types.Block {
	return &types.Block{
		Header: &types.Header{
			ParentHash:   parentHash,
			MinerAddress: common.Address{},
			Height:       height,
			Time:         timestamp,
		},
		Txs: txs,
	}
}
