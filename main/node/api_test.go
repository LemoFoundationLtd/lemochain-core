package node

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestAccountAPI_api account api test
func TestAccountAPI_api(t *testing.T) {
	db := newDB()
	defer store.ClearData()
	am := account.NewManager(common.Hash{}, db)
	acc := NewPublicAccountAPI(am)
	priAcc := NewPrivateAccountAPI(am)
	// Create key pair
	addressKeyPair, err := priAcc.NewKeyPair()
	assert.NoError(t, err)
	t.Log(addressKeyPair)

	// getBalance api
	b01 := acc.manager.GetCanonicalAccount(common.HexToAddress("0x015780F8456F9c1532645087a19DcF9a7e0c7F97")).GetBalance().String()
	bb01, err := acc.GetBalance("0x015780F8456F9c1532645087a19DcF9a7e0c7F97")
	assert.NoError(t, err)
	assert.Equal(t, b01, bb01)

	address, err := common.StringToAddress("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG")
	assert.NoError(t, err)
	b02 := acc.manager.GetCanonicalAccount(address).GetBalance().String()
	bb02, err := acc.GetBalance("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG")
	assert.NoError(t, err)
	assert.Equal(t, b02, bb02)

	b03 := acc.manager.GetCanonicalAccount(testAddr).GetBalance().String()
	bb03, err := acc.GetBalance(testAddr.String())
	assert.NoError(t, err)
	assert.Equal(t, b03, bb03)

	// get account api
	account01, err := acc.GetAccount("0x015780F8456F9c1532645087a19DcF9a7e0c7F97")
	assert.NoError(t, err)
	addr, err := common.StringToAddress("0x015780F8456F9c1532645087a19DcF9a7e0c7F97")
	assert.NoError(t, err)
	assert.Equal(t, acc.manager.GetCanonicalAccount(addr), account01)

}

// TestChainAPI_api chain api test
func TestChainAPI_api(t *testing.T) {
	bc := newChain()
	defer store.ClearData()
	c := NewPublicChainAPI(bc)

	// getBlockByHash
	exBlock1 := c.chain.GetBlockByHash(common.HexToHash("0x3f4c3152fb02a7673bf804b1ddeb75542b6ef9a5a87501d9cfbbcf6c3632a211"))
	assert.Equal(t, exBlock1, c.GetBlockByHash("0x3f4c3152fb02a7673bf804b1ddeb75542b6ef9a5a87501d9cfbbcf6c3632a211", true))
	Block1 := &types.Block{
		Header: exBlock1.Header,
	}
	assert.Equal(t, Block1, c.GetBlockByHash("0x3f4c3152fb02a7673bf804b1ddeb75542b6ef9a5a87501d9cfbbcf6c3632a211", false))

	// getBlockByHeight
	exBlock2 := c.chain.GetBlockByHeight(1)
	assert.Equal(t, exBlock2, c.GetBlockByHeight(1, true))
	Block2 := &types.Block{
		Header: exBlock2.Header,
	}
	assert.Equal(t, Block2, c.GetBlockByHeight(1, false))

	// get chain ID api
	assert.Equal(t, c.chain.ChainID(), c.ChainID())

	// get genesis block api
	assert.Equal(t, c.chain.Genesis(), c.Genesis())

	// get current block api
	curBlock := c.chain.CurrentBlock()
	assert.Equal(t, curBlock, c.CurrentBlock(true))
	cBlock := &types.Block{
		Header: curBlock.Header,
	}
	assert.Equal(t, cBlock, c.CurrentBlock(false))

	// get stable block api
	StaBlock := c.chain.StableBlock()
	assert.Equal(t, StaBlock, c.LatestStableBlock(true))
	sBlock := &types.Block{
		Header: StaBlock.Header,
	}
	assert.Equal(t, sBlock, c.LatestStableBlock(false))

	// get current chain height api
	assert.Equal(t, c.chain.CurrentBlock().Height(), c.CurrentHeight())

	// get latest stable block height
	assert.Equal(t, c.chain.StableBlock().Height(), c.LatestStableHeight())

	// get suggest gas price
	// todo

	// get nodeVersion
	// todo

}

// TestTxAPI_api send tx api test
func TestTxAPI_api(t *testing.T) {
	testTx := types.NewTransaction(common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 200, uint64(1544596), "aa", string("send a Tx"))
	signTx := signTransaction(testTx, testPrivate)
	// txCh := make(chan types.Transactions, 100)
	pool := chain.NewTxPool(nil)
	txAPI := NewPublicTxAPI(pool)

	sendTxHash, err := txAPI.SendTx(signTx)
	assert.Nil(t, err)
	assert.Equal(t, signTx.Hash(), sendTxHash)
}

// // TestMineAPI_api miner api test // todo
// func TestMineAPI_api(t *testing.T) {
// 	lemoConf := &LemoConfig{
// 		Genesis:   chain.DefaultGenesisBlock(),
// 		NetworkId: 1,
// 		MaxPeers:  1000,
// 		Port:      7001,
// 		NodeKey:   "0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa",
// 		ExtraData: []byte{},
// 	}
// 	testNode, err := New(lemoConf, &DefaultNodeConfig, flag.NewCmdFlags(&cli.Context{}, []cli.Flag{}))
// 	t.Error(err)
//
// 	miner := (*testNode).miner
// 	m := NewMineAPI(miner)
// 	t.Log("after:", m.IsMining())
// 	m.MineStart()
// 	t.Log("then:", m.IsMining())
// 	// todo
// 	m.MineStop()
// 	t.Log("last:", m.IsMining())
//
// 	assert.Equal(t, "0x015780F8456F9c1532645087a19DcF9a7e0c7F97", m.MinerAddress())
//
// }
//
