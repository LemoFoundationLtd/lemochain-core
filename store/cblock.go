package store

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
)

type CBlock struct {
	Block  *types.Block
	TrieDB *AccountTrieDB
	Top30  []*Candidate
}

func (block *CBlock) SetTop(src []*Candidate) {
	if len(src) <= 0 {
		block.Top30 = make([]*Candidate, 0)
	} else {

	}
}
