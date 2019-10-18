package chain

import (
	"crypto/ecdsa"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/consensus"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/transaction"
	"github.com/LemoFoundationLtd/lemochain-core/chain/txpool"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/flag"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/subscribe"
	"github.com/LemoFoundationLtd/lemochain-core/network"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"os"
	"reflect"
	"testing"
	"time"
)

var (
	ErrChannelTimeout        = errors.New("channel is timeout")
	blockType                = reflect.TypeOf(&types.Block{})
	confirmType              = reflect.TypeOf(&network.BlockConfirmData{})
	testChainID       uint16 = 99
	mineTimeout              = 10000
	nodeKeys                 = []*ecdsa.PrivateKey{
		parseKey("c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"),
		parseKey("9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb"),
		parseKey("ba9b51e59ec57d66b30b9b868c76d6f4d386ce148d9c6c1520360d92ef0f27ae"),
	}
)

func GetStorePath() string {
	return "../testdata/blockchain"
}

func ClearData() {
	_ = os.RemoveAll(GetStorePath())
}

func newTestBlockChain(attendedDeputyCount int) *BlockChain {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())

	// init nodeKey
	priv := parseKey("c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa")
	deputynode.SetSelfNodeKey(priv)

	// init genesis block
	genesis := DefaultGenesisConfig()
	genesis.DeputyNodesInfo = genesis.DeputyNodesInfo[:attendedDeputyCount]
	SetupGenesisBlock(db, genesis)

	// max deputy count is 5
	dm := deputynode.NewManager(5, db)
	blockChain, err := NewBlockChain(Config{testChainID, 10000}, dm, db, flag.CmdFlags{}, txpool.NewTxPool())
	if err != nil {
		panic(err)
	}
	return blockChain
}

type resultWithErr struct {
	data interface{}
	err  error
}

func parseKey(key string) *ecdsa.PrivateKey {
	private, err := crypto.HexToECDSA(key)
	if err != nil {
		panic(err)
	}
	return private
}

func getTestNodeID(deputyIndex int) []byte {
	return crypto.PrivateKeyToNodeID(nodeKeys[deputyIndex])
}

func createTestTx() *types.Transaction {
	testPrivate, _ := crypto.HexToECDSA("432a86ab8765d82415a803e29864dcfc1ed93dac949abf6f95a583179f27e4bb")
	testAddr := crypto.PubkeyToAddress(testPrivate.PublicKey) // 0x0107134b9cdd7d89f83efa6175f9b3552f29094c
	testSigner := types.DefaultSigner{}
	tx := types.NewTransaction(testAddr, common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 0, testChainID, 1544584596, "aa", "aaa")
	tx, err := testSigner.SignTx(tx, testPrivate)
	if err != nil {
		panic(err)
	}
	return tx
}

func newTestBlock(bc *BlockChain) *types.Block {
	processor := transaction.NewTxProcessor(bc.Founder(), testChainID, bc, bc.am, bc.db, bc.dm)
	assembler := consensus.NewBlockAssembler(bc.am, bc.dm, processor, bc.engine)
	parent := bc.CurrentBlock()
	header, err := assembler.PrepareHeader(parent.Header, nil)
	if err != nil {
		panic(err)
	}
	// testTx := createTestTx()
	block, _, err := assembler.MineBlock(header, nil, 1000)
	if err != nil {
		panic(err)
	}

	// maybe it is not in our turn to mine now, so try to find the correct miner
	for i := range nodeKeys {
		block.Header.MinerAddress = bc.dm.GetDeputyByNodeID(block.Height(), getTestNodeID(i)).MinerAddress
		if err := consensus.VerifyMineSlot(block.Header, parent.Header, uint64(mineTimeout), bc.dm); err == nil {
			hash := block.Hash()
			block.Header.SignData, err = crypto.Sign(hash[:], nodeKeys[i])
			log.Info("sign a new block", "deputyIndex", i, "height", block.Height())
			return block
		}
	}
	panic("can't find a proper miner")
}

func TestNewBlockChain(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()

	// no genesis
	dm := deputynode.NewManager(5, db)
	_, err := NewBlockChain(Config{testChainID, 10000}, dm, db, flag.CmdFlags{}, txpool.NewTxPool())
	assert.Equal(t, ErrNoGenesis, err)

	// success
	genesisBlock := SetupGenesisBlock(db, nil)
	blockChain, err := NewBlockChain(Config{testChainID, 10000}, dm, db, flag.CmdFlags{}, txpool.NewTxPool())
	assert.NoError(t, err)
	assert.Equal(t, genesisBlock, blockChain.engine.StableBlock())
	assert.Equal(t, genesisBlock, blockChain.engine.CurrentBlock())
	assert.Equal(t, genesisBlock.Hash(), blockChain.Genesis().Hash())
	assert.Equal(t, genesisBlock.MinerAddress(), blockChain.Founder())
	assert.Equal(t, testChainID, blockChain.ChainID())

	blockChain.Stop()
}

// toTimeoutChannel wait to read from a channel, or time out
func toTimeoutChannel(ch interface{}, timeout time.Duration) <-chan interface{} {
	notify := make(chan interface{})
	go func() {
		queueTimer := time.NewTimer(timeout)
		caseList := []reflect.SelectCase{
			// case data := <-ch:
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)},
			// case <-queueTimer.C:
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(queueTimer.C)},
		}
		switch caseIndex, recv, _ := reflect.Select(caseList); caseIndex {
		case 0:
			notify <- recv.Interface()
		case 1:
			notify <- ErrChannelTimeout
		default:
			notify <- errors.New("select channel error")
		}
	}()
	return notify
}

func subscribeEvent(subscribeName string, chType reflect.Type) <-chan resultWithErr {
	ch := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, chType), 0).Interface()
	subscribe.Sub(subscribeName, ch)
	testFinish := make(chan resultWithErr)
	go func() {
		result := resultWithErr{
			data: <-toTimeoutChannel(ch, 1000*time.Millisecond),
		}
		result.err, _ = result.data.(error)
		subscribe.UnSub(subscribeName, ch)
		testFinish <- result
	}()
	return testFinish
}

func TestBlockChain_MineBlock(t *testing.T) {
	// Only one deputy node so that we can test stable event
	bc := newTestBlockChain(1)
	defer bc.db.Close()
	defer bc.Stop()

	// mine and become stable
	current := bc.CurrentBlock()
	mineEvent := subscribeEvent(subscribe.NewMinedBlock, blockType)
	currentEvent := subscribeEvent(subscribe.NewCurrentBlock, blockType)
	stableEvent := subscribeEvent(subscribe.NewStableBlock, blockType)
	confirmEvent := subscribeEvent(subscribe.NewConfirm, confirmType)
	bc.MineBlock(1000)

	assertBlockChannelByParent(t, mineEvent, current)
	assertBlockChannelByParent(t, currentEvent, current)
	assertBlockChannelByParent(t, stableEvent, current)
	assert.Equal(t, ErrChannelTimeout, (<-confirmEvent).err)
}

func assertBlockChannelByParent(t *testing.T, ch <-chan resultWithErr, parent *types.Block) {
	result := <-ch
	assert.NoError(t, result.err)
	assert.Equal(t, parent.Hash(), result.data.(*types.Block).ParentHash())
	assert.Equal(t, parent.Height()+1, result.data.(*types.Block).Height())
}

func assertBlockChannel(t *testing.T, ch <-chan resultWithErr, block *types.Block) {
	result := <-ch
	assert.NoError(t, result.err)
	assert.Equal(t, block.Hash(), result.data.(*types.Block).Hash())
	assert.Equal(t, block.Height(), result.data.(*types.Block).Height())
}

func assertConfirmChannel(t *testing.T, ch <-chan resultWithErr, block *types.Block) {
	result := <-ch
	assert.NoError(t, result.err)
	confirm := result.data.(*network.BlockConfirmData)
	assert.Equal(t, block.Hash(), confirm.Hash)
	assert.Equal(t, block.Height(), confirm.Height)
	confirmNodeID, err := confirm.SignInfo.RecoverNodeID(confirm.Hash)
	assert.NoError(t, err)
	assert.Equal(t, deputynode.GetSelfNodeID(), confirmNodeID)
}

func TestBlockChain_InsertBlock(t *testing.T) {
	// Only 2 deputy nodes so that we can test stable event and confirm event
	bc := newTestBlockChain(2)
	defer bc.db.Close()
	defer bc.Stop()

	// insert and become stable
	newBlock := newTestBlock(bc)
	newBlockFork := newTestBlock(bc)
	mineEvent := subscribeEvent(subscribe.NewMinedBlock, blockType)
	currentEvent := subscribeEvent(subscribe.NewCurrentBlock, blockType)
	stableEvent := subscribeEvent(subscribe.NewStableBlock, blockType)
	confirmEvent := subscribeEvent(subscribe.NewConfirm, confirmType)
	go func() {
		err := bc.InsertBlock(newBlock)
		assert.NoError(t, err)
	}()

	assert.Equal(t, ErrChannelTimeout, (<-mineEvent).err)
	assertBlockChannel(t, currentEvent, newBlock)
	if consensus.IsMinedByself(newBlock) {
		// if it is mined by self, then it is unstable
		assert.Equal(t, ErrChannelTimeout, (<-stableEvent).err)
		assert.Equal(t, ErrChannelTimeout, (<-confirmEvent).err)
	} else {
		assertBlockChannel(t, stableEvent, newBlock)
		assertConfirmChannel(t, confirmEvent, newBlock)
	}

	// insert to another fork
	mineEvent = subscribeEvent(subscribe.NewMinedBlock, blockType)
	currentEvent = subscribeEvent(subscribe.NewCurrentBlock, blockType)
	stableEvent = subscribeEvent(subscribe.NewStableBlock, blockType)
	confirmEvent = subscribeEvent(subscribe.NewConfirm, confirmType)
	go func() {
		err := bc.InsertBlock(newBlockFork)
		assert.Equal(t, consensus.ErrExistBlock, err)
	}()

	assert.Equal(t, ErrChannelTimeout, (<-mineEvent).err)
	assert.Equal(t, ErrChannelTimeout, (<-currentEvent).err)
	assert.Equal(t, ErrChannelTimeout, (<-stableEvent).err)
	assert.Equal(t, ErrChannelTimeout, (<-confirmEvent).err)

	// mined by next miner and insert synchronously
	newBlock = newTestBlock(bc)
	mineEvent = subscribeEvent(subscribe.NewMinedBlock, blockType)
	currentEvent = subscribeEvent(subscribe.NewCurrentBlock, blockType)
	stableEvent = subscribeEvent(subscribe.NewStableBlock, blockType)
	confirmEvent = subscribeEvent(subscribe.NewConfirm, confirmType)
	err := bc.InsertBlock(newBlock)
	assert.NoError(t, err)

	assert.Equal(t, ErrChannelTimeout, (<-mineEvent).err)
	assertBlockChannel(t, currentEvent, newBlock)
	if consensus.IsMinedByself(newBlock) {
		// if it is mined by self, then it is unstable
		assert.Equal(t, ErrChannelTimeout, (<-stableEvent).err)
		assert.Equal(t, ErrChannelTimeout, (<-confirmEvent).err)
	} else {
		assertBlockChannel(t, stableEvent, newBlock)
		assertConfirmChannel(t, confirmEvent, newBlock)
	}
}

func TestBlockChain_Stop(t *testing.T) {
	// Only one deputy node so that we can test stable event
	bc := newTestBlockChain(1)
	defer bc.db.Close()
	bc.Stop()

	// mine and become stable
	mineEvent := subscribeEvent(subscribe.NewMinedBlock, blockType)
	currentEvent := subscribeEvent(subscribe.NewCurrentBlock, blockType)
	stableEvent := subscribeEvent(subscribe.NewStableBlock, blockType)
	confirmEvent := subscribeEvent(subscribe.NewConfirm, confirmType)
	bc.MineBlock(1000)

	assert.Equal(t, ErrChannelTimeout, (<-mineEvent).err)
	assert.Equal(t, ErrChannelTimeout, (<-currentEvent).err)
	assert.Equal(t, ErrChannelTimeout, (<-stableEvent).err)
	assert.Equal(t, ErrChannelTimeout, (<-confirmEvent).err)
}

func TestBlockChain_IsInBlackList(t *testing.T) {

}

func TestBlockChain_Founder(t *testing.T) {

}

func TestBlockChain_GetBlockByHeight(t *testing.T) {

}

func TestBlockChain_GetParentByHeight(t *testing.T) {

}
