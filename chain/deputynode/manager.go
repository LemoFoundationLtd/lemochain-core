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
	ErrNotDeputy             = errors.New("not a deputy address in specific height")
	ErrMineGenesis           = errors.New("can not mine genesis block")
)

// Manager 代理节点管理器
type Manager struct {
	DeputyCount int // Max deputy count. Not include candidate nodes

	termList []*TermRecord
	lock     sync.Mutex
}

// NewManager creates a new Manager. It is used to maintain term record list
func NewManager(deputyCount int) *Manager {
	return &Manager{
		DeputyCount: deputyCount,
		termList:    make([]*TermRecord, 0),
	}
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
		log.Warn("missing term", "expectIndex", expectIndex, "newTermIndex", newTerm.TermIndex)
		panic(ErrMissingTerm)
	} else if expectIndex == newTerm.TermIndex {
		// correct! congratulation~~~
		m.termList = append(m.termList, newTerm)
		log.Info("new term", "index", newTerm.TermIndex, "nodesCount", len(newTerm.Nodes))
	} else {
		// may exist when fork at term snapshot height
		m.termList[newTerm.TermIndex] = newTerm
		log.Info("overwrite existed term", "index", newTerm.TermIndex, "nodesCount", len(newTerm.Nodes))
		if termCount > int(newTerm.TermIndex+1) {
			log.Warnf("drop %d terms because the older term is overwritten", termCount-int(newTerm.TermIndex+1))
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
		log.Warn("query future term", "current term count", termCount, "looking for term index", termIndex)
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
	return findDeputyByAddress(nodes, addr)
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

func findDeputyByAddress(deputies []*DeputyNode, addr common.Address) *DeputyNode {
	for _, node := range deputies {
		if node.MinerAddress == addr {
			return node
		}
	}
	return nil
}

// GetMinerDistance get miner index distance in same term
func (m *Manager) GetMinerDistance(targetHeight uint32, lastBlockMiner, targetMiner common.Address) (uint32, error) {
	if targetHeight == 0 {
		return 0, ErrMineGenesis
	}
	termDeputies := m.GetDeputiesByHeight(targetHeight)

	// find target block miner deputy
	targetDeputy := findDeputyByAddress(termDeputies, targetMiner)
	if targetDeputy == nil {
		return 0, ErrNotDeputy
	}

	// only one deputy
	nodeCount := uint32(len(termDeputies))
	if nodeCount == 1 {
		return 1, nil
	}

	// Genesis block is pre-set, not belong to any deputy node. So only blocks start with height 1 is mined by deputies
	// The reward block changes deputy nodes, so we need recompute the slot
	if targetHeight == 1 || IsRewardBlock(targetHeight) {
		return targetDeputy.Rank + 1, nil
	}

	// find last block miner deputy
	lastDeputy := findDeputyByAddress(termDeputies, lastBlockMiner)
	if lastDeputy == nil {
		return 0, ErrNotDeputy
	}

	return (targetDeputy.Rank - lastDeputy.Rank + nodeCount) % nodeCount, nil
}

// IsSelfDeputyNode
func (m *Manager) IsSelfDeputyNode(height uint32) bool {
	return m.IsNodeDeputy(height, GetSelfNodeID())
}

// IsNodeDeputy
func (m *Manager) IsNodeDeputy(height uint32, nodeID []byte) bool {
	return m.GetDeputyByNodeID(height, nodeID) != nil
}
