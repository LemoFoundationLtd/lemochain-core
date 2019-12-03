package consensus

import (
	"crypto/ecdsa"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"math/big"
	"math/rand"
	"os"
	"time"
)

var (
	// The first deputy's private is set to "selfNodeKey" which means my miner private
	testDeputies        = generateDeputies(17)
	testBlocks          = generateBlocks()
	testChainID  uint16 = 123
)

func init() {
	log.Setup(log.LevelDebug, false, false)
}

func GetStorePath() string {
	return "../../testdata/consensus"
}

func ClearData() {
	err := os.RemoveAll(GetStorePath())
	failCnt := 1
	for err != nil {
		log.Errorf("CLEAR DATA BASE FAIL.%s, SLEEP(%ds) AND CONTINUE", err.Error(), failCnt)
		time.Sleep(time.Duration(failCnt) * time.Second)
		err = os.RemoveAll(GetStorePath())
		failCnt++
	}
}

type deputyTestData struct {
	types.DeputyNode
	*ecdsa.PrivateKey
}

type deputyTestDatas []deputyTestData

func (dta deputyTestDatas) ToDeputyNodes() types.DeputyNodes {
	result := make(types.DeputyNodes, len(dta))
	for i := range dta {
		result[i] = &dta[i].DeputyNode
	}
	return result
}

func (dta deputyTestDatas) FindByMiner(miner common.Address) *deputyTestData {
	for _, deputy := range dta {
		if deputy.MinerAddress == miner {
			return &deputy
		}
	}
	return nil
}

type testBlockLoader struct {
	Blocks []*types.Block
	Stable *types.Block
}

func (bl *testBlockLoader) IterateUnConfirms(fn func(*types.Block)) {
	for i := 0; i < len(bl.Blocks); i++ {
		block := bl.Blocks[i]
		if bl.isUnstable(block) {
			fn(block)
		}
	}
}

func (bl *testBlockLoader) GetUnConfirmByHeight(height uint32, leafBlockHash common.Hash) (*types.Block, error) {
	block, _ := bl.GetBlockByHash(leafBlockHash)
	if bl.isUnstable(block) {
		return block, nil
	} else {
		return nil, store.ErrNotExist
	}
}

func (bl *testBlockLoader) isUnstable(block *types.Block) bool {
	if bl.Stable == nil {
		return true
	}
	if block == nil || block.Height() <= bl.Stable.Height() {
		return false
	}
	parent := block
	for {
		parent, _ = bl.GetBlockByHash(parent.ParentHash())
		if parent == nil {
			return false
		} else if parent == bl.Stable {
			return true
		}
	}
}

func (bl *testBlockLoader) GetBlockByHash(hash common.Hash) (*types.Block, error) {
	for i := 0; i < len(bl.Blocks); i++ {
		if bl.Blocks[i].Hash() == hash {
			return bl.Blocks[i], nil
		}
	}
	return nil, store.ErrNotExist
}

func (bl *testBlockLoader) GetBlockByHeight(height uint32) (*types.Block, error) {
	for i := 0; i < len(bl.Blocks); i++ {
		if bl.Blocks[i].Height() == height {
			return bl.Blocks[i], nil
		}
	}
	return nil, store.ErrNotExist
}

// AppendBlock create a new test block then append to testBlockLoader
func (bl *testBlockLoader) AppendBlock(height, time uint32, parentIndex int) {
	block := &types.Block{Header: &types.Header{Height: height, Time: time}}
	if parentIndex >= 0 {
		block.Header.ParentHash = bl.Blocks[parentIndex].Hash()
	}
	bl.Blocks = append(bl.Blocks, block)
}

// createBlockLoader picks some test blocks by index
func createBlockLoader(blockIndexList []int, stableIndex int) *testBlockLoader {
	result := &testBlockLoader{}
	for _, blockIndex := range blockIndexList {
		result.Blocks = append(result.Blocks, testBlocks[blockIndex])
	}
	if stableIndex != -1 {
		result.Stable = testBlocks[stableIndex]
	}
	return result
}

// createUnstableLoader create a block loader
func createUnstableLoader(blocks ...*types.Block) *testBlockLoader {
	return &testBlockLoader{Blocks: blocks}
}

type testCandidateLoader types.DeputyNodes

func (cl testCandidateLoader) LoadTopCandidates(blockHash common.Hash) types.DeputyNodes {
	return types.DeputyNodes(cl)
}

func (cl testCandidateLoader) LoadRefundCandidates(height uint32) ([]common.Address, error) {
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
func generateDeputies(num int) deputyTestDatas {
	var result deputyTestDatas
	for i := 0; i < num; i++ {
		private, _ := crypto.GenerateKey()
		node := types.DeputyNode{
			MinerAddress: crypto.PubkeyToAddress(private.PublicKey),
			NodeID:       crypto.PrivateKeyToNodeID(private),
			Rank:         uint32(i),
			Votes:        big.NewInt(int64(10000000000 - i)),
		}
		result = append(result, deputyTestData{DeputyNode: node, PrivateKey: private})
		// let me to be the first deputy
		if i == 0 {
			deputynode.SetSelfNodeKey(private)
		}
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

func initDeputyManager(deputyCount int) *deputynode.Manager {
	dm := deputynode.NewManager(deputyCount, &testBlockLoader{})
	nodeIndexList := make([]int, deputyCount)
	for i := range nodeIndexList {
		nodeIndexList[i] = i
	}
	nodes := pickNodes(nodeIndexList...)
	dm.SaveSnapshot(0, nodes)
	return dm
}

// generateBlocks generate block forks like this:
//       ┌─2
// 0───1─┼─3───6
//       ├─4─┬─7───9
//       │   └─8
//       └─5
func generateBlocks() []*types.Block {
	loader := testBlockLoader{}
	loader.AppendBlock(100, 0, -1) // 0 757227e1
	loader.AppendBlock(101, 1, 0)  // 1 1e4ef847
	loader.AppendBlock(102, 2, 1)  // 2 42b341d2
	loader.AppendBlock(102, 3, 1)  // 3 6937c4b0
	loader.AppendBlock(102, 4, 1)  // 4 9919ec3c
	loader.AppendBlock(102, 5, 1)  // 5 aff9c979
	loader.AppendBlock(103, 6, 3)  // 6 3b6c49af
	loader.AppendBlock(103, 7, 4)  // 7 6ee786b3
	loader.AppendBlock(103, 8, 4)  // 8 8cfd42ce
	loader.AppendBlock(104, 9, 7)  // 9 1bbff42c
	return []*types.Block(loader.Blocks)
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

func (tp txPoolForValidator) ExistPendingTx(time uint32) bool {
	panic("implement me")
}

func (tp txPoolForValidator) PushTx(tx *types.Transaction) bool {
	panic("implement me")
}

func (tp txPoolForValidator) SetTxsFlag(txs []*types.Transaction, isPending bool) bool {
	panic("implement me")
}

func (txPoolForValidator) Get(time uint32, size int) []*types.Transaction {
	panic("implement me")
}

func (txPoolForValidator) DelInvalidTxs(txs []*types.Transaction) {
	panic("implement me")
}

func (txPoolForValidator) RecvBlock(block *types.Block) {
	panic("implement me")
}

type parentLoader struct {
	Db protocol.ChainDB
}

func (t *parentLoader) GetParentByHeight(height uint32, sonBlockHash common.Hash) *types.Block {
	block, err := t.Db.GetUnConfirmByHeight(height, sonBlockHash)
	if err == store.ErrNotExist {
		block, err = t.Db.GetBlockByHeight(height)
	}

	if err != nil {
		log.Error("load block by height fail", "height", height, "err", err)
		return nil
	}
	return block
}

func MakeTxFast(fromPrivate *ecdsa.PrivateKey) *types.Transaction {
	from := crypto.PubkeyToAddress(fromPrivate.PublicKey)
	r := rand.New(rand.NewSource(1234))
	expiration := uint64(time.Now().Unix() + 100 + r.Int63n(200))
	tx := types.NewTransaction(from, from, common.Lemo2Mo("100"), 1000000, big.NewInt(100), []byte{}, params.OrdinaryTx, testChainID, expiration, "", string("aaa"))
	return SignTx(tx, fromPrivate)
}

func MakeTx(fromPrivate *ecdsa.PrivateKey, to common.Address, amount *big.Int, expiration uint64) *types.Transaction {
	from := crypto.PubkeyToAddress(fromPrivate.PublicKey)
	tx := types.NewTransaction(from, to, amount, 1000000, big.NewInt(100), []byte{}, params.OrdinaryTx, testChainID, expiration, "", string("aaa"))
	return SignTx(tx, fromPrivate)
}

func SignTx(tx *types.Transaction, private *ecdsa.PrivateKey) *types.Transaction {
	tx, err := types.DefaultSigner{}.SignTx(tx, private)
	if err != nil {
		panic(err)
	}
	return tx
}

// initGenesis create and save a genesis block to db, then set some balance to the address
func initGenesis(db *store.ChainDatabase, am *account.Manager, deputies types.DeputyNodes) *types.Block {
	genesisBlock := &types.Block{
		Header:      &types.Header{MinerAddress: deputies[0].MinerAddress},
		DeputyNodes: deputies,
	}
	am.Reset(common.Hash{})
	am.GetAccount(deputies[0].MinerAddress).SetBalance(common.Lemo2Mo("10000"))
	err := am.Finalise()
	if err != nil {
		panic(err)
	}
	err = db.SetBlock(genesisBlock.Hash(), genesisBlock)
	if err != nil {
		panic(err)
	}
	err = am.Save(genesisBlock.Hash())
	if err != nil {
		panic(err)
	}
	if _, err := db.SetStableBlock(genesisBlock.Hash()); err != nil {
		log.Errorf("setup genesis block failed: %v", err)
		panic(err)
	}
	return genesisBlock
}
