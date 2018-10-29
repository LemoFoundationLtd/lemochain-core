package chain

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/stretchr/testify/assert"
	"testing"
)

type EngineTest struct{}

func (engine *EngineTest) VerifyHeader(block *types.Block) error { return nil }

func (engine *EngineTest) Seal(header *types.Header, txs []*types.Transaction, changeLog []*types.ChangeLog, events []*types.Event) (*types.Block, error) {
	return nil, nil
}

func (engine *EngineTest) Finalize(header *types.Header, am *account.Manager) {}

func broadcastStableBlock(block *types.Block) {}

func broadcastConfirmInfo(hash common.Hash, height uint32) {}

func newBlockChain() (*BlockChain, chan *types.Block, error) {
	store.ClearData()

	chainId := uint16(99)
	db, err := store.NewCacheChain(store.GetStorePath())
	if err != nil {
		return nil, nil, err
	}

	genesis := DefaultGenesisBlock()
	_, err = SetupGenesisBlock(db, genesis)
	if err != nil {
		return nil, nil, err
	}

	var engine EngineTest
	ch := make(chan *types.Block)
	blockChain, err := NewBlockChain(chainId, &engine, db, ch, nil)
	if err != nil {
		return nil, nil, err
	}

	blockChain.BroadcastStableBlock = broadcastStableBlock
	blockChain.BroadcastConfirmInfo = broadcastConfirmInfo

	deputynode.Instance().Add(0, genesis.DeputyNodes)
	return blockChain, ch, nil
}

func TestBlockChain_Reorg8ABC(t *testing.T) {
	store.ClearData()

	var info blockInfo

	blockChain, _, err := newBlockChain()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), genesis.Hash())

	info.parentHash = genesis.Hash()
	info.height = 1
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block1, true)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block1.Hash())
	assert.Nil(t, blockChain.chainForksHead[genesis.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block1.Hash()])

	info.parentHash = block1.Hash()
	info.height = 2
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block2, true)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block2.Hash())
	assert.Nil(t, blockChain.chainForksHead[block1.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block2.Hash()])

	hash := block2.Hash()

	info.parentHash = hash
	info.height = 3
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block31, true)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block31.Hash())
	assert.Nil(t, blockChain.chainForksHead[block2.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block31.Hash()])

	info.parentHash = hash
	info.height = 3
	info.gasLimit = 1000
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block32, true)
	assert.NoError(t, err)
	assert.NotNil(t, blockChain.chainForksHead[block31.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block32.Hash()])
	if block31.Hash().Big().Cmp(block32.Hash().Big()) <= 0 {
		assert.Equal(t, blockChain.CurrentBlock().Hash(), block31.Hash())
	} else {
		assert.Equal(t, blockChain.CurrentBlock().Hash(), block32.Hash())
	}

	info.parentHash = block32.Hash()
	info.height = 4
	block41 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block41, true)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block41.Hash())
	assert.Nil(t, blockChain.chainForksHead[block32.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block41.Hash()])

	info.parentHash = block41.Hash()
	info.height = 5
	block5 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block5, true)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block5.Hash())
	assert.Nil(t, blockChain.chainForksHead[block41.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block5.Hash()])

	info.parentHash = block31.Hash()
	info.height = 4
	block42 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block42, true)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block5.Hash())
	assert.Nil(t, blockChain.chainForksHead[block31.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block42.Hash()])
}

func TestBlockChain_Reorg8Len(t *testing.T) {
	store.ClearData()

	var info blockInfo

	blockChain, _, err := newBlockChain()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	info.parentHash = genesis.Hash()
	info.height = 1
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block1, true)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = 2
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block2, true)
	assert.NoError(t, err)

	hash := block2.Hash()

	info.parentHash = hash
	info.height = 3
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block31, true)
	assert.NoError(t, err)

	info.parentHash = hash
	info.height = 3
	info.gasLimit = 1000
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block32, true)
	assert.NoError(t, err)

	info.parentHash = block31.Hash()
	info.height = 4
	block41 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block41, true)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block41.Hash())

	info.parentHash = block41.Hash()
	info.height = 5
	block51 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block51, true)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block51.Hash())

	info.parentHash = block32.Hash()
	info.height = 4
	block42 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block42, true)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block51.Hash())

	info.parentHash = block42.Hash()
	info.height = 5
	block52 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block52, true)
	assert.NoError(t, err)
	if block51.Hash().Big().Cmp(block52.Hash().Big()) <= 0 {
		assert.Equal(t, blockChain.CurrentBlock().Hash(), block51.Hash())
	} else {
		assert.Equal(t, blockChain.CurrentBlock().Hash(), block52.Hash())
	}
}

func TestBlockChain_GetBlockByHeight(t *testing.T) {
	store.ClearData()

	blockChain, _, err := newBlockChain()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	for index := 1; index < 16; index++ {
		block := makeBlock(blockChain.db, info, false)
		err = blockChain.InsertChain(block, true)
		assert.NoError(t, err)

		info.height = uint32(index) + 1
		info.parentHash = block.Hash()

		result := blockChain.GetBlockByHeight(uint32(index))
		assert.Equal(t, block.Hash(), result.Hash())
	}

	result := blockChain.GetBlockByHeight(1)
	assert.Equal(t, genesis.Hash(), result.ParentHash())
}

func TestBlockChain_SetStableBlock(t *testing.T) {

	store.ClearData()

	blockChain, _, err := newBlockChain()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block1, true)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = uint32(2)
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block2, true)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block31, true)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.gasLimit = 1000
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block32, true)
	assert.NoError(t, err)

	info.parentHash = block32.Hash()
	info.height = uint32(4)
	block4 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block4, true)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(5)
	block5 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block5, true)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block4.Hash(), 4, false)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block5.Hash()])
}
