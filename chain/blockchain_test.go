package chain

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/stretchr/testify/assert"
	"testing"
)

type EngineTest struct {
	//
}

func (engine *EngineTest) VerifyHeader(block *types.Block) error {
	return nil
}

func (engine *EngineTest) Seal(header *types.Header, txs []*types.Transaction, changeLog []*types.ChangeLog, events []*types.Event) (*types.Block, error) {
	return nil, nil
}

func (engine *EngineTest) Finalize(header *types.Header) {
	//
}

func initDeputyNodes() error {
	manager := deputynode.Instance()
	privateKey, err := crypto.HexToECDSA("432a86ab8765d82415a803e29864dcfc1ed93dac949abf6f95a583179f27e4bb")
	if err != nil {

	}

	nodes := make([]deputynode.DeputyNode, 2)
	nodes[0] = deputynode.DeputyNode{
		LemoBase: common.HexToAddress("0x10000"),
		NodeID:   crypto.FromECDSAPub(&privateKey.PublicKey)[1:],
		IP:       []byte{'e', 'e', 'e', 'e'},
		Port:     60000,
		Rank:     0,
		Votes:    100000,
	}

	nodes[1] = deputynode.DeputyNode{
		LemoBase: common.HexToAddress("0x20000"),
		NodeID:   crypto.FromECDSAPub(&privateKey.PublicKey)[1:],
		IP:       []byte{'f', 'f', 'f', 'f'},
		Port:     60000,
		Rank:     0,
		Votes:    100000,
	}

	manager.Add(0, nodes)

	return nil
}

func newGenesis(db *store.CacheChain) *types.Block {
	genesis := DefaultGenesisBlock()
	// am := account.NewManager(common.Hash{}, db)
	return genesis.ToBlock()
}

func broadcastStableBlock(block *types.Block) {
	//
}

func broadcastConfirmInfo(hash common.Hash, height uint32) {
	//
}

func newBlockChain() (*BlockChain, chan *types.Block, error) {
	store.ClearData()

	chainId := uint64(99)
	db, err := store.NewCacheChain(store.GetStorePath())
	if err != nil {
		return nil, nil, err
	}

	gBlock := newGenesis(db)
	err = db.SetBlock(gBlock.Hash(), gBlock)
	if err != nil {
		return nil, nil, err
	}

	am := account.NewManager(common.Hash{}, db)
	err = am.Save(gBlock.Hash())
	if err != nil {
		return nil, nil, err
	}

	err = db.SetStableBlock(gBlock.Hash())
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
	return blockChain, ch, nil
}

func TestBlockChain_Genesis(t *testing.T) {
	store.ClearData()

	err := initDeputyNodes()
	assert.NoError(t, err)

	blockChain, _, err := newBlockChain()
	assert.NoError(t, err)

	genesis := blockChain.GetBlockByHeight(0)

	info := blockInfo{
		parentHash: genesis.Hash(),
		height:     1,
	}
	block := makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block)
	assert.NoError(t, err)

	info.parentHash = block.Hash()
	info.height = 2
	block = makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block)
	assert.NoError(t, err)

	hash := block.Hash()

	info.parentHash = hash
	info.height = 3
	block = makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block)
	assert.NoError(t, err)

	info.parentHash = hash
	info.height = 3
	info.gasLimit = 1000
	block = makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block)
	assert.NoError(t, err)

	info.parentHash = block.Hash()
	info.height = 4
	block = makeBlock(blockChain.db, info, false)
	err = blockChain.InsertChain(block)
	assert.NoError(t, err)
}
