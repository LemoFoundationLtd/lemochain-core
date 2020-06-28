package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

// TxTracer use transaction hash to be the key, and the value is block hashes which the transaction appeared
type TxTracer map[common.Hash]HashSet

func (t TxTracer) AddTrace(tx *types.Transaction, blockHash common.Hash) {
	if tx == nil {
		return
	}

	t.addTrace(tx.Hash(), blockHash)

	// add sub transactions in box transaction
	for _, subTx := range getSubTxs(tx) {
		t.addTrace(subTx.Hash(), blockHash)
	}
}

func (t TxTracer) addTrace(txHash, blockHash common.Hash) {
	if _, ok := t[txHash]; !ok {
		t[txHash] = make(HashSet)
	}
	t[txHash].Add(blockHash)
}

func (t TxTracer) DelTrace(tx *types.Transaction) {
	if tx == nil {
		return
	}

	delete(t, tx.Hash())

	// delete sub transactions in box transaction
	// the sub transactions must be expired cause the box transaction is expired
	for _, subTx := range getSubTxs(tx) {
		delete(t, subTx.Hash())
	}
}

// loadTraces load the block hash list which txs appeared
func (t TxTracer) LoadTraces(txs types.Transactions) HashSet {
	trace := make(HashSet)
	for _, tx := range txs {
		if t, ok := t[tx.Hash()]; ok {
			trace.Merge(t)
		}
		// load sub transactions in box transaction
		for _, subTx := range getSubTxs(tx) {
			if t, ok := t[subTx.Hash()]; ok {
				trace.Merge(t)
			}
		}
	}
	return trace
}
