package txpool

import "github.com/LemoFoundationLtd/lemochain-core/common"

// HashSet 是hash的集合，用map实现高效查询
type HashSet map[common.Hash]struct{}

func (set HashSet) Add(hash common.Hash) {
	set[hash] = struct{}{}
}

func (set HashSet) Del(hash common.Hash) {
	delete(set, hash)
}

func (set HashSet) Collect() []common.Hash {
	result := make([]common.Hash, len(set))
	i := 0
	for k, _ := range set {
		result[i] = k
		i++
	}
	return result
}

func (set HashSet) Has(hash common.Hash) bool {
	_, ok := set[hash]
	return ok
}
