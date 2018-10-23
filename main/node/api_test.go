package node

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/flag"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/testify/assert"
	"gopkg.in/urfave/cli.v1"
	"math/big"
	"testing"
)

// TestAccountAPI_api account api test
func TestAccountAPI_api(t *testing.T) {
	db := newDB()
	am := account.NewManager(common.Hash{}, db)
	acc := NewAccountAPI(am)
	// Create key pair
	addressKeyPair, err := acc.NewKeyPair()
	if err != nil {
		t.Error(err)
	}
	t.Log(addressKeyPair)
	// getBalance api
	t.Log(acc.GetBalance("0x10000"))
	t.Log(acc.GetBalance("0x20000"))
	t.Log(acc.GetBalance(testAddr.String()))
	// get account api
	t.Log(acc.GetAccount("0x10000"))
	t.Log(acc.GetAccount("0x20000"))
	t.Log(acc.GetAccount(testAddr.String()))
}

// TestChainAPI_api chain api test
func TestChainAPI_api(t *testing.T) {
	bc := newChain()
	c := NewChainAPI(bc)
	// getBlock (via block height or block hash)
	t.Log(c.GetBlockByHash("0x16019ad7c4d4ecf5163906339048ac73a7aa7131b1154fefeb865c0d523d23f5", false))
	t.Log(c.GetBlockByHeight(0, true))
	t.Log(c.GetBlockByHash("0xd67857de0f447554c94712d9c0016a8d9e4974d6c3b14b9b062226637d968449", true))
	t.Log(c.GetBlockByHeight(1, false))
	t.Log(c.GetBlockByHash("0x5afb6907e01a243325ce7c6e56e463f777080f6e5277ba2ec83928329c8dce61", false))
	t.Log(c.GetBlockByHeight(2, false))
	t.Log(c.GetBlockByHash("0x1889ca33d2ea9bfe68b171258e19f3034e9518c47d15b1d484797458e96cfb96", false)) // block03 did not insert db
	t.Log(c.GetBlockByHeight(3, true))
	// get chain ID api
	t.Log(c.ChainID())
	// get genesis block api
	t.Log(c.Genesis())
	// get current block api
	t.Log(c.CurrentBlock(false))
	// get stable block api
	t.Log(c.LatestStableBlock(true))
	// get current chain height api
	t.Log(c.CurrentHeight())
	// get latest stable block height
	t.Log(c.LatestStableHeight())
	// get suggest gas price
	t.Log(c.GasPriceAdvice())
	// get nodeVersion
	t.Log(c.NodeVersion())

}

// TestTxAPI_api send tx api test
func TestTxAPI_api(t *testing.T) {
	testTx := types.NewTransaction(common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 200, big.NewInt(1544584596), "aa", []byte{34})
	txCh := make(chan types.Transactions, 100)
	pool := chain.NewTxPool(nil, txCh)
	txAPI := NewTxAPI(pool)
	encodeTx, err := rlp.EncodeToBytes(testTx)
	if err != nil {
		t.Error(t, err)
	}
	byteTx, err := txAPI.SendTx(encodeTx)
	if err != nil {
		t.Error(t, err)
	}

	assert.Equal(t, testTx.Hash(), byteTx)
}

// TestMineAPI_api miner api test // todo
func TestMineAPI_api(t *testing.T) {
	lemoConf := &LemoConfig{
		Genesis:   chain.DefaultGenesisBlock(),
		NetworkId: 1,
		MaxPeers:  1000,
		Port:      7001,
		NodeKey:   "0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa",
		ExtraData: []byte{},
	}
	testNode, err := New(lemoConf, &DefaultNodeConfig, flag.NewCmdFlags(&cli.Context{}, []cli.Flag{}))
	if err != nil {
		t.Error(err)
	}
	miner := (*testNode).miner
	m := NewMineAPI(miner)
	t.Log("after:", m.IsMining())
	m.MineStart()
	t.Log("then:", m.IsMining())
	// todo
	m.MineStop()
	t.Log("last:", m.IsMining())

	assert.Equal(t, "0x015780F8456F9c1532645087a19DcF9a7e0c7F97", m.LemoBase())

}
