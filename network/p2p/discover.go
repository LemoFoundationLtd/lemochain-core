package p2p

import (
	"bufio"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"net"
	"os"
	"path/filepath"
	"strconv"
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
	ErrHasStared     = errors.New("has been started")
	ErrNotStart      = errors.New("not start")
)

// RawNode wrap node connection info for discovery
type RawNode struct {
	NodeID      *NodeID
	Endpoint    string
	IsReconnect bool
	ConnCounter int8
	Sequence    int32 // fresh: >0; stale: <0; connecting: 0
}

func newRawNode(nodeID *NodeID, endpoint string) *RawNode {
	return &RawNode{
		NodeID:   nodeID,
		Endpoint: endpoint,
	}
}

// String string formatter
func (n *RawNode) String() string {
	idStr := common.Bytes2Hex(n.NodeID[:])
	return idStr + "@" + n.Endpoint
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

func NewDiscoverManager(dataDir string) *DiscoverManager {
	m := &DiscoverManager{
		dataDir:     dataDir,
		sequence:    0,
		foundNodes:  make(map[common.Hash]*RawNode, 100),
		whiteNodes:  make(map[common.Hash]*RawNode, 20),
		deputyNodes: make(map[common.Hash]*RawNode, 20),

		status: 0,
	}
	return m
}

// Start
func (m *DiscoverManager) Start() error {
	if atomic.CompareAndSwapInt32(&m.status, 0, 1) {
		m.setWhiteList()
		m.initDiscoverList()
	} else {
		return ErrHasStared
	}
	log.Info("discover start ok")
	return nil
}

// Stop
func (m *DiscoverManager) Stop() error {
	if atomic.CompareAndSwapInt32(&m.status, 1, 0) {
		m.writeFindFile()
	} else {
		return ErrNotStart
	}
	log.Info("discover stop ok")
	return nil
}

// connectedNodes get connected nodes ever
func (m *DiscoverManager) connectedNodes() []string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	res := make([]string, 0, MaxNodeCount)
	for _, node := range m.whiteNodes {
		if node.Sequence > 0 {
			res = append(res, node.String())
		}
	}
	for _, node := range m.deputyNodes {
		if node.Sequence > 0 {
			res = append(res, node.String())
		}
	}
	for _, node := range m.foundNodes {
		if node.Sequence > 0 {
			res = append(res, node.String())
		}
	}
	return res
}

// connectingNodes to be connected nodes
func (m *DiscoverManager) connectingNodes() []string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	res := make([]string, 0, MaxNodeCount)
	for _, node := range m.whiteNodes {
		if node.Sequence == 0 {
			res = append(res, node.String())
		}
	}
	for _, node := range m.deputyNodes {
		if node.Sequence == 0 {
			res = append(res, node.String())
		}
	}
	for _, node := range m.foundNodes {
		if node.Sequence == 0 {
			res = append(res, node.String())
		}
	}
	return res
}

// staleNodes connect failed nodes
func (m *DiscoverManager) staleNodes() []string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	res := make([]string, 0, MaxNodeCount)
	for _, node := range m.whiteNodes {
		if node.Sequence < 0 {
			res = append(res, node.String())
		}
	}
	for _, node := range m.deputyNodes {
		if node.Sequence < 0 {
			res = append(res, node.String())
		}
	}
	for _, node := range m.foundNodes {
		if node.Sequence < 0 {
			res = append(res, node.String())
		}
	}
	return res
}

// resetState reset state
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

	for _, node := range nodes {
		nodeID, endpoint := checkNodeString(node)
		if nodeID == nil {
			continue
		}
		key := crypto.Keccak256Hash(nodeID[:])
		if n, ok := m.whiteNodes[key]; ok {
			if n.Sequence < 0 {
				m.resetState(n)
			}
			continue
		}
		if n, ok := m.deputyNodes[key]; ok {
			if n.Sequence < 0 {
				m.resetState(n)
			}
			continue
		}
		if n, ok := m.foundNodes[key]; ok {
			if n.Sequence < 0 {
				m.resetState(n)
			}
			continue
		}
		if n := newRawNode(nodeID, endpoint); n != nil {
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
		nodeID, endpoint := checkNodeString(node)
		if nodeID == nil {
			continue
		}
		key := crypto.Keccak256Hash(nodeID[:])
		if _, ok := m.deputyNodes[key]; ok {
			continue
		}
		if n = newRawNode(nodeID, endpoint); n != nil {
			m.deputyNodes[key] = n
		}
	}
}

// SetConnectResult set connect result
func (m *DiscoverManager) SetConnectResult(nodeID *NodeID, success bool) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	key := crypto.Keccak256Hash((*nodeID)[:])
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
			n.ConnCounter++
			if n.ConnCounter == MaxReconnectCount {
				n.Sequence = -1
				n.IsReconnect = false
				return ErrMaxReconnect
			}
		}
	}
	return nil
}

// SetReconnect start reconnect
func (m *DiscoverManager) SetReconnect(nodeID *NodeID) error {
	log.Debugf("discover: set reconnect: %s", common.ToHex((*nodeID)[:]))
	m.lock.Lock()
	defer m.lock.Unlock()

	key := crypto.Keccak256Hash((*nodeID)[:])
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
	// if n.IsReconnect {
	// 	if n.ConnCounter == MaxReconnectCount {
	// 		log.Infof("node: %s has reconnect %d, but not success", node, MaxReconnectCount)
	// 		return ErrMaxReconnect
	// 	}
	// } else {
	// 	n.IsReconnect = true
	// 	n.Sequence = 0
	// }
	// n.ConnCounter++
	n.IsReconnect = true
	n.Sequence = 0
	n.ConnCounter = 1
	return nil
}

// getAvailableNodes get available nodes
func (m *DiscoverManager) getAvailableNodes() []string {
	list := m.connectedNodes()
	if len(list) < MaxNodeCount {
		list = append(list, m.connectingNodes()...)
	}
	// if len(list) < MaxNodeCount {
	// 	list = append(list, m.staleNodes()...)
	// }
	if len(list) > MaxNodeCount {
		list = list[:MaxNodeCount]
	}
	return list
}

// GetNodesForDiscover get available nodes for node discovery
func (m *DiscoverManager) GetNodesForDiscover(sequence uint) []string {
	// sequence for revert
	return m.getAvailableNodes()
}

// readFile read file function
func readFile(path string) []string {
	f, err := os.OpenFile(path, os.O_RDONLY, 666)
	if err != nil {
		return nil
	}
	defer f.Close()

	list := make([]string, 0, MaxNodeCount)
	buf := bufio.NewReader(f)
	line, _, err := buf.ReadLine()
	for err == nil {
		if strings.Index(string(line), "@") > -1 {
			list = append(list, string(line))
		}
		line, _, err = buf.ReadLine()
		if len(list) == MaxNodeCount {
			break
		}
	}
	return list
}

// setWhiteList set white list nodes
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
		nodeID, endpoint := checkNodeString(node)
		if nodeID == nil {
			continue
		}
		key := crypto.Keccak256Hash(nodeID[:])
		if _, ok := m.whiteNodes[key]; ok {
			continue
		}
		if n = newRawNode(nodeID, endpoint); n != nil {
			m.whiteNodes[key] = n
		}
	}
}

// initDiscoverList read initial node from file
func (m *DiscoverManager) initDiscoverList() {
	path := filepath.Join(m.dataDir, FindFile)
	list := readFile(path)
	m.addDiscoverNodes(list)
}

// AddNewList for discovery
func (m *DiscoverManager) AddNewList(nodes []string) error {
	m.addDiscoverNodes(nodes)
	return nil
}

// writeFindFile write invalid node to file
func (m *DiscoverManager) writeFindFile() {
	// create list
	list := m.getAvailableNodes()

	path := filepath.Join(m.dataDir, FindFile)
	// open file
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 666) // read and write
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

// checkNodeString verify invalid
func checkNodeString(node string) (*NodeID, string) {
	tmp := strings.Split(node, "@")
	if len(tmp) != 2 {
		return nil, ""
	}
	if len(tmp[0]) != 128 {
		return nil, ""
	}
	nodeID := BytesToNodeID(common.FromHex(tmp[0]))
	_, err := nodeID.PubKey()
	if err != nil {
		return nil, ""
	}
	if !verifyIP(tmp[1]) {
		return nil, ""
	}
	return nodeID, tmp[1]
}

// verifyIP  verify ipv4
func verifyIP(input string) bool {
	tmp := strings.Split(input, ":")
	if len(tmp) != 2 {
		return false
	}
	if ip := net.ParseIP(tmp[0]); ip == nil {
		return false
	}
	p, err := strconv.Atoi(tmp[1])
	if err != nil {
		return false
	}
	if p < 1024 || p > 65535 {
		return false
	}
	return true
}
