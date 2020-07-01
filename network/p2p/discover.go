package p2p

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
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
	MaxNodeCount           = 100

	WhiteFile = "nodewhitelist"
	FindFile  = "findnode"
	BlackFile = "nodeblacklist"
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
	Sequence    int32 // fresh(已连接): >0; stale(连接过并失败): <0; connecting(可以连接): 0
}

func newRawNode(nodeID *NodeID, endpoint string) *RawNode {
	return &RawNode{
		NodeID:   nodeID,
		Endpoint: endpoint,
	}
}

// String string formatter
func (n *RawNode) String() string {
	idStr := fmt.Sprintf("%x", n.NodeID[:])
	return idStr + "@" + n.Endpoint
}

type DiscoverManager struct {
	sequence    int32
	foundNodes  map[common.Hash]*RawNode // total nodes. contains: 'add peer', 'receive nodes from nodes find request'
	whiteNodes  map[common.Hash]*RawNode // white list nodes
	blackNodes  map[common.Hash]*RawNode // black list nodes
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
		blackNodes:  make(map[common.Hash]*RawNode, 20),
		deputyNodes: make(map[common.Hash]*RawNode, 20),

		status: 0,
	}
	return m
}

// Start
func (m *DiscoverManager) Start() error {
	if atomic.CompareAndSwapInt32(&m.status, 0, 1) {
		m.initBlackList()
		m.initWhiteList()
		m.initDiscoverList()
	} else {
		return ErrHasStared
	}
	log.Info("Discover manager start")
	return nil
}

// Stop
func (m *DiscoverManager) Stop() error {
	if atomic.CompareAndSwapInt32(&m.status, 1, 0) {
		// write find node to file
		m.writeFindNodeToFile()
	} else {
		return ErrNotStart
	}
	log.Info("Discover stop ok")
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
	// limit max connect number
	if len(res) > MaxNodeCount {
		res = res[:MaxNodeCount]
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
		nodeID, endpoint := ParseNodeString(node)
		if nodeID == nil {
			continue
		}
		key := nodeID.Hash()
		if _, ok := m.blackNodes[key]; ok {
			continue
		}
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
		m.foundNodes[key] = newRawNode(nodeID, endpoint)
		log.Debug("Found new node", "nodeID", common.ToHex(nodeID[:4]), "endpoint", endpoint)
	}
}

// SetDeputyNodes add deputy nodes
func (m *DiscoverManager) SetDeputyNodes(nodes []string) {
	if nodes == nil || len(nodes) == 0 {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	for _, node := range nodes {
		nodeID, endpoint := ParseNodeString(node)
		if nodeID == nil {
			continue
		}
		key := nodeID.Hash()
		if _, ok := m.deputyNodes[key]; ok {
			continue
		}
		m.deputyNodes[key] = newRawNode(nodeID, endpoint)
	}
}

// SetConnectResult set connect result
func (m *DiscoverManager) SetConnectResult(nodeID *NodeID, success bool) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	key := nodeID.Hash()
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
	log.Debugf("Discover: set reconnect: %s", common.ToHex((*nodeID)[:]))
	m.lock.Lock()
	defer m.lock.Unlock()

	key := nodeID.Hash()
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
func (m *DiscoverManager) GetNodesForDiscover(sequence uint, rPeerNodeID string) []string {
	// sequence for revert
	nodes := m.getAvailableNodes()
	newNodes := make([]string, 0)
	// judge that sending nodes for discovery cannot contain the remote peer node
	for _, node := range nodes {
		tmp := strings.Split(node, "@")
		if len(tmp) != 2 {
			continue
		}
		if tmp[0] == rPeerNodeID {
			continue
		}
		newNodes = append(newNodes, node)
	}
	return newNodes
}

// readFile read file function
func readFile(path string) []string {
	f, err := os.OpenFile(path, os.O_RDONLY, 666)
	if err != nil {
		return nil
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Infof("Close file failed: %v", err)
		}
	}()

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

// is black list node
func (m *DiscoverManager) IsBlackNode(nodeID *NodeID) bool {
	key := nodeID.Hash()

	if n := m.getBlackNode(key); n != nil {
		return true
	} else {
		return false
	}
}

// getBlackNode
func (m *DiscoverManager) getBlackNode(key common.Hash) *RawNode {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.blackNodes[key]
}

// PutBlackNode
func (m *DiscoverManager) PutBlackNode(nodeID *NodeID, endpoint string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	key := nodeID.Hash()
	if _, ok := m.blackNodes[key]; ok {
		return
	}
	m.blackNodes[key] = newRawNode(nodeID, endpoint)
}

// initBlackList set black list nodes
func (m *DiscoverManager) initBlackList() {
	path := filepath.Join(m.dataDir, BlackFile)
	nodes := readFile(path)
	if nodes == nil || len(nodes) == 0 {
		return
	}

	for _, node := range nodes {
		nodeID, endpoint := ParseNodeString(node)
		if nodeID == nil {
			continue
		}
		m.PutBlackNode(nodeID, endpoint)
	}
}

// writeBlackListToFile
func (m *DiscoverManager) writeBlackListToFile() {
	list := make([]string, 0, MaxNodeCount)
	for _, node := range m.blackNodes {
		list = append(list, node.String())
	}
	m.writeToFile(list, BlackFile)
}

// initWhiteList set white list nodes
func (m *DiscoverManager) initWhiteList() {
	path := filepath.Join(m.dataDir, WhiteFile)
	nodes := readFile(path)
	if nodes == nil || len(nodes) == 0 {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	for _, node := range nodes {
		nodeID, endpoint := ParseNodeString(node)
		if nodeID == nil {
			continue
		}
		key := nodeID.Hash()
		if _, ok := m.whiteNodes[key]; ok {
			continue
		}
		if _, ok := m.blackNodes[key]; ok {
			continue
		}
		m.whiteNodes[key] = newRawNode(nodeID, endpoint)
	}
}

// initDiscoverList read initial node from file
func (m *DiscoverManager) initDiscoverList() {
	path := filepath.Join(m.dataDir, FindFile)
	list := readFile(path)
	m.addDiscoverNodes(list)
}

// AddNewList for discovery
func (m *DiscoverManager) AddNewList(nodes []string) {
	m.addDiscoverNodes(nodes)
}

// writeFindNodeToFile write invalid node to file
func (m *DiscoverManager) writeFindNodeToFile() {
	// create list
	list := m.getAvailableNodes()
	// write find nodes to "findnode" file
	m.writeToFile(list, FindFile)
}

// writeToFile write node list to file
func (m *DiscoverManager) writeToFile(nodeList []string, fileName string) {
	path := filepath.Join(m.dataDir, fileName)
	// open or create file
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666) // first clear file to read and write
	if err != nil {
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Infof("Close file failed: %v", err)
		}
	}()

	// write file
	buf := bufio.NewWriter(f)
	for _, n := range nodeList {
		if _, err := buf.WriteString(n + "\n"); err != nil {
			log.Infof("Write file failed: %v", err)
		}
	}
	if err := buf.Flush(); err != nil {
		log.Infof("Write file failed: %v", err)
	}
}

// InWhiteList node in white list
func (m *DiscoverManager) InWhiteList(nodeID *NodeID) (ok bool) {
	key := nodeID.Hash()
	_, ok = m.whiteNodes[key]
	return
}

// ParseNodeString verify invalid
func ParseNodeString(node string) (*NodeID, string) {
	trunks := strings.Split(node, "@")
	if len(trunks) != 2 {
		return nil, ""
	}
	if len(trunks[0]) != 128 {
		return nil, ""
	}
	nodeID := BytesToNodeID(common.FromHex(trunks[0]))
	_, err := nodeID.PubKey()
	if err != nil {
		return nil, ""
	}
	if bytes.Compare(nodeID[:], deputynode.GetSelfNodeID()) == 0 {
		return nil, ""
	}
	if !verifyIP(trunks[1]) {
		return nil, ""
	}
	return nodeID, trunks[1]
}

// verifyIP  verify ipv4
func verifyIP(input string) bool {
	trunks := strings.Split(input, ":")
	if len(trunks) != 2 {
		return false
	}
	if ip := net.ParseIP(trunks[0]); ip == nil {
		return false
	}
	p, err := strconv.Atoi(trunks[1])
	if err != nil {
		return false
	}
	if p < 0 || p > 65535 {
		return false
	}
	return true
}
