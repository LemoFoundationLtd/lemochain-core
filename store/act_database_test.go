package store

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestPatriciaTrie_Min(t *testing.T) {
	val := min(10, 11)
	assert.Equal(t, val, 10)

	val = min(10, 9, 10, 11)
	assert.Equal(t, val, 9)

	val = min(10, 10, 9)
	assert.Equal(t, val, 9)
}

func TestPatriciaTrie_Max(t *testing.T) {
	val := max(10, 11)
	assert.Equal(t, val, 11)

	val = max(10, 9, 10, 11)
	assert.Equal(t, val, 11)

	val = max(10, 10, 9)
	assert.Equal(t, val, 10)

	val = max(10)
	assert.Equal(t, val, 10)
}

func TestPatriciaTrie_Substring(t *testing.T) {
	assert.Equal(t, "", substring("", 0, 0))

	str := "123456789987654321"

	val := substring(str, 0, -1)
	assert.Equal(t, "", val)

	val = substring(str, 0, 0)
	assert.Equal(t, "", val)

	val = substring(str, 0, 1)
	assert.Equal(t, "1", val)

	val = substring(str, 0, len(str))
	assert.Equal(t, str, val)

	val = substring(str, 0, len(str)+1)
	assert.Equal(t, str, val)

	val = substring(str, -2, len(str)+1)
	assert.Equal(t, "321", val)

	val = substring(str, len(str)-1-5, len(str))
	assert.Equal(t, "654321", val)
}

func TestPatriciaNode_Insert(t *testing.T) {
	node := &PatriciaNode{dye: 0}

	nodes := insert(nil, 0, node)
	assert.Equal(t, node.dye, nodes[0].dye)

	node = &PatriciaNode{dye: 1}
	nodes = insert(nodes, 1, node)
	assert.Equal(t, node.dye, nodes[1].dye)

	node = &PatriciaNode{dye: 2}
	nodes = insert(nodes, 2, node)
	assert.Equal(t, node.dye, nodes[2].dye)

	node = &PatriciaNode{dye: 3}
	nodes = insert(nodes, 1, node)
	assert.Equal(t, node.dye, nodes[1].dye)

	node = &PatriciaNode{dye: 4}
	nodes = insert(nodes, 0, node)
	assert.Equal(t, node.dye, nodes[0].dye)
}

type TestReader struct {
}

func (reader *TestReader) Get(key []byte) (value []byte, err error) {
	return nil, nil
}

// Has retrieves whether a key is present in the database.
func (reader *TestReader) Has(key []byte) (bool, error) {
	return false, nil
}

func TestPatriciaTrie_Put1(t *testing.T) {
	trie := NewEmptyDatabase(new(TestReader))

	account1 := &types.AccountData{
		Address: common.HexToAddress("0x1"),
		Balance: big.NewInt(1),
	}
	account2 := &types.AccountData{
		Address: common.HexToAddress("0x2"),
		Balance: big.NewInt(2),
	}
	trie.Insert(account1.Address[:], account1)
	trie.Insert(account2.Address[:], account2)

	tmp1 := NewActDatabase(new(TestReader), trie)
	account3 := &types.AccountData{
		Address: common.HexToAddress("0x11"),
		Balance: big.NewInt(3),
	}
	tmp1.Put(account3, 1)

	account4 := &types.AccountData{
		Address: common.HexToAddress("0x400000000000"),
		Balance: big.NewInt(400000000),
	}
	tmp1.Put(account4, 1)
}

func TestPatriciaTrie_Put(t *testing.T) {
	trie := NewEmptyDatabase(new(TestReader))

	account1 := &types.AccountData{
		Address: common.HexToAddress("0x1"),
		Balance: big.NewInt(1),
	}
	account2 := &types.AccountData{
		Address: common.HexToAddress("0x2"),
		Balance: big.NewInt(2),
	}
	trie.Insert(account1.Address[:], account1)
	trie.Insert(account2.Address[:], account2)

	tmp1 := trie.Clone()
	account3 := &types.AccountData{
		Address: common.HexToAddress("0x3"),
		Balance: big.NewInt(3),
	}
	tmp1.Put(account3, 1)

	account4 := &types.AccountData{
		Address: common.HexToAddress("0x400000000000"),
		Balance: big.NewInt(400000000),
	}
	tmp1.Put(account4, 1)

	account5 := &types.AccountData{
		Address: common.HexToAddress("0x500000000000"),
		Balance: big.NewInt(500000000),
	}
	tmp1.Put(account5, 1)

	result := trie.Find(account3.Address[:])
	assert.Nil(t, result)

	result = tmp1.Find(account3.Address[:])
	assert.Equal(t, result.Address, common.HexToAddress("0x3"))

	result = trie.Find(account5.Address[:])
	assert.Nil(t, result)

	result = tmp1.Find(account5.Address[:])
	assert.Equal(t, result.Address, common.HexToAddress("0x500000000000"))

	tmp2 := tmp1.Clone()
	account6 := &types.AccountData{
		Address: common.HexToAddress("0x600000000000"),
		Balance: big.NewInt(600000000),
	}
	tmp2.Put(account6, 2)

	result = trie.Find(account6.Address[:])
	assert.Nil(t, result)

	result = tmp1.Find(account6.Address[:])
	assert.Nil(t, result)

	result = tmp2.Find(account6.Address[:])
	assert.Equal(t, account6.Address, result.Address)
}
