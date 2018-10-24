package node

import (
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/stretchr/testify/assert"
	"math/big"
	"strconv"
	"testing"
	"time"
)

// Test_AddPoint test AddPoint function
func Test_AddPoint(t *testing.T) {
	test01 := "11111111111111111111"
	test02 := "111111111111"
	test03 := "111111111111111111111111111111111111111111111111"
	assert.Equal(t, "11.111111111111111111", AddPoint(test01))
	assert.Equal(t, "0.000000111111111111", AddPoint(test02))
	assert.Equal(t, "Warning : Your account balance is abnormal. Please stop any operation.", AddPoint(test03))
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
	B01 := acc.manager.GetCanonicalAccount(common.HexToAddress("0x10000")).GetBalance().String()
	assert.Equal(t, AddPoint(B01), acc.GetBalance("0x10000"))

	B02 := acc.manager.GetCanonicalAccount(crypto.RestoreOriginalAddress("Lemo20000")).GetBalance().String()
	assert.Equal(t, AddPoint(B02), acc.GetBalance("Lemo20000"))

	B03 := acc.manager.GetCanonicalAccount(testAddr).GetBalance().String()
	assert.Equal(t, AddPoint(B03), acc.GetBalance(testAddr.String()))

	// get account api
	assert.Equal(t, acc.manager.GetCanonicalAccount(common.HexToAddress("0x10000")), acc.GetAccount("0x10000"))
	assert.Equal(t, acc.manager.GetCanonicalAccount(common.HexToAddress("0x20000")), acc.GetAccount("0x20000"))
	assert.Equal(t, acc.manager.GetCanonicalAccount(common.HexToAddress(testAddr.String())), acc.GetAccount(testAddr.String()))

}

// TestChainAPI_api chain api test
func TestChainAPI_api(t *testing.T) {
	bc := newChain()
	c := NewChainAPI(bc)

	// getBlockByHash
	exBlock1 := c.chain.GetBlockByHash(common.HexToHash("0x5afb6907e01a243325ce7c6e56e463f777080f6e5277ba2ec83928329c8dce61"))
	assert.Equal(t, exBlock1, c.GetBlockByHash("0x5afb6907e01a243325ce7c6e56e463f777080f6e5277ba2ec83928329c8dce61", true))
	exBlock1.Txs = []*types.Transaction{} // set block txs to null
	assert.Equal(t, exBlock1, c.GetBlockByHash("0x5afb6907e01a243325ce7c6e56e463f777080f6e5277ba2ec83928329c8dce61", false))

	// getBlockByHeight
	exBlock2 := c.chain.GetBlockByHeight(1)
	assert.Equal(t, exBlock2, c.GetBlockByHeight(1, true))
	exBlock2.Txs = []*types.Transaction{} // set block txs to null
	assert.Equal(t, exBlock2, c.GetBlockByHeight(1, false))

	// get chain ID api
	assert.Equal(t, strconv.Itoa(int(c.chain.ChainID())), c.ChainID())

	// get genesis block api
	assert.Equal(t, c.chain.Genesis(), c.Genesis())

	// get current block api
	curBlock := c.chain.CurrentBlock()
	assert.Equal(t, curBlock, c.CurrentBlock(true))
	curBlock.Txs = []*types.Transaction{} // set block txs to null
	assert.Equal(t, curBlock, c.CurrentBlock(false))

	// get stable block api
	StaBlock := c.chain.StableBlock()
	assert.Equal(t, StaBlock, c.LatestStableBlock(true))
	StaBlock.Txs = []*types.Transaction{} // set block txs to null
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
	testTx := types.NewTransaction(common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 200, big.NewInt(time.Now().Unix()+230000), "aa", nil)
	signTxs := signTransaction(testTx, testPrivate)
	txCh := make(chan types.Transactions, 100)
	pool := chain.NewTxPool(nil, txCh)
	txAPI := NewTxAPI(pool)

	tx, err := json.Marshal(signTxs)
	assert.Nil(t, err)

	sendTxHash, err := txAPI.SendTx(string(tx))
	assert.Nil(t, err)

	assert.Equal(t, signTxs.Hash(), sendTxHash)
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
