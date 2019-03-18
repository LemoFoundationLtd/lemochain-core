package merkle

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"math"
)

// NodeTypeFlag 节点类型标识
type NodeTypeFlag int

// 用在伴随节点
const (
	LeftNode NodeTypeFlag = iota
	RightNode
	RootNode
)

// MerkleNode 用在获取与验证伴随节点
type MerkleNode struct {
	Hash     common.Hash
	NodeType NodeTypeFlag
}

type MerkleTree struct {
	leafHashes []common.Hash // 叶子hash，外界传入
	nodes      []common.Hash // 所有节点hash
	offset     int           // 下一个要计算父节点
}

// New 新建一个merkle tree
func New(leafHashes []common.Hash) *MerkleTree {
	m := &MerkleTree{
		leafHashes: leafHashes,
	}
	return m
}

// VersionRoot 获取根Hash
func (m *MerkleTree) Root() common.Hash {
	if m.nodes == nil {
		m.calculateNodes()
	}
	return m.nodes[len(m.nodes)-1]
}

// HashNodes 获取所有的hash，从叶子节点到根root
func (m *MerkleTree) HashNodes() []common.Hash {
	if m.nodes == nil {
		m.calculateNodes()
	}
	return m.nodes
}

// calculateNodes 计算中间节点
func (m *MerkleTree) calculateNodes() {
	m.nodes = make([]common.Hash, 0, len(m.leafHashes)*2)
	m.nodes = append(m.nodes, m.leafHashes...)
	for ; m.offset < len(m.nodes)-1; m.offset += 2 {
		hash := crypto.Keccak256Hash(append(m.nodes[m.offset][:], m.nodes[m.offset+1][:]...))
		m.nodes = append(m.nodes, hash)
	}
}

// FindSiblingNodes 查找伴随节点
func FindSiblingNodes(src common.Hash, srcNodes []common.Hash) ([]MerkleNode, error) {
	if srcNodes == nil {
		return nil, errors.New("src nodes can't be nil")
	}
	var index = 0
	for ; index < len(srcNodes); index++ {
		if bytes.Compare(src[:], srcNodes[index][:]) == 0 {
			break
		}
	}
	if index == len(srcNodes) {
		return nil, fmt.Errorf("can't find hash:%s in src nodes", common.ToHex(src[:]))
	}
	nodesLen := (len(srcNodes) + 1) / 2
	var findPath func(n int, result []MerkleNode) []MerkleNode
	findPath = func(n int, result []MerkleNode) []MerkleNode {
		if n == len(srcNodes)-1 {
			result = append(result, MerkleNode{Hash: srcNodes[n], NodeType: RootNode})
			return result
		} else if n%2 == 1 {
			result = append(result, MerkleNode{Hash: srcNodes[n-1], NodeType: LeftNode})
		} else {
			result = append(result, MerkleNode{Hash: srcNodes[n+1], NodeType: RightNode})
		}
		return findPath(nodesLen+int(math.Floor(float64(n)/float64(2))), result)
	}
	result := make([]MerkleNode, 0)
	result = findPath(index, result)
	return result, nil
}
