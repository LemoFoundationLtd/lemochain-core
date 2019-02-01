package store

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

type TestReader struct {
}

func (reader *TestReader) Get(key []byte) (value []byte, err error) {
	return nil, nil
}

// Has retrieves whether a key is present in the database.
func (reader *TestReader) Has(key []byte) (bool, error) {
	return false, nil
}

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

// e.g. child = "e"(curNode = "abc")
//	"abc"		insert("c")		"abc"
//	/	\		   ====>  		/ |	\
// e	 f				   	   c# e  f
func TestPatriciaTrie_Put1(t *testing.T) {
	trie := NewEmptyDatabase(new(TestReader))

	account1 := &types.AccountData{
		Address: common.HexToAddress("0x1"),
		Balance: big.NewInt(1),
	}

	account12 := &types.AccountData{
		Address: common.HexToAddress("0x12"),
		Balance: big.NewInt(12),
	}

	account13 := &types.AccountData{
		Address: common.HexToAddress("0x13"),
		Balance: big.NewInt(13),
	}
	trie.Insert(account1.Address[:], account1)
	trie.Insert(account12.Address[:], account12)
	trie.Insert(account13.Address[:], account13)

	tmp1 := NewActDatabase(new(TestReader), trie)

	account11 := &types.AccountData{
		Address: common.HexToAddress("0x11"),
		Balance: big.NewInt(11),
	}
	tmp1.Put(account11, 1)

	account10 := &types.AccountData{
		Address: common.HexToAddress("0x10"),
		Balance: big.NewInt(10),
	}
	tmp1.Put(account10, 1)

	result := trie.Find(account10.Address[:])
	assert.Nil(t, result)
	result = tmp1.Find(account10.Address[:])
	assert.Equal(t, result.Address, account10.Address)

	result = trie.Find(account11.Address[:])
	assert.Nil(t, result)
	result = tmp1.Find(account11.Address[:])
	assert.Equal(t, result.Address, account11.Address)

	result = trie.Find(account1.Address[:])
	assert.Equal(t, result.Address, account1.Address)
	result = tmp1.Find(account1.Address[:])
	assert.Equal(t, result.Address, account1.Address)
	result = trie.Find(account12.Address[:])
	assert.Equal(t, result.Address, account12.Address)
	result = tmp1.Find(account12.Address[:])
	assert.Equal(t, result.Address, account12.Address)
	result = trie.Find(account13.Address[:])
	assert.Equal(t, result.Address, account13.Address)
	result = tmp1.Find(account13.Address[:])
	assert.Equal(t, result.Address, account13.Address)
}

// e.g. child = "ab"
// 	   ab    insert("ab")    ab#
//    /  \    =========>    /   \
//   e    f     		   e     f
func TestPatriciaTrie_Put2(t *testing.T) {
	trie := NewEmptyDatabase(new(TestReader))

	account12 := &types.AccountData{
		Address: common.HexToAddress("0x112"),
		Balance: big.NewInt(12),
	}

	account13 := &types.AccountData{
		Address: common.HexToAddress("0x113"),
		Balance: big.NewInt(13),
	}

	trie.Insert(account12.Address[:], account12)
	trie.Insert(account13.Address[:], account13)

	tmp1 := NewActDatabase(new(TestReader), trie)
	account1 := &types.AccountData{
		Address: common.HexToAddress("0x11"),
		Balance: big.NewInt(1),
	}

	str := "0x000000000000000000000000000000000000011"
	node := tmp1.put(tmp1.root, str, account1, 1)
	if node != nil {
		tmp1.root = node
	}
	result := trie.find(trie.root, str)
	assert.Nil(t, result)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account1)

	node = tmp1.put(tmp1.root, str, account1, 1)
	if node != nil {
		tmp1.root = node
	}
	result = trie.find(trie.root, str)
	assert.Nil(t, result)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account1)

	result = trie.Find(account12.Address[:])
	assert.Equal(t, result, account12)
	result = tmp1.Find(account12.Address[:])
	assert.Equal(t, result, account12)
	result = trie.Find(account13.Address[:])
	assert.Equal(t, result, account13)
	result = tmp1.Find(account13.Address[:])
	assert.Equal(t, result, account13)
}

// e.g. child = "ab"
// 	   ab    insert("ab")    ab#
//    /  \    =========>    /   \
//   e    f     		   e     f
func TestPatriciaTrie_Put3(t *testing.T) {
	trie := NewEmptyDatabase(new(TestReader))

	account12 := &types.AccountData{
		Address: common.HexToAddress("0x1112"),
		Balance: big.NewInt(12),
	}

	account13 := &types.AccountData{
		Address: common.HexToAddress("0x1113"),
		Balance: big.NewInt(13),
	}

	trie.Insert(account12.Address[:], account12)
	trie.Insert(account13.Address[:], account13)

	tmp1 := NewActDatabase(new(TestReader), trie)
	account1222 := &types.AccountData{
		Address: common.HexToAddress("0x1222"),
		Balance: big.NewInt(1222),
	}
	tmp1.Put(account1222, 1)
	result := trie.Find(account1222.Address[:])
	assert.Nil(t, result)
	result = tmp1.Find(account1222.Address[:])
	assert.Equal(t, result, account1222)

	account111 := &types.AccountData{
		Address: common.HexToAddress("0x111"),
		Balance: big.NewInt(1),
	}

	str := "0x000000000000000000000000000000000000111"
	node := tmp1.put(tmp1.root, str, account111, 1)
	if node != nil {
		tmp1.root = node
	}
	result = trie.find(trie.root, str)
	assert.Nil(t, result)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account111)

	result = trie.Find(account12.Address[:])
	assert.Equal(t, result, account12)
	result = tmp1.Find(account12.Address[:])
	assert.Equal(t, result, account12)
	result = trie.Find(account13.Address[:])
	assert.Equal(t, result, account13)
	result = tmp1.Find(account13.Address[:])
	assert.Equal(t, result, account13)
}

// 	e.g. child = "ab#"
// 	   ab#    insert("abc")   ab#
//    /  \    ==========>    / | \
//   e    f     			c# e  f
func TestPatriciaTrie_Put4(t *testing.T) {
	trie := NewEmptyDatabase(new(TestReader))

	account1112 := &types.AccountData{
		Address: common.HexToAddress("0x1112"),
		Balance: big.NewInt(12),
	}

	account1113 := &types.AccountData{
		Address: common.HexToAddress("0x1113"),
		Balance: big.NewInt(13),
	}

	account111 := &types.AccountData{
		Address: common.HexToAddress("0x111"),
		Balance: big.NewInt(1),
	}
	str := "0x000000000000000000000000000000000000111"

	trie.Insert(account1112.Address[:], account1112)
	trie.Insert(account1113.Address[:], account1113)
	trie.insert(trie.root, str, account111)

	account1111 := &types.AccountData{
		Address: common.HexToAddress("0x1111"),
		Balance: big.NewInt(1111),
	}
	tmp1 := NewActDatabase(new(TestReader), trie)
	tmp1.Put(account1111, 1)

	result := trie.Find(account1111.Address[:])
	assert.Nil(t, result)
	result = tmp1.Find(account1111.Address[:])
	assert.Equal(t, result, account1111)

	result = trie.Find(account1112.Address[:])
	assert.Equal(t, result, account1112)
	result = tmp1.Find(account1112.Address[:])
	assert.Equal(t, result, account1112)
	result = trie.Find(account1113.Address[:])
	assert.Equal(t, result, account1113)
	result = tmp1.Find(account1113.Address[:])
	assert.Equal(t, result, account1113)

	result = trie.find(trie.root, str)
	assert.Equal(t, result, account111)
	result = tmp1.find(trie.root, str)
	assert.Equal(t, result, account111)
}

// 	e.g. child = "ab#"
// 	   ab#    insert("abc")   ab#
//    /  \    ==========>    / | \
//   e    f     			c# e  f
func TestPatriciaTrie_Put5(t *testing.T) {
	trie := NewEmptyDatabase(new(TestReader))

	account1112 := &types.AccountData{
		Address: common.HexToAddress("0x1112"),
		Balance: big.NewInt(12),
	}

	account1113 := &types.AccountData{
		Address: common.HexToAddress("0x1113"),
		Balance: big.NewInt(13),
	}

	trie.Insert(account1112.Address[:], account1112)
	trie.Insert(account1113.Address[:], account1113)

	tmp1 := NewActDatabase(new(TestReader), trie)
	account111 := &types.AccountData{
		Address: common.HexToAddress("0x111"),
		Balance: big.NewInt(1),
	}
	str := "0x000000000000000000000000000000000000111"
	node := tmp1.put(tmp1.root, str, account111, 1)
	if node != nil {
		tmp1.root = node
	}
	result := trie.find(trie.root, str)
	assert.Nil(t, result)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account111)

	account1111 := &types.AccountData{
		Address: common.HexToAddress("0x1111"),
		Balance: big.NewInt(1111),
	}
	tmp1.Put(account1111, 1)

	result = trie.Find(account1111.Address[:])
	assert.Nil(t, result)
	result = tmp1.Find(account1111.Address[:])
	assert.Equal(t, result, account1111)

	result = trie.Find(account1112.Address[:])
	assert.Equal(t, result, account1112)
	result = tmp1.Find(account1112.Address[:])
	assert.Equal(t, result, account1112)
	result = trie.Find(account1113.Address[:])
	assert.Equal(t, result, account1113)
	result = tmp1.Find(account1113.Address[:])
	assert.Equal(t, result, account1113)
}

// 	e.g. child="abc#"
// 	   abc#      insert("ab")    ab#
//    /   \      =========>      /
//   e     f     			    c#
// 					           /  \
//                            e    f
func TestPatriciaTrie_Put6(t *testing.T) {
	trie := NewEmptyDatabase(new(TestReader))

	account1112 := &types.AccountData{
		Address: common.HexToAddress("0x1112"),
		Balance: big.NewInt(12),
	}

	account1113 := &types.AccountData{
		Address: common.HexToAddress("0x1113"),
		Balance: big.NewInt(13),
	}

	account111 := &types.AccountData{
		Address: common.HexToAddress("0x111"),
		Balance: big.NewInt(1),
	}
	str := "0x000000000000000000000000000000000000111"
	trie.insert(trie.root, str, account111)
	trie.Insert(account1112.Address[:], account1112)
	trie.Insert(account1113.Address[:], account1113)

	tmp1 := NewActDatabase(new(TestReader), trie)
	result := trie.find(trie.root, str)
	assert.Equal(t, result, account111)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account111)

	account11 := &types.AccountData{
		Address: common.HexToAddress("0x11"),
		Balance: big.NewInt(1),
	}
	str = "0x00000000000000000000000000000000000011"
	node := tmp1.put(tmp1.root, str, account11, 1)
	if node != nil {
		tmp1.root = node
	}
	result = trie.find(trie.root, str)
	assert.Nil(t, result)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account11)

	result = trie.Find(account1112.Address[:])
	assert.Equal(t, result, account1112)
	result = trie.Find(account1113.Address[:])
	assert.Equal(t, result, account1113)
	result = tmp1.Find(account1112.Address[:])
	assert.Equal(t, result, account1112)
	result = tmp1.Find(account1113.Address[:])
	assert.Equal(t, result, account1113)
}

// 	e.g. child="abc#"
// 	   abc#      insert("ab")    ab#
//    /   \      =========>      /
//   e     f     			    c#
// 					           /  \
//                            e    f
func TestPatriciaTrie_Put7(t *testing.T) {
	trie := NewEmptyDatabase(new(TestReader))

	account1112 := &types.AccountData{
		Address: common.HexToAddress("0x1112"),
		Balance: big.NewInt(12),
	}

	account1113 := &types.AccountData{
		Address: common.HexToAddress("0x1113"),
		Balance: big.NewInt(13),
	}

	trie.Insert(account1112.Address[:], account1112)
	trie.Insert(account1113.Address[:], account1113)

	tmp1 := NewActDatabase(new(TestReader), trie)
	account111 := &types.AccountData{
		Address: common.HexToAddress("0x111"),
		Balance: big.NewInt(1),
	}
	str := "0x000000000000000000000000000000000000111"
	node := tmp1.put(tmp1.root, str, account111, 1)
	if node != nil {
		tmp1.root = node
	}
	result := trie.find(trie.root, str)
	assert.Nil(t, result)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account111)

	account11 := &types.AccountData{
		Address: common.HexToAddress("0x11"),
		Balance: big.NewInt(1),
	}
	str = "0x00000000000000000000000000000000000011"
	node = tmp1.put(tmp1.root, str, account11, 1)
	if node != nil {
		tmp1.root = node
	}
	result = trie.find(trie.root, str)
	assert.Nil(t, result)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account11)

	result = trie.Find(account1112.Address[:])
	assert.Equal(t, result, account1112)
	result = trie.Find(account1113.Address[:])
	assert.Equal(t, result, account1113)
	result = tmp1.Find(account1112.Address[:])
	assert.Equal(t, result, account1112)
	result = tmp1.Find(account1113.Address[:])
	assert.Equal(t, result, account1113)
}

//	e.g. child="abc#"
//	   abc#    insert("abd")         ab
//    /   \     ==========>     	/  \
//   e     f   				       c#  d#
//                           	  /  \
//                          	 e    f
// split at j
func TestPatriciaTrie_Put8(t *testing.T) {
	trie := NewEmptyDatabase(new(TestReader))

	account1112 := &types.AccountData{
		Address: common.HexToAddress("0x1112"),
		Balance: big.NewInt(12),
	}

	account1113 := &types.AccountData{
		Address: common.HexToAddress("0x1113"),
		Balance: big.NewInt(13),
	}

	account111 := &types.AccountData{
		Address: common.HexToAddress("0x111"),
		Balance: big.NewInt(1),
	}
	str := "0x000000000000000000000000000000000000111"
	trie.insert(trie.root, str, account111)
	trie.Insert(account1112.Address[:], account1112)
	trie.Insert(account1113.Address[:], account1113)

	tmp1 := NewActDatabase(new(TestReader), trie)
	result := trie.find(trie.root, str)
	assert.Equal(t, result, account111)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account111)

	account112 := &types.AccountData{
		Address: common.HexToAddress("0x112"),
		Balance: big.NewInt(1),
	}
	str = "0x000000000000000000000000000000000000112"
	node := tmp1.put(tmp1.root, str, account112, 1)
	if node != nil {
		tmp1.root = node
	}

	result = trie.find(trie.root, str)
	assert.Nil(t, result)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account112)

	result = trie.Find(account1112.Address[:])
	assert.Equal(t, result, account1112)
	result = trie.Find(account1113.Address[:])
	assert.Equal(t, result, account1113)
	result = tmp1.Find(account1112.Address[:])
	assert.Equal(t, result, account1112)
	result = tmp1.Find(account1113.Address[:])
	assert.Equal(t, result, account1113)
}

//	e.g. child="abc#"
//	   abc#    insert("abd")         ab
//    /   \     ==========>     	/  \
//   e     f   				       1#  c#
//                           	  	   /  \
//                          	 	  e    f
// split at j
func TestPatriciaTrie_Put9(t *testing.T) {
	trie := NewEmptyDatabase(new(TestReader))

	account1112 := &types.AccountData{
		Address: common.HexToAddress("0x1112"),
		Balance: big.NewInt(12),
	}

	account1113 := &types.AccountData{
		Address: common.HexToAddress("0x1113"),
		Balance: big.NewInt(13),
	}

	account111 := &types.AccountData{
		Address: common.HexToAddress("0x111"),
		Balance: big.NewInt(1),
	}
	str := "0x000000000000000000000000000000000000111"
	trie.insert(trie.root, str, account111)
	trie.Insert(account1112.Address[:], account1112)
	trie.Insert(account1113.Address[:], account1113)

	tmp1 := NewActDatabase(new(TestReader), trie)
	result := trie.find(trie.root, str)
	assert.Equal(t, result, account111)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account111)

	account110 := &types.AccountData{
		Address: common.HexToAddress("0x110"),
		Balance: big.NewInt(1),
	}
	str = "0x000000000000000000000000000000000000110"
	node := tmp1.put(tmp1.root, str, account110, 1)
	if node != nil {
		tmp1.root = node
	}

	result = trie.find(trie.root, str)
	assert.Nil(t, result)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account110)

	result = trie.Find(account1112.Address[:])
	assert.Equal(t, result, account1112)
	result = trie.Find(account1113.Address[:])
	assert.Equal(t, result, account1113)
	result = tmp1.Find(account1112.Address[:])
	assert.Equal(t, result, account1112)
	result = tmp1.Find(account1113.Address[:])
	assert.Equal(t, result, account1113)
}

//	e.g. child="abc#"
//	   abc#    insert("abd")         ab
//    /   \     ==========>     	/  \
//   e     f   				       c#  d#
//                           	  /  \
//                          	 e    f
// split at j
func TestPatriciaTrie_Put10(t *testing.T) {
	trie := NewEmptyDatabase(new(TestReader))

	account1112 := &types.AccountData{
		Address: common.HexToAddress("0x1112"),
		Balance: big.NewInt(12),
	}

	account1113 := &types.AccountData{
		Address: common.HexToAddress("0x1113"),
		Balance: big.NewInt(13),
	}

	trie.Insert(account1112.Address[:], account1112)
	trie.Insert(account1113.Address[:], account1113)

	tmp1 := NewActDatabase(new(TestReader), trie)
	account111 := &types.AccountData{
		Address: common.HexToAddress("0x111"),
		Balance: big.NewInt(1),
	}
	str := "0x000000000000000000000000000000000000111"
	node := tmp1.put(tmp1.root, str, account111, 1)
	if node != nil {
		tmp1.root = node
	}

	result := trie.find(trie.root, str)
	assert.Nil(t, result)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account111)

	account112 := &types.AccountData{
		Address: common.HexToAddress("0x112"),
		Balance: big.NewInt(1),
	}
	str = "0x000000000000000000000000000000000000112"
	node = tmp1.put(tmp1.root, str, account112, 1)
	if node != nil {
		tmp1.root = node
	}

	result = trie.find(trie.root, str)
	assert.Nil(t, result)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account112)

	result = trie.Find(account1112.Address[:])
	assert.Equal(t, result, account1112)
	result = trie.Find(account1113.Address[:])
	assert.Equal(t, result, account1113)
	result = tmp1.Find(account1112.Address[:])
	assert.Equal(t, result, account1112)
	result = tmp1.Find(account1113.Address[:])
	assert.Equal(t, result, account1113)
}

func TestPatriciaTrie_Put11(t *testing.T) {
	trie := NewEmptyDatabase(new(TestReader))
	tmp1 := NewActDatabase(new(TestReader), trie)
	account1112 := &types.AccountData{
		Address: common.HexToAddress("0x1112"),
		Balance: big.NewInt(12),
	}
	tmp1.Put(account1112, 1)

	result := trie.Find(account1112.Address[:])
	assert.Nil(t, result)

	result = tmp1.Find(account1112.Address[:])
	assert.Equal(t, result, account1112)
}

func TestPatriciaTrie_Put12(t *testing.T) {
	trie := NewEmptyDatabase(new(TestReader))

	tmp1 := NewActDatabase(new(TestReader), trie)
	account1112 := &types.AccountData{
		Address: common.HexToAddress("0x1112"),
		Balance: big.NewInt(12),
	}
	str := "1112"
	node := tmp1.put(tmp1.root, str, account1112, 1)
	if node != nil {
		tmp1.root = node
	}
	result := trie.find(trie.root, str)
	assert.Nil(t, result)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account1112)

	account2223 := &types.AccountData{
		Address: common.HexToAddress("0x2223"),
		Balance: big.NewInt(12),
	}
	str = "2223"
	node = tmp1.put(tmp1.root, str, account2223, 1)
	if node != nil {
		tmp1.root = node
	}

	result = trie.find(trie.root, str)
	assert.Nil(t, result)
	result = tmp1.find(tmp1.root, str)
	assert.Equal(t, result, account2223)
}

func TestPatriciaTrie_Collected(t *testing.T) {

	trie := NewEmptyDatabase(new(TestReader))
	addr01, _ := common.StringToAddress("Lemo8888888888888888888888888888884SD4Q6")
	addr02, _ := common.StringToAddress("Lemo836BQKCBZ8Z7B7N4G4N4SNGBT24ZZSJQD24D")
	addr03, _ := common.StringToAddress("Lemo8888888888888888888888885PKQARFNQWYR")
	addr04, _ := common.StringToAddress("Lemo83W59DHT7FD4KSB3HWRJ5T4JD82TZW27ZKHJ")
	addr05, _ := common.StringToAddress("Lemo83F96RQR3J5GW8CS35JWP2A4QBQ3CYHHQJAK")
	addr06, _ := common.StringToAddress("Lemo843A8K22PDK9BSZT8SDN95GASSRSDW2DJZ3S")
	account01 := &types.AccountData{
		Address: addr01,
		Balance: big.NewInt(12),
	}

	account02 := &types.AccountData{
		Address: addr02,
		Balance: big.NewInt(13),
	}

	account03 := &types.AccountData{
		Address: addr03,
		Balance: big.NewInt(13),
	}

	account04 := &types.AccountData{
		Address: addr04,
		Balance: big.NewInt(13),
	}

	account05 := &types.AccountData{
		Address: addr05,
		Balance: big.NewInt(13),
	}

	account06 := &types.AccountData{
		Address: addr06,
		Balance: big.NewInt(13),
	}
	trie.Put(account01, 2)
	trie.Put(account02, 2)
	trie.Put(account03, 2)
	trie.Put(account04, 2)
	trie.Put(account05, 2)
	trie.Put(account06, 2)
	result := trie.Collected(2)
	assert.Equal(t, 6, len(result))

	addr1, _ := common.StringToAddress("Lemo8888888888888888888888885PKQARFNQWYR")
	addr2, _ := common.StringToAddress("Lemo8888888888888888888888888888884SD4Q6")
	addr3, _ := common.StringToAddress("Lemo83W59DHT7FD4KSB3HWRJ5T4JD82TZW27ZKHJ")
	addr4, _ := common.StringToAddress("Lemo836BQKCBZ8Z7B7N4G4N4SNGBT24ZZSJQD24D")
	addr5, _ := common.StringToAddress("Lemo83F96RQR3J5GW8CS35JWP2A4QBQ3CYHHQJAK")
	account1 := &types.AccountData{
		Address: addr1,
		Balance: big.NewInt(12),
	}

	account2 := &types.AccountData{
		Address: addr2,
		Balance: big.NewInt(13),
	}

	account3 := &types.AccountData{
		Address: addr3,
		Balance: big.NewInt(13),
	}

	account4 := &types.AccountData{
		Address: addr4,
		Balance: big.NewInt(13),
	}

	account5 := &types.AccountData{
		Address: addr5,
		Balance: big.NewInt(13),
	}

	tmp1 := NewActDatabase(new(TestReader), trie)
	tmp1.Put(account1, 3)
	tmp1.Put(account2, 3)
	tmp1.Put(account3, 3)
	tmp1.Put(account4, 3)
	tmp1.Put(account5, 3)

	result = tmp1.Collected(3)
	assert.Equal(t, 5, len(result))
}
