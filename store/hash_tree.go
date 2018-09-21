package store

import "bytes"

var (
	flgValid = uint8(0x01)
)

type HItem struct {
	Flg uint8
	Num uint32
	Pos uint32 // 文件位置
}

type TItem struct {
	Flg uint8
	Num uint32
	Pos uint32 // 文件位置
	Key []byte
}

type Node struct {
	IsNode  bool
	Depth   uint32
	MaxCnt  uint8
	UsedCnt uint8
	Header  uint32
}

type Tree struct {
	Root    *Node
	DataBuf *CachePool
	NodeBuf *CachePool
}

type familyNode struct {
	Ancestor *Node
	Index    uint32 // the DataBuf index in Parents
	Parents  []*Node
	DataBuf  []*TItem
}

func (tree *Tree) getFamily(num uint32) (*familyNode, error) {
	family := new(familyNode)
	family.Ancestor = tree.Root

	child, err := GetNodes(tree.NodeBuf, tree.Root.Header)
	if err != nil {
		return nil, err
	} else {
		// root's depth is 0
		family.Index = num & 0x0f
		family.Parents = child
	}

	for {
		node := family.Parents[family.Index]
		if !node.IsNode {
			if node.UsedCnt <= 0 {
				return family, nil
			}

			items, err := GetItems(tree.DataBuf, node.Header)
			if err != nil {
				return nil, err
			} else {
				family.DataBuf = items
				return family, nil
			}
		} else {
			family.Ancestor = node
			child, err := GetNodes(tree.NodeBuf, family.Ancestor.Header)
			if err != nil {
				return nil, err
			} else {
				depth := family.Ancestor.Depth
				family.Index = (num >> (depth * 4)) & 0x0f
				family.Parents = child
			}
		}
	}
}

func (tree *Tree) initNodes(isNode bool, depth uint32, maxCnt uint8, usedCnt uint8) []*Node {
	nodes := make([]*Node, 16)
	for index := 0; index < 16; index++ {
		nodes[index] = &Node{
			IsNode:  isNode,
			Depth:   depth,
			MaxCnt:  maxCnt,
			UsedCnt: usedCnt,
			Header:  0,
		}
	}
	return nodes
}

func (tree *Tree) initItems() []*TItem {
	items := make([]*TItem, 16)
	key := make([]byte, keySize, keySize)
	for index := 0; index < 16; index++ {
		items[index] = &TItem{
			Flg: 0,
			Num: 0,
			Pos: 0,
			Key: key,
		}
	}
	return items
}

func (tree *Tree) split(family *familyNode) error {
	nodes := tree.initNodes(false, family.Parents[family.Index].Depth+1, 16, 0)

	oldItems := family.DataBuf
	newItems := make([][]*TItem, 16)
	for index := 0; index < 16; index++ {

		item := oldItems[index]
		num := item.Num
		depth := family.Parents[family.Index].Depth
		val := num >> (4 * (depth + 1))
		hash := val & 0x0f

		if newItems[hash] == nil {
			newItems[hash] = tree.initItems()
		}

		usedCnt := nodes[hash].UsedCnt
		newItems[hash][usedCnt] = item
		nodes[hash].UsedCnt = usedCnt + 1
	}

	for index := 0; index < 16; index++ {
		items := newItems[index]
		if items == nil {
			continue
		} else {
			header, err := SetItems(tree.DataBuf, items)
			if err != nil {
				return err
			} else {
				nodes[index].Header = header
				nodes[index].IsNode = false
			}
		}
	}

	header, err := SetNodes(tree.NodeBuf, nodes)
	if err != nil {
		return err
	}

	family.Parents[family.Index].IsNode = true
	family.Parents[family.Index].Header = header

	return UpdateNodes(tree.NodeBuf, family.Ancestor.Header, family.Parents)
}

func (tree *Tree) Get(key []byte) (*HItem, error) {
	num := Byte2Uint32(key)
	family, err := tree.getFamily(num)
	if err != nil {
		return nil, err
	}

	items := family.DataBuf
	if items == nil || len(items) == 0 {
		return nil, nil
	}

	for index := 0; index < 16; index++ {
		item := items[index]
		if item.Flg&flgValid != 1 {
			return nil, nil
		}

		if bytes.Equal(item.Key, key) {
			return &HItem{
				Flg: item.Flg,
				Num: item.Num,
				Pos: item.Pos,
			}, nil
		}
	}

	return nil, nil
}

func (tree *Tree) Add(key []byte, pos uint32) error {
	num := Byte2Uint32(key)
	family, err := tree.getFamily(num)
	if err != nil {
		return err
	}

	items := family.DataBuf
	if items != nil {
		index := 0
		for index = 0; index < 16; index++ {
			item := items[index]
			if item.Flg&flgValid != 1 {
				break
			}

			if bytes.Equal(item.Key, key) {
				item.Pos = pos
				return UpdateItems(tree.DataBuf, family.Parents[family.Index].Header, items)
			}
		}

		if family.Parents[family.Index].UsedCnt >= 16 {
			err := tree.split(family)
			if err != nil {
				return err
			} else {
				return tree.Add(key, pos)
			}
		} else {
			usedCnt := family.Parents[family.Index].UsedCnt
			items[usedCnt].Flg = items[usedCnt].Flg | flgValid
			items[usedCnt].Num = num
			items[usedCnt].Pos = pos
			items[usedCnt].Key = key
			err := UpdateItems(tree.DataBuf, family.Parents[family.Index].Header, items)
			if err != nil {
				return err
			} else {
				family.Parents[family.Index].UsedCnt = usedCnt + 1
				return UpdateNodes(tree.NodeBuf, family.Ancestor.Header, family.Parents)
			}
		}
	} else {
		items = tree.initItems()
		items[0].Flg = items[0].Flg | flgValid
		items[0].Num = num
		items[0].Pos = pos
		items[0].Key = key
		header, err := SetItems(tree.DataBuf, items)
		if err != nil {
			return err
		} else {
			family.Parents[family.Index].UsedCnt = 1
			family.Parents[family.Index].Header = header
			return UpdateNodes(tree.NodeBuf, family.Ancestor.Header, family.Parents)
		}
	}
}

func NewTree() (*Tree, error) {
	tree := &Tree{}
	tree.Root = new(Node)
	tree.Root.IsNode = true
	tree.Root.Depth = 0
	tree.Root.MaxCnt = 16
	tree.Root.UsedCnt = 16

	tree.NodeBuf = NewCachePool(uint32(256*256), uint8(nodeSize))
	nodes := tree.initNodes(false, 1, 16, 0)
	header, err := SetNodes(tree.NodeBuf, nodes)
	if err != nil {
		return nil, err
	} else {
		tree.Root.Header = header
	}

	tree.DataBuf = NewCachePool(uint32(256*256), uint8(itemHeaderSize)+uint8(keySize))
	return tree, nil
}
