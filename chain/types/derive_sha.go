package types

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/merkle"
)

type Hashable interface {
	Hash() common.Hash
}

// DeriveTxsSha compute the root hash of transactions merkle trie
func DeriveTxsSha(rawList []*Transaction) common.Hash {
	leaves := make([]common.Hash, len(rawList))
	for i, item := range rawList {
		leaves[i] = item.Hash()
	}
	return merkle.New(leaves).Root()
}

// DeriveChangeLogsSha compute the root hash of changelogs merkle trie
func DeriveChangeLogsSha(rawList []*ChangeLog) common.Hash {
	leaves := make([]common.Hash, len(rawList))
	for i, item := range rawList {
		leaves[i] = item.Hash()
	}
	return merkle.New(leaves).Root()
}

// DeriveDeputyRootSha compute the root hash of deputy nodes merkle trie
func DeriveDeputyRootSha(rawList deputynode.DeputyNodes) common.Hash {
	leaves := make([]common.Hash, len(rawList))
	for i, item := range rawList {
		leaves[i] = item.Hash()
	}
	return merkle.New(leaves).Root()
}
