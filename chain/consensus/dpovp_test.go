package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/consensus"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/merkle"
	"github.com/LemoFoundationLtd/lemochain-core/network"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

func init() {
	prv, _ := crypto.ToECDSA(common.FromHex("0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"))
	deputynode.SetSelfNodeKey(prv)

}

func TestBlockChain_Reorg8ABC(t *testing.T) {
	ClearData()

	var info blockInfo
	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), genesis.Hash())

	info.parentHash = genesis.Hash()
	info.height = 1
	info.time = uint32(time.Now().Unix() - 22)
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block1)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block1.Hash())
	assert.Nil(t, blockChain.chainForksHead[genesis.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block1.Hash()])

	info.parentHash = block1.Hash()
	info.height = 2
	info.time = uint32(time.Now().Unix() - 19)
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block2)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block2.Hash())
	assert.Nil(t, blockChain.chainForksHead[block1.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block2.Hash()])

	hash := block2.Hash()

	info.parentHash = hash
	info.height = 3
	info.time = uint32(time.Now().Unix() - 16)
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block31)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block31.Hash())
	assert.Nil(t, blockChain.chainForksHead[block2.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block31.Hash()])

	info.parentHash = hash
	info.height = 3
	info.time = uint32(time.Now().Unix() - 9)
	info.gasLimit = 1000
	block32 := makeBlock(blockChain.db, info, false)

	err = blockChain.InsertBlock(block32)
	assert.NoError(t, err)
	assert.NotNil(t, blockChain.chainForksHead[block31.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block32.Hash()])
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block31.Hash())

	info.parentHash = block32.Hash()
	info.height = 4
	info.time = uint32(time.Now().Unix() - 6)
	block41 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block41)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block31.Hash())
	assert.Nil(t, blockChain.chainForksHead[block32.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block41.Hash()])

	info.parentHash = block41.Hash()
	info.height = 5
	info.time = uint32(time.Now().Unix() - 3)
	block5 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block5)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block31.Hash())
	assert.Nil(t, blockChain.chainForksHead[block41.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block5.Hash()])

	info.parentHash = block31.Hash()
	info.height = 4
	info.time = uint32(time.Now().Unix() - 13)
	block42 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block42)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block42.Hash())
	assert.Nil(t, blockChain.chainForksHead[block31.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block42.Hash()])
}

func TestBlockChain_Reorg8Len(t *testing.T) {
	ClearData()

	var info blockInfo

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()

	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	info.parentHash = genesis.Hash()
	info.height = 1
	info.time = uint32(time.Now().Unix() - 22)
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block1)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = 2
	info.time = uint32(time.Now().Unix() - 19)
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block2)
	assert.NoError(t, err)

	hash := block2.Hash()

	info.parentHash = hash
	info.height = 3
	info.time = uint32(time.Now().Unix() - 16)
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block31)
	assert.NoError(t, err)

	info.parentHash = hash
	info.height = 3
	info.time = uint32(time.Now().Unix() - 9)
	info.gasLimit = 1000
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block32)
	assert.NoError(t, err)

	info.parentHash = block31.Hash()
	info.height = 4
	info.time = uint32(time.Now().Unix() - 13)
	block41 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block41)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block41.Hash())

	info.parentHash = block41.Hash()
	info.height = 5
	info.time = uint32(time.Now().Unix() - 10)
	block51 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block51)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block51.Hash())

	info.parentHash = block32.Hash()
	info.height = 4
	info.time = uint32(time.Now().Unix() - 6)
	block42 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block42)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block51.Hash())

	info.parentHash = block42.Hash()
	info.height = 5
	info.time = uint32(time.Now().Unix() - 3)
	block52 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block52)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block51.Hash())
}

func TestBlockChain_GetBlockByHeight(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	for index := 1; index < 16; index++ {
		block := makeBlock(blockChain.db, info, false)
		err = blockChain.InsertBlock(block)
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
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.time = uint32(time.Now().Unix() - 22)
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block1)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = uint32(2)
	info.time = uint32(time.Now().Unix() - 19)
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block2)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.time = uint32(time.Now().Unix() - 9)
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block31)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.time = uint32(time.Now().Unix() - 16)
	info.gasLimit = 1000
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block32)
	assert.NoError(t, err)
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block32.Hash())

	info.parentHash = block32.Hash()
	info.height = uint32(4)
	info.time = uint32(time.Now().Unix() - 13)
	block4 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block4)
	assert.NoError(t, err)

	info.parentHash = block4.Hash()
	info.height = uint32(5)
	info.time = uint32(time.Now().Unix() - 10)
	block5 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block5)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block2.Hash(), 2)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block5.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block31.Hash()])
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block5.Hash())
}

// 1、2、31、32{42、52}，set stable #42
func TestBlockChain_SetStableBlockCurBranch12(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.time = uint32(time.Now().Unix() - 22)
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block1)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = uint32(2)
	info.time = uint32(time.Now().Unix() - 19)
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block2)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.time = uint32(time.Now().Unix() - 16)
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block31)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.gasLimit = 1000
	info.time = uint32(time.Now().Unix() - 9)
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block32)
	assert.NoError(t, err)

	info.parentHash = block32.Hash()
	info.height = uint32(4)
	info.time = uint32(time.Now().Unix() - 6)
	block4 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block4)
	assert.NoError(t, err)

	info.parentHash = block4.Hash()
	info.height = uint32(5)
	info.time = uint32(time.Now().Unix() - 3)
	block5 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block5)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block4.Hash(), 4)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block5.Hash()])
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block5.Hash())
}

// 1、2、31、32{42、52}，set stable #52
func TestBlockChain_SetStableBlockCurBranch13(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.time = uint32(time.Now().Unix() - 22)
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block1)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = uint32(2)
	info.time = uint32(time.Now().Unix() - 19)
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block2)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.time = uint32(time.Now().Unix() - 16)
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block31)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.time = uint32(time.Now().Unix() - 9)
	info.gasLimit = 1000
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block32)
	assert.NoError(t, err)

	info.parentHash = block32.Hash()
	info.height = uint32(4)
	info.time = uint32(time.Now().Unix() - 6)
	block4 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block4)
	assert.NoError(t, err)

	info.parentHash = block4.Hash()
	info.height = uint32(5)
	info.time = uint32(time.Now().Unix() - 3)
	block5 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block5)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block5.Hash(), 5)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block5.Hash()])
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block5.Hash())
}

// 1、2、31{41、51}、32{42、52} set stable #42
func TestBlockChain_SetStableBlockCurBranch21(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.time = uint32(time.Now().Unix() - 22)
	info.gasLimit = 10000
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block1)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = uint32(2)
	info.time = uint32(time.Now().Unix() - 19)
	info.gasLimit = 20000
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block2)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.time = uint32(time.Now().Unix() - 16)
	info.gasLimit = 30000
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block31)
	assert.NoError(t, err)

	info.parentHash = block31.Hash()
	info.height = uint32(4)
	info.time = uint32(time.Now().Unix() - 13)
	info.gasLimit = 40000
	block41 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block41)
	assert.NoError(t, err)

	info.parentHash = block41.Hash()
	info.height = uint32(5)
	info.time = uint32(time.Now().Unix() - 10)
	info.gasLimit = 50000
	block51 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block51)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.gasLimit = 6000
	info.time = uint32(time.Now().Unix() - 9)
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block32)
	assert.NoError(t, err)

	info.parentHash = block32.Hash()
	info.height = uint32(4)
	info.gasLimit = 70000
	info.time = uint32(time.Now().Unix() - 6)
	block42 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block42)
	assert.NoError(t, err)

	info.parentHash = block42.Hash()
	info.height = uint32(5)
	info.gasLimit = 80000
	info.time = uint32(time.Now().Unix() - 3)
	block52 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block52)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block42.Hash(), 4)
	assert.NoError(t, err)

	assert.Equal(t, blockChain.CurrentBlock().Hash(), block52.Hash())

	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block52.Hash()])
}

// 1、2、31{41、51}、32{42、52} set stable #2
func TestBlockChain_SetStableBlockCurBranch22(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.time = uint32(time.Now().Unix() - 22)
	info.gasLimit = 10000
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block1)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = uint32(2)
	info.time = uint32(time.Now().Unix() - 19)
	info.gasLimit = 20000
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block2)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.time = uint32(time.Now().Unix() - 16)
	info.gasLimit = 30000
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block31)
	assert.NoError(t, err)

	info.parentHash = block31.Hash()
	info.height = uint32(4)
	info.time = uint32(time.Now().Unix() - 13)
	info.gasLimit = 40000
	block41 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block41)
	assert.NoError(t, err)

	info.parentHash = block41.Hash()
	info.height = uint32(5)
	info.time = uint32(time.Now().Unix() - 10)
	info.gasLimit = 50000
	block51 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block51)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.time = uint32(time.Now().Unix() - 9)
	info.gasLimit = 6000
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block32)
	assert.NoError(t, err)

	info.parentHash = block32.Hash()
	info.height = uint32(4)
	info.time = uint32(time.Now().Unix() - 6)
	info.gasLimit = 70000
	block42 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block42)
	assert.NoError(t, err)

	info.parentHash = block42.Hash()
	info.height = uint32(5)
	info.time = uint32(time.Now().Unix() - 3)
	info.gasLimit = 80000
	block52 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block52)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block2.Hash(), 2)
	assert.NoError(t, err)

	assert.Equal(t, blockChain.CurrentBlock().Hash(), block51.Hash())

	assert.Equal(t, 2, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block51.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block52.Hash()])
}

// 1、2、31{41、51}、32{42、52} set stable #52
func TestBlockChain_SetStableBlockCurBranch23(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.time = uint32(time.Now().Unix() - 22)
	info.gasLimit = 10000
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block1)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = uint32(2)
	info.time = uint32(time.Now().Unix() - 19)
	info.gasLimit = 20000
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block2)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.time = uint32(time.Now().Unix() - 16)
	info.gasLimit = 30000
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block31)
	assert.NoError(t, err)

	info.parentHash = block31.Hash()
	info.height = uint32(4)
	info.time = uint32(time.Now().Unix() - 13)
	info.gasLimit = 40000
	block41 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block41)
	assert.NoError(t, err)

	info.parentHash = block41.Hash()
	info.height = uint32(5)
	info.time = uint32(time.Now().Unix() - 10)
	info.gasLimit = 50000
	block51 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block51)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.time = uint32(time.Now().Unix() - 9)
	info.gasLimit = 6000
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block32)
	assert.NoError(t, err)

	info.parentHash = block32.Hash()
	info.height = uint32(4)
	info.time = uint32(time.Now().Unix() - 6)
	info.gasLimit = 70000
	block42 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block42)
	assert.NoError(t, err)

	info.parentHash = block42.Hash()
	info.height = uint32(5)
	info.time = uint32(time.Now().Unix() - 3)
	info.gasLimit = 80000
	block52 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block52)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block52.Hash(), 5)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block52.Hash()])
}

// 1、2、31{41、51}、32{42} set stable #42
func TestBlockChain_SetStableBlockCurBranch31(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.time = uint32(time.Now().Unix() - 22)
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block1)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = uint32(2)
	info.time = uint32(time.Now().Unix() - 19)
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block2)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.time = uint32(time.Now().Unix() - 16)
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block31)
	assert.NoError(t, err)

	info.parentHash = block31.Hash()
	info.height = uint32(4)
	info.time = uint32(time.Now().Unix() - 13)
	block41 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block41)
	assert.NoError(t, err)

	info.parentHash = block41.Hash()
	info.height = uint32(5)
	info.time = uint32(time.Now().Unix() - 10)
	block51 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block51)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.time = uint32(time.Now().Unix() - 9)
	info.gasLimit = 1000
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block32)
	assert.NoError(t, err)

	info.parentHash = block32.Hash()
	info.height = uint32(4)
	info.time = uint32(time.Now().Unix() - 6)
	block42 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block42)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block42.Hash(), 4)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block42.Hash()])
}

// 1、2、31{41、51、61}、32{42、52} set stable #42
func TestBlockChain_SetStableBlockCurBranch32(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.time = uint32(time.Now().Unix() - 22)
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block1)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = uint32(2)
	info.time = uint32(time.Now().Unix() - 19)
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block2)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.time = uint32(time.Now().Unix() - 16)
	block31 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block31)
	assert.NoError(t, err)

	info.parentHash = block31.Hash()
	info.height = uint32(4)
	info.time = uint32(time.Now().Unix() - 13)
	block41 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block41)
	assert.NoError(t, err)

	info.parentHash = block41.Hash()
	info.height = uint32(5)
	info.time = uint32(time.Now().Unix() - 10)
	block51 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block51)
	assert.NoError(t, err)

	info.parentHash = block51.Hash()
	info.height = uint32(6)
	info.time = uint32(time.Now().Unix() - 7)
	block61 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block61)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.time = uint32(time.Now().Unix() - 9)
	info.gasLimit = 1000
	block32 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block32)
	assert.NoError(t, err)

	info.parentHash = block32.Hash()
	info.height = uint32(4)
	info.time = uint32(time.Now().Unix() - 6)
	block42 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block42)
	assert.NoError(t, err)

	info.parentHash = block42.Hash()
	info.height = uint32(5)
	info.time = uint32(time.Now().Unix() - 3)
	block52 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block52)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block42.Hash(), 4)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block52.Hash()])
}

// 1、2{213、214、215、216}、{3{{314、315、316}、{324、325, 326}}}， set stable #3
func TestBlockChain_SetStableBlockCurBranch41(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	// Block1
	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.time = uint32(time.Now().Unix() - 22)
	info.gasLimit = 1000
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block1)
	assert.NoError(t, err)

	// Block2
	info.parentHash = block1.Hash()
	info.height = 2
	info.time = uint32(time.Now().Unix() - 19)
	info.gasLimit = 1001
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block2)
	assert.NoError(t, err)

	// Block213
	info.parentHash = block2.Hash()
	info.height = 3
	info.time = uint32(time.Now().Unix() - 16)
	info.gasLimit = 1002
	block213 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block213)
	assert.NoError(t, err)

	// Block24
	info.parentHash = block213.Hash()
	info.height = 4
	info.time = uint32(time.Now().Unix() - 13)
	info.gasLimit = 1003
	block214 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block214)
	assert.NoError(t, err)

	// Block215
	info.parentHash = block214.Hash()
	info.height = 5
	info.time = uint32(time.Now().Unix() - 10)
	info.gasLimit = 1004
	block215 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block215)
	assert.NoError(t, err)

	// Block3
	info.parentHash = block2.Hash()
	info.height = 3
	info.time = uint32(time.Now().Unix() - 9)
	info.gasLimit = 1005
	block3 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block3)
	assert.NoError(t, err)

	// Block314
	info.parentHash = block3.Hash()
	info.height = 4
	info.time = uint32(time.Now().Unix() - 6)
	info.gasLimit = 1006
	block314 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block314)
	assert.NoError(t, err)

	// Block315
	info.parentHash = block314.Hash()
	info.height = 5
	info.time = uint32(time.Now().Unix() - 3)
	info.gasLimit = 1007
	block315 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block315)
	assert.NoError(t, err)

	// Block316
	info.parentHash = block315.Hash()
	info.height = 6
	info.time = uint32(time.Now().Unix() - 0)
	info.gasLimit = 1008
	block316 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block316)
	assert.NoError(t, err)

	// Block324
	info.parentHash = block3.Hash()
	info.height = 4
	info.time = uint32(time.Now().Unix() + 1)
	info.gasLimit = 1009
	block324 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block324)
	assert.NoError(t, err)

	// Block325
	info.parentHash = block324.Hash()
	info.height = 5
	info.time = uint32(time.Now().Unix() + 4)
	info.gasLimit = 1010
	block325 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block325)
	assert.NoError(t, err)

	// Block326
	info.parentHash = block325.Hash()
	info.height = 6
	info.time = uint32(time.Now().Unix() + 7)
	info.gasLimit = 1011
	block326 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block326)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block3.Hash(), 3)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block316.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block326.Hash()])
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block316.Hash())
}

// 1、2{213、214、215、216}、{3{{314、315、316}、{324、325, 326}}}， set stable #2
func TestBlockChain_SetStableBlockCurBranch42(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	// Block1
	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.time = uint32(time.Now().Unix() - 22)
	info.gasLimit = 1000
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block1)
	assert.NoError(t, err)

	// Block2
	info.parentHash = block1.Hash()
	info.height = 2
	info.time = uint32(time.Now().Unix() - 19)
	info.gasLimit = 1001
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block2)
	assert.NoError(t, err)

	// Block213
	info.parentHash = block2.Hash()
	info.height = 3
	info.time = uint32(time.Now().Unix() - 16)
	info.gasLimit = 1002
	block213 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block213)
	assert.NoError(t, err)

	// Block24
	info.parentHash = block213.Hash()
	info.height = 4
	info.time = uint32(time.Now().Unix() - 13)
	info.gasLimit = 1003
	block214 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block214)
	assert.NoError(t, err)

	// Block215
	info.parentHash = block214.Hash()
	info.height = 5
	info.time = uint32(time.Now().Unix() - 10)
	info.gasLimit = 1004
	block215 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block215)
	assert.NoError(t, err)

	// Block3
	info.parentHash = block2.Hash()
	info.height = 3
	info.time = uint32(time.Now().Unix() - 9)
	info.gasLimit = 1005
	block3 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block3)
	assert.NoError(t, err)

	// Block314
	info.parentHash = block3.Hash()
	info.height = 4
	info.time = uint32(time.Now().Unix() - 6)
	info.gasLimit = 1006
	block314 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block314)
	assert.NoError(t, err)

	// Block315
	info.parentHash = block314.Hash()
	info.height = 5
	info.time = uint32(time.Now().Unix() - 3)
	info.gasLimit = 1007
	block315 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block315)
	assert.NoError(t, err)

	// Block316
	info.parentHash = block315.Hash()
	info.height = 6
	info.time = uint32(time.Now().Unix() - 0)
	info.gasLimit = 1008
	block316 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block316)
	assert.NoError(t, err)

	// Block324
	info.parentHash = block3.Hash()
	info.height = 4
	info.time = uint32(time.Now().Unix() + 1)
	info.gasLimit = 1009
	block324 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block324)
	assert.NoError(t, err)

	// Block325
	info.parentHash = block324.Hash()
	info.height = 5
	info.time = uint32(time.Now().Unix() + 4)
	info.gasLimit = 1010
	block325 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block325)
	assert.NoError(t, err)

	// Block326
	info.parentHash = block325.Hash()
	info.height = 6
	info.time = uint32(time.Now().Unix() + 7)
	info.gasLimit = 1011
	block326 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block326)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block2.Hash(), 2)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block215.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block316.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block326.Hash()])
	assert.Equal(t, blockChain.CurrentBlock().Hash().Prefix(), block215.Hash().Prefix())

	err = blockChain.SetStableBlock(block3.Hash(), 3)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block316.Hash()])
	assert.NotNil(t, blockChain.chainForksHead[block326.Hash()])
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block316.Hash())

	err = blockChain.SetStableBlock(block325.Hash(), 5)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block326.Hash()])
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block326.Hash())
}

// 1、2{213、214、215、216}、{3{{314、315、316}、{324、325, 326}}}， set stable #316
func TestBlockChain_SetStableBlockCurBranch43(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	// Block1
	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.time = uint32(time.Now().Unix() - 22)
	info.gasLimit = 1000
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block1)
	assert.NoError(t, err)

	// Block2
	info.parentHash = block1.Hash()
	info.height = 2
	info.time = uint32(time.Now().Unix() - 19)
	info.gasLimit = 1001
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block2)
	assert.NoError(t, err)

	// Block213
	info.parentHash = block2.Hash()
	info.height = 3
	info.time = uint32(time.Now().Unix() - 16)
	info.gasLimit = 1002
	block213 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block213)
	assert.NoError(t, err)

	// Block24
	info.parentHash = block213.Hash()
	info.height = 4
	info.time = uint32(time.Now().Unix() - 13)
	info.gasLimit = 1003
	block214 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block214)
	assert.NoError(t, err)

	// Block215
	info.parentHash = block214.Hash()
	info.height = 5
	info.time = uint32(time.Now().Unix() - 10)
	info.gasLimit = 1004
	block215 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block215)
	assert.NoError(t, err)

	// Block3
	info.parentHash = block2.Hash()
	info.height = 3
	info.time = uint32(time.Now().Unix() - 9)
	info.gasLimit = 1005
	block3 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block3)
	assert.NoError(t, err)

	// Block314
	info.parentHash = block3.Hash()
	info.height = 4
	info.time = uint32(time.Now().Unix() - 6)
	info.gasLimit = 1006
	block314 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block314)
	assert.NoError(t, err)

	// Block315
	info.parentHash = block314.Hash()
	info.height = 5
	info.time = uint32(time.Now().Unix() - 3)
	info.gasLimit = 1007
	block315 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block315)
	assert.NoError(t, err)

	// Block316
	info.parentHash = block315.Hash()
	info.height = 6
	info.time = uint32(time.Now().Unix() - 0)
	info.gasLimit = 1008
	block316 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block316)
	assert.NoError(t, err)

	// Block324
	info.parentHash = block3.Hash()
	info.height = 4
	info.time = uint32(time.Now().Unix() + 1)
	info.gasLimit = 1009
	block324 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block324)
	assert.NoError(t, err)

	// Block325
	info.parentHash = block324.Hash()
	info.height = 5
	info.time = uint32(time.Now().Unix() + 4)
	info.gasLimit = 1010
	block325 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block325)
	assert.NoError(t, err)

	// Block326
	info.parentHash = block325.Hash()
	info.height = 6
	info.time = uint32(time.Now().Unix() + 7)
	info.gasLimit = 1011
	block326 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block326)
	assert.NoError(t, err)

	err = blockChain.SetStableBlock(block316.Hash(), 6)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(blockChain.chainForksHead))
	assert.NotNil(t, blockChain.chainForksHead[block316.Hash()])
	assert.Equal(t, blockChain.CurrentBlock().Hash(), block316.Hash())
}

func buildConfirm(hash common.Hash, privateKey string) (*network.BlockConfirmData, error) {
	tmp, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, err
	}

	signData, err := crypto.Sign(hash[:], tmp)
	if err != nil {
		return nil, err
	}

	confirm := &network.BlockConfirmData{
		Height: 1,
		Hash:   hash,
	}
	copy(confirm.SignInfo[:], signData[:])

	return confirm, nil
}

func TestBlockChain_ReceiveConfirm(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	info := blockInfo{
		parentHash: genesis.Hash(),
		height:     1,
		gasLimit:   1000,
		time:       1540893799,
	}
	block1 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block1)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = 2
	info.gasLimit = 2000
	info.time = 1540893798
	block2 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block2)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = 3
	info.gasLimit = 3000
	info.time = 1540893799
	block3 := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertBlock(block3)
	assert.NoError(t, err)

	// recovery failed
	// confirm, err := buildConfirm(block1.Hash(), "c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa")
	// assert.NoError(t, err)
	//
	// confirm.SignInfo[2] = '1'
	// err = blockChain.ReceiveConfirm(confirm)
	// assert.Equal(t, ErrInvalidConfirmSigner, err)

	// unavailable confirm info.
	confirm, err := buildConfirm(block1.Hash(), "cbe9fa7c8721b8103e5af1ee5a40ac60c0c2b8c3c762e4e2c6ee0965917b1d86")
	assert.NoError(t, err)
	err = blockChain.ReceiveConfirm(confirm)
	assert.Equal(t, ErrInvalidConfirmSigner, err)

	// has block consensus
	err = blockChain.SetStableBlock(block1.Hash(), 1)
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

func TestBlockChain_VerifyAfterTxProcessNormal(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	tmp, err := crypto.HexToECDSA("c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa")
	assert.NoError(t, err)

	accounts := store.GetAccounts()
	info := blockInfo{
		parentHash:  genesis.Hash(),
		height:      1,
		gasLimit:    1000000000,
		deputyNodes: genesis.DeputyNodes,
		txList: []*types.Transaction{
			makeTx(tmp, accounts[0].Address, params.OrdinaryTx, big.NewInt(30000)),
			makeTx(tmp, accounts[1].Address, params.OrdinaryTx, big.NewInt(40000)),
		},
		time:   1540893799,
		author: genesis.MinerAddress(),
	}
	block := makeBlock(blockChain.db, info, false)
	newBlock, err := blockChain.VerifyAndSeal(block)
	assert.NoError(t, err)
	assert.NotEqual(t, merkle.EmptyTrieHash, newBlock.Header.VersionRoot)
	assert.NotEqual(t, merkle.EmptyTrieHash, newBlock.Header.LogRoot)
	assert.NotEqual(t, merkle.EmptyTrieHash, newBlock.Header.TxRoot)
	assert.NotEqual(t, 0, newBlock.Header.GasUsed)
}

func TestBlockChain_VerifyBlockBalanceNotEnough(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	tmp, err := crypto.HexToECDSA("9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb")
	assert.NoError(t, err)

	accounts := store.GetAccounts()
	info := blockInfo{
		parentHash:  genesis.Hash(),
		height:      1,
		gasLimit:    1000000000,
		deputyNodes: genesis.DeputyNodes,
		time:        1540893799,
		author:      genesis.MinerAddress(),
	}
	block := makeBlock(blockChain.db, info, false)
	block.Txs = []*types.Transaction{
		makeTx(tmp, accounts[0].Address, params.OrdinaryTx, big.NewInt(30000)),
		makeTx(tmp, accounts[1].Address, params.OrdinaryTx, big.NewInt(40000)),
	}
	block.Header.TxRoot = block.Txs.MerkleRootSha()
	_, err = blockChain.VerifyAndSeal(block)
	assert.Equal(t, err, consensus.ErrInvalidTxInBlock)
}

func TestBlockChain_VerifyBlockBalanceNotSign(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	accounts := store.GetAccounts()
	info := blockInfo{
		parentHash:  genesis.Hash(),
		height:      1,
		gasLimit:    1000000000,
		deputyNodes: genesis.DeputyNodes,
		time:        1540893799,
		author:      genesis.MinerAddress(),
	}
	block := makeBlock(blockChain.db, info, false)
	block.Txs = []*types.Transaction{
		types.NewTransaction(accounts[0].Address, common.Big2, 30000, common.Big2, []byte{}, 0, 200, 1538210398, "", ""),
	}
	block.Header.TxRoot = block.Txs.MerkleRootSha()
	_, err = blockChain.VerifyAndSeal(block)
	assert.Equal(t, err, consensus.ErrInvalidTxInBlock)
}

func TestBlockChain_VerifyBlockBalanceValidDeputy(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	tmp, err := crypto.HexToECDSA("c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa")
	assert.NoError(t, err)

	accounts := store.GetAccounts()
	info := blockInfo{
		parentHash:  genesis.Hash(),
		height:      100000000,
		gasLimit:    1000000000,
		deputyNodes: genesis.DeputyNodes,
		txList: []*types.Transaction{
			makeTx(tmp, accounts[0].Address, params.OrdinaryTx, big.NewInt(30000)),
			makeTx(tmp, accounts[1].Address, params.OrdinaryTx, big.NewInt(40000)),
		},
		time:   1540893799,
		author: genesis.MinerAddress(),
	}
	block := makeBlock(blockChain.db, info, false)
	block.DeputyNodes = append(block.DeputyNodes[:1], block.DeputyNodes[2:]...)
	_, err = blockChain.VerifyAndSeal(block)
	assert.Equal(t, err, consensus.ErrVerifyBlockFailed)
}

func TestBlockChain_VerifyBlockBalanceValidTx(t *testing.T) {
	ClearData()

	blockChain, _, err := NewBlockChainForTest()
	defer blockChain.db.Close()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	tmp, err := crypto.HexToECDSA("c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa")
	assert.NoError(t, err)

	accounts := store.GetAccounts()
	info := blockInfo{
		parentHash:  genesis.Hash(),
		height:      1,
		gasLimit:    1000000000,
		deputyNodes: genesis.DeputyNodes,
		txList: []*types.Transaction{
			makeTx(tmp, accounts[0].Address, params.OrdinaryTx, big.NewInt(30000)),
			makeTx(tmp, accounts[1].Address, params.OrdinaryTx, big.NewInt(40000)),
		},
		time:   1540893799,
		author: genesis.MinerAddress(),
	}
	block := makeBlock(blockChain.db, info, false)
	block.Txs = append(block.Txs[:1], block.Txs[2:]...)
	_, err = blockChain.VerifyAndSeal(block)
	assert.Equal(t, err, consensus.ErrVerifyBlockFailed)
}

// 1->2->{31,32} 32
func TestInsertChain_1(t *testing.T) {
	ClearData()

	bc, _, err := NewBlockChainForTest()
	defer bc.db.Close()
	assert.NoError(t, err)

	genesis := bc.GetBlockByHeight(0)
	assert.NotNil(t, genesis)

	var info blockInfo
	info.parentHash = genesis.Hash()
	info.height = uint32(1)
	info.author = common.HexToAddress(consensus.block01MinerAddress)
	info.gasLimit = 1000
	info.time = uint32(time.Now().Unix() - 22)
	block1 := makeBlock(bc.db, info, false)
	err = bc.InsertBlock(block1)
	assert.NoError(t, err)

	info.parentHash = block1.Hash()
	info.height = uint32(2)
	info.author = common.HexToAddress(consensus.block02MinerAddress)
	info.gasLimit = 1000
	info.time = uint32(time.Now().Unix() - 19)
	block2 := makeBlock(bc.db, info, false)
	err = bc.InsertBlock(block2)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.author = common.HexToAddress(consensus.block04MinerAddress)
	info.gasLimit = 1000
	info.time = uint32(time.Now().Unix() - 9)
	block32 := makeBlock(bc.db, info, false)
	err = bc.InsertBlock(block32)
	assert.NoError(t, err)

	info.parentHash = block2.Hash()
	info.height = uint32(3)
	info.author = common.HexToAddress(consensus.block03MinerAddress)
	info.gasLimit = 1000
	info.time = uint32(time.Now().Unix() - 13)
	block31 := makeBlock(bc.db, info, false)
	err = bc.InsertBlock(block31)
	assert.NoError(t, err)

	assert.Equal(t, bc.CurrentBlock().Hash(), block31.Hash())

	_ = bc.SetStableBlock(block32.Hash(), block32.Height())

	cb := bc.CurrentBlock()
	assert.Equal(t, cb.Hash(), block32.Hash())
}