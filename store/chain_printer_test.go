package store

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

// makeForkBlocks make blocks and setup the tree struct like this:
//       ┌─2
// 0───1─┼─3───6
//       ├─4─┬─7───9
//       │   └─8
//       └─5
func makeForkBlocks() []*CBlock {
	rawBlock0 := &types.Block{Header: &types.Header{Height: 99}}
	block0 := &CBlock{Block: rawBlock0}
	rawBlock1 := &types.Block{Header: &types.Header{Height: 100}}
	block1 := &CBlock{Block: rawBlock1}
	rawBlock2 := &types.Block{Header: &types.Header{Height: 101, Time: 1}}
	block2 := &CBlock{Block: rawBlock2}
	rawBlock3 := &types.Block{Header: &types.Header{Height: 101, Time: 2}}
	block3 := &CBlock{Block: rawBlock3}
	rawBlock4 := &types.Block{Header: &types.Header{Height: 101, Time: 3}}
	block4 := &CBlock{Block: rawBlock4}
	rawBlock5 := &types.Block{Header: &types.Header{Height: 101, Time: 4}}
	block5 := &CBlock{Block: rawBlock5}
	rawBlock6 := &types.Block{Header: &types.Header{Height: 102, Time: 5}}
	block6 := &CBlock{Block: rawBlock6}
	rawBlock7 := &types.Block{Header: &types.Header{Height: 102, Time: 6}}
	block7 := &CBlock{Block: rawBlock7}
	rawBlock8 := &types.Block{Header: &types.Header{Height: 102, Time: 7}}
	block8 := &CBlock{Block: rawBlock8}
	rawBlock9 := &types.Block{Header: &types.Header{Height: 103, Time: 8}}
	block9 := &CBlock{Block: rawBlock9}

	block1.BeChildOf(block0)
	block2.BeChildOf(block1)
	block3.BeChildOf(block1)
	block4.BeChildOf(block1)
	block5.BeChildOf(block1)
	block6.BeChildOf(block3)
	block7.BeChildOf(block4)
	block8.BeChildOf(block4)
	block9.BeChildOf(block7)
	return []*CBlock{block0, block1, block2, block3, block4, block5, block6, block7, block8, block9}
}

func ExampleSerializeForks() {
	blocks := makeForkBlocks()
	blockMap := make(map[common.Hash]*CBlock, len(blocks))
	for _, block := range blocks {
		blockMap[block.Block.Hash()] = block
	}

	fmt.Printf(SerializeForks(blockMap, blocks[9].Block.Hash()))

	// Output: ─[ 99]5bd69f─[100]757227┬[101]1ba62c
	//                         ├[101]1dc055
	//                         ├[101]1f5603┬[102]44ae7c
	//                         │           └[102]6490a0─[103]db4799 <-Current
	//                         └[101]29d8a5─[102]379da9
}
