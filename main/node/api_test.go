package node

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/testify/assert"
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
	// get version api
	t.Log(acc.GetVersion(testAddr.String(), 0))
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
	t.Log(c.GetBlock("0x16019ad7c4d4ecf5163906339048ac73a7aa7131b1154fefeb865c0d523d23f5"))
	t.Log(c.GetBlock(0.0)) // input type must float64,this is a bug.// todo
	t.Log(c.GetBlock("0xd67857de0f447554c94712d9c0016a8d9e4974d6c3b14b9b062226637d968449"))
	t.Log(c.GetBlock(1.0))
	t.Log(c.GetBlock("0x5afb6907e01a243325ce7c6e56e463f777080f6e5277ba2ec83928329c8dce61"))
	t.Log(c.GetBlock(2.0))
	t.Log(c.GetBlock("0x1889ca33d2ea9bfe68b171258e19f3034e9518c47d15b1d484797458e96cfb96")) // block03 did not insert db
	t.Log(c.GetBlock(3.0))
	// get chain ID api
	t.Log(c.GetChainID())
	// get genesis block api
	t.Log(c.GetGenesis())
	// get current block api
	t.Log(c.GetCurrentBlock())
	// get stable block api
	t.Log(c.GetLatestStableBlock())
	// get current chain height api
	t.Log(c.GetCurrentHeight())

}

// TestTxAPI_SendTx send tx api test
func TestTxAPI_SendTx(t *testing.T) {
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
