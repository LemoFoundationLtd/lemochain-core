package deputynode

import (
	"bytes"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"sync"
)

const (
	// max deputy count
	TotalCount = 5
)

var (
	ErrEmptyDeputies         = errors.New("can't save empty deputy nodes")
	ErrExistSnapshotHeight   = errors.New("exist snapshot block height")
	ErrInvalidSnapshotHeight = errors.New("invalid snapshot block height")
	ErrNoDeputies            = errors.New("can't access deputy nodes before SaveSnapshot")
)

//go:generate gencodec -type TermRecord --field-override termRecordMarshaling -out gen_term_record_json.go
type TermRecord struct {
	// 0, 100W+1K+1, 200W+1K+1, 300W+1K+1, 400W+1K+1...
	StartHeight uint32      `json:"height"`
	Nodes       DeputyNodes `json:"nodes"`
}

type termRecordMarshaling struct {
	StartHeight hexutil.Uint32
}

// Manager 代理节点管理器
type Manager struct {
	termList []*TermRecord
	lock     sync.Mutex
}

var managerInstance = &Manager{
	termList: make([]*TermRecord, 0, 1),
}

func Instance() *Manager {
	return managerInstance
}

// SaveSnapshot add deputy nodes record by snapshot block data
func (m *Manager) SaveSnapshot(snapshotHeight uint32, nodes DeputyNodes) {
	// check nodes to make sure it is not empty
	if nodes == nil || len(nodes) == 0 {
		log.Error("can't save empty deputy nodes", "height", snapshotHeight)
		panic(ErrEmptyDeputies)
	}

	// compute term start block height
	var termStart uint32
	if snapshotHeight == 0 {
		termStart = 0
	} else {
		termStart = snapshotHeight + params.InterimDuration + 1
	}
	record := &TermRecord{StartHeight: termStart, Nodes: nodes}

	// save
	if err := m.addDeputyRecord(record); err != nil {
		if err == ErrExistSnapshotHeight {
			log.Warn("ignore exist snapshot block error")
		} else {
			panic(err)
		}
	}
}

// addDeputyRecord add a deputy nodes record
func (m *Manager) addDeputyRecord(record *TermRecord) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	// TODO may exist when fork at term snapshot height
	for _, term := range m.termList {
		if record.StartHeight <= term.StartHeight {
			log.Warn("exist snapshot block height", "new record height", record.StartHeight, "exit height", term.StartHeight)
			return ErrExistSnapshotHeight
		}
	}
	// TODO if check skip term
	m.termList = append(m.termList, record)

	log.Info("new deputy nodes", "start height", record.StartHeight, "nodes count", len(record.Nodes))
	return nil
}

// GetDeputiesByHeight 通过height获取对应的节点列表
func (m *Manager) GetDeputiesByHeight(height uint32, total bool) DeputyNodes {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.termList == nil || len(m.termList) == 0 {
		panic(ErrNoDeputies)
	}

	// find record
	var record *TermRecord
	if len(m.termList) == 1 {
		record = m.termList[0]
	} else {
		for i := 0; i < len(m.termList)-1; i++ {
			nextTermStart := m.termList[i+1].StartHeight
			// the height is after next term
			if height >= nextTermStart {
				continue
			}
			// the height is in current term
			record = m.termList[i]
			break
		}
		// the height is after last term
		if record == nil {
			record = m.termList[len(m.termList)-1]
		}
	}

	// find nodes
	nodes := record.Nodes
	// if not total, then result nodes must be less than TotalCount
	if !total && len(nodes) > TotalCount {
		nodes = nodes[:TotalCount]
	}

	return nodes
}

// GetDeputiesCount 获取共识节点数量
func (m *Manager) GetDeputiesCount(height uint32) int {
	nodes := m.GetDeputiesByHeight(height, false)
	return len(nodes)
}

// GetDeputyByAddress 获取address对应的节点
func (m *Manager) GetDeputyByAddress(height uint32, addr common.Address) *DeputyNode {
	nodes := m.GetDeputiesByHeight(height, false)
	for _, node := range nodes {
		if node.MinerAddress == addr {
			return node
		}
	}
	return nil
}

// GetDeputyByNodeID 根据nodeID获取对应的节点
func (m *Manager) GetDeputyByNodeID(height uint32, nodeID []byte) *DeputyNode {
	nodes := m.GetDeputiesByHeight(height, false)
	for _, node := range nodes {
		if bytes.Compare(node.NodeID, nodeID) == 0 {
			return node
		}
	}
	return nil
}

// GetSlot 获取最新块的出块者序号与本节点序号差
func (m *Manager) GetSlot(height uint32, firstAddress, nextAddress common.Address) int {
	firstNode := m.GetDeputyByAddress(height, firstAddress)
	nextNode := m.GetDeputyByAddress(height, nextAddress)
	if ((height == 1) || (height > params.TermDuration && height%params.TermDuration == params.InterimDuration+1)) && nextNode != nil {
		log.Debugf("GetSlot: change term. rank: %d", nextNode.Rank)
		return int(nextNode.Rank + 1)
	}
	if firstNode == nil || nextNode == nil {
		return -1
	}
	nodeCount := m.GetDeputiesCount(height)
	// 只有一个主节点
	if nodeCount == 1 {
		log.Debug("getSlot: only one star node")
		return 1
	}
	return (int(nextNode.Rank) - int(firstNode.Rank) + nodeCount) % nodeCount
}

// IsRewardBlock 是否该发出块奖励了
func (m *Manager) IsRewardBlock(height uint32) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	if height < params.TermDuration+params.InterimDuration+1 {
		// in genesis term
		return false
	} else if height%params.TermDuration == params.InterimDuration+1 {
		// term start block
		return true
	} else {
		// other normal block
		return false
	}
}

// IsSelfDeputyNode
func (m *Manager) IsSelfDeputyNode(height uint32) bool {
	node := m.GetDeputyByNodeID(height, GetSelfNodeID())
	return node != nil
}

// IsNodeDeputy
func (m *Manager) IsNodeDeputy(height uint32, nodeID []byte) bool {
	node := m.GetDeputyByNodeID(height, nodeID)
	return node != nil
}

// Clear for test
func (m *Manager) Clear() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.termList = make([]*TermRecord, 0, 1)
}

// Clear for test
func (m *Manager) GetTermList() []*TermRecord {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.termList[:]
}

// TODO move this out
// GetDeputiesInCharge for api
func (m *Manager) GetDeputiesInCharge(currentHeight uint32) []string {
	nodes := m.GetDeputiesByHeight(currentHeight, true)

	res := make([]string, 0)
	for _, n := range nodes {
		res = append(res, n.NodeAddrString())
	}
	return res
}
