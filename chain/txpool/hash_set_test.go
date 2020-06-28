package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHashSet_Add_Del_Has(t *testing.T) {
	set := make(HashSet)

	assert.Equal(t, false, set.Has(common.HexToHash("1")))

	set.Del(common.HexToHash("1"))
	assert.Equal(t, false, set.Has(common.HexToHash("1")))

	set.Add(common.HexToHash("1"))
	assert.Equal(t, true, set.Has(common.HexToHash("1")))

	set.Add(common.HexToHash("2"))
	assert.Equal(t, true, set.Has(common.HexToHash("2")))

	set.Del(common.HexToHash("1"))
	assert.Equal(t, false, set.Has(common.HexToHash("1")))

	set.Add(common.HexToHash("1"))
	assert.Equal(t, true, set.Has(common.HexToHash("1")))
}

func TestHashSet_Collect(t *testing.T) {
	set := make(HashSet)

	hashes := set.Collect()
	assert.Equal(t, 0, len(hashes))

	set.Add(common.HexToHash("1"))
	hashes = set.Collect()
	assert.Equal(t, 1, len(hashes))
	assert.Equal(t, common.HexToHash("1"), hashes[0])

	set.Add(common.HexToHash("2"))
	hashes = set.Collect()
	assert.Equal(t, 2, len(hashes))

	set.Del(common.HexToHash("1"))
	assert.Equal(t, false, set.Has(common.HexToHash("1")))
	hashes = set.Collect()
	assert.Equal(t, 1, len(hashes))
	assert.Equal(t, common.HexToHash("2"), hashes[0])
}

func TestHashSet_Merge(t *testing.T) {
	set1 := make(HashSet)
	set2 := make(HashSet)

	set1.Merge(set2)
	hashes := set1.Collect()
	assert.Equal(t, 0, len(hashes))

	set2.Add(common.HexToHash("1"))
	set1.Merge(set2)
	hashes = set1.Collect()
	assert.Equal(t, 1, len(hashes))

	set2.Add(common.HexToHash("1"))
	set2.Add(common.HexToHash("2"))
	set1.Merge(set2)
	hashes = set1.Collect()
	assert.Equal(t, 2, len(hashes))
}
