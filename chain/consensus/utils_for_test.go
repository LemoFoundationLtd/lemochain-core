package consensus

import (
	"crypto/ecdsa"
	"crypto/rand"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto/secp256k1"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"math/big"
)

var (
	testDeputies = generateDeputies(17)
	testBlocks   = generateBlocks()
)

type testBlockLoader []*types.Block

func (bl testBlockLoader) IterateUnConfirms(fn func(*types.Block)) {
	for i := 0; i < len(bl); i++ {
		fn(bl[i])
	}
}

func (bl testBlockLoader) GetBlockByHash(hash common.Hash) (*types.Block, error) {
	for i := 0; i < len(bl); i++ {
		if bl[i].Hash() == hash {
			return bl[i], nil
		}
	}
	return nil, store.ErrNotExist
}

func (bl testBlockLoader) GetBlockByHeight(height uint32) (*types.Block, error) {
	for i := 0; i < len(bl); i++ {
		if bl[i].Height() == height {
			return bl[i], nil
		}
	}
	return nil, store.ErrNotExist
}

// AppendBlock create a new test block then append to testBlockLoader
func (bl testBlockLoader) AppendBlock(height, time uint32, parentIndex int) testBlockLoader {
	block := &types.Block{Header: &types.Header{Height: height, Time: time}}
	if parentIndex >= 0 {
		block.Header.ParentHash = bl[parentIndex].Hash()
	}
	newTestBlockLoader := append(bl, block)
	return newTestBlockLoader
}

// createUnconfirmBlockLoader picks some test blocks by index
func createUnconfirmBlockLoader(blockIndexList []int) testBlockLoader {
	var result testBlockLoader
	for _, blockIndex := range blockIndexList {
		result = append(result, testBlocks[blockIndex])
	}
	return result
}

type testCandidateLoader types.DeputyNodes

func (cl testCandidateLoader) LoadTopCandidates(blockHash common.Hash) types.DeputyNodes {
	return types.DeputyNodes(cl)
}

func (cl testCandidateLoader) LoadRefundCandidates() ([]common.Address, error) {
	var result []common.Address
	for i := 0; i < len(cl); i++ {
		result = append(result, cl[i].MinerAddress)
	}
	return result, nil
}

// createCandidateLoader picks some test deputies by index
func createCandidateLoader(nodeIndexList ...int) testCandidateLoader {
	return testCandidateLoader(pickNodes(nodeIndexList...))
}

// GenerateDeputies generate random deputy nodes
func generateDeputies(num int) types.DeputyNodes {
	var result []*types.DeputyNode
	for i := 0; i < num; i++ {
		private, _ := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
		result = append(result, &types.DeputyNode{
			MinerAddress: crypto.PubkeyToAddress(private.PublicKey),
			NodeID:       (crypto.FromECDSAPub(&private.PublicKey))[1:],
			Rank:         uint32(i),
			Votes:        big.NewInt(int64(10000000000 - i)),
		})
	}
	return result
}

// pickNodes picks some test deputy nodes by index
func pickNodes(nodeIndexList ...int) types.DeputyNodes {
	var result []*types.DeputyNode
	for i, nodeIndex := range nodeIndexList {
		newDeputy := testDeputies[nodeIndex].Copy()
		// reset rank
		newDeputy.Rank = uint32(i)
		result = append(result, newDeputy)
	}
	return result
}

// generateBlocks generate block forks like this:
//       ┌─2
// 0───1─┼─3───6
//       ├─4─┬─7───9
//       │   └─8
//       └─5
func generateBlocks() []*types.Block {
	loader := testBlockLoader{}
	loader = loader.AppendBlock(100, 0, -1) // 0 757227e1
	loader = loader.AppendBlock(101, 1, 0)  // 1 1e4ef847
	loader = loader.AppendBlock(102, 2, 1)  // 2 42b341d2
	loader = loader.AppendBlock(102, 3, 1)  // 3 6937c4b0
	loader = loader.AppendBlock(102, 4, 1)  // 4 9919ec3c
	loader = loader.AppendBlock(102, 5, 1)  // 5 aff9c979
	loader = loader.AppendBlock(103, 6, 3)  // 6 3b6c49af
	loader = loader.AppendBlock(103, 7, 4)  // 7 6ee786b3
	loader = loader.AppendBlock(103, 8, 4)  // 8 8cfd42ce
	loader = loader.AppendBlock(104, 9, 7)  // 9 1bbff42c
	return []*types.Block(loader)
}

func getTestBlockIndex(targetBlock *types.Block) int {
	for i, block := range testBlocks {
		if block == targetBlock {
			return i
		}
	}
	return -1
}