package txpool

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/stretchr/testify/assert"
)

func TestTxPool_AddTx(t *testing.T) {
	pool := NewTxPool()

	// invalid tx
	err := pool.AddTx(nil)
	assert.Equal(t, ErrInvalidTx, err)

	// normal
	tx1 := makeTx(101, 0)
	err = pool.AddTx(tx1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pool.txs))
	assert.Equal(t, 1, len(pool.hashIndexMap))
	assert.Equal(t, 0, pool.hashIndexMap[tx1.Hash()])
	// big amount
	tx2 := makeTx(blockTradeAmount.Int64()+102, 0)
	err = pool.AddTx(tx2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(pool.txs))
	assert.Equal(t, 2, len(pool.hashIndexMap))
	assert.Equal(t, 0, pool.hashIndexMap[tx1.Hash()])
	assert.Equal(t, 1, pool.hashIndexMap[tx2.Hash()])
	// box
	subTx30 := makeTx(1030, 0)
	subTx31 := makeTx(1031, 0)
	tx3 := makeBoxTx(103, 0, subTx30, subTx31)
	err = pool.AddTx(tx3)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(pool.txs))
	assert.Equal(t, 5, len(pool.hashIndexMap))
	assert.Equal(t, 0, pool.hashIndexMap[tx1.Hash()])
	assert.Equal(t, 1, pool.hashIndexMap[tx2.Hash()])
	assert.Equal(t, 2, pool.hashIndexMap[tx3.Hash()])
	assert.Equal(t, 2, pool.hashIndexMap[subTx30.Hash()])
	assert.Equal(t, 2, pool.hashIndexMap[subTx31.Hash()])

	// exist tx
	err = pool.AddTx(tx1)
	assert.Equal(t, ErrTxIsExist, err)
	// exist sub tx
	err = pool.AddTx(subTx30)
	assert.Equal(t, ErrTxIsExist, err)
	err = pool.AddTx(subTx31)
	assert.Equal(t, ErrTxIsExist, err)
	// exist box tx
	tx4 := makeBoxTx(104, 0, makeTx(1040, 0), subTx30)
	err = pool.AddTx(tx4)
	assert.Equal(t, ErrTxIsExist, err)

	// extend storage
	pool = NewTxPool()
	assert.Equal(t, defaultPoolCap, pool.cap)
	for i := 0; i < defaultPoolCap; i++ {
		err = pool.AddTx(makeTx(int64(i*10000), 0))
		assert.NoError(t, err)
		assert.Equal(t, defaultPoolCap, pool.cap)
		assert.Equal(t, i+1, len(pool.txs))
	}
	err = pool.AddTx(makeTx(0, 1))
	assert.NoError(t, err)
	assert.Equal(t, defaultPoolCap*2, pool.cap)
	assert.Equal(t, 1, len(pool.txs))
}

func TestTxPool_AddTxs1(t *testing.T) {
	pool := NewTxPool()

	successCount := pool.AddTxs(nil)
	assert.Equal(t, 0, successCount)

	tx1 := makeTx(101, 0)
	successCount = pool.AddTxs(types.Transactions{tx1})
	assert.Equal(t, 1, successCount)

	successCount = pool.AddTxs(types.Transactions{tx1, makeTx(1021, 0), makeTx(1022, 0)})
	assert.Equal(t, 2, successCount)
}

func TestTxPool_AddTxs2(t *testing.T) {
	type testInfo struct {
		txs types.Transactions
	}
	pool := NewTxPool()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// make 100 test cases
	var tests = make([]testInfo, 100)
	for i := 0; i < 100; i++ {
		// up to 100 txs in every case
		count := r.Intn(100)
		tests[i].txs = make(types.Transactions, count)
		for j := 0; j < count; j++ {
			tests[i].txs[j] = makeTx(int64(i*100+j), 0)
		}
	}

	for i, test := range tests {
		caseName := fmt.Sprintf("case %d. txs count=%d", i, len(test.txs))
		t.Run(caseName, func(t *testing.T) {
			test := test // capture range variable
			t.Parallel()

			successCount := pool.AddTxs(test.txs)
			assert.Equal(t, len(test.txs), successCount)
		})
	}
}

func TestTxPool_DelTxs(t *testing.T) {
	pool := NewTxPool()

	// invalid tx
	pool.DelTxs(types.Transactions{})
	pool.DelTxs(types.Transactions{nil})

	// success
	tx1 := makeTx(101, 0)
	tx2 := makeTx(102, 0)
	_ = pool.AddTxs(types.Transactions{tx1, tx2})
	pool.DelTxs(types.Transactions{tx1})
	assert.Equal(t, 2, len(pool.txs))
	assert.Equal(t, 1, len(pool.hashIndexMap))
	assert.Nil(t, pool.txs[0]) // delTx just put nil into txs
	assert.Equal(t, tx2, pool.txs[1])
	assert.Equal(t, 1, pool.hashIndexMap[tx2.Hash()])
	// not exist
	pool.DelTxs(types.Transactions{tx1})

	// some tx are not exist
	tx3 := makeTx(103, 0)
	_ = pool.AddTxs(types.Transactions{tx1}) // append txs. now the txs is [nil, tx2, tx1]
	assert.Equal(t, 3, len(pool.txs))
	assert.Equal(t, 2, len(pool.hashIndexMap))
	assert.Equal(t, tx1, pool.txs[2])
	pool.DelTxs(types.Transactions{tx1, tx3}) // now the txs is [nil, tx2, nil]
	assert.Equal(t, 3, len(pool.txs))
	assert.Equal(t, 1, len(pool.hashIndexMap))
	assert.Nil(t, pool.txs[2])
	assert.Equal(t, 1, pool.hashIndexMap[tx2.Hash()])

	// remove all txs and active gc logic
	pool.DelTxs(types.Transactions{tx1, tx2, tx3})
	assert.Equal(t, 0, len(pool.txs))
	assert.Equal(t, 0, len(pool.hashIndexMap))
	assert.Equal(t, defaultPoolCap, pool.cap)
	// decrease cap
	pool.cap = defaultPoolCap * 2
	pool.DelTxs(types.Transactions{nil})
	assert.Equal(t, defaultPoolCap*2-1, pool.cap)

	// box tx
	pool = NewTxPool()
	tx4 := makeTx(1040, 0)
	subTx50 := makeTx(1050, 0)
	subTx51 := makeTx(1051, 0)
	tx5 := makeBoxTx(105, 0, subTx50, subTx51)
	_ = pool.AddTx(tx4)
	_ = pool.AddTx(tx5)
	// delete sub tx
	pool.DelTxs(types.Transactions{subTx50})
	assert.Equal(t, 2, len(pool.txs))
	assert.Equal(t, 3, len(pool.hashIndexMap))
	assert.Equal(t, 0, pool.hashIndexMap[tx4.Hash()])
	assert.Equal(t, 1, pool.hashIndexMap[tx5.Hash()])
	assert.Equal(t, 1, pool.hashIndexMap[subTx51.Hash()])
	assert.Equal(t, tx4, pool.txs[0])
	assert.Equal(t, (*types.Transaction)(nil), pool.txs[1])
	// delete box tx
	pool.DelTxs(types.Transactions{tx5})
	assert.Equal(t, 2, len(pool.txs))
	assert.Equal(t, 1, len(pool.hashIndexMap))
	assert.Equal(t, 0, pool.hashIndexMap[tx4.Hash()])
	assert.Equal(t, tx4, pool.txs[0])
}

func TestTxPool_IsEmpty(t *testing.T) {
	pool := NewTxPool()

	assert.Equal(t, true, pool.IsEmpty())

	tx1 := makeTx(100, 0)
	err := pool.AddTx(tx1)
	assert.NoError(t, err)
	assert.Equal(t, false, pool.IsEmpty())
	pool.DelTxs(types.Transactions{tx1})
	assert.Equal(t, true, pool.IsEmpty())

	// box tx
	subTx20 := makeTx(1020, 0)
	subTx21 := makeTx(1021, 0)
	tx2 := makeBoxTx(102, 0, subTx20, subTx21)
	err = pool.AddTx(tx2)
	assert.NoError(t, err)
	assert.Equal(t, false, pool.IsEmpty())
	pool.DelTxs(types.Transactions{subTx20})
	assert.Equal(t, false, pool.IsEmpty())
	pool.DelTxs(types.Transactions{tx2})
	assert.Equal(t, true, pool.IsEmpty())
}

func TestTxPool_GetTxs(t *testing.T) {
	pool := NewTxPool()
	cur := uint32(time.Now().Unix())
	tx1 := makeTx(101, 0) // not expired
	tx2 := makeTx(102, 0)
	tx3 := makeTx(103, -1) // expired
	tx4 := makeTx(104, 1)

	// no tx
	txs := pool.GetTxs(cur, 10)
	assert.Equal(t, 0, len(txs))

	// 1 tx
	_ = pool.AddTxs(types.Transactions{tx1, tx2})
	txs = pool.GetTxs(cur, 1)
	assert.Equal(t, 1, len(txs))
	assert.Equal(t, tx1, txs[0])
	// 0 size
	txs = pool.GetTxs(cur, 0)
	assert.Equal(t, 0, len(txs))
	// 2 txs
	txs = pool.GetTxs(cur, 10)
	assert.Equal(t, 2, len(txs))
	assert.Equal(t, tx1, txs[0])
	assert.Equal(t, tx2, txs[1])

	// pool contains nil
	pool = NewTxPool()
	_ = pool.AddTxs(types.Transactions{tx1, tx2})
	pool.DelTxs(types.Transactions{tx1})
	txs = pool.GetTxs(cur, 10)
	assert.Equal(t, 1, len(txs))
	assert.Equal(t, tx2, txs[0])

	// pool contains expired tx
	pool = NewTxPool()
	_ = pool.AddTxs(types.Transactions{tx1, tx2, tx3, tx4})
	txs = pool.GetTxs(cur, 10)
	assert.Equal(t, 3, len(txs))
	assert.Equal(t, tx1, txs[0])
	assert.Equal(t, tx2, txs[1])
	assert.Equal(t, tx4, txs[2])
	assert.Equal(t, 3, len(pool.hashIndexMap))
	assert.Equal(t, (*types.Transaction)(nil), pool.txs[2])

	// some sub txs in box are expired
	pool = NewTxPool()
	boxTx1 := makeBoxTx(105, 0, tx1, tx3)
	boxTx2 := makeBoxTx(106, 0, tx2, tx4)
	_ = pool.AddTxs(types.Transactions{boxTx1, boxTx2})
	txs = pool.GetTxs(cur, 10)
	assert.Equal(t, 1, len(txs))
	assert.Equal(t, boxTx2, txs[0])
	assert.Equal(t, 2, len(pool.txs))
	assert.Equal(t, (*types.Transaction)(nil), pool.txs[0])
}
