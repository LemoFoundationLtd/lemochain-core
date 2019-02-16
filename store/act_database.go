package store

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
)

func min(first int, args ...int) int {
	for _, v := range args {
		if first > v {
			first = v
		}
	}
	return first
}

func max(first int, args ...int) int {
	for _, v := range args {
		if first < v {
			first = v
		}
	}
	return first
}

func substring(str string, start int, length int) string {
	rs := []rune(str)
	rl := len(rs)
	end := 0

	if start < 0 {
		start = rl - 1 + start
	}
	end = start + length

	if start > end {
		start, end = end, start
	}

	if start < 0 {
		start = 0
	}

	if start > rl {
		start = rl
	}

	if end < 0 {
		end = 0
	}

	if end > rl {
		end = rl
	}

	return string(rs[start:end])
}

type PatriciaNode struct {
	key      string
	dye      uint32
	data     types.NodeData
	terminal bool
	children []*PatriciaNode
}

func insert(nodes []*PatriciaNode, pos int, node *PatriciaNode) []*PatriciaNode {
	if (pos < 0) || (pos > len(nodes)) {
		panic("valid pos.")
	}

	if node == nil {
		panic("node == nil.")
	}

	if (nodes == nil) && (pos != 0) {
		panic("nodes == nil and pos != 0.")
	}

	if nodes == nil {
		nodes = make([]*PatriciaNode, 1)
		nodes[0] = node
		return nodes
	} else {
		nodes = append(nodes, node)
		for index := len(nodes) - 1; index > pos; index-- {
			nodes[index] = nodes[index-1]
		}
		nodes[pos] = node
		return nodes
	}
}

func (node *PatriciaNode) Clone() *PatriciaNode {
	result := &PatriciaNode{
		key:      node.key,
		dye:      node.dye,
		terminal: node.terminal,
	}

	if node.data != nil {
		result.data = node.data.Clone()
	}

	if len(node.children) > 0 {
		result.children = make([]*PatriciaNode, len(node.children))
		for index := 0; index < len(node.children); index++ {
			result.children[index] = node.children[index]
		}
	}

	return result
}

type PatriciaTrie struct {
	root   *PatriciaNode
	total  int
	minDye uint32
	maxDye uint32
}

func NewEmptyDatabase() *PatriciaTrie {
	return &PatriciaTrie{
		root:   new(PatriciaNode),
		total:  0,
		minDye: 0,
		maxDye: 0,
	}
}

func NewActDatabase(root *PatriciaTrie) *PatriciaTrie {
	return &PatriciaTrie{
		root:   root.root,
		total:  root.total,
		minDye: root.minDye,
		maxDye: root.maxDye,
	}
}

func (trie *PatriciaTrie) Clone() *PatriciaTrie {
	return &PatriciaTrie{
		root:   trie.root,
		total:  trie.total,
		minDye: trie.minDye,
		maxDye: trie.maxDye,
	}
}

func (trie *PatriciaTrie) Find(key string) types.NodeData {
	if len(key) <= 0 {
		return nil
	}

	return trie.find(trie.root, key)
}

func (trie *PatriciaTrie) find(curNode *PatriciaNode, key string) types.NodeData {
	for i := 0; i < len(curNode.children); i++ {
		child := curNode.children[i]

		length := min(len(child.key), len(key))
		j := 0
		for ; j < length; j++ {
			if key[j] != child.key[j] {
				break
			}
		}

		if j == 0 {
			if key[0] < child.key[0] {
				//	e.g. child="e", key="c" (currNode="abc")
				//	   abc
				//    /   \
				//   e     h
				return nil
			} else {
				continue
			}
		} else {
			if j == length {
				if len(key) == len(child.key) {
					if child.terminal {
						//	e.g. child="ab", key="ab"
						//	   ab#
						//    /
						//   f#
						return child.data
					} else {
						//	e.g. child="ab", key="ab"
						//	   ab
						//    /
						//   f#
						return nil
					}
				} else if len(key) > len(child.key) {
					//	e.g. child="ab#", key="abc"
					//	   ab#
					//    /   \
					//   a     c#
					sub := substring(key, j, len(key))
					return trie.find(child, sub)
				} else {
					//	e.g. child="abc", key="ab"
					//	   abc
					//    /   \
					//   e     f
					return nil
				}
			} else {
				//	e.g. child="abc", key="abd"
				//	   abc
				//    /   \
				//   e     f
				return nil
			}
		}

		return nil
	}

	return nil
}

func (trie *PatriciaTrie) Insert(key string, data types.NodeData) error {
	if len(key) <= 0 {
		return nil
	}

	return trie.insert(trie.root, key, data)
}

func (trie *PatriciaTrie) insert(curNode *PatriciaNode, key string, data types.NodeData) error {
	done := false

	for i := 0; i < len(curNode.children); i++ {
		child := curNode.children[i]

		length := min(len(child.key), len(key))
		j := 0
		for ; j < length; j++ {
			if child.key[j] != key[j] {
				break
			}
		}

		if j == 0 {
			if key[0] < child.key[0] {
				// e.g. child = "e"(curNode = "abc")
				//	"abc"		insert("c")		"abc"
				//	/	\		   ====>  		/ |	\
				// e	 f				   	   c# e  f

				node := &PatriciaNode{
					key:      key,
					dye:      0,
					terminal: true,
					data:     data,
					children: nil,
				}

				curNode.children = insert(curNode.children, i, node)

				done = true
				break
			} else {
				continue
			}
		} else {
			if j == length {
				if len(key) == len(child.key) {
					if child.terminal { // duplicate key
						// nil
					} else {
						// e.g. child = "ab"
						// 	   ab    insert("ab")    ab#
						//    /  \    =========>    /   \
						//   e    f     		   e     f
						child.terminal = true
						child.data = data
					}
				} else if len(key) > len(child.key) {
					// 	e.g. child = "ab#"
					// 	   ab#    insert("abc")   ab#
					//    /  \    ==========>    / | \
					//   e    f     			c# e  f
					sub := substring(key, j, len(key))
					trie.insert(child, sub, data)
				} else {
					// 	e.g. child="abc#"
					// 	   abc#      insert("ab")    ab#
					//    /   \      =========>      /
					//   e     f     			    c#
					// 					           /  \
					//                            e    f
					sub := substring(child.key, j, len(child.key))
					node := &PatriciaNode{
						key:      sub,
						dye:      0,
						terminal: child.terminal,
						data:     child.data,
						children: child.children,
					}

					child.key = key
					child.terminal = true
					child.data = data
					child.children = insert(child.children, 0, node)
				}
			} else {
				//	e.g. child="abc#"
				//	   abc#    insert("abd")         ab
				//    /   \     ==========>     	/  \
				//   e     f   				       c#  d#
				//                           	  /  \
				//                          	 e    f
				// split at j
				childSub := substring(child.key, j, len(child.key))
				sub := substring(key, j, len(key))
				childNode := &PatriciaNode{
					key:      childSub,
					dye:      child.dye,
					terminal: child.terminal,
					data:     child.data,
					children: child.children,
				}

				child.key = substring(child.key, 0, j)
				child.terminal = false
				child.children = make([]*PatriciaNode, 0)

				node := &PatriciaNode{
					key:      sub,
					terminal: true,
					data:     data,
					children: make([]*PatriciaNode, 0),
				}

				if sub[0] < childSub[0] {
					child.children = append(child.children, node)
					child.children = append(child.children, childNode)
				} else {
					child.children = append(child.children, childNode)
					child.children = append(child.children, node)
				}
			}

			done = true
			break
		}

	}

	if !done {
		node := &PatriciaNode{
			key:      key,
			dye:      0,
			terminal: true,
			data:     data,
		}

		curNode.children = append(curNode.children, node)
	}

	return nil
}

func (trie *PatriciaTrie) free(curNode *PatriciaNode, data *types.AccountData) {

}

func (trie *PatriciaTrie) DelDye(dye uint32) {

}

// delete all dyed node
func (trie *PatriciaTrie) delDye(curNode *PatriciaNode, dye uint32) {
	if curNode.dye != dye {
		return
	}

	for i := 0; i < len(curNode.children); i++ {
		trie.delDye(curNode.children[i], dye)
	}

	for i := len(curNode.children) - 1; i >= 0; i-- {
		curNode.children[i] = nil
	}
	curNode.children = nil
}

func (trie *PatriciaTrie) delLess(curNode *PatriciaNode, dye uint32) {
	if curNode.dye > dye {
		return
	}

	for i := 0; i < len(curNode.children); i++ {
		trie.delDye(curNode.children[i], dye)
	}

	for i := len(curNode.children) - 1; i >= 0; i-- {
		curNode.children[i] = nil
	}
	curNode.children = nil
}

// dye the node's path
func (trie *PatriciaTrie) Put(key string, data types.NodeData, dye uint32) {
	result := trie.put(trie.root, key, data, dye)
	if result != nil {
		trie.root = result
	}
}

func (trie *PatriciaTrie) put(curNode *PatriciaNode, key string, data types.NodeData, dye uint32) *PatriciaNode {
	for i := 0; i < len(curNode.children); i++ {
		child := curNode.children[i]

		length := min(len(child.key), len(key))
		j := 0
		for ; j < length; j++ {
			if child.key[j] != key[j] {
				break
			}
		}

		if j == 0 {
			if key[0] < child.key[0] {
				// e.g. child = "e"(curNode = "abc")
				//	"abc"		insert("c")		"abc"
				//	/	\		   ====>  		/ |	\
				// e	 f				   	   c# e  f

				node := &PatriciaNode{
					key:      key,
					dye:      dye,
					terminal: true,
					data:     data,
					children: nil,
				}

				if curNode.dye == dye {
					curNode.children = insert(curNode.children, i, node)
					return nil
				} else {
					tmpCurNode := curNode.Clone()
					tmpCurNode.children = insert(tmpCurNode.children, i, node)
					tmpCurNode.dye = dye
					return tmpCurNode
				}
			} else {
				continue
			}
		} else {
			if j == length {
				if len(key) == len(child.key) { // duplicate key
					// e.g. child = "ab"
					// 	   ab    insert("ab")    ab#
					//    /  \    =========>    /   \
					//   e    f     		   e     f
					if child.dye == dye {
						return nil
					} else {
						tmpChild := child.Clone()
						tmpChild.dye = dye
						tmpChild.data = data
						tmpChild.terminal = true
						if curNode.dye == dye {
							curNode.children[i] = tmpChild
							return nil
						} else {
							tmpCurNode := curNode.Clone()
							tmpCurNode.dye = dye
							tmpCurNode.children[i] = tmpChild
							return tmpCurNode
						}
					}
				} else if len(key) > len(child.key) {
					// 	e.g. child = "ab#"
					// 	   ab#    insert("abc")   ab#
					//    /  \    ==========>    / | \
					//   e    f     			c# e  f
					sub := substring(key, j, len(key))
					result := trie.put(child, sub, data, dye)
					if result == nil {
						return nil
					}

					if curNode.dye == dye {
						curNode.children[i] = result
						return nil
					} else {
						tmpCurNode := curNode.Clone() // current node can't
						tmpCurNode.dye = dye
						tmpCurNode.children[i] = result
						return tmpCurNode
					}
				} else {
					// 	e.g. child="abc#"
					// 	   abc#      insert("ab")    ab#
					//    /   \      =========>      /
					//   e     f     			    c#
					// 					           /  \
					//                            e    f
					sub := substring(child.key, j, len(child.key))
					node := &PatriciaNode{
						key:      sub,
						dye:      dye,
						terminal: child.terminal,
						data:     child.data,
						children: child.children,
					}

					tmpChild := child.Clone()
					tmpChild.key = key
					tmpChild.dye = dye
					tmpChild.terminal = true
					tmpChild.data = data
					tmpChild.children = insert(child.children, 0, node)

					if curNode.dye == dye {
						curNode.children[i] = tmpChild
						return nil
					} else {
						tmpCurNode := curNode.Clone()
						tmpCurNode.dye = dye
						tmpCurNode.children[i] = tmpChild
						return tmpCurNode
					}
				}
			} else {
				//	e.g. child="abc#"
				//	   abc#    insert("abd")         ab
				//    /   \     ==========>     	/  \
				//   e     f   				       c#  d#
				//                           	  /  \
				//                          	 e    f
				// split at j
				childSub := substring(child.key, j, len(child.key))
				sub := substring(key, j, len(key))
				childNode := &PatriciaNode{ // c#
					key:      childSub,
					dye:      child.dye,
					terminal: child.terminal,
					data:     child.data,
					children: child.children,
				}

				node := &PatriciaNode{ // d#
					key:      sub,
					terminal: true,
					dye:      dye,
					data:     data,
					children: make([]*PatriciaNode, 0),
				}

				tmpChild := child.Clone()
				tmpChild.key = substring(child.key, 0, j) // ab
				tmpChild.dye = dye
				tmpChild.terminal = false
				tmpChild.children = make([]*PatriciaNode, 0)

				if sub[0] < childSub[0] {
					tmpChild.children = append(tmpChild.children, node)
					tmpChild.children = append(tmpChild.children, childNode)
				} else {
					tmpChild.children = append(tmpChild.children, childNode)
					tmpChild.children = append(tmpChild.children, node)
				}

				if curNode.dye == dye {
					curNode.children[i] = tmpChild
					return nil
				} else {
					tmpCurNode := curNode.Clone()
					tmpCurNode.dye = dye
					tmpCurNode.children[i] = tmpChild
					return tmpCurNode
				}
			}
		}
	}

	node := &PatriciaNode{
		key:      key,
		dye:      dye,
		terminal: true,
		data:     data,
	}

	if curNode.dye == dye {
		curNode.children = append(curNode.children, node)
		return nil
	} else {
		tmpCurNode := curNode.Clone()
		tmpCurNode.dye = dye
		tmpCurNode.children = append(tmpCurNode.children, node)
		return tmpCurNode
	}
}

func (trie *PatriciaTrie) Collected(dye uint32) []types.NodeData {
	all := make([]types.NodeData, 0)
	return trie.collected(trie.root, dye, all)
}

func (trie *PatriciaTrie) collected(curNode *PatriciaNode, dye uint32, all []types.NodeData) []types.NodeData {
	if curNode.dye != dye {
		return all
	} else {
		for i := 0; i < len(curNode.children); i++ {
			all = trie.collected(curNode.children[i], dye, all)
		}

		if curNode.data != nil && curNode.terminal {
			return append(all, curNode.data)
		} else {
			return all
		}
	}
}

func (trie *PatriciaTrie) All() []types.NodeData {
	all := make([]types.NodeData, 0, trie.total)
	return trie.all(trie.root, all)
}

func (trie *PatriciaTrie) all(curNode *PatriciaNode, all []types.NodeData) []types.NodeData {
	for i := 0; i < len(curNode.children); i++ {
		all = trie.all(curNode.children[i], all)
	}

	if curNode.terminal && curNode.data != nil {
		return append(all, curNode.data)
	} else {
		return all
	}
}

// AccountAPI
type AccountTrieDB struct {
	trie   *PatriciaTrie
	reader DatabaseReader
}

func NewEmptyAccountTrieDB(reader DatabaseReader) *AccountTrieDB {
	return &AccountTrieDB{
		trie:   NewEmptyDatabase(),
		reader: reader,
	}
}

func NewAccountTrieDB(trie *PatriciaTrie, reader DatabaseReader) *AccountTrieDB {
	return &AccountTrieDB{
		trie:   trie,
		reader: reader,
	}
}

func (db *AccountTrieDB) Clone() *AccountTrieDB {
	return &AccountTrieDB{
		reader: db.reader,
		trie:   NewActDatabase(db.trie),
	}
}

func (db *AccountTrieDB) SetTrie(trie *PatriciaTrie) {
	db.trie = trie
}

func (db *AccountTrieDB) GetTrie() *PatriciaTrie {
	return db.trie
}

func (db *AccountTrieDB) SetReader(reader DatabaseReader) {
	db.reader = reader
}

func (db *AccountTrieDB) GetReader() DatabaseReader {
	return db.reader
}

func (db *AccountTrieDB) Set(account *types.AccountData) {
	if account == nil {
		return
	}

	key := account.Address.Hex()
	db.trie.Insert(key, account.Copy())
}

func (db *AccountTrieDB) Get(address common.Address) (*types.AccountData, error) {
	key := address.Hex()
	data := db.trie.Find(key)
	if data == nil {
		val, err := db.reader.Get(address[:])
		if err != nil {
			return nil, err
		}
		if val == nil {
			return nil, nil
		}

		var account types.AccountData
		err = rlp.DecodeBytes(val, &account)
		if err != nil {
			panic("trie.reader.Get(key):" + err.Error())
		} else {
			db.trie.insert(db.trie.root, key, &account)
			return (&account).Copy(), nil
		}
	} else {
		val, ok := data.(*types.AccountData)
		if !ok {
			panic(fmt.Sprintf("expected val *AccountData, got %T", data))
		} else {
			return val.Copy(), nil
		}
	}
}

func (db *AccountTrieDB) Put(account *types.AccountData, dye uint32) {
	if account == nil {
		return
	}

	key := account.Address.Hex()
	db.trie.Put(key, account.Copy(), dye)
}

func (db *AccountTrieDB) Collect(dye uint32) []*types.AccountData {
	all := db.trie.Collected(dye)
	if len(all) <= 0 {
		return make([]*types.AccountData, 0)
	}

	accounts := make([]*types.AccountData, len(all))
	for index := 0; index < len(all); index++ {
		account, ok := all[index].(*types.AccountData)
		if !ok {
			panic(fmt.Sprintf("expected val *AccountData, got %T", all[index]))
		} else {
			accounts[index] = account
		}
	}
	return accounts
}

// CandidateAPI
type CandidateTrieDB struct {
	trie   *PatriciaTrie
	reader DatabaseReader
}

func NewEmptyCandidateTrieDB() *CandidateTrieDB {
	return &CandidateTrieDB{
		trie:   NewEmptyDatabase(),
		reader: nil,
	}
}

func (db *CandidateTrieDB) Clone() *CandidateTrieDB {
	return &CandidateTrieDB{
		reader: db.reader,
		trie:   NewActDatabase(db.trie),
	}
}

func (db *CandidateTrieDB) SetTrie(trie *PatriciaTrie) {
	db.trie = trie
}

func (db *CandidateTrieDB) GetTrie() *PatriciaTrie {
	return db.trie
}

func (db *CandidateTrieDB) SetReader(reader DatabaseReader) {
	db.reader = reader
}

func (db *CandidateTrieDB) GetReader() DatabaseReader {
	return db.reader
}

func (db *CandidateTrieDB) key(address common.Address) string {
	return "candidate:" + address.Hex()
}

func (db *CandidateTrieDB) Set(candidate *Candidate) {
	if candidate == nil {
		return
	}

	db.trie.Insert(db.key(candidate.address), candidate.Clone())
}

func (db *CandidateTrieDB) Get(address common.Address) (*Candidate, error) {
	data := db.trie.Find(db.key(address))
	val, ok := data.(*Candidate)
	if !ok {
		panic(fmt.Sprintf("expected val *Candidate, got %T", data))
	}

	if val == nil {
		return nil, nil
	} else {
		return val.Copy(), nil
	}
}

func (db *CandidateTrieDB) GetAll() []*Candidate {
	result := db.trie.All()
	if len(result) <= 0 {
		return make([]*Candidate, 0)
	}

	candidates := make([]*Candidate, len(result))
	for index := 0; index < len(result); index++ {
		candidate, ok := result[index].(*Candidate)
		if !ok {
			panic(fmt.Sprintf("expected val *Candidate, got %T", result[index]))
		}

		if candidate == nil {
			panic("got nil candidate.")
		}

		candidates[index] = candidate
	}
	return candidates
}

func (db *CandidateTrieDB) Put(candidate *Candidate, dye uint32) {
	if candidate == nil {
		return
	}

	db.trie.Put(db.key(candidate.address), candidate.Copy(), dye)
}

func (db *CandidateTrieDB) Flush() error {
	return nil
}

func (db *CandidateTrieDB) Loading() error {
	return nil
}
