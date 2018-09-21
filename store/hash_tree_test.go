package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTree_Add(t *testing.T) {
	tree, err := NewTree()
	assert.NoError(t, err)

	// set
	key1, pos1 := NewKey1()
	err = tree.Add(key1, pos1)
	assert.NoError(t, err)

	// get
	item, err := tree.Get(key1)
	assert.NoError(t, err)
	assert.Equal(t, item.Pos, pos1)

	// update
	err = tree.Add(key1, 5000)
	assert.NoError(t, err)

	item, err = tree.Get(key1)
	assert.NoError(t, err)
	assert.Equal(t, item.Pos, uint32(5000))
}

func TestTree_Split(t *testing.T) {
	tree, err := NewTree()
	assert.NoError(t, err)

	testCnt := 256 * 256
	key1, _ := NewKey1()
	keys, err := CreateBufWithNumberBatch(testCnt, key1)
	assert.NoError(t, err)
	for index := 0; index < testCnt; index++ {
		key := keys[index]
		pos := uint32(index)
		err = tree.Add(key, pos)
		assert.NoError(t, err)

		item, err := tree.Get(key)
		assert.NoError(t, err)
		assert.NoError(t, err)
		assert.Equal(t, item.Pos, pos)
	}
}

func BenchmarkTree_Add(b *testing.B) {
	tree, _ := NewTree()

	testCnt := 256
	key1, _ := NewKey1()
	keys, _ := CreateBufWithNumberBatch(testCnt, key1)
	for index := 0; index < testCnt; index++ {
		key := keys[index]
		pos := uint32(index)
		tree.Add(key, pos)
		tree.Get(key)
	}
}
