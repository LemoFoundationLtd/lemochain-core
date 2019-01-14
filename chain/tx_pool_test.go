package chain

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

// type TestManager
//
// func (am *Manager) GetAccount(address common.Address) types.AccountAccessor {

func CreateTx(to string, amount int64, gasPrice int64, expiration uint64) *types.Transaction {
	testSigner := types.MakeSigner()
	testPrivate, _ := crypto.HexToECDSA("432a86ab8765d82415a803e29864dcfc1ed93dac949abf6f95a583179f27e4bb") // secp256k1.V = 1

	address := common.HexToAddress(to)

	bigAmount := new(big.Int)
	bigAmount.SetInt64(amount)

	gasLimit := uint64(2500000)

	bigGasPrice := new(big.Int)
	bigGasPrice.SetInt64(gasPrice)

	tx := types.NewTransaction(address, bigAmount, gasLimit, bigGasPrice, nil, 100, expiration, "paul xie", "")
	txV, err := types.SignTx(tx, testSigner, testPrivate)

	if err != nil {
		return nil
	} else {
		return txV
	}
}

func TestTxPool_AddTx(t *testing.T) {
	// store.ClearData()

	// db, err := store.NewCacheChain("../../testdata/db_account")
	// if err != nil {
	// 	panic(err)
	// }
	//
	// am := account.NewManager(common.Hash{}, db)
	pool := NewTxPool(100)
	tx := CreateTx("0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 1000, 2000, 3000)

	err := pool.AddTx(tx)
	assert.NoError(t, err)

	err = pool.AddTx(tx)
	assert.NoError(t, err)

	pending := pool.Pending(100)
	assert.Equal(t, 1, len(pending))
}

func TestTxPool_Pending(t *testing.T) {
	// txCh := make(chan types.Transactions, 100)
	pool := NewTxPool(100)

	// is not exist
	tx := CreateTx("0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 1000, 2000, 3000)
	err := pool.AddTx(tx)
	assert.NoError(t, err)

	// is not exist
	tx = CreateTx("0x2d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 4000, 5000, 6000)
	err = pool.AddTx(tx)
	assert.NoError(t, err)

	//time.Sleep(time.Duration(10) * time.Second)

	// is not exist
	tx = CreateTx("0x3d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 7000, 8000, 9000)
	err = pool.AddTx(tx)
	assert.NoError(t, err)

	// exist
	tx = CreateTx("0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 1000, 2000, 3000)
	err = pool.AddTx(tx)
	assert.NoError(t, err)

	// exist
	tx = CreateTx("0x2d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 4000, 5000, 6000)
	err = pool.AddTx(tx)
	assert.NoError(t, err)

	//time.Sleep(time.Duration(10) * time.Second)

	tx = CreateTx("0x3d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 7000, 8000, 9000)
	err = pool.AddTx(tx)
	assert.NoError(t, err)

	result := pool.Pending(10)
	assert.Equal(t, 3, len(result))

	tx = CreateTx("0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 1000, 2000, 3000)
	err = pool.AddTx(tx)
	assert.NoError(t, err)

	tx = CreateTx("0x2d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 4000, 5000, 6000)
	err = pool.AddTx(tx)
	assert.NoError(t, err)

	// tx = CreateTx("0x3d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 7000, 8000, 9000)
	// err = pool.AddTx(tx)
	// assert.NoError(t, err)

	keys := []common.Hash{tx.Hash()}
	pool.Remove(keys)

	result = pool.Pending(10)
	assert.Equal(t, 2, len(result))
}

func TestTxPool_Remove(t *testing.T) {
	pool := NewTxPool(100)

	tx1 := CreateTx("0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 1000, 2000, 3000)
	err := pool.AddTx(tx1)
	assert.NoError(t, err)

	tx2 := CreateTx("0x2d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 2000, 5000, 6000)
	err = pool.AddTx(tx2)
	assert.NoError(t, err)

	tx3 := CreateTx("0x3d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 3000, 8000, 9000)
	err = pool.AddTx(tx3)
	assert.NoError(t, err)

	tx4 := CreateTx("0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 4000, 2000, 3000)
	err = pool.AddTx(tx4)
	assert.NoError(t, err)

	tx5 := CreateTx("0x2d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 5000, 5000, 6000)
	err = pool.AddTx(tx5)
	assert.NoError(t, err)

	tx6 := CreateTx("0x3d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 6001, 8000, 9000)
	err = pool.AddTx(tx6)
	assert.NoError(t, err)

	tx7 := CreateTx("0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 7002, 2000, 3000)
	err = pool.AddTx(tx7)
	assert.NoError(t, err)

	tx8 := CreateTx("0x2d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 8003, 5000, 6000)
	err = pool.AddTx(tx8)
	assert.NoError(t, err)

	keys := []common.Hash{tx2.Hash()}
	pool.Remove(keys)
	assert.Equal(t, 8, pool.txsCache.len())

	result := pool.Pending(3)
	assert.Equal(t, 3, len(result))
	assert.Equal(t, 8, pool.txsCache.len())

	keys = []common.Hash{tx1.Hash(), tx3.Hash(), tx4.Hash()}
	pool.Remove(keys)
	assert.Equal(t, 8, pool.txsCache.len())

	result = pool.Pending(10)
	assert.Equal(t, 4, len(result))
	assert.Equal(t, 8, pool.txsCache.len())

	keys = []common.Hash{tx1.Hash(), tx2.Hash(), tx3.Hash(), tx4.Hash(), tx5.Hash(), tx6.Hash(), tx7.Hash(), tx8.Hash()}
	pool.Remove(keys)
	assert.Equal(t, 8, pool.txsCache.len())

	result = pool.Pending(10)
	assert.Equal(t, 0, len(result))
	assert.Equal(t, 0, pool.txsCache.len())
}
