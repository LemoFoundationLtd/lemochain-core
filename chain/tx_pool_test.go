package chain

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

func CreateTx(to string, amount int64, gasPrice int64, expiration int64) *types.Transaction {
	address := common.HexToAddress(to)

	bigAmount := new(big.Int)
	bigAmount.SetInt64(amount)

	gasLimit := uint64(2500000)

	bigGasPrice := new(big.Int)
	bigGasPrice.SetInt64(gasPrice)

	bigExpiration := new(big.Int)
	bigExpiration.SetInt64(expiration)

	return types.NewTransaction(address, bigAmount, gasLimit, bigGasPrice, nil, 0, bigExpiration, "paul xie", nil)
}

func TestTxPool_AddTx(t *testing.T) {
	pool := NewTxPool()
	tx := CreateTx("0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 1000, 2000, 3000)

	err := pool.AddTx(tx)
	assert.NoError(t, err)

	err = pool.AddTx(tx)
	assert.NoError(t, err)

	pending := pool.Pending(100)
	assert.Equal(t, 1, len(pending))
}

func TestTxPool_Pending(t *testing.T) {
	pool := NewTxPool()

	// is not exist
	tx := CreateTx("0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 1000, 2000, 3000)
	err := pool.AddTx(tx)
	assert.NoError(t, err)

	// is not exist
	tx = CreateTx("0x2d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e", 4000, 5000, 6000)
	err = pool.AddTx(tx)
	assert.NoError(t, err)

	time.Sleep(time.Duration(10) * time.Second)

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

	time.Sleep(time.Duration(10) * time.Second)

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

	result = pool.Pending(10)
	assert.Equal(t, 0, len(result))
}
