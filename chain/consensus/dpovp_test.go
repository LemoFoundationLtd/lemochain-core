package consensus

import (
	"fmt"
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

	txGuard := txpool.NewTxGuard(stable.Time())
	dp := NewDPoVP(testDpovpCfg, db, nil, nil, nil, nil, txGuard)
	assert.Equal(t, testDpovpCfg.ChainID, dp.TxProcessor().ChainID)
	assert.Equal(t, testDpovpCfg.MinerExtra, dp.minerExtra)
	assert.Equal(t, testDpovpCfg.LogForks, dp.logForks)
	assert.Equal(t, testDpovpCfg.MineTimeout, dp.validator.mineTimeout)
	assert.Equal(t, txGuard, dp.TxGuard())
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
	assert.Equal(t, testDpovpCfg.MinerExtra, block.Extra())

	// mine success with tx
	tx1 := MakeTxFast(deputyInfos[0].PrivateKey, 100)
	tx2 := MakeTxFast(deputyInfos[0].PrivateKey, 20)
	invalidTx := MakeTx(deputynode.GetSelfNodeKey(), common.HexToAddress("0x88"), common.Lemo2Mo("100000000000000"), uint64(time.Now().Unix()+100))
	dp.txPool.AddTxs(types.Transactions{tx1, tx2, invalidTx})
	assert.Equal(t, 3, len(dp.txPool.GetTxs(uint32(time.Now().Unix()), 100)))
	parentBlock = dp.CurrentBlock()
	miner, err = GetCorrectMiner(parentBlock.Header, time.Now().Unix()*1000, int64(testDpovpCfg.MineTimeout), dp.dm)
	assert.NoError(t, err)
	deputynode.SetSelfNodeKey(deputyInfos.FindByMiner(miner).PrivateKey)
	block, err = dp.MineBlock(3000)
	assert.NoError(t, err)
	assert.Equal(t, parentBlock.Height()+1, block.Height())
	assert.Equal(t, 0, len(dp.txPool.GetTxs(uint32(time.Now().Unix()), 100)))
	assert.Equal(t, true, dp.txGuard.ExistTx(block.Hash(), tx1))
	assert.Equal(t, false, dp.txGuard.ExistTx(block.Hash(), invalidTx))

	// VerifyMiner error: same miner mine again, not in turn
	_, err = dp.MineBlock(3000)
	assert.Equal(t, ErrVerifyHeaderFailed, err)

	// prepareHeader error: I'm not miner
	randomPrivate, _ := crypto.GenerateKey()
	deputynode.SetSelfNodeKey(randomPrivate)
	_, err = dp.MineBlock(3000)
	assert.Equal(t, ErrNotDeputy, err)

	// assembler.MineBlock error: all txs are invalid
	dp.txPool.AddTxs(types.Transactions{invalidTx})
	assert.Equal(t, 1, len(dp.txPool.GetTxs(uint32(time.Now().Unix()), 100)))
	miner, err = GetCorrectMiner(block.Header, time.Now().Unix()*1000, int64(testDpovpCfg.MineTimeout), dp.dm)
	assert.NoError(t, err)
	deputynode.SetSelfNodeKey(deputyInfos.FindByMiner(miner).PrivateKey)
	block, err = dp.MineBlock(3000)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(dp.txPool.GetTxs(uint32(time.Now().Unix()), 100)))
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
	block, invalidTxs, err := dp.assembler.MineBlock(header, txs, 3000)
	if err != nil {
		panic(err)
	}
	if len(invalidTxs) > 0 {
		panic(fmt.Errorf("found %d invalid txs", len(invalidTxs)))
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
	newBlock, err = dp.InsertBlock(block) // became stable
	assert.NoError(t, err)
	assert.Equal(t, block.Hash(), newBlock.Hash())
	assert.Equal(t, block.Hash(), dp.CurrentBlock().Hash())
	assert.Equal(t, block.Hash(), dp.StableBlock().Hash())
	assert.Equal(t, block.Height(), dp.confirmer.lastSig.Height) // confirmed
	assert.Equal(t, 1, len(newBlock.Confirms))

	// same height and same miner
	tx1 := MakeTxFast(deputyInfos[0].PrivateKey, 100)
	tx2 := MakeTxFast(deputyInfos[0].PrivateKey, 20)
	block = newTestBlock(dp, newBlock.Header, deputyInfos, types.Transactions{tx1})
	sameHeightBlock := newTestBlock(dp, newBlock.Header, deputyInfos, types.Transactions{tx2})
	// let me be the other miner
	avoidMiner(block.MinerAddress(), deputyInfos)
	newBlock, err = dp.InsertBlock(block) // became stable
	assert.NoError(t, err)
	assert.Equal(t, block.Hash(), newBlock.Hash())
	_, err = dp.InsertBlock(sameHeightBlock)
	assert.Equal(t, ErrIgnoreBlock, err)
	assert.Equal(t, true, dp.TxGuard().ExistTx(dp.CurrentBlock().Hash(), tx1))
	assert.Equal(t, false, dp.TxGuard().ExistTx(dp.CurrentBlock().Hash(), tx2))

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

	// switch fork
	block3q := newTestBlock(dp, block2q.Header, deputyInfos, nil)
	// 0─┬─1───2 <-current
	//   └─1'──2'──3'
	_, err = dp.InsertBlock(block3q) // insert new block on other fork
	assert.NoError(t, err)
	block4q := newTestBlock(dp, block3q.Header, deputyInfos, nil)
	// 0─┬─1───2
	//   └─1'──2'──3'──4' <-current
	newBlock4q, err := dp.InsertBlock(block4q) // insert new block on other fork
	assert.NoError(t, err)
	assert.Equal(t, newBlock4q.Hash(), dp.CurrentBlock().Hash())
}

func TestDPoVP_saveNewBlock(t *testing.T) {
	dp, deputyInfos := newTestDPoVP(3)
	defer dp.db.Close()

	tx1 := MakeTxFast(deputyInfos[0].PrivateKey, 100)
	tx2 := MakeTxFast(deputyInfos[0].PrivateKey, 20)
	tx3 := MakeTxFast(deputyInfos[0].PrivateKey, 3)
	err := dp.txPool.AddTx(tx1)
	assert.NoError(t, err)
	err = dp.txPool.AddTx(tx2)
	assert.NoError(t, err)
	block1 := newTestBlock(dp, dp.CurrentBlock().Header, deputyInfos, types.Transactions{tx1})
	time.Sleep(time.Duration(testDpovpCfg.MineTimeout) * time.Millisecond)
	block1q := newTestBlock(dp, dp.CurrentBlock().Header, deputyInfos, types.Transactions{tx2, tx3})

	// insert my block success
	// let me be the miner of block1
	deputynode.SetSelfNodeKey(deputyInfos.FindByMiner(block1.MinerAddress()).PrivateKey)
	// 0───1 <-current
	err = dp.saveNewBlock(block1)
	assert.NoError(t, err)
	blockInDb, err := dp.db.GetUnConfirmByHeight(block1.Height(), block1.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block1, blockInDb)
	assert.Equal(t, true, dp.TxGuard().ExistTx(block1.Hash(), tx1))
	assert.Equal(t, false, dp.TxGuard().ExistTx(block1.Hash(), tx2))
	assert.Equal(t, block1.Hash(), dp.confirmer.lastSig.Hash)
	assert.Equal(t, block1.Hash(), dp.CurrentBlock().Hash())
	assert.Equal(t, block1.ParentHash(), dp.StableBlock().Hash())
	txInPool := dp.txPool.GetTxs(block1.Time(), 100)
	assert.Equal(t, 1, len(txInPool))
	assert.Equal(t, tx2, txInPool[0])

	// insert other's block succes
	// let me be the other miner of block1q
	avoidMiner(block1q.MinerAddress(), deputyInfos)
	// 0─┬─1 <-current
	//   └─1'
	err = dp.saveNewBlock(block1q)
	assert.NoError(t, err)
	blockInDb, err = dp.db.GetUnConfirmByHeight(block1q.Height(), block1q.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block1q, blockInDb)
	assert.Equal(t, false, dp.TxGuard().ExistTx(block1.Hash(), tx2))
	assert.Equal(t, false, dp.TxGuard().ExistTx(block1q.Hash(), tx1))
	assert.Equal(t, true, dp.TxGuard().ExistTx(block1q.Hash(), tx2))
	assert.Equal(t, true, dp.TxGuard().ExistTx(block1q.Hash(), tx3))
	assert.Equal(t, block1.Hash(), dp.confirmer.lastSig.Hash)
	assert.Equal(t, block1.Hash(), dp.CurrentBlock().Hash())
	assert.Equal(t, block1q.ParentHash(), dp.StableBlock().Hash())
	assert.Equal(t, 2, len(dp.txPool.GetTxs(block1.Time(), 100))) // tx2,tx3

	// switch fork
	block2q := newTestBlock(dp, block1q.Header, deputyInfos, nil)
	// 0─┬─1 <-current
	//   └─1'──2'
	err = dp.saveNewBlock(block2q)
	assert.NoError(t, err)
	block3q := newTestBlock(dp, block2q.Header, deputyInfos, nil)
	// 0─┬─1
	//   └─1'──2'──3' <-current
	err = dp.saveNewBlock(block3q) // insert new block on other fork
	assert.NoError(t, err)
	txInPool = dp.txPool.GetTxs(block3q.Time(), 100)
	assert.Equal(t, 1, len(txInPool))
	assert.Equal(t, tx1, txInPool[0])

	// stable changed
	block4q := newTestBlock(dp, block3q.Header, deputyInfos, nil)
	avoidMiner(block4q.MinerAddress(), deputyInfos)
	sig, _ := dp.confirmer.TryConfirm(block4q)
	block4q.Confirms = append(block4q.Confirms, sig)
	// 4' <-current
	err = dp.saveNewBlock(block4q)
	assert.NoError(t, err)
	assert.Equal(t, block4q.Hash(), dp.StableBlock().Hash())

	// saveToStore error: block is exist
	err = dp.saveNewBlock(block4q)
	assert.Equal(t, ErrSaveBlock, err)
}

func TestDPoVP_onCurrentChanged(t *testing.T) {
	dp, deputyInfos := newTestDPoVP(3)
	defer dp.db.Close()

	tx1 := MakeTxFast(deputyInfos[0].PrivateKey, 101)
	tx2 := MakeTxFast(deputyInfos[0].PrivateKey, 102)
	tx3 := MakeTxFast(deputyInfos[0].PrivateKey, 103)
	dp.txPool.AddTxs(types.Transactions{tx1, tx2, tx3})
	block1 := newTestBlock(dp, dp.CurrentBlock().Header, deputyInfos, nil)
	err := dp.saveToStore(block1)
	assert.NoError(t, err)
	time.Sleep(time.Duration(testDpovpCfg.MineTimeout) * time.Millisecond)
	block2 := newTestBlock(dp, block1.Header, deputyInfos, types.Transactions{tx2})

	// fork grow
	assert.Equal(t, 3, len(dp.txPool.GetTxs(uint32(time.Now().Unix()), 100)))
	dp.onCurrentChanged(block1, block2)
	assert.Equal(t, 2, len(dp.txPool.GetTxs(uint32(time.Now().Unix()), 100)))

	// fork switch
	block2q := newTestBlock(dp, block1.Header, deputyInfos, types.Transactions{tx3})
	dp.TxGuard().SaveBlock(block1)
	dp.TxGuard().SaveBlock(block2)
	dp.TxGuard().SaveBlock(block2q)
	dp.onCurrentChanged(block2, block2q)
	txsInPool := dp.txPool.GetTxs(uint32(time.Now().Unix()), 100)
	assert.Equal(t, 2, len(txsInPool))
	assert.Equal(t, tx1, txsInPool[0])
	assert.Equal(t, tx2, txsInPool[1])
}

func TestDPoVP_UpdateStable(t *testing.T) {
	dp, deputyInfos := newTestDPoVP(3)
	defer dp.db.Close()

	// not changed. confirms are not enough
	block1 := newTestBlock(dp, dp.CurrentBlock().Header, deputyInfos, nil)
	err := dp.saveToStore(block1)
	assert.NoError(t, err)
	changed, err := dp.UpdateStable(block1)
	assert.NoError(t, err)
	assert.Equal(t, false, changed)

	// UpdateStable error: block is not in db
	block2 := newTestBlock(dp, block1.Header, deputyInfos, nil)
	block2.Confirms = append(block2.Confirms, types.SignData{0x12})
	avoidMiner(block2.MinerAddress(), deputyInfos)
	_, err = dp.UpdateStable(block2)
	assert.Equal(t, ErrSetStableBlockToDB, err)

	// changed
	err = dp.saveToStore(block2)
	assert.NoError(t, err)
	changed, err = dp.UpdateStable(block2)
	assert.NoError(t, err)
	assert.Equal(t, true, changed)

	// TODO move params.TermDuration to be not global variable
	// can't test saveSnapshot. Because the db requires insert block by height order. And if we change the snapshot height, the other case may be failed. So we can't reach the snapshot height in test case
}

func TestDPoVP_VerifyAndSeal(t *testing.T) {
	// TODO
}

func TestDPoVP_InsertConfirms(t *testing.T) {
	// TODO InsertConfirms insertConfirms
}

func TestDPoVP_LoadTopCandidates(t *testing.T) {
	// TODO
}

func TestDPoVP_LoadRefundCandidates(t *testing.T) {
	// TODO
}
