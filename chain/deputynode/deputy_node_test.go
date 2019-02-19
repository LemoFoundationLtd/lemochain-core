package deputynode

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto/secp256k1"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

const (
	block01MinerAddress = "0x015780F8456F9c1532645087a19DcF9a7e0c7F97"
	deputy01Privkey     = "0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"
	block02MinerAddress = "0x016ad4Fc7e1608685Bf5fe5573973BF2B1Ef9B8A"
	deputy02Privkey     = "0x9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb"
	block03MinerAddress = "0x01f98855Be9ecc5c23A28Ce345D2Cc04686f2c61"
	deputy03Privkey     = "0xba9b51e59ec57d66b30b9b868c76d6f4d386ce148d9c6c1520360d92ef0f27ae"
	block04MinerAddress = "0x0112fDDcF0C08132A5dcd9ED77e1a3348ff378D2"
	deputy04Privkey     = "0xb381bad69ad4b200462a0cc08fcb8ba64d26efd4f49933c2c2448cb23f2cd9d0"
	block05MinerAddress = "0x016017aF50F4bB67101CE79298ACBdA1A3c12C15"
	deputy05Privkey     = "0x56b5fe1b8c40f0dec29b621a16ffcbc7a1bb5c0b0f910c5529f991273cd0569c"
)

// Test_SelfNodeKey 测试GetSelfNodeKey、GetSelfNodeID、SetSelfNodeKey
func Test_SelfNodeKey(t *testing.T) {

	key := func() *ecdsa.PrivateKey {
		pri, _ := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
		return pri
	}()
	SetSelfNodeKey(key)
	assert.Equal(t, key, GetSelfNodeKey())
	assert.Equal(t, crypto.FromECDSAPub(&key.PublicKey)[1:], GetSelfNodeID())
}

// NewDeputyNode 创建一个代理节点
func NewDeputyNode(MinerAddress common.Address, nodeID []byte, port uint32) *DeputyNode {
	return &DeputyNode{
		MinerAddress: MinerAddress,
		NodeID:       nodeID,
		IP:           nil,
		Port:         port,
		Rank:         0,
		Votes:        big.NewInt(0),
	}
}

// TestDeputyNode_Check
func TestDeputyNode_Check(t *testing.T) {

	node01 := NewDeputyNode(common.HexToAddress("Lemo838888888888888888888888888888888888"),
		common.FromHex("0x5e3600755f9b512a65603b38e30885c98845bf37a3b437831871b48fd3"),
		7002)
	assert.Equal(t, errors.New("incorrect field: 'NodeID'"), node01.Check())

	node02 := NewDeputyNode(common.HexToAddress("Lemo888888888888888888888888888888888888"),
		common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"),
		666666)
	assert.Equal(t, errors.New("max deputy node's port is 65535"), node02.Check())

	node03 := NewDeputyNode(common.HexToAddress("Lemo888888888888888888888888888888888888"),
		common.FromHex("0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0"),
		7002)
	assert.Nil(t, node03.Check())
}

// deputyNodes 初始化代理节点,numNode为选择共识节点数量，取值为[1,5],height为发放奖励高度
func deputyNodes(nodeNum int) (DeputyNodes, error) {

	privarte01, err := crypto.ToECDSA(common.FromHex(deputy01Privkey))
	if err != nil {
		return nil, err
	}
	privarte02, err := crypto.ToECDSA(common.FromHex(deputy02Privkey))
	if err != nil {
		return nil, err
	}
	privarte03, err := crypto.ToECDSA(common.FromHex(deputy03Privkey))
	if err != nil {
		return nil, err
	}
	privarte04, err := crypto.ToECDSA(common.FromHex(deputy04Privkey))
	if err != nil {
		return nil, err
	}
	privarte05, err := crypto.ToECDSA(common.FromHex(deputy05Privkey))
	if err != nil {
		return nil, err
	}

	var nodes = make([]*DeputyNode, 5)
	nodes[0] = &DeputyNode{
		MinerAddress: common.HexToAddress(block01MinerAddress),
		NodeID:       (crypto.FromECDSAPub(&privarte01.PublicKey))[1:],
		IP:           nil,
		Port:         7001,
		Rank:         0,
		Votes:        big.NewInt(120),
	}
	nodes[1] = &DeputyNode{
		MinerAddress: common.HexToAddress(block02MinerAddress),
		NodeID:       (crypto.FromECDSAPub(&privarte02.PublicKey))[1:],
		IP:           nil,
		Port:         7002,
		Rank:         1,
		Votes:        big.NewInt(110),
	}
	nodes[2] = &DeputyNode{
		MinerAddress: common.HexToAddress(block03MinerAddress),
		NodeID:       (crypto.FromECDSAPub(&privarte03.PublicKey))[1:],
		IP:           nil,
		Port:         7003,
		Rank:         2,
		Votes:        big.NewInt(100),
	}
	nodes[3] = &DeputyNode{
		MinerAddress: common.HexToAddress(block04MinerAddress),
		NodeID:       (crypto.FromECDSAPub(&privarte04.PublicKey))[1:],
		IP:           nil,
		Port:         7004,
		Rank:         3,
		Votes:        big.NewInt(90),
	}
	nodes[4] = &DeputyNode{
		MinerAddress: common.HexToAddress(block05MinerAddress),
		NodeID:       (crypto.FromECDSAPub(&privarte05.PublicKey))[1:],
		IP:           nil,
		Port:         7005,
		Rank:         4,
		Votes:        big.NewInt(80),
	}

	return nodes[:nodeNum], nil
}

// TestManager_Add
func TestManager_Add(t *testing.T) {
	ma := Instance()
	ma.Clear()
	deputyNodes01, err := deputyNodes(3)
	assert.NoError(t, err)
	addDeputyNodes01 := []*DeputyNodesRecord{&DeputyNodesRecord{height: 0, nodes: deputyNodes01}}
	ma.Add(0, deputyNodes01)

	assert.Equal(t, addDeputyNodes01, ma.DeputyNodesList)
}

// TestDeputyNode_getDeputiesByHeight
func TestManager_getDeputiesByHeight(t *testing.T) {
	ma := Instance()
	ma.Clear()

	nodes01, err := deputyNodes(1)
	assert.NoError(t, err)
	ma.Add(0, nodes01)

	nodes02, err := deputyNodes(2)
	assert.NoError(t, err)
	ma.Add(100, nodes02)

	nodes03, err := deputyNodes(3)
	assert.NoError(t, err)
	ma.Add(200, nodes03)
	// 获取第一个代理节点表
	assert.Equal(t, nodes01, ma.getDeputiesByHeight(0))
	assert.Equal(t, nodes01, ma.getDeputiesByHeight(99))
	// 获取第二个代理节点表
	assert.Equal(t, nodes02, ma.getDeputiesByHeight(100))
	assert.Equal(t, nodes02, ma.getDeputiesByHeight(199))
	// 获取最后一个节点表
	assert.Equal(t, nodes03, ma.getDeputiesByHeight(200))
	assert.Equal(t, nodes03, ma.getDeputiesByHeight(1000000000)) // height为无穷大时则默认为最后一个节点列表

}

// TestManager_GetDeputyByAddress
func TestManager_GetDeputyByAddress(t *testing.T) {
	ma := Instance()
	ma.Clear()
	nodes00, err := deputyNodes(5)
	assert.NoError(t, err)
	ma.Add(0, nodes00)
	assert.Equal(t, nodes00[0], ma.GetDeputyByAddress(0, common.HexToAddress(block01MinerAddress)))
	assert.Equal(t, nodes00[1], ma.GetDeputyByAddress(0, common.HexToAddress(block02MinerAddress)))
	assert.Equal(t, nodes00[2], ma.GetDeputyByAddress(0, common.HexToAddress(block03MinerAddress)))

}

// TestManager_GetDeputyByNodeID
func TestManager_GetDeputyByNodeID(t *testing.T) {
	ma := Instance()
	ma.Clear()
	nodes01, err := deputyNodes(5)
	assert.NoError(t, err)
	ma.Add(0, nodes01)

	privarte01, err := crypto.ToECDSA(common.FromHex(deputy01Privkey))
	assert.NoError(t, err)

	assert.Equal(t, nodes01[0], ma.GetDeputyByNodeID(0, (crypto.FromECDSAPub(&privarte01.PublicKey))[1:]))
}

// TestManager_GetSlot
func TestManager_GetSlot(t *testing.T) {
	ma := Instance()
	ma.Clear()

	nodes00, err := deputyNodes(3)
	assert.NoError(t, err)
	ma.Add(0, nodes00) // 创建3个代理节点的节点列表，列表高度为0

	nodes01, err := deputyNodes(5)
	assert.NoError(t, err)
	ma.Add(params.SnapshotBlock, nodes01) // 创建5个代理节点的节点列表，高度为100000

	// 测试height==1的情况，此情况为上一个块为创世块
	assert.Equal(t, 1, ma.GetSlot(1, common.Address{}, common.HexToAddress(block01MinerAddress)))
	assert.Equal(t, 2, ma.GetSlot(1, common.Address{}, common.HexToAddress(block02MinerAddress)))
	assert.Equal(t, 3, ma.GetSlot(1, common.Address{}, common.HexToAddress(block03MinerAddress)))

	// 测试换届的情况
	assert.Equal(t, 1, ma.GetSlot(params.SnapshotBlock+params.PeriodBlock+1, common.Address{}, common.HexToAddress(block01MinerAddress)))
	assert.Equal(t, 2, ma.GetSlot(params.SnapshotBlock+params.PeriodBlock+1, common.Address{}, common.HexToAddress(block02MinerAddress)))
	assert.Equal(t, 3, ma.GetSlot(params.SnapshotBlock+params.PeriodBlock+1, common.Address{}, common.HexToAddress(block03MinerAddress)))

	// 测试firstNode和NextNode为空的情况
	assert.Equal(t, -1, ma.GetSlot(22, common.Address{}, common.Address{}))

	// 测试只有一个共识节点的情况
	ma.Clear()
	nodes03, err := deputyNodes(1) // 生成只有一个共识节点的节点列表
	assert.NoError(t, err)
	ma.Add(1, nodes03)
	assert.Equal(t, 1, ma.GetSlot(1, common.HexToAddress(block01MinerAddress), common.HexToAddress(block01MinerAddress)))

	// 正常情况下、
	ma.Clear()
	nodes04, err := deputyNodes(5)
	ma.Add(1, nodes04)
	assert.Equal(t, 4, ma.GetSlot(11, common.HexToAddress(block01MinerAddress), common.HexToAddress(block05MinerAddress)))
	assert.Equal(t, 3, ma.GetSlot(11, common.HexToAddress(block02MinerAddress), common.HexToAddress(block05MinerAddress)))
	assert.Equal(t, 2, ma.GetSlot(11, common.HexToAddress(block02MinerAddress), common.HexToAddress(block04MinerAddress)))
	assert.Equal(t, 0, ma.GetSlot(11, common.HexToAddress(block01MinerAddress), common.HexToAddress(block01MinerAddress)))
}

// TestManager_TimeToHandOutRewards
func TestManager_TimeToHandOutRewards(t *testing.T) {
	ma := Instance()
	ma.Clear()
	nodes05, err := deputyNodes(1)
	assert.NoError(t, err)
	ma.Add(0, nodes05)

	nodes06, err := deputyNodes(2)
	assert.NoError(t, err)
	ma.Add(100000, nodes06)

	nodes07, err := deputyNodes(3)
	assert.NoError(t, err)
	ma.Add(200000, nodes07)

	assert.Equal(t, true, ma.TimeToHandOutRewards(100000+1000+1))
	assert.Equal(t, true, ma.TimeToHandOutRewards(200000+1000+1))
	assert.Equal(t, false, ma.TimeToHandOutRewards(111111+1000+1))
}
