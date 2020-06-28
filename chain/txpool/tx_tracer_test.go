package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/LemoFoundationLtd/lemochain-core/common"
)

func TestTxTracer_AddTrace(t *testing.T) {
	tracer := make(TxTracer)

	tracer.AddTrace(nil, common.HexToHash("1"))

	// add 1 tx
	tx1 := makeTx(101, 0)
	bHash1 := common.HexToHash("1")
	tracer.AddTrace(tx1, bHash1) // now tx1:[b1]
	assert.Equal(t, 1, len(tracer))
	assert.Equal(t, 1, len(tracer[tx1.Hash()].Collect()))
	assert.Equal(t, bHash1, tracer[tx1.Hash()].Collect()[0])
	// add another tx
	tx2 := makeTx(102, 0)
	bHash2 := common.HexToHash("2")
	tracer.AddTrace(tx2, bHash2) // now tx1:[b1], tx2:[b2]
	assert.Equal(t, 2, len(tracer))
	assert.Equal(t, 1, len(tracer[tx2.Hash()].Collect()))
	assert.Equal(t, bHash2, tracer[tx2.Hash()].Collect()[0])
	// add same tx for different block
	tracer.AddTrace(tx1, bHash2) // now tx1:[b1,b2], tx2:[b2]
	assert.Equal(t, 2, len(tracer))
	assert.Equal(t, 2, len(tracer[tx1.Hash()].Collect()))
	assert.Equal(t, true, tracer[tx1.Hash()].Has(bHash1))
	assert.Equal(t, true, tracer[tx1.Hash()].Has(bHash2))
	// add same tx for same block
	tracer.AddTrace(tx1, bHash1) // now tx1:[b1,b2], tx2:[b2]
	assert.Equal(t, 2, len(tracer))
	assert.Equal(t, 2, len(tracer[tx1.Hash()].Collect()))

	// add box tx
	tx3 := makeTx(103, 0)
	tx4 := makeBoxTx(104, 0, tx1, tx2, tx3)
	bHash3 := common.HexToHash("3")
	// one tx would not appear twice in a block. so we should add it tu another block
	tracer.AddTrace(tx4, bHash3) // now tx1:[b1,b2,b3], tx2:[b2,b3], tx3:[b3], tx4:[b3]
	assert.Equal(t, 4, len(tracer))
	assert.Equal(t, 3, len(tracer[tx1.Hash()].Collect()))
	assert.Equal(t, true, tracer[tx1.Hash()].Has(bHash3))
	assert.Equal(t, 2, len(tracer[tx2.Hash()].Collect()))
	assert.Equal(t, true, tracer[tx2.Hash()].Has(bHash3))
	assert.Equal(t, 1, len(tracer[tx3.Hash()].Collect()))
	assert.Equal(t, bHash3, tracer[tx3.Hash()].Collect()[0])
	assert.Equal(t, 1, len(tracer[tx4.Hash()].Collect()))
	assert.Equal(t, bHash3, tracer[tx4.Hash()].Collect()[0])
	// add sub tx in another block
	tracer.AddTrace(tx3, bHash2) // now tx1:[b1,b2,b3], tx2:[b2,b3], tx3:[b2,b3], tx4:[b3]
	assert.Equal(t, 4, len(tracer))
	assert.Equal(t, 2, len(tracer[tx3.Hash()].Collect()))
}

func TestTxTracer_DelTrace(t *testing.T) {
	tracer := make(TxTracer)

	// not exist
	tracer.DelTrace(nil)
	tracer.DelTrace(makeTx(0, 0))

	// del 1 tx
	tx1 := makeTx(101, 0)
	tx2 := makeTx(102, 0)
	bHash1 := common.HexToHash("1")
	bHash2 := common.HexToHash("2")
	tracer.AddTrace(tx1, bHash1) // now tx1:[b1]
	tracer.AddTrace(tx2, bHash2) // now tx1:[b1], tx2:[b2]
	tracer.DelTrace(tx1)
	assert.Equal(t, 1, len(tracer))
	assert.Equal(t, 1, len(tracer[tx2.Hash()].Collect()))

	// del box tx
	tracer = make(TxTracer)
	tx3 := makeTx(103, 0)
	tx4 := makeBoxTx(104, 0, tx1, tx2, tx3)
	bHash3 := common.HexToHash("3")
	tracer.AddTrace(tx1, bHash1) // now tx1:[b1]
	tracer.AddTrace(tx2, bHash2) // now tx1:[b1], tx2:[b2]
	tracer.AddTrace(tx4, bHash3) // now tx1:[b1,b3], tx2:[b2,b3], tx3:[b3], tx4:[b3]
	tracer.DelTrace(tx4)
	assert.Equal(t, 0, len(tracer))

	// del sub tx
	tracer = make(TxTracer)
	tracer.AddTrace(tx1, bHash1) // now tx1:[b1]
	tracer.AddTrace(tx2, bHash2) // now tx1:[b1], tx2:[b2]
	tracer.AddTrace(tx4, bHash3) // now tx1:[b1,b3], tx2:[b2,b3], tx3:[b3], tx4:[b3]
	tracer.DelTrace(tx1)
	assert.Equal(t, 3, len(tracer))
	assert.Equal(t, 2, len(tracer[tx2.Hash()].Collect()))
	assert.Equal(t, bHash3, tracer[tx3.Hash()].Collect()[0])
	assert.Equal(t, bHash3, tracer[tx4.Hash()].Collect()[0])
}

func TestTxTracer_LoadTraces(t *testing.T) {
	tracer := make(TxTracer)

	// not exist
	trace := tracer.LoadTraces(nil)
	assert.Equal(t, 0, len(trace))

	// init tracer
	tx1 := makeTx(101, 0)
	tx2 := makeTx(102, 0)
	tx3 := makeTx(103, 0)
	tx4 := makeBoxTx(104, 0, tx1, tx2, tx3)
	bHash1 := common.HexToHash("1")
	bHash2 := common.HexToHash("2")
	bHash3 := common.HexToHash("3")
	tracer.AddTrace(tx1, bHash1) // now tx1:[b1]
	tracer.AddTrace(tx2, bHash2) // now tx1:[b1], tx2:[b2]
	tracer.AddTrace(tx4, bHash3) // now tx1:[b1,b3], tx2:[b2,b3], tx3:[b3], tx4:[b3]

	// not exist
	trace = tracer.LoadTraces(types.Transactions{makeTx(0, 0)})
	assert.Equal(t, 0, len(trace))

	// load 1 tx
	trace = tracer.LoadTraces(types.Transactions{tx1, makeTx(0, 0)})
	assert.Equal(t, 2, len(trace))
	assert.Equal(t, true, trace.Has(bHash1))
	assert.Equal(t, true, trace.Has(bHash3))

	// load box tx
	trace = tracer.LoadTraces(types.Transactions{tx4})
	assert.Equal(t, 3, len(trace))
	assert.Equal(t, true, trace.Has(bHash1))
	assert.Equal(t, true, trace.Has(bHash2))
	assert.Equal(t, true, trace.Has(bHash3))

	// load sub tx
	trace = tracer.LoadTraces(types.Transactions{tx3})
	assert.Equal(t, 1, len(trace))
	assert.Equal(t, bHash3, trace.Collect()[0])
}
