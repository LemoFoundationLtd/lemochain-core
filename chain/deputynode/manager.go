package deputynode

import (
	"bytes"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"sync"
)

var (
	ErrNoDeputyInBlock       = errors.New("there is no deputy nodes in snapshot block")
	ErrInvalidDeputyRank     = errors.New("deputy nodes should be sorted by rank and start from 0")
	ErrInvalidDeputyVotes    = errors.New("there is a conflict between deputy node' rank and votes")
	ErrMissingTerm           = errors.New("some term is missing")
	ErrInvalidSnapshotHeight = errors.New("invalid snapshot block height")
	ErrNoTerms               = errors.New("can't access deputy nodes before SaveSnapshot")
	ErrQueryFutureTerm       = errors.New("can't query future term")
)

// Manager 代理节点管理器
type Manager struct {
	DeputyCount int // Max deputy count. Not include candidate nodes

	termList []*TermRecord
	lock     sync.Mutex

	evilDeputies map[common.Address]uint32 // key is minerAddress, value is release height
	edLock       sync.Mutex
}

// NewManager creates a new Manager. It is used to maintain term record list
func NewManager(deputyCount int) *Manager {
	return &Manager{
		DeputyCount:  deputyCount,
		termList:     make([]*TermRecord, 0),
		evilDeputies: make(map[common.Address]uint32),
	}
}

// IsEvilDeputyNode currentHeight is current block height
func (m *Manager) IsEvilDeputyNode(minerAddress common.Address, currentHeight uint32) bool {
	m.edLock.Lock()
	defer m.edLock.Unlock()
	if height, exit := m.evilDeputies[minerAddress]; exit {
		if currentHeight >= height {
			delete(m.evilDeputies, minerAddress)
			return false
		}
		return true
	}
	return false
}

// SetAbnormalDeputyNode height is release height
func (m *Manager) PutEvilDeputyNode(minerAddress common.Address, height uint32) {
	m.edLock.Lock()
	defer m.edLock.Unlock()
	m.evilDeputies[minerAddress] = height
}

// SaveSnapshot add deputy nodes record by snapshot block data
func (m *Manager) SaveSnapshot(snapshotHeight uint32, nodes DeputyNodes) {
	newTerm := NewTermRecord(snapshotHeight, nodes)

	m.lock.Lock()
	defer m.lock.Unlock()

	termCount := len(m.termList)
	if termCount == 0 {
		m.termList = append(m.termList, newTerm)
		log.Info("first term", "nodesCount", len(newTerm.Nodes))
		return
	}

	// expect term index is the last term index +1
	expectIndex := m.termList[termCount-1].TermIndex + 1
	if expectIndex < newTerm.TermIndex {
		log.Warn("Missing term", "expectIndex", expectIndex, "newTermIndex", newTerm.TermIndex)
		panic(ErrMissingTerm)
	} else if expectIndex == newTerm.TermIndex {
		// correct! congratulation~~~
		m.termList = append(m.termList, newTerm)
		log.Info("New term", "index", newTerm.TermIndex, "nodesCount", len(newTerm.Nodes))
	} else {
		// may exist when fork at term snapshot height
		m.termList[newTerm.TermIndex] = newTerm
		log.Info("Overwrite existed term", "index", newTerm.TermIndex, "nodesCount", len(newTerm.Nodes))
		if termCount > int(newTerm.TermIndex+1) {
			log.Warnf("Drop %d terms because the older term is overwritten", termCount-int(newTerm.TermIndex+1))
			m.termList = m.termList[:newTerm.TermIndex+1]
		}
	}
}

// GetTermByHeight 通过height获取对应的任期信息
func (m *Manager) GetTermByHeight(height uint32) (*TermRecord, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	termCount := len(m.termList)
	if m.termList == nil || termCount == 0 {
		return nil, ErrNoTerms
	}

	termIndex := GetTermIndexByHeight(height)
	if termCount > int(termIndex) {
		return m.termList[termIndex], nil
	} else {
		// the height is after last term
		log.Warn("Query future term", "current term count", termCount, "looking for term index", termIndex)
		return nil, ErrQueryFutureTerm
	}
}

// GetDeputiesByHeight 通过height获取对应的节点列表
func (m *Manager) GetDeputiesByHeight(height uint32) DeputyNodes {
	term, err := m.GetTermByHeight(height)
	if err != nil {
		// TODO
		panic(err)
		// m.lock.Lock()
		// term = m.termList[len(m.termList)-1]
		// m.lock.Unlock()
	}
	return term.GetDeputies(m.DeputyCount)
}

// GetDeputiesCount 获取共识节点数量
func (m *Manager) GetDeputiesCount(height uint32) int {
	nodes := m.GetDeputiesByHeight(height)
	return len(nodes)
}

// GetDeputyByAddress 获取address对应的节点
func (m *Manager) GetDeputyByAddress(height uint32, addr common.Address) *DeputyNode {
	nodes := m.GetDeputiesByHeight(height)
	for _, node := range nodes {
		if node.MinerAddress == addr {
			return node
		}
	}
	return nil
}

// GetDeputyByNodeID 根据nodeID获取对应的节点
func (m *Manager) GetDeputyByNodeID(height uint32, nodeID []byte) *DeputyNode {
	nodes := m.GetDeputiesByHeight(height)
	for _, node := range nodes {
		if bytes.Compare(node.NodeID, nodeID) == 0 {
			return node
		}
	}
	return nil
}

// GetMyDeputyInfo 获取自己在某一届高度的共识节点信息
func (m *Manager) GetMyDeputyInfo(height uint32) *DeputyNode {
	return m.GetDeputyByNodeID(height, GetSelfNodeID())
}

// IsSelfDeputyNode
func (m *Manager) IsSelfDeputyNode(height uint32) bool {
	return m.IsNodeDeputy(height, GetSelfNodeID())
}

// IsNodeDeputy
func (m *Manager) IsNodeDeputy(height uint32, nodeID []byte) bool {
	return m.GetDeputyByNodeID(height, nodeID) != nil
}
