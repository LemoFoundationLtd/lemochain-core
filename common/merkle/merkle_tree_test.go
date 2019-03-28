package merkle

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	src = []common.Hash{
		common.HexToHash("0x5f30cc80133b9394156e24b233f0c4be32b24e44bb3381f02c7ba52619d0febc"),
		common.HexToHash("0xbdd637c523ed5c0eab792b986db18850c239a2e23802b36aff26bb68fb3fe008"),
		common.HexToHash("0x2ed873ae0dd2372777c57a685d907083ca97290e508f15478ca98bad3225ebbc"),
		common.HexToHash("0x1e4c3d286aada9b795029df6a2da044699943ea25a18adfa2995ad798dbb922b"),
		common.HexToHash("0x4b7471ea1795646eac817cf8a39e94a0f3a120e3affe4ad69240b97c74896c25"),
	}
)

func Test_HashNodes(t *testing.T) {
	// empty
	m := New([]common.Hash{})
	hashes := m.HashNodes()
	assert.Equal(t, 0, len(hashes))

	// 1 element
	m = New([]common.Hash{src[0]})
	hashes = m.HashNodes()
	assert.Equal(t, 1, len(hashes))

	// 5 elements
	m = New(src)
	hashes = m.HashNodes()
	assert.Equal(t, 9, len(hashes))
	// for _, hash := range hashes {
	// 	fmt.Println(common.ToHex(hash[:]))
	// }
}

func Test_Root(t *testing.T) {
	// empty
	m := New([]common.Hash{})
	root := m.Root()
	assert.Equal(t, EmptyTrieHash, root)

	// 1 element
	m = New([]common.Hash{src[0]})
	root = m.Root()
	assert.Equal(t, src[0], root)

	// 5 elements
	m = New(src)
	root = m.Root()
	assert.Equal(t, common.HexToHash("0x4755fbecb3a0ba3df2be677d35b309ea398bc10c08bbb89962c72e8c6b6cff2a"), root)
}

func Test_FindSiblingNodes(t *testing.T) {
	m := New(src)
	hashes := m.HashNodes()
	target := src[len(src)-1]
	sibling, err := FindSiblingNodes(target, hashes)
	assert.NoError(t, err)

	valid := Verify(target, m.Root(), sibling)
	assert.Equal(t, true, valid)
}
