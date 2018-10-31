package node

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

// Test_AddPoint test AddPoint function
func Test_AddPoint(t *testing.T) {
	test01 := "11111111111111111111"
	test02 := "111111111111"
	test03 := "111111111111111111111111111111111111111111111111"

	B01 := addPoint(test01)
	assert.Equal(t, "11.111111111111111111", B01)

	B02 := addPoint(test02)
	assert.Equal(t, "0.000000111111111111", B02)

	B03 := addPoint(test03)
	assert.Equal(t, "111111111111111111111111111111.111111111111111111", B03)
}

// TestAccountAPI_api account api test
func TestAccountAPI_api(t *testing.T) {
	db := newDB()
	am := account.NewManager(common.Hash{}, db)
	acc := NewAccountAPI(am)
	// Create key pair
	addressKeyPair, err := acc.NewKeyPair()
	assert.Nil(t, err)
	t.Log(addressKeyPair)

	// getBalance api
	B01 := acc.manager.GetCanonicalAccount(common.HexToAddress("0x015780F8456F9c1532645087a19DcF9a7e0c7F97")).GetBalance().String()
	b01 := addPoint(B01)
	bb01, err := acc.GetBalance("0x015780F8456F9c1532645087a19DcF9a7e0c7F97")
	assert.Nil(t, err)
	assert.Equal(t, b01, bb01)

	address, err := common.RestoreOriginalAddress("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG")
	assert.Nil(t, err)
	B02 := acc.manager.GetCanonicalAccount(address).GetBalance().String()
	b02 := addPoint(B02)
	bb02, err := acc.GetBalance("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG")
	assert.Nil(t, err)
	assert.Equal(t, b02, bb02)

	B03 := acc.manager.GetCanonicalAccount(testAddr).GetBalance().String()
	b03 := addPoint(B03)
	bb03, err := acc.GetBalance(testAddr.String())
	assert.Nil(t, err)
	assert.Equal(t, b03, bb03)

	// get account api
	account01, err := acc.GetAccount("0x016ad4Fc7e1608685Bf5fe5573973BF2B1Ef9B8A")
	assert.Nil(t, err)
	assert.Equal(t, acc.manager.GetCanonicalAccount(common.HexToAddress("0x016ad4Fc7e1608685Bf5fe5573973BF2B1Ef9B8A")), account01)

}

// TestChainAPI_api chain api test
func TestChainAPI_api(t *testing.T) {
	bc := newChain()
	c := NewChainAPI(bc)

	// getBlockByHash
	exBlock1 := c.chain.GetBlockByHash(common.HexToHash("0x3f4c3152fb02a7673bf804b1ddeb75542b6ef9a5a87501d9cfbbcf6c3632a211"))
	assert.Equal(t, exBlock1, c.GetBlockByHash("0x3f4c3152fb02a7673bf804b1ddeb75542b6ef9a5a87501d9cfbbcf6c3632a211", true))
	exBlock1.SetTxs([]*types.Transaction{}) // set block txs to null
	assert.Equal(t, exBlock1, c.GetBlockByHash("0x3f4c3152fb02a7673bf804b1ddeb75542b6ef9a5a87501d9cfbbcf6c3632a211", false))

	// getBlockByHeight
	exBlock2 := c.chain.GetBlockByHeight(1)
	assert.Equal(t, exBlock2, c.GetBlockByHeight(1, true))
	exBlock2.SetTxs([]*types.Transaction{}) // set block txs to null
	assert.Equal(t, exBlock2, c.GetBlockByHeight(1, false))

	// get chain ID api
	assert.Equal(t, strconv.Itoa(int(c.chain.ChainID())), c.ChainID())

	// get genesis block api
	assert.Equal(t, c.chain.Genesis(), c.Genesis())

	// get current block api
	curBlock := c.chain.CurrentBlock()
	assert.Equal(t, curBlock, c.CurrentBlock(true))
	curBlock.SetTxs([]*types.Transaction{}) // set block txs to null
	assert.Equal(t, curBlock, c.CurrentBlock(false))

	// get stable block api
	StaBlock := c.chain.StableBlock()
	assert.Equal(t, StaBlock, c.LatestStableBlock(true))
	StaBlock.SetTxs([]*types.Transaction{}) // set block txs to null
	assert.Equal(t, StaBlock, c.LatestStableBlock(false))

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
	txCh := make(chan types.Transactions, 100)
	pool := chain.NewTxPool(nil, txCh)
	txAPI := NewTxAPI(pool)

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
// 	assert.Equal(t, "0x015780F8456F9c1532645087a19DcF9a7e0c7F97", m.LemoBase())
//
// }
//
