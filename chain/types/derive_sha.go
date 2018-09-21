package types

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/merkle"
)

var emptyHash = crypto.Keccak256Hash(nil)

// DeriveTxsSha 计算交易的根HASH
func DeriveTxsSha(txs []*Transaction) common.Hash {
	if txs == nil || len(txs) == 0 {
		return emptyHash
	}
	leaves := make([]common.Hash, 0, len(txs))
	for _, tx := range txs {
		leaves = append(leaves, tx.Hash())
	}
	m := merkle.New(leaves)
	return m.Root()
}

// DeriveEventsSha 计算event的根HASH
func DeriveEventsSha(events []*Event) common.Hash {
	if events == nil || len(events) == 0 {
		return emptyHash
	}
	leaves := make([]common.Hash, 0, len(events))
	for _, event := range events {
		leaves = append(leaves, event.Hash())
	}
	m := merkle.New(leaves)
	return m.Root()
}

// DeriveChangeLogsSha 计算changelog的根HASH
func DeriveChangeLogsSha(logs []*ChangeLog) common.Hash {
	if logs == nil || len(logs) == 0 {
		return emptyHash
	}
	leaves := make([]common.Hash, 0, len(logs))
	for _, log := range logs {
		leaves = append(leaves, log.Hash())
	}
	m := merkle.New(leaves)
	return m.Root()
}
