package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHashSet_Add(t *testing.T) {
	set := make(HashSet)

	set.Add(common.HexToHash("0x01"))
	assert.Equal(t, true, set.Has(common.HexToHash("0x01")))

	set.Add(common.HexToHash("0x02"))
	assert.Equal(t, true, set.Has(common.HexToHash("0x02")))

	set.Add(common.HexToHash("0x03"))
	assert.Equal(t, true, set.Has(common.HexToHash("0x03")))

	set.Del(common.HexToHash("0x01"))
	assert.Equal(t, false, set.Has(common.HexToHash("0x01")))

	hashes := set.Collect()
	assert.Equal(t, 2, len(hashes))
}
