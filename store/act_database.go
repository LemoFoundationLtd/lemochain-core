package store

import (
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
	data     *types.AccountData
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
		result.data = node.data.Copy()
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
	reader DatabaseReader
}

func NewEmptyDatabase(reader DatabaseReader) *PatriciaTrie {
	return &PatriciaTrie{
		root:   new(PatriciaNode),
		total:  0,
		minDye: 0,
		maxDye: 0,
		reader: reader,
	}
}

func NewActDatabase(reader DatabaseReader, root *PatriciaTrie) *PatriciaTrie {
	return &PatriciaTrie{
		root:   root.root,
		total:  root.total,
		minDye: root.minDye,
		maxDye: root.maxDye,
		reader: reader,
	}
}

func (trie *PatriciaTrie) Clone() *PatriciaTrie {
	return &PatriciaTrie{
		root:   trie.root,
		total:  trie.total,
		minDye: trie.minDye,
		maxDye: trie.maxDye,
		reader: trie.reader,
	}
}

func (trie *PatriciaTrie) Find(key []byte) *types.AccountData {
	if len(key) <= 0 {
		return nil
	}

	tmp := common.ToHex(key[:])
	act := trie.find(trie.root, tmp)
	if act != nil {
		return act
	}

	val, err := trie.reader.Get(key)
	if err != nil {
		panic("trie.reader.Get(key):" + err.Error())
	}

	if val == nil {
		return nil
	}

	var account types.AccountData
	err = rlp.DecodeBytes(val, &account)
	if err != nil {
		panic("trie.reader.Get(key):" + err.Error())
	} else {
		trie.insert(trie.root, tmp, &account)
	}

	return &account
}

func (trie *PatriciaTrie) find(curNode *PatriciaNode, key string) *types.AccountData {
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

func (trie *PatriciaTrie) Insert(key []byte, data *types.AccountData) error {
	if len(key) <= 0 {
		return nil
	}

	tmp := common.ToHex(key[:])
	return trie.insert(trie.root, tmp, data)
}

func (trie *PatriciaTrie) insert(curNode *PatriciaNode, key string, data *types.AccountData) error {
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
func (trie *PatriciaTrie) Put(account *types.AccountData, dye uint32) {
	key := common.ToHex(account.Address[:])

	result := trie.put(trie.root, key, account, dye)
	if result != nil {
		trie.root = result
	}
}

func (trie *PatriciaTrie) put(curNode *PatriciaNode, key string, account *types.AccountData, dye uint32) *PatriciaNode {
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
					data:     account,
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
						tmpChild.data = account
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
					result := trie.put(child, sub, account, dye)
					if result == nil {
						return nil
					}

					if curNode.dye == dye {
						curNode.children[i] = result
						return nil
					} else {
						tmpCurNode := curNode.Clone()
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
					tmpChild.data = account
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
					data:     account,
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
		data:     account,
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

func (trie *PatriciaTrie) Collected(dye uint32) []*types.AccountData {
	accounts := make([]*types.AccountData, 0)
	return trie.collected(trie.root, dye, accounts)
}

func (trie *PatriciaTrie) collected(curNode *PatriciaNode, dye uint32, accounts []*types.AccountData) []*types.AccountData {
	if curNode.dye != dye {
		return accounts
	} else {
		for i := 0; i < len(curNode.children); i++ {
			accounts = trie.collected(curNode.children[i], dye, accounts)
		}

		if curNode.data != nil && curNode.terminal {
			return append(accounts, curNode.data)
		} else {
			return accounts
		}
	}
}
