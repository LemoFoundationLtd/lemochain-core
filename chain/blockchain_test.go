package chain

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/network/synchronise/protocol"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/stretchr/testify/assert"
	"math/big"
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
	//store.ClearData()

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
	defer store.ClearData()

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
	defer store.ClearData()

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
	defer store.ClearData()

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

// 1、2、31、32{42、52}，set stable #2
func TestBlockChain_SetStableBlockCurBranch11(t *testing.T) {
	store.ClearData()
	defer store.ClearData()

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

	info.parentHash = block4.Hash()
	info.height = uint32(5)
	block5 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block5, true)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block2.Hash(), 2, false)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block5.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block31.Hash()])
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block5.Hash())
}

// 1、2、31、32{42、52}，set stable #42
func TestBlockChain_SetStableBlockCurBranch12(t *testing.T) {
	store.ClearData()
	defer store.ClearData()

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

	info.parentHash = block4.Hash()
	info.height = uint32(5)
	block5 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block5, true)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block4.Hash(), 4, false)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block5.Hash()])
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block5.Hash())
}

// 1、2、31、32{42、52}，set stable #52
func TestBlockChain_SetStableBlockCurBranch13(t *testing.T) {
	store.ClearData()
	defer store.ClearData()

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

	info.parentHash = block4.Hash()
	info.height = uint32(5)
	block5 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block5, true)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block5.Hash(), 5, false)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block5.Hash()])
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block5.Hash())
}

// 1、2、31{41、51}、32{42、52} set stable #42
func TestBlockChain_SetStableBlockCurBranch21(t *testing.T) {
	store.ClearData()
	defer store.ClearData()

	blockChain, _, err := newBlockChain()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.gasLimit = 10000
	info.time = big.NewInt(1540864589)
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block1, true)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = uint32(2)
	info.gasLimit = 20000
	info.time = big.NewInt(1540864590)
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block2, true)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.gasLimit = 30000
	info.time = big.NewInt(1540864591)
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block31, true)
	assert.NoError(t, err)

	info.parentHash = block31.Hash()
	info.height = uint32(4)
	info.gasLimit = 40000
	info.time = big.NewInt(1540864592)
	block41 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block41, true)
	assert.NoError(t, err)

	info.parentHash = block41.Hash()
	info.height = uint32(5)
	info.gasLimit = 50000
	info.time = big.NewInt(1540864593)
	block51 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block51, true)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.gasLimit = 6000
	info.time = big.NewInt(1540864594)
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block32, true)
	assert.NoError(t, err)

	info.parentHash = block32.Hash()
	info.height = uint32(4)
	info.gasLimit = 70000
	info.time = big.NewInt(1540864595)
	block42 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block42, true)
	assert.NoError(t, err)

	info.parentHash = block42.Hash()
	info.height = uint32(5)
	info.gasLimit = 80000
	info.time = big.NewInt(1540864596)
	block52 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block52, true)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block42.Hash(), 4, false)
	assert.NoError(t, err)

	assert.Equal(t, blockChain.CurrentBlock().Hash(), block52.Hash())

	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block52.Hash()])
}

// 1、2、31{41、51}、32{42、52} set stable #2
func TestBlockChain_SetStableBlockCurBranch22(t *testing.T) {
	store.ClearData()
	defer store.ClearData()

	blockChain, _, err := newBlockChain()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.gasLimit = 10000
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block1, true)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = uint32(2)
	info.gasLimit = 20000
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block2, true)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.gasLimit = 30000
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block31, true)
	assert.NoError(t, err)

	info.parentHash = block31.Hash()
	info.height = uint32(4)
	info.gasLimit = 40000
	block41 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block41, true)
	assert.NoError(t, err)

	info.parentHash = block41.Hash()
	info.height = uint32(5)
	info.gasLimit = 50000
	block51 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block51, true)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.gasLimit = 6000
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block32, true)
	assert.NoError(t, err)

	info.parentHash = block32.Hash()
	info.height = uint32(4)
	info.gasLimit = 70000
	block42 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block42, true)
	assert.NoError(t, err)

	info.parentHash = block42.Hash()
	info.height = uint32(5)
	info.gasLimit = 80000
	block52 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block52, true)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block2.Hash(), 2, false)
	assert.NoError(t, err)

	if block51.Hash().Big().Cmp(block52.Hash().Big()) <= 0 {
		assert.Equal(t, blockChain.CurrentBlock().Hash(), block51.Hash())
	} else {
		assert.Equal(t, blockChain.CurrentBlock().Hash(), block52.Hash())
	}

	assert.Equal(t, 2, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block51.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block52.Hash()])
}

// 1、2、31{41、51}、32{42、52} set stable #52
func TestBlockChain_SetStableBlockCurBranch23(t *testing.T) {
	store.ClearData()
	defer store.ClearData()

	blockChain, _, err := newBlockChain()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.gasLimit = 10000
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block1, true)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = uint32(2)
	info.gasLimit = 20000
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block2, true)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.gasLimit = 30000
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block31, true)
	assert.NoError(t, err)

	info.parentHash = block31.Hash()
	info.height = uint32(4)
	info.gasLimit = 40000
	block41 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block41, true)
	assert.NoError(t, err)

	info.parentHash = block41.Hash()
	info.height = uint32(5)
	info.gasLimit = 50000
	block51 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block51, true)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.gasLimit = 6000
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block32, true)
	assert.NoError(t, err)

	info.parentHash = block32.Hash()
	info.height = uint32(4)
	info.gasLimit = 70000
	block42 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block42, true)
	assert.NoError(t, err)

	info.parentHash = block42.Hash()
	info.height = uint32(5)
	info.gasLimit = 80000
	block52 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block52, true)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block52.Hash(), 5, false)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block52.Hash()])
}

// 1、2、31{41、51}、32{42} set stable #42
func TestBlockChain_SetStableBlockCurBranch31(t *testing.T) {
	store.ClearData()
	defer store.ClearData()

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

	info.parentHash = block31.Hash()
	info.height = uint32(4)
	block41 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block41, true)
	assert.NoError(t, err)

	info.parentHash = block41.Hash()
	info.height = uint32(4)
	block51 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block51, true)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.gasLimit = 1000
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block32, true)
	assert.NoError(t, err)

	info.parentHash = block32.Hash()
	info.height = uint32(4)
	block42 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block42, true)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block42.Hash(), 4, false)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block42.Hash()])
}

// 1、2、31{41、51、61}、32{42、52} set stable #42
func TestBlockChain_SetStableBlockCurBranch32(t *testing.T) {
	store.ClearData()
	defer store.ClearData()

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

	info.parentHash = block31.Hash()
	info.height = uint32(4)
	block41 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block41, true)
	assert.NoError(t, err)

	info.parentHash = block41.Hash()
	info.height = uint32(4)
	block51 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block51, true)
	assert.NoError(t, err)

	info.parentHash = block51.Hash()
	info.height = uint32(6)
	block61 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block61, true)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.gasLimit = 1000
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block32, true)
	assert.NoError(t, err)

	info.parentHash = block32.Hash()
	info.height = uint32(4)
	block42 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block42, true)
	assert.NoError(t, err)

	info.parentHash = block42.Hash()
	info.height = uint32(5)
	block52 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block52, true)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block42.Hash(), 4, false)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block52.Hash()])
}

// 1、2{213、214、215、216}、{3{{314、315、316}、{324、325, 326}}}， set stable #3
func TestBlockChain_SetStableBlockCurBranch41(t *testing.T) {
	store.ClearData()
	defer store.ClearData()

	blockChain, _, err := newBlockChain()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	// Block1
	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.gasLimit = 1000
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block1, true)
	assert.NoError(t, err)

	// Block2
	info.parentHash = block1.Hash()
	info.height = 2
	info.gasLimit = 1001
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block2, true)
	assert.NoError(t, err)

	// Block213
	info.parentHash = block2.Hash()
	info.height = 3
	info.gasLimit = 1002
	block213 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block213, true)
	assert.NoError(t, err)

	// Block24
	info.parentHash = block213.Hash()
	info.height = 4
	info.gasLimit = 1003
	block214 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block214, true)
	assert.NoError(t, err)

	// Block215
	info.parentHash = block214.Hash()
	info.height = 5
	info.gasLimit = 1004
	block215 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block215, true)
	assert.NoError(t, err)

	// Block3
	info.parentHash = block2.Hash()
	info.height = 3
	info.gasLimit = 1005
	block3 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block3, true)
	assert.NoError(t, err)

	// Block314
	info.parentHash = block3.Hash()
	info.height = 4
	info.gasLimit = 1006
	block314 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block314, true)
	assert.NoError(t, err)

	// Block315
	info.parentHash = block314.Hash()
	info.height = 5
	info.gasLimit = 1007
	block315 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block315, true)
	assert.NoError(t, err)

	// Block316
	info.parentHash = block315.Hash()
	info.height = 6
	info.gasLimit = 1008
	block316 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block316, true)
	assert.NoError(t, err)

	// Block324
	info.parentHash = block3.Hash()
	info.height = 4
	info.gasLimit = 1009
	block324 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block324, true)
	assert.NoError(t, err)

	// Block325
	info.parentHash = block324.Hash()
	info.height = 5
	info.gasLimit = 1010
	block325 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block325, true)
	assert.NoError(t, err)

	// Block326
	info.parentHash = block325.Hash()
	info.height = 6
	info.gasLimit = 1011
	block326 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block326, true)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block3.Hash(), 3, false)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block316.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block326.Hash()])
}

// 1、2{213、214、215、216}、{3{{314、315、316}、{324、325, 326}}}， set stable #2
func TestBlockChain_SetStableBlockCurBranch42(t *testing.T) {
	store.ClearData()
	defer store.ClearData()

	blockChain, _, err := newBlockChain()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	// Block1
	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.gasLimit = 1000
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block1, true)
	assert.NoError(t, err)

	// Block2
	info.parentHash = block1.Hash()
	info.height = 2
	info.gasLimit = 1001
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block2, true)
	assert.NoError(t, err)

	// Block213
	info.parentHash = block2.Hash()
	info.height = 3
	info.gasLimit = 1002
	block213 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block213, true)
	assert.NoError(t, err)

	// Block24
	info.parentHash = block213.Hash()
	info.height = 4
	info.gasLimit = 1003
	block214 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block214, true)
	assert.NoError(t, err)

	// Block215
	info.parentHash = block214.Hash()
	info.height = 5
	info.gasLimit = 1004
	block215 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block215, true)
	assert.NoError(t, err)

	// Block3
	info.parentHash = block2.Hash()
	info.height = 3
	info.gasLimit = 1005
	block3 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block3, true)
	assert.NoError(t, err)

	// Block314
	info.parentHash = block3.Hash()
	info.height = 4
	info.gasLimit = 1006
	block314 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block314, true)
	assert.NoError(t, err)

	// Block315
	info.parentHash = block314.Hash()
	info.height = 5
	info.gasLimit = 1007
	block315 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block315, true)
	assert.NoError(t, err)

	// Block316
	info.parentHash = block315.Hash()
	info.height = 6
	info.gasLimit = 1008
	block316 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block316, true)
	assert.NoError(t, err)

	// Block324
	info.parentHash = block3.Hash()
	info.height = 4
	info.gasLimit = 1009
	block324 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block324, true)
	assert.NoError(t, err)

	// Block325
	info.parentHash = block324.Hash()
	info.height = 5
	info.gasLimit = 1010
	block325 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block325, true)
	assert.NoError(t, err)

	// Block326
	info.parentHash = block325.Hash()
	info.height = 6
	info.gasLimit = 1011
	block326 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block326, true)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block2.Hash(), 2, false)
	assert.NoError(t, err)

	assert.Equal(t, 3, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block215.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block316.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block326.Hash()])
}

// 1、2{213、214、215、216}、{3{{314、315、316}、{324、325, 326}}}， set stable #316
func TestBlockChain_SetStableBlockCurBranch43(t *testing.T) {
	store.ClearData()
	defer store.ClearData()

	blockChain, _, err := newBlockChain()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	// Block1
	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.gasLimit = 1000
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block1, true)
	assert.NoError(t, err)

	// Block2
	info.parentHash = block1.Hash()
	info.height = 2
	info.gasLimit = 1001
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block2, true)
	assert.NoError(t, err)

	// Block213
	info.parentHash = block2.Hash()
	info.height = 3
	info.gasLimit = 1002
	block213 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block213, true)
	assert.NoError(t, err)

	// Block24
	info.parentHash = block213.Hash()
	info.height = 4
	info.gasLimit = 1003
	block214 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block214, true)
	assert.NoError(t, err)

	// Block215
	info.parentHash = block214.Hash()
	info.height = 5
	info.gasLimit = 1004
	block215 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block215, true)
	assert.NoError(t, err)

	// Block3
	info.parentHash = block2.Hash()
	info.height = 3
	info.gasLimit = 1005
	block3 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block3, true)
	assert.NoError(t, err)

	// Block314
	info.parentHash = block3.Hash()
	info.height = 4
	info.gasLimit = 1006
	block314 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block314, true)
	assert.NoError(t, err)

	// Block315
	info.parentHash = block314.Hash()
	info.height = 5
	info.gasLimit = 1007
	block315 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block315, true)
	assert.NoError(t, err)

	// Block316
	info.parentHash = block315.Hash()
	info.height = 6
	info.gasLimit = 1008
	block316 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block316, true)
	assert.NoError(t, err)

	// Block324
	info.parentHash = block3.Hash()
	info.height = 4
	info.gasLimit = 1009
	block324 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block324, true)
	assert.NoError(t, err)

	// Block325
	info.parentHash = block324.Hash()
	info.height = 5
	info.gasLimit = 1010
	block325 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block325, true)
	assert.NoError(t, err)

	// Block326
	info.parentHash = block325.Hash()
	info.height = 6
	info.gasLimit = 1011
	block326 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block326, true)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block316.Hash(), 6, false)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block316.Hash()])
}

func buildConfirm(hash common.Hash, privateKey string) (*protocol.BlockConfirmData, error) {
	tmp, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, err
	}

	signData, err := crypto.Sign(hash[:], tmp)
	if err != nil {
		return nil, err
	}

	confirm := &protocol.BlockConfirmData{
		Height: 1,
		Hash:   hash,
	}
	copy(confirm.SignInfo[:], signData[:])

	return confirm, nil
}

func TestBlockChain_ReceiveConfirm(t *testing.T) {
	store.ClearData()
	defer store.ClearData()

	blockChain, _, err := newBlockChain()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	info := blockInfo{
		parentHash: genesis.Hash(),
		height:     1,
		gasLimit:   1000,
		time:       big.NewInt(1540893799),
	}
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block1, true)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = 2
	info.gasLimit = 2000
	info.time = big.NewInt(1540893798)
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block2, true)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = 3
	info.gasLimit = 3000
	info.time = big.NewInt(1540893799)
	block3 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block3, true)
	assert.NoError(t, err)

	// recovery failed
	confirm, err := buildConfirm(block1.Hash(), "c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa")
	assert.NoError(t, err)

	confirm.SignInfo[2] = '1'
	err = blockChain.ReceiveConfirm(confirm)
	assert.Equal(t, ErrInvalidConfirmInfo, err)

	// unavailable confirm info.
	confirm, err = buildConfirm(block1.Hash(), "cbe9fa7c8721b8103e5af1ee5a40ac60c0c2b8c3c762e4e2c6ee0965917b1d86")
	assert.NoError(t, err)
	err = blockChain.ReceiveConfirm(confirm)
	assert.Equal(t, ErrInvalidConfirmInfo, err)

	// has block consensus
	err = blockChain.SetStableBlock(block1.Hash(), 1, false)
	assert.NoError(t, err)

	confirm, err = buildConfirm(block2.Hash(), "c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa")
	assert.NoError(t, err)
	err = blockChain.ReceiveConfirm(confirm)
	assert.NoError(t, err)

	confirm, err = buildConfirm(block2.Hash(), "9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb")
	assert.NoError(t, err)
	err = blockChain.ReceiveConfirm(confirm)
	assert.NoError(t, err)

	confirm, err = buildConfirm(block2.Hash(), "ba9b51e59ec57d66b30b9b868c76d6f4d386ce148d9c6c1520360d92ef0f27ae")
	assert.NoError(t, err)
	err = blockChain.ReceiveConfirm(confirm)
	assert.NoError(t, err)

	confirm, err = buildConfirm(block2.Hash(), "b381bad69ad4b200462a0cc08fcb8ba64d26efd4f49933c2c2448cb23f2cd9d0")
	assert.NoError(t, err)
	err = blockChain.ReceiveConfirm(confirm)
	assert.NoError(t, err)

	confirm, err = buildConfirm(block2.Hash(), "56b5fe1b8c40f0dec29b621a16ffcbc7a1bb5c0b0f910c5529f991273cd0569c")
	assert.NoError(t, err)
	err = blockChain.ReceiveConfirm(confirm)
	assert.NoError(t, err)

	result := blockChain.GetBlockByHash(block2.Hash())
	assert.NotNil(t, result)
}

func TestBlockChain_VerifyBodyNormal(t *testing.T) {
	store.ClearData()
	defer store.ClearData()

	blockChain, _, err := newBlockChain()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	tmp, err := crypto.HexToECDSA("c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa")
	assert.NoError(t, err)

	accounts := store.GetAccounts()
	info := blockInfo{
		parentHash:  genesis.Hash(),
		height:      1,
		gasLimit:    1000,
		deputyNodes: genesis.DeputyNodes,
		txList: []*types.Transaction{
			signTransaction(makeTransaction(tmp, accounts[0].Address, common.Big2, common.Big2, 1538210398, 30000), tmp),
			signTransaction(makeTransaction(tmp, accounts[1].Address, common.Big2, common.Big3, 1538210425, 30000), tmp),
		},
		time: big.NewInt(1540893799),
	}
	block := makeBlock(blockChain.db, info, false)
	err = blockChain.Verify(block)
	assert.NoError(t, err)
}

func TestBlockChain_VerifyBlockBalanceNotEnough(t *testing.T) {
	store.ClearData()
	defer store.ClearData()

	blockChain, _, err := newBlockChain()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	tmp, err := crypto.HexToECDSA("b1960f67176431d708684e243fc2a6474f3924194290c6b10ea4734f2a150894")
	assert.NoError(t, err)

	accounts := store.GetAccounts()
	info := blockInfo{
		parentHash:  genesis.Hash(),
		height:      1,
		gasLimit:    1000,
		deputyNodes: genesis.DeputyNodes,
		txList: []*types.Transaction{
			signTransaction(makeTransaction(tmp, accounts[0].Address, common.Big2, common.Big2, 1538210398, 30000), tmp),
			signTransaction(makeTransaction(tmp, accounts[1].Address, common.Big2, common.Big3, 1538210425, 30000), tmp),
			// makeTransaction(tmp, accounts[0].Address, common.Big2, common.Big2, 1538210398, 30000),
			// makeTransaction(tmp, accounts[1].Address, common.Big2, common.Big3, 1538210425, 30000),
		},
		time: big.NewInt(1540893799),
	}
	block := makeBlock(blockChain.db, info, false)

	err = blockChain.Verify(block)
	assert.NoError(t, err)
}

func TestBlockChain_VerifyBlockBalanceNotSign(t *testing.T) {
	store.ClearData()
	defer store.ClearData()

	blockChain, _, err := newBlockChain()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	tmp, err := crypto.HexToECDSA("b1960f67176431d708684e243fc2a6474f3924194290c6b10ea4734f2a150894")
	assert.NoError(t, err)

	accounts := store.GetAccounts()
	info := blockInfo{
		parentHash:  genesis.Hash(),
		height:      1,
		gasLimit:    1000,
		deputyNodes: genesis.DeputyNodes,
		txList: []*types.Transaction{
			makeTransaction(tmp, accounts[0].Address, common.Big2, common.Big2, 1538210398, 30000),
			makeTransaction(tmp, accounts[1].Address, common.Big2, common.Big3, 1538210425, 30000),
		},
		time: big.NewInt(1540893799),
	}
	block := makeBlock(blockChain.db, info, false)

	err = blockChain.Verify(block)
	assert.NoError(t, err)
}
