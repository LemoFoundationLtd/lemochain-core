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
	mathRand "math/rand"
	"time"
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

// txPoolForValidator is a txPool for test. It only contains a bool which will be returned by VerifyTxInBlock
type txPoolForValidator struct {
	blockIsValid bool
}

func (txPoolForValidator) Get(time uint32, size int) []*types.Transaction {
	panic("implement me")
}

func (txPoolForValidator) DelInvalidTxs(txs []*types.Transaction) {
	panic("implement me")
}

func (tp txPoolForValidator) VerifyTxInBlock(block *types.Block) bool {
	return tp.blockIsValid
}

func (txPoolForValidator) RecvBlock(block *types.Block) {
	panic("implement me")
}

func (txPoolForValidator) PruneBlock(block *types.Block) {
	panic("implement me")
}

// 用于New deputyNode manager初始化deputy
type loader struct {
	Nodes types.DeputyNodes
}

func (l loader) GetBlockByHeight(height uint32) (*types.Block, error) {
	if height > 0 {
		return nil, store.ErrNotExist
	}
	return &types.Block{
		DeputyNodes: l.Nodes,
	}, nil
}

var (
	minerAddr, _ = common.StringToAddress("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG")
	minerPrivate = "c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"
	minerNodeId  = common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0")

	addr02, _ = common.StringToAddress("Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY")
	private02 = "9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb"
	nodeId02  = common.FromHex("0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0")
)

// newBlockForVerifySigner 需要构造出区块签名数据、MinerAddress、区块高度
func newBlockForVerifySigner(height uint32, private string) *types.Block {
	privateKey, _ := crypto.HexToECDSA(private)
	minerAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	header := &types.Header{
		MinerAddress: minerAddress,
		Height:       height,
	}
	block := &types.Block{
		Header: header,
	}
	hash := block.Hash()
	signData, _ := crypto.Sign(hash[:], privateKey)
	block.Header.SignData = signData
	return block
}

func makeTx(from, to common.Address, txTime uint64) *types.Transaction {
	return types.NewTransaction(from, to, big.NewInt(100), uint64(1000), big.NewInt(100), nil, 0, 1, txTime, "", "")
}

// newBlockForVerifyTxRoot
func newBlockForVerifyTxRoot(txs types.Transactions, txRoot common.Hash) *types.Block {
	header := &types.Header{
		TxRoot: txRoot,
	}
	return &types.Block{
		Header: header,
		Txs:    txs,
	}
}

func newBlockForVerifyTxs(txs types.Transactions, time uint32) *types.Block {
	header := &types.Header{
		Time: time,
	}
	return &types.Block{
		Header: header,
		Txs:    txs,
	}
}

func newBlockForVerifyHeight(height uint32) *types.Block {
	header := &types.Header{
		Height: height,
	}
	return &types.Block{
		Header: header,
	}
}

func newBlockForVerifyTime(time uint32) *types.Block {
	header := &types.Header{
		Time: time,
	}
	return &types.Block{
		Header: header,
	}
}

func newBlockForVerifyDeputy(height uint32, deputyNodes types.DeputyNodes, deputyRoot []byte) *types.Block {
	header := &types.Header{
		Height:     height,
		DeputyRoot: deputyRoot,
	}
	return &types.Block{
		Header:      header,
		DeputyNodes: deputyNodes,
	}
}

func newBlockForVerifyExtraData(extraData []byte) *types.Block {
	return &types.Block{
		Header: &types.Header{
			Extra: extraData,
		},
	}
}

// time单位:s
func newBlockForVerifyMineSlot(height uint32, minerAddress common.Address, time uint32) *types.Block {
	return &types.Block{
		Header: &types.Header{
			MinerAddress: minerAddress,
			Height:       height,
			Time:         time,
		},
	}
}

// assembleBlockForVerifyMineSlot
func assembleBlockForVerifyMineSlot(passTime, oneLoopTime uint32, parentMiner, currentMiner common.Address) (parentBlock *types.Block, currentBlock *types.Block) {
	mathRand.Seed(time.Now().UnixNano())
	parentTime := uint32(mathRand.Intn(500)) + 1
	blockTime := parentTime + passTime + oneLoopTime*uint32(mathRand.Intn(5)) // blockTime为parentTime + 正确的相差时间 + 随机的轮数
	parentBlock = newBlockForVerifyMineSlot(1, parentMiner, parentTime)
	currentBlock = newBlockForVerifyMineSlot(2, currentMiner, blockTime)
	return
}
