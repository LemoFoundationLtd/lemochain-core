package deputynode

import (
	"crypto/ecdsa"
	"crypto/rand"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto/secp256k1"
	"github.com/stretchr/testify/assert"
	"math/big"
	"net"
	"testing"
)

var (
	testDeputies = GenerateDeputies(17)
)

// GenerateDeputies generate random deputy nodes
func GenerateDeputies(num int) DeputyNodes {
	var result []*DeputyNode
	for i := 0; i < num; i++ {
		private, _ := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
		result = append(result, &DeputyNode{
			MinerAddress: crypto.PubkeyToAddress(private.PublicKey),
			NodeID:       (crypto.FromECDSAPub(&private.PublicKey))[1:],
			IP:           net.IPv4(127, 0, 0, byte(i%256)),
			Port:         uint32(i % 9999),
			Rank:         uint32(i),
			Votes:        big.NewInt(int64(10000000000 - i)),
		})
	}
	return result
}

// pickNodes picks some test deputy nodes by index
func pickNodes(nodeIndexList ...int) DeputyNodes {
	var result []*DeputyNode
	for i := range nodeIndexList {
		result = append(result, testDeputies[i])
	}
	return result
}

func TestManager_SaveSnapshot(t *testing.T) {
	m := Instance()
	m.Clear()

	// save genesis
	height := uint32(0)
	nodes := pickNodes(0, 1)
	m.SaveSnapshot(height, nodes)
	assert.Len(t, m.termList, 1)
	assert.Equal(t, uint32(0), m.termList[0].StartHeight)
	assert.Equal(t, nodes, m.termList[0].Nodes)

	// save snapshot
	height = uint32(params.TermDuration * 1)
	nodes = pickNodes(2)
	m.SaveSnapshot(height, nodes)
	assert.Len(t, m.termList, 2)
	assert.Equal(t, uint32(0), m.termList[0].StartHeight)
	assert.Equal(t, height+params.InterimDuration+1, m.termList[1].StartHeight)
	assert.Equal(t, nodes, m.termList[1].Nodes)

	// save exist node
	height = uint32(params.TermDuration * 2)
	nodes = pickNodes(1, 3)
	m.SaveSnapshot(height, nodes)
	assert.Len(t, m.termList, 3)
	assert.Equal(t, height+params.InterimDuration+1, m.termList[2].StartHeight)
	assert.Equal(t, nodes, m.termList[2].Nodes)

	// save nothing
	height = uint32(params.TermDuration * 3)
	nodes = pickNodes()
	assert.PanicsWithValue(t, ErrEmptyDeputies, func() {
		m.SaveSnapshot(height, nodes)
	})

	// save exist snapshot height
	height = uint32(params.TermDuration * 2)
	nodes = pickNodes(4)
	m.SaveSnapshot(height, nodes)
	assert.Len(t, m.termList, 3)

	// save skipped snapshot
	height = uint32(params.TermDuration * 4)
	nodes = pickNodes(4)
	assert.PanicsWithValue(t, ErrInvalidSnapshotHeight, func() {
		m.SaveSnapshot(height, nodes)
	})

	// save invalid snapshot height
	height = uint32(params.TermDuration*2 + 1)
	nodes = pickNodes(4)
	assert.PanicsWithValue(t, ErrInvalidSnapshotHeight, func() {
		m.SaveSnapshot(height, nodes)
	})
}

func TestManager_GetDeputiesByHeight(t *testing.T) {
	m := Instance()
	m.Clear()

	// no any terms
	assert.PanicsWithValue(t, ErrNoDeputies, func() {
		m.GetDeputiesByHeight(0, false)
	})

	nodes0 := pickNodes(0, 1)
	m.SaveSnapshot(0, nodes0)
	nodes1 := pickNodes(0, 1, 2)
	m.SaveSnapshot(params.TermDuration*1, nodes1)
	nodes2 := pickNodes(1, 2, 3, 4, 5, 6)
	m.SaveSnapshot(params.TermDuration*2, nodes2)

	// genesis term
	assert.Equal(t, nodes0, m.GetDeputiesByHeight(0, false))
	assert.Equal(t, nodes0, m.GetDeputiesByHeight(0, true))
	assert.Equal(t, nodes0, m.GetDeputiesByHeight(1, false))
	assert.Equal(t, nodes0, m.GetDeputiesByHeight(1, true))
	assert.Equal(t, nodes0, m.GetDeputiesByHeight(params.TermDuration+params.InterimDuration, false))
	assert.Equal(t, nodes0, m.GetDeputiesByHeight(params.TermDuration+params.InterimDuration, true))

	// second term
	assert.Equal(t, nodes1, m.GetDeputiesByHeight(params.TermDuration+params.InterimDuration+1, false))
	assert.Equal(t, nodes1, m.GetDeputiesByHeight(params.TermDuration+params.InterimDuration+1, true))
	assert.Equal(t, nodes1, m.GetDeputiesByHeight(params.TermDuration*2+params.InterimDuration, false))
	assert.Equal(t, nodes1, m.GetDeputiesByHeight(params.TermDuration*2+params.InterimDuration, true))

	// third term
	assert.Equal(t, nodes2[:TotalCount], m.GetDeputiesByHeight(params.TermDuration*2+params.InterimDuration+1, false))
	assert.Equal(t, nodes2, m.GetDeputiesByHeight(params.TermDuration*2+params.InterimDuration+1, true))

	// current term
	assert.Equal(t, nodes2, m.GetDeputiesByHeight(1000000000, true))
}

func TestManager_GetDeputyByAddress(t *testing.T) {
	m := Instance()
	m.Clear()

	nodes := pickNodes(0, 1, 2)
	m.SaveSnapshot(0, nodes)

	assert.Equal(t, nodes[0], m.GetDeputyByAddress(0, testDeputies[0].MinerAddress))
	assert.Equal(t, nodes[2], m.GetDeputyByAddress(0, testDeputies[2].MinerAddress))
	// not exist
	assert.Nil(t, m.GetDeputyByAddress(0, testDeputies[5].MinerAddress))
	assert.Nil(t, m.GetDeputyByAddress(0, common.Address{}))
}

func TestManager_GetDeputyByNodeID(t *testing.T) {
	m := Instance()
	m.Clear()

	nodes := pickNodes(0, 1, 2)
	m.SaveSnapshot(0, nodes)

	assert.Equal(t, nodes[0], m.GetDeputyByNodeID(0, testDeputies[0].NodeID))
	assert.Equal(t, nodes[2], m.GetDeputyByNodeID(0, testDeputies[2].NodeID))
	// not exist
	assert.Nil(t, m.GetDeputyByNodeID(0, testDeputies[5].NodeID))
	assert.Nil(t, m.GetDeputyByNodeID(0, []byte{}))
	assert.Nil(t, m.GetDeputyByNodeID(0, nil))
}

// TODO
func TestManager_GetSlot(t *testing.T) {
	m := Instance()
	m.Clear()

	nodes00 := pickNodes(0, 1, 2)
	m.SaveSnapshot(0, nodes00) // 创建3个代理节点的节点列表，列表高度为0

	nodes01 := pickNodes(0, 1, 2, 3, 4)
	m.SaveSnapshot(params.TermDuration, nodes01) // 创建5个代理节点的节点列表，高度为100000

	// 测试height==1的情况，此情况为上一个块为创世块
	assert.Equal(t, 1, m.GetSlot(1, common.Address{}, testDeputies[0].MinerAddress))
	assert.Equal(t, 2, m.GetSlot(1, common.Address{}, testDeputies[1].MinerAddress))
	assert.Equal(t, 3, m.GetSlot(1, common.Address{}, testDeputies[2].MinerAddress))

	// 测试换届的情况
	assert.Equal(t, 1, m.GetSlot(params.TermDuration+params.InterimDuration+1, common.Address{}, testDeputies[0].MinerAddress))
	assert.Equal(t, 2, m.GetSlot(params.TermDuration+params.InterimDuration+1, common.Address{}, testDeputies[1].MinerAddress))
	assert.Equal(t, 3, m.GetSlot(params.TermDuration+params.InterimDuration+1, common.Address{}, testDeputies[2].MinerAddress))

	// 测试firstNode和NextNode为空的情况
	assert.Equal(t, -1, m.GetSlot(22, common.Address{}, common.Address{}))

	// 测试只有一个共识节点的情况
	m.Clear()
	nodes03 := pickNodes(0) // 生成只有一个共识节点的节点列表
	m.SaveSnapshot(1, nodes03)
	assert.Equal(t, 1, m.GetSlot(1, testDeputies[0].MinerAddress, testDeputies[0].MinerAddress))

	// 正常情况下
	m.Clear()
	nodes04 := pickNodes(0, 1, 2, 3, 4)
	m.SaveSnapshot(1, nodes04)
	assert.Equal(t, 4, m.GetSlot(11, testDeputies[0].MinerAddress, testDeputies[4].MinerAddress))
	assert.Equal(t, 3, m.GetSlot(11, testDeputies[1].MinerAddress, testDeputies[4].MinerAddress))
	assert.Equal(t, 2, m.GetSlot(11, testDeputies[1].MinerAddress, testDeputies[3].MinerAddress))
	assert.Equal(t, 0, m.GetSlot(11, testDeputies[0].MinerAddress, testDeputies[0].MinerAddress))
}

func TestManager_IsRewardBlock(t *testing.T) {
	m := Instance()
	m.Clear()

	m.SaveSnapshot(0, pickNodes(0))
	m.SaveSnapshot(params.TermDuration*1, pickNodes(0))
	m.SaveSnapshot(params.TermDuration*2, pickNodes(0))

	assert.Equal(t, false, m.IsRewardBlock(0))
	assert.Equal(t, false, m.IsRewardBlock(1))
	assert.Equal(t, false, m.IsRewardBlock(params.TermDuration))
	assert.Equal(t, true, m.IsRewardBlock(params.TermDuration+params.InterimDuration+1))
	assert.Equal(t, true, m.IsRewardBlock(params.TermDuration*2+params.InterimDuration+1))
	assert.Equal(t, false, m.IsRewardBlock(params.TermDuration*2+params.InterimDuration+2))
	assert.Equal(t, true, m.IsRewardBlock(params.TermDuration*3+params.InterimDuration+1))
}

func Test_GetDeputiesInCharge(t *testing.T) {
	m := Instance()
	m.Clear()

	nodes := pickNodes(1, 2, 3)
	m.SaveSnapshot(0, nodes)

	assert.Len(t, m.GetDeputiesInCharge(1), 3)
}
