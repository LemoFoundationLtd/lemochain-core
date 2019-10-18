package deputynode

import (
	"bytes"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"math"
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

type BlockLoader interface {
	GetBlockByHeight(height uint32) (*types.Block, error)
}

// Manager 代理节点管理器
type Manager struct {
	DeputyCount int // Max deputy count. Not include candidate nodes

	termList []*TermRecord
	lock     sync.RWMutex

	evilDeputies map[common.Address]uint32 // key is minerAddress, value is release height(release height = block height + InterimDuration)
	edLock       sync.Mutex
}

// NewManager creates a new Manager. It is used to maintain term record list
func NewManager(deputyCount int, blockLoader BlockLoader) *Manager {
	manager := &Manager{
		DeputyCount:  deputyCount,
		termList:     make([]*TermRecord, 0),
		evilDeputies: make(map[common.Address]uint32),
	}
	manager.init(blockLoader)
	return manager
}

// initDeputyNodes init deputy nodes information
func (m *Manager) init(blockLoader BlockLoader) {
	snapshotHeight := uint32(0)
	for ; ; snapshotHeight += params.TermDuration {
		block, err := blockLoader.GetBlockByHeight(snapshotHeight)
		if err != nil {
			if err == store.ErrNotExist {
				break
			}
			log.Errorf("Load snapshot block error: %v", err)
			panic(err)
		}

		m.SaveSnapshot(snapshotHeight, block.DeputyNodes)
	}

	if snapshotHeight == 0 {
		log.Warn("Deputy manager is ready. But there is no genesis block, so no deputies")
	} else {
		lastSnapshotHeight := snapshotHeight - params.TermDuration
		currentDeputyCount := m.GetDeputiesCount(lastSnapshotHeight + params.InterimDuration + 1)
		log.Info("Deputy manager is ready", "lastSnapshotHeight", snapshotHeight, "deputyCount", currentDeputyCount)
	}
}

// IsEvilDeputyNode currentHeight is current block height
func (m *Manager) IsEvilDeputyNode(minerAddress common.Address, currentHeight uint32) bool {
	m.edLock.Lock()
	defer m.edLock.Unlock()
	if height, exist := m.evilDeputies[minerAddress]; exist {
		if currentHeight >= height {
			delete(m.evilDeputies, minerAddress)
			return false
		}
		return true
	}
	return false
}

// SetAbnormalDeputyNode height is block height
func (m *Manager) PutEvilDeputyNode(minerAddress common.Address, blockHeight uint32) {
	m.edLock.Lock()
	defer m.edLock.Unlock()
	m.evilDeputies[minerAddress] = blockHeight + params.ReleaseEvilNodeDuration
}

// SaveSnapshot add deputy nodes record by snapshot block data
func (m *Manager) SaveSnapshot(snapshotHeight uint32, nodes types.DeputyNodes) {
	newTerm := NewTermRecord(snapshotHeight, nodes)

	m.lock.Lock()
	defer m.lock.Unlock()

	termCount := len(m.termList)
	if termCount == 0 {
		m.termList = append(m.termList, newTerm)
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
	log.Debug("save new term", "deputies", log.Lazy{Fn: func() string {
		return nodes.String()
	}})
}

// GetTermByHeight 通过height获取对应的任期信息
func (m *Manager) GetTermByHeight(height uint32) (*TermRecord, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	termCount := len(m.termList)
	if m.termList == nil || termCount == 0 {
		return nil, ErrNoTerms
	}

	termIndex := GetTermIndexByHeight(height)
	if termCount > int(termIndex) {
		return m.termList[termIndex], nil
	} else {
		// the height is after last term
		log.Warn("Query future term", "current stable term count", termCount, "looking for term index", termIndex, "query height", height)
		return nil, ErrQueryFutureTerm
	}
}

// GetDeputiesByHeight 通过height获取对应的节点列表
func (m *Manager) GetDeputiesByHeight(height uint32) types.DeputyNodes {
	term, err := m.GetTermByHeight(height)
	if err != nil {
		// panic(err)
		return types.DeputyNodes{}
	}
	return term.GetDeputies(m.DeputyCount)
}

// GetDeputiesCount 获取共识节点数量
func (m *Manager) GetDeputiesCount(height uint32) int {
	nodes := m.GetDeputiesByHeight(height)
	return len(nodes)
}

// TwoThirdDeputyCount return the deputy nodes count * 2/3
func (m *Manager) TwoThirdDeputyCount(height uint32) uint32 {
	nodes := m.GetDeputiesByHeight(height)
	return uint32(math.Ceil(float64(len(nodes)) * 2.0 / 3.0))
}

// GetDeputyByAddress 获取address对应的节点
func (m *Manager) GetDeputyByAddress(height uint32, addr common.Address) *types.DeputyNode {
	nodes := m.GetDeputiesByHeight(height)
	for _, node := range nodes {
		if node.MinerAddress == addr {
			return node
		}
	}
	return nil
}

// GetDeputyByNodeID 根据nodeID获取对应的节点
func (m *Manager) GetDeputyByNodeID(height uint32, nodeID []byte) *types.DeputyNode {
	nodes := m.GetDeputiesByHeight(height)
	for _, node := range nodes {
		if bytes.Compare(node.NodeID, nodeID) == 0 {
			return node
		}
	}
	return nil
}

// GetMyDeputyInfo 获取自己在某一届高度的共识节点信息
func (m *Manager) GetMyDeputyInfo(height uint32) *types.DeputyNode {
	return m.GetDeputyByNodeID(height, GetSelfNodeID())
}

// GetMyMinerAddress 获取自己在某一届高度的矿工账号
func (m *Manager) GetMyMinerAddress(height uint32) (common.Address, bool) {
	deputy := m.GetDeputyByNodeID(height, GetSelfNodeID())
	if deputy != nil {
		return deputy.MinerAddress, true
	}
	return common.Address{}, false
}

// IsSelfDeputyNode
func (m *Manager) IsSelfDeputyNode(height uint32) bool {
	return m.IsNodeDeputy(height, GetSelfNodeID())
}

// IsNodeDeputy
func (m *Manager) IsNodeDeputy(height uint32, nodeID []byte) bool {
	return m.GetDeputyByNodeID(height, nodeID) != nil
}
