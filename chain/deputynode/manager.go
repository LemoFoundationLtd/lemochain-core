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
	ErrNoStableTerm          = errors.New("term is not stable")
	ErrMineGenesis           = errors.New("can not mine genesis block")
	ErrNotDeputy             = errors.New("the miner address is not a deputy")
	ErrInvalidDistance       = errors.New("deputy distance should be greater than 0")
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
			if err == store.ErrBlockNotExist {
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

// PutEvilDeputyNode put a deputy node into blacklist for a while
func (m *Manager) PutEvilDeputyNode(minerAddress common.Address, blockHeight uint32) {
	m.edLock.Lock()
	defer m.edLock.Unlock()
	m.evilDeputies[minerAddress] = blockHeight + params.ReleaseEvilNodeDuration
}

// IsEvilDeputyNode test if a deputy node is in blacklist
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

// GetTermByHeight 通过height获取对应的任期信息。若onlyBlockSigner为true则获取当前可签名的节点的任期信息，false则获取当前已当选的节点的任期信息
func (m *Manager) GetTermByHeight(height uint32, onlyBlockSigner bool) (*TermRecord, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	termCount := len(m.termList)
	if m.termList == nil || termCount == 0 {
		log.Warn("No terms", "query height", height)
		return nil, ErrNoStableTerm
	}

	var termIndex uint32
	if onlyBlockSigner {
		termIndex = GetSignerTermIndexByHeight(height)
	} else {
		termIndex = GetDeputyTermIndexByHeight(height)
	}
	if termCount > int(termIndex) {
		return m.termList[termIndex], nil
	} else {
		log.Warn("Term is not stable", "stableTermCount", termCount, "queryTermIndex", termIndex, "queryHeight", height, "needStableHeight", uint32(termCount+1)*params.TermDuration)
		return nil, ErrNoStableTerm
	}
}

// GetDeputiesByHeight 通过height获取对应的节点列表。若onlyBlockSigner为true则获取当前可签名的节点的任期信息，false则获取当前已当选的节点的任期信息
func (m *Manager) GetDeputiesByHeight(height uint32, onlyBlockSigner bool) types.DeputyNodes {
	term, err := m.GetTermByHeight(height, onlyBlockSigner)
	if err != nil {
		// panic(err)
		return types.DeputyNodes{}
	}
	return term.GetDeputies(m.DeputyCount)
}

// GetDeputiesCount 获取共识节点数量
func (m *Manager) GetDeputiesCount(height uint32) int {
	nodes := m.GetDeputiesByHeight(height, true)
	return len(nodes)
}

// TwoThirdDeputyCount return the deputy nodes count * 2/3
func (m *Manager) TwoThirdDeputyCount(height uint32) uint32 {
	nodes := m.GetDeputiesByHeight(height, true)
	return uint32(math.Ceil(float64(len(nodes)) * 2.0 / 3.0))
}

// GetDeputyByAddress 获取address对应的节点
func (m *Manager) GetDeputyByAddress(height uint32, addr common.Address) *types.DeputyNode {
	nodes := m.GetDeputiesByHeight(height, true)
	for _, node := range nodes {
		if node.MinerAddress == addr {
			return node
		}
	}
	return nil
}

// GetDeputyByNodeID 根据nodeID获取对应的节点
func (m *Manager) GetDeputyByNodeID(height uint32, nodeID []byte) *types.DeputyNode {
	nodes := m.GetDeputiesByHeight(height, true)
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

// GetDeputyByDistance find a deputy from parent block miner by miner index distance. The distance should always greater than 0
func (m *Manager) GetDeputyByDistance(targetHeight uint32, parentBlockMiner common.Address, distance uint32) (*types.DeputyNode, error) {
	if targetHeight == 0 {
		panic(ErrMineGenesis)
	}
	if distance < 1 {
		panic(ErrInvalidDistance)
	}
	deputies := m.GetDeputiesByHeight(targetHeight, true)
	nodeCount := uint32(len(deputies))
	if nodeCount == 0 {
		return nil, ErrNotDeputy
	}

	if targetHeight == 1 || IsRewardBlock(targetHeight) {
		// Genesis block is pre-set, not belong to any deputy node. So only blocks start with height 1 is mined by deputies
		// The reward block changes deputy nodes, so we need recompute the slot
		targetIndex := (distance - 1 + nodeCount) % nodeCount
		return deputies[targetIndex], nil
	} else {
		// find parent block miner deputy after the IsRewardBlock logic, to make the distance calculation correct in the scene of crossing terms
		for index, node := range deputies {
			if node.MinerAddress == parentBlockMiner {
				targetIndex := (uint32(index) + distance + nodeCount) % nodeCount
				return deputies[targetIndex], nil
			}
		}
	}
	return nil, ErrNotDeputy
}

// GetMinerDistance get miner index distance. It is always greater than 0 and not greater than deputy count
func (m *Manager) GetMinerDistance(targetHeight uint32, parentBlockMiner, targetMiner common.Address) (uint32, error) {
	if targetHeight == 0 {
		panic(ErrMineGenesis)
	}
	deputies := m.GetDeputiesByHeight(targetHeight, true)
	nodeCount := uint32(len(deputies))

	// find target block miner deputy
	targetDeputy := findDeputyByAddress(deputies, targetMiner)
	if targetDeputy == nil {
		return 0, ErrNotDeputy
	}

	// Genesis block is pre-set, not belong to any deputy node. So only blocks start with height 1 is mined by deputies
	// The reward block changes deputy nodes, so we need recompute the slot
	if targetHeight == 1 || IsRewardBlock(targetHeight) {
		return targetDeputy.Rank + 1, nil
	}

	// if they are same miner, then return deputy count
	if targetMiner == parentBlockMiner {
		return nodeCount, nil
	}

	// find last block miner deputy
	lastDeputy := findDeputyByAddress(deputies, parentBlockMiner)
	if lastDeputy == nil {
		return 0, ErrNotDeputy
	}
	return (nodeCount + targetDeputy.Rank - lastDeputy.Rank) % nodeCount, nil
}

func findDeputyByAddress(deputies []*types.DeputyNode, addr common.Address) *types.DeputyNode {
	for _, node := range deputies {
		if node.MinerAddress == addr {
			return node
		}
	}
	return nil
}
