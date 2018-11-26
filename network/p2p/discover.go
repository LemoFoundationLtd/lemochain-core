package p2p

import (
	"bufio"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	MaxReconnectCount int8 = 5
	MaxNodeCount           = 200

	WhiteFile = "whitelist"
	FindFile  = "findnode"
)

var (
	ErrMaxReconnect  = errors.New("reconnect has reached max count")
	ErrNoSpecialNode = errors.New("doesn't have this special node")
)

type RawNode struct {
	NodeID      string
	Endpoint    string
	IsReconnect bool
	ConnCounter int8
	Sequence    int32 // fresh: >0; stale: <0; connecting: 0
}

func newRawNode(node string) *RawNode {
	// tmp := strings.Split(node, "@")
	// if len(tmp) != 2 || len(tmp[0]) != 64 || strings.Index(tmp[1], ":") < 0 {
	// 	return nil
	// }
	// n := &RawNode{
	// 	NodeID:      tmp[0],
	// 	Endpoint:    tmp[1],
	// 	IsReconnect: false,
	// 	ConnCounter: 0,
	// 	Sequence:    0,
	// }
	//return n

	// todo delete this and use behand code
	return &RawNode{
		Endpoint: node,
	}
}

type DiscoverManager struct {
	sequence    int32
	foundNodes  map[common.Hash]*RawNode // total nodes. contains: 'add peer', 'receive nodes from nodes find request'
	whiteNodes  map[common.Hash]*RawNode // white list nodes
	deputyNodes map[common.Hash]*RawNode // deputy nodes

	dataDir string
	status  int32

	lock sync.RWMutex
}

func NewDiscoverManager(datadir string) *DiscoverManager {
	m := &DiscoverManager{
		dataDir:     datadir,
		sequence:    0,
		foundNodes:  make(map[common.Hash]*RawNode, 100),
		whiteNodes:  make(map[common.Hash]*RawNode, 20),
		deputyNodes: make(map[common.Hash]*RawNode, 20),

		status: 0,
	}
	return m
}

func (m *DiscoverManager) Start() {
	if atomic.CompareAndSwapInt32(&m.status, 0, 1) {
		m.setWhiteList()
		m.initDiscoverList()
	} else {
		log.Warn("DiscoverManager has been started.")
	}
}

func (m *DiscoverManager) Stop() error {
	if atomic.CompareAndSwapInt32(&m.status, 1, 0) {
		m.writeFindFile()
	} else {
		log.Warn("DiscoverManager has not been start.")
	}
	return nil
}

// connectedNodes get connected nodes ever
func (m *DiscoverManager) connectedNodes() []string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	res := make([]string, 0, len(m.foundNodes))
	for _, node := range m.whiteNodes {
		if node.Sequence > 0 {
			res = append(res, node.Endpoint)
		}
	}
	for _, node := range m.deputyNodes {
		if node.Sequence > 0 {
			res = append(res, node.Endpoint)
		}
	}
	for _, node := range m.foundNodes {
		if node.Sequence > 0 {
			res = append(res, node.Endpoint)
		}
	}
	return res
}

// connectingNodes to be connected nodes
func (m *DiscoverManager) connectingNodes() []string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	res := make([]string, 0, len(m.foundNodes))
	for _, node := range m.whiteNodes {
		if node.Sequence == 0 {
			res = append(res, node.Endpoint)
		}
	}
	for _, node := range m.deputyNodes {
		if node.Sequence == 0 {
			res = append(res, node.Endpoint)
		}
	}
	for _, node := range m.foundNodes {
		if node.Sequence == 0 {
			res = append(res, node.Endpoint)
		}
	}
	return res
}

// staleNodes connect failed nodes
func (m *DiscoverManager) staleNodes() []string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	res := make([]string, 0, len(m.foundNodes))
	for _, node := range m.whiteNodes {
		if node.Sequence < 0 {
			res = append(res, node.Endpoint)
		}
	}
	for _, node := range m.deputyNodes {
		if node.Sequence < 0 {
			res = append(res, node.Endpoint)
		}
	}
	for _, node := range m.foundNodes {
		if node.Sequence < 0 {
			res = append(res, node.Endpoint)
		}
	}
	return res
}

func (m *DiscoverManager) resetState(n *RawNode) {
	n.IsReconnect = false
	n.Sequence = 0
	n.ConnCounter = 0
}

// addDiscoverNodes add nodes to DiscoverManager.foundNodes
func (m *DiscoverManager) addDiscoverNodes(nodes []string) {
	if nodes == nil || len(nodes) == 0 {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	var n *RawNode
	for _, node := range nodes {
		key := crypto.Keccak256Hash([]byte(node))
		if n, ok := m.whiteNodes[key]; ok {
			if n.Sequence < 0 {
				m.resetState(n)
			}
			continue
		}
		if _, ok := m.deputyNodes[key]; ok {
			if n.Sequence < 0 {
				m.resetState(n)
			}
			continue
		}
		if _, ok := m.foundNodes[key]; ok {
			if n.Sequence < 0 {
				m.resetState(n)
			}
			continue
		}
		if n = newRawNode(node); n != nil {
			m.foundNodes[key] = n
		}
	}
}

// SetDeputyNodes add deputy nodes
func (m *DiscoverManager) SetDeputyNodes(nodes []string) {
	if nodes == nil || len(nodes) == 0 {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	var n *RawNode
	for _, node := range nodes {
		key := crypto.Keccak256Hash([]byte(node))
		if _, ok := m.deputyNodes[key]; ok {
			continue
		}
		if n = newRawNode(node); n != nil {
			m.deputyNodes[key] = n
		}
	}
}

// SetConnectResult set connect result
func (m *DiscoverManager) SetConnectResult(node string, success bool) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	key := crypto.Keccak256Hash([]byte(node))
	n, ok := m.deputyNodes[key]
	if !ok {
		n, ok = m.whiteNodes[key]
	}
	if !ok {
		n, ok = m.foundNodes[key]
	}
	if !ok {
		return ErrNoSpecialNode
	}
	if success {
		m.sequence++
		n.Sequence = m.sequence
		n.IsReconnect = false
		n.ConnCounter = 0
	} else {
		if !n.IsReconnect {
			n.Sequence = -1
		} else {
			n.Sequence = 0
			// n.ConnCounter++
			// if n.ConnCounter == MaxReconnectCount {
			// 	return ErrMaxReconnect
			// }
		}
	}
	return nil
}

// SetReconnect start reconnect
func (m *DiscoverManager) SetReconnect(node string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	key := crypto.Keccak256Hash([]byte(node))
	n, ok := m.deputyNodes[key]
	if !ok {
		n, ok = m.whiteNodes[key]
	}
	if !ok {
		n, ok = m.foundNodes[key]
	}
	if !ok {
		return
	}
	if n.IsReconnect {
		if n.ConnCounter == MaxReconnectCount {
			log.Infof("node: %s has reconnect %d, but not success", node, MaxReconnectCount)
			return
		}
		n.ConnCounter++
	} else {
		n.IsReconnect = true
		n.Sequence = 0
	}
}

func (m *DiscoverManager) getAvailableNodes() []string {
	list := m.connectedNodes()
	if len(list) < MaxNodeCount {
		list = append(list, m.connectingNodes()...)
	}
	if len(list) < MaxNodeCount {
		list = append(list, m.staleNodes()...)
	}
	if len(list) > MaxNodeCount {
		list = list[:MaxNodeCount]
	}
	return list
}

func (m *DiscoverManager) GetNodesForDiscover(sequence int32) []string {
	// sequence for revert
	return m.getAvailableNodes()
}

func readFile(path string) []string {
	f, err := os.OpenFile(path, os.O_RDONLY, 777)
	if err != nil {
		return nil
	}
	defer f.Close()

	list := make([]string, 0, MaxNodeCount)
	buf := bufio.NewReader(f)
	line, _, err := buf.ReadLine()
	count := 0
	for err == nil {
		count++
		if strings.Index(string(line), ":") > -1 { // todo
			list = append(list, string(line))
		}
		line, _, err = buf.ReadLine()
		if len(list) == MaxNodeCount {
			break
		}
	}
	return list
}

func (m *DiscoverManager) setWhiteList() {
	path := filepath.Join(m.dataDir, WhiteFile)
	nodes := readFile(path)
	if nodes == nil || len(nodes) == 0 {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	var n *RawNode
	for _, node := range nodes {
		key := crypto.Keccak256Hash([]byte(node))
		if _, ok := m.whiteNodes[key]; ok {
			continue
		}
		if n = newRawNode(node); n != nil {
			m.whiteNodes[key] = n
		}
	}
}

func (m *DiscoverManager) initDiscoverList() {
	path := filepath.Join(m.dataDir, FindFile)
	list := readFile(path)
	m.addDiscoverNodes(list)
}

func (m *DiscoverManager) AddNewList(nodes []string) {
	m.addDiscoverNodes(nodes)
}

func (m *DiscoverManager) writeFindFile() {
	// create list
	list := m.getAvailableNodes()

	path := filepath.Join(m.dataDir, FindFile)
	// open file
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 777) // todo
	if err != nil {
		return
	}
	defer f.Close()

	// write file
	buf := bufio.NewWriter(f)
	for _, n := range list {
		buf.WriteString(n + "\n")
	}
	buf.Flush()
}
