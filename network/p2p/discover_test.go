package p2p

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func writeFile(file string, context string) {
	datadir := filepath.Join(os.TempDir(), "datadir")
	file = filepath.Join(datadir, file)
	_ = ioutil.WriteFile(file, []byte(context), 777)
}

func removeFile(file string) {
	datadir := filepath.Join(os.TempDir(), "datadir")
	file = filepath.Join(datadir, file)
	_ = os.Remove(file)
}

func newDiscover() *DiscoverManager {
	datadir := filepath.Join(os.TempDir(), "datadir")
	if _, err := os.Stat(datadir); err != nil {
		_ = os.MkdirAll(datadir, 777)
	}
	return NewDiscoverManager(datadir)
}

var nodeIDs = []*NodeID{
	BytesToNodeID(common.FromHex("5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0")),
	BytesToNodeID(common.FromHex("c7021a9c903da38ed499f486dba4539fbe12b8878d43e566674beebd36746e77c827a2849db3c1289e0adf25fce294253be5e7c9bb65d0b94cf8a7ec34c91468")),
	BytesToNodeID(common.FromHex("7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2d0347692f36ef736f342fb5ce4710f7e337f062cc2110d134b63a9575f78cb167bfae2f43")),
	BytesToNodeID(common.FromHex("34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f")),
	BytesToNodeID(common.FromHex("5b980ffb1b463fce4773a22ebf376c07c6207023b016b36ccfaba7be1cd1ab4a91737741cd43b7fcb10879e0fcf314d69fa953daec0f02be0f8f9cedb0cb3797")),
	BytesToNodeID(common.FromHex("0e53292ab5a51286d64422344c6b0751dc1429497fe72820a0a273c70e35bbbe8196af0c5526588fee62f1b68558773501d32e5d552fd9863d740f30ed41f4b0")),
}

var table = []struct {
	k common.Hash
	v string
	n *NodeID
}{
	{
		k: nodeIDs[0].Hash(),
		v: "127.0.0.1:7001",
		n: nodeIDs[0],
	},
	{
		k: nodeIDs[1].Hash(),
		v: "127.0.0.1:7002",
		n: nodeIDs[1],
	},
	{
		k: nodeIDs[2].Hash(),
		v: "127.0.0.1:7003",
		n: nodeIDs[2],
	},
	{
		k: nodeIDs[3].Hash(),
		v: "127.0.0.1:7004",
		n: nodeIDs[3],
	},
	{
		k: nodeIDs[4].Hash(),
		v: "127.0.0.1:7005",
		n: nodeIDs[4],
	},
	{
		k: nodeIDs[5].Hash(),
		v: "127.0.0.1:7006",
		n: nodeIDs[5],
	},
}

func Test_connectedNodes_1(t *testing.T) {
	dis := newDiscover()

	correct := make(map[string]struct{})
	correct[table[1].n.String()+"@"+table[1].v] = struct{}{}
	correct[table[2].n.String()+"@"+table[2].v] = struct{}{}

	dis.foundNodes[table[0].k] = newRawNode(table[0].n, table[0].v)
	dis.foundNodes[table[0].k].Sequence = 0

	dis.foundNodes[table[1].k] = newRawNode(table[1].n, table[1].v)
	dis.foundNodes[table[1].k].Sequence = 1

	dis.foundNodes[table[2].k] = newRawNode(table[2].n, table[2].v)
	dis.foundNodes[table[2].k].Sequence = 2

	list := dis.connectedNodes()
	assert.Len(t, list, len(correct))

	for i := 0; i < len(correct); i++ {
		assert.Contains(t, correct, list[i])
	}
}

func Test_connectedNodes_2(t *testing.T) {
	correct := make(map[string]struct{})
	correct[table[1].n.String()+"@"+table[1].v] = struct{}{}
	correct[table[3].n.String()+"@"+table[3].v] = struct{}{}
	correct[table[4].n.String()+"@"+table[4].v] = struct{}{}

	dis := newDiscover()

	dis.foundNodes[table[0].k] = newRawNode(table[0].n, table[0].v)
	dis.foundNodes[table[0].k].Sequence = -1

	dis.foundNodes[table[1].k] = newRawNode(table[1].n, table[1].v)
	dis.foundNodes[table[1].k].Sequence = 1

	dis.whiteNodes[table[2].k] = newRawNode(table[2].n, table[2].v)
	dis.whiteNodes[table[2].k].Sequence = -1

	dis.whiteNodes[table[3].k] = newRawNode(table[3].n, table[3].v)
	dis.whiteNodes[table[3].k].Sequence = 3

	dis.deputyNodes[table[4].k] = newRawNode(table[4].n, table[4].v)
	dis.deputyNodes[table[4].k].Sequence = 4

	dis.deputyNodes[table[5].k] = newRawNode(table[5].n, table[5].v)
	dis.deputyNodes[table[5].k].Sequence = -1

	list := dis.connectedNodes()
	assert.Len(t, list, len(correct))

	for i := 0; i < len(correct); i++ {
		assert.Contains(t, correct, list[i])
	}
}

func Test_connectingNodes_1(t *testing.T) {
	dis := newDiscover()

	correct := make(map[string]struct{})
	correct[table[0].n.String()+"@"+table[0].v] = struct{}{}
	correct[table[3].n.String()+"@"+table[3].v] = struct{}{}

	dis.foundNodes[table[0].k] = newRawNode(table[0].n, table[0].v)
	dis.foundNodes[table[0].k].Sequence = 0

	dis.foundNodes[table[1].k] = newRawNode(table[1].n, table[1].v)
	dis.foundNodes[table[1].k].Sequence = 1

	dis.foundNodes[table[2].k] = newRawNode(table[2].n, table[2].v)
	dis.foundNodes[table[2].k].Sequence = -1

	dis.foundNodes[table[3].k] = newRawNode(table[3].n, table[3].v)
	dis.foundNodes[table[3].k].Sequence = 0

	list := dis.connectingNodes()
	assert.Len(t, list, len(correct))

	for i := 0; i < len(correct); i++ {
		assert.Contains(t, correct, list[i])
	}
}

func Test_connectingNodes_2(t *testing.T) {
	correct := make(map[string]struct{})
	correct[table[0].n.String()+"@"+table[0].v] = struct{}{}
	correct[table[3].n.String()+"@"+table[3].v] = struct{}{}

	dis := newDiscover()

	dis.foundNodes[table[0].k] = newRawNode(table[0].n, table[0].v)
	dis.foundNodes[table[0].k].Sequence = 0

	dis.foundNodes[table[1].k] = newRawNode(table[1].n, table[1].v)
	dis.foundNodes[table[1].k].Sequence = 1

	dis.whiteNodes[table[2].k] = newRawNode(table[2].n, table[2].v)
	dis.whiteNodes[table[2].k].Sequence = -1

	dis.whiteNodes[table[3].k] = newRawNode(table[3].n, table[3].v)
	dis.whiteNodes[table[3].k].Sequence = 0

	dis.deputyNodes[table[4].k] = newRawNode(table[4].n, table[4].v)
	dis.deputyNodes[table[4].k].Sequence = 2

	dis.deputyNodes[table[5].k] = newRawNode(table[5].n, table[5].v)
	dis.deputyNodes[table[5].k].Sequence = -1

	list := dis.connectingNodes()
	assert.Len(t, list, len(correct))

	for i := 0; i < len(correct); i++ {
		assert.Contains(t, correct, list[i])
	}
}

func Test_staleNodes_1(t *testing.T) {
	dis := newDiscover()

	correct := make(map[string]struct{})
	correct[table[2].n.String()+"@"+table[2].v] = struct{}{}

	dis.foundNodes[table[0].k] = newRawNode(table[0].n, table[0].v)
	dis.foundNodes[table[0].k].Sequence = 0

	dis.foundNodes[table[1].k] = newRawNode(table[1].n, table[1].v)
	dis.foundNodes[table[1].k].Sequence = 1

	dis.foundNodes[table[2].k] = newRawNode(table[2].n, table[2].v)
	dis.foundNodes[table[2].k].Sequence = -1

	dis.foundNodes[table[3].k] = newRawNode(table[3].n, table[3].v)
	dis.foundNodes[table[3].k].Sequence = 0

	list := dis.staleNodes()
	assert.Len(t, list, len(correct))

	for i := 0; i < len(correct); i++ {
		assert.Contains(t, correct, list[i])
	}
}

func Test_staleNodes_2(t *testing.T) {
	correct := make(map[string]struct{})
	correct[table[0].n.String()+"@"+table[0].v] = struct{}{}
	correct[table[1].n.String()+"@"+table[1].v] = struct{}{}
	correct[table[2].n.String()+"@"+table[2].v] = struct{}{}
	correct[table[3].n.String()+"@"+table[3].v] = struct{}{}
	correct[table[4].n.String()+"@"+table[4].v] = struct{}{}
	correct[table[5].n.String()+"@"+table[5].v] = struct{}{}

	dis := newDiscover()

	dis.foundNodes[table[0].k] = newRawNode(table[0].n, table[0].v)
	dis.foundNodes[table[0].k].Sequence = -1

	dis.foundNodes[table[1].k] = newRawNode(table[1].n, table[1].v)
	dis.foundNodes[table[1].k].Sequence = -1

	dis.whiteNodes[table[2].k] = newRawNode(table[2].n, table[2].v)
	dis.whiteNodes[table[2].k].Sequence = -1

	dis.whiteNodes[table[3].k] = newRawNode(table[3].n, table[3].v)
	dis.whiteNodes[table[3].k].Sequence = -1

	dis.deputyNodes[table[4].k] = newRawNode(table[4].n, table[4].v)
	dis.deputyNodes[table[4].k].Sequence = -1

	dis.deputyNodes[table[5].k] = newRawNode(table[5].n, table[5].v)
	dis.deputyNodes[table[5].k].Sequence = -1

	list := dis.staleNodes()
	assert.Len(t, list, len(correct))

	for i := 0; i < len(correct); i++ {
		assert.Contains(t, correct, list[i])
	}
}

func Test_getAvailableNodes(t *testing.T) {
	dis := newDiscover()

	for i := 1; i < 18; i++ {
		var s string
		if i < 10 {
			s = "adb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb" + strconv.Itoa(i)
		} else {
			s = "adb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74af" + strconv.Itoa(i)
		}

		n := BytesToNodeID(common.FromHex(s))
		v := fmt.Sprintf("160.0.0.1:70%d", i)
		k := n.Hash()

		dis.deputyNodes[k] = newRawNode(n, v)
		if i%3 == 0 {
			dis.deputyNodes[k].Sequence = -1
		}
	}

	for i := 1; i < 101; i++ {
		var s string
		if i < 10 {
			s = "bdb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb" + strconv.Itoa(i)
		} else if i < 100 {
			s = "bdb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74af" + strconv.Itoa(i)
		} else {
			s = "bdb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74a" + strconv.Itoa(i)
		}

		n := BytesToNodeID(common.FromHex(s))
		v := fmt.Sprintf("170.0.0.1:70%d", i)
		k := n.Hash()

		dis.whiteNodes[k] = newRawNode(n, v)
		if i%3 == 0 {
			dis.whiteNodes[k].Sequence = -1
		}
	}

	for i := 1; i < 201; i++ {
		var s string
		if i < 10 {
			s = "cdb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb" + strconv.Itoa(i)
		} else if i < 100 {
			s = "cdb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74af" + strconv.Itoa(i)
		} else {
			s = "cdb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74a" + strconv.Itoa(i)
		}

		n := BytesToNodeID(common.FromHex(s))
		v := fmt.Sprintf("180.0.0.1:70%d", i)
		k := n.Hash()

		dis.foundNodes[k] = newRawNode(n, v)
		if i%3 == 0 {
			dis.foundNodes[k].Sequence = -1
		}
	}

	list := dis.getAvailableNodes()
	assert.Len(t, list, 100)
}

func Test_initWhiteList_initBlackeList_ok(t *testing.T) {
	list := make([]string, 0)

	for i := 0; i < 5; i++ {
		prv, _ := crypto.GenerateKey()
		nodeid := PubKeyToNodeID(&prv.PublicKey)
		hex := common.Bytes2Hex(nodeid[:])
		list = append(list, hex+"@127.0.0.1:1234"+strconv.Itoa(i))
	}

	content := strings.Join(list, "\r\n")
	writeFile(WhiteFile, content)
	defer removeFile(WhiteFile)

	writeFile(BlackFile, content)
	defer removeFile(BlackFile)
	dis := newDiscover()
	dis.initWhiteList()
	dis.initBlackList()
	assert.Len(t, dis.whiteNodes, 5)
	assert.Len(t, dis.blackNodes, 5)

	for _, v := range dis.whiteNodes {
		assert.Contains(t, list, v.NodeID.String()+"@"+v.Endpoint)
	}
	for _, v := range dis.blackNodes {
		assert.Contains(t, list, v.NodeID.String()+"@"+v.Endpoint)
	}
}

func Test_initBlackList_err(t *testing.T) {
	dis := newDiscover()
	dis.initBlackList()
	assert.Len(t, dis.blackNodes, 0)
}

func Test_initWhiteList_err(t *testing.T) {
	dis := newDiscover()
	dis.initWhiteList()

	assert.Len(t, dis.whiteNodes, 0)
}

func Test_writeFindFile(t *testing.T) {
	dis := newDiscover()

	for i := 1; i < 18; i++ {
		prv, _ := crypto.GenerateKey()
		n := PubKeyToNodeID(&prv.PublicKey)
		v := fmt.Sprintf("160.0.0.1:110%d", i)
		k := n.Hash()

		dis.deputyNodes[k] = newRawNode(&n, v)

		if i%3 == 0 {
			dis.deputyNodes[k].Sequence = -1
		}
	}

	for i := 1; i < 100; i++ {
		prv, _ := crypto.GenerateKey()
		n := PubKeyToNodeID(&prv.PublicKey)
		v := fmt.Sprintf("170.0.0.1:110%d", i)
		k := n.Hash()

		dis.whiteNodes[k] = newRawNode(&n, v)
		if i%3 == 0 {
			dis.whiteNodes[k].Sequence = -1
		}
	}

	for i := 1; i < 200; i++ {
		prv, _ := crypto.GenerateKey()
		n := PubKeyToNodeID(&prv.PublicKey)
		v := fmt.Sprintf("180.0.0.1:110%d", i%100)
		k := n.Hash()

		dis.foundNodes[k] = newRawNode(&n, v)
		if i%3 == 0 {
			dis.foundNodes[k].Sequence = -1
		}
	}

	dis.writeFindNodeToFile()

	dis = newDiscover()
	dis.initDiscoverList()

	list := dis.getAvailableNodes()
	assert.Len(t, list, 100)
	removeFile(FoundFile)
}

func Test_SetDeputyNodes(t *testing.T) {
	list := make([]string, 0)

	for i := 0; i < 5; i++ {
		prv, _ := crypto.GenerateKey()
		nodeid := PubKeyToNodeID(&prv.PublicKey)
		hex := common.Bytes2Hex(nodeid[:])
		list = append(list, hex+"@127.0.0.1:1234"+strconv.Itoa(i))
	}

	dis := newDiscover()
	dis.SetDeputyNodes(list)

	assert.Len(t, dis.deputyNodes, len(list))

	for _, v := range dis.deputyNodes {
		assert.Contains(t, list, v.NodeID.String()+"@"+v.Endpoint)
	}
}

func Test_SetReconnect(t *testing.T) {
	dis := newDiscover()

	dis.foundNodes[table[0].k] = newRawNode(table[0].n, table[0].v)
	dis.foundNodes[table[0].k].Sequence = -1

	dis.foundNodes[table[1].k] = newRawNode(table[1].n, table[1].v)
	dis.foundNodes[table[1].k].Sequence = 1

	dis.foundNodes[table[2].k] = newRawNode(table[2].n, table[2].v)
	dis.foundNodes[table[2].k].Sequence = -1

	dis.foundNodes[table[3].k] = newRawNode(table[3].n, table[3].v)
	dis.foundNodes[table[3].k].Sequence = 0

	assert.Equal(t, dis.foundNodes[table[0].k].IsReconnect, false)
	assert.NoError(t, dis.SetReconnect(table[0].n))
	assert.Equal(t, dis.foundNodes[table[0].k].Sequence, int32(0))
	assert.Equal(t, dis.foundNodes[table[0].k].IsReconnect, true)
	assert.NoError(t, dis.SetConnectResult(table[0].n, false))
	assert.Equal(t, dis.foundNodes[table[0].k].ConnCounter, int8(2))
	assert.Equal(t, dis.foundNodes[table[0].k].Sequence, int32(0))
	assert.Equal(t, dis.foundNodes[table[0].k].IsReconnect, true)

	for i := 0; i < 2; i++ {
		assert.NoError(t, dis.SetConnectResult(table[0].n, false))
	}
	assert.Equal(t, ErrMaxReconnect, dis.SetConnectResult(table[0].n, false))

	assert.Equal(t, ErrNoSpecialNode, dis.SetReconnect(&NodeID{}))
}

func Test_GetNodesForDiscover(t *testing.T) {
	dis := newDiscover()

	for i := 1; i < 18; i++ {
		prv, _ := crypto.GenerateKey()
		n := PubKeyToNodeID(&prv.PublicKey)
		v := fmt.Sprintf("160.0.0.1:110%d", i)
		k := n.Hash()

		dis.deputyNodes[k] = newRawNode(&n, v)

		if i%3 == 0 {
			dis.deputyNodes[k].Sequence = -1
		} else if i%3 == 1 {
			dis.deputyNodes[k].Sequence = 1
		}
	}

	for i := 1; i < 100; i++ {
		prv, _ := crypto.GenerateKey()
		n := PubKeyToNodeID(&prv.PublicKey)
		v := fmt.Sprintf("170.0.0.1:110%d", i%100)
		k := n.Hash()

		dis.whiteNodes[k] = newRawNode(&n, v)
		if i%3 == 0 {
			dis.whiteNodes[k].Sequence = -1
		} else if i%3 == 1 {
			dis.whiteNodes[k].Sequence = 1
		}
	}

	for i := 1; i < 200; i++ {
		prv, _ := crypto.GenerateKey()
		n := PubKeyToNodeID(&prv.PublicKey)
		v := fmt.Sprintf("180.0.0.1:110%d", i%100)
		k := n.Hash()

		dis.foundNodes[k] = newRawNode(&n, v)
		if i%3 == 0 {
			dis.foundNodes[k].Sequence = -1
		} else if i%3 == 1 {
			dis.foundNodes[k].Sequence = 1
		}
	}

	nodes := dis.GetNodesForDiscover(1, "")

	assert.Len(t, nodes, 100)
}

func setupRawNode(nodes map[common.Hash]*RawNode, i int) {
	nodes[table[i].k] = newRawNode(table[i].n, table[i].v)
	nodes[table[i].k].IsReconnect = true
	nodes[table[i].k].Sequence = -1
	nodes[table[i].k].ConnCounter = 5
}

func Test_AddNewList(t *testing.T) {
	dis := newDiscover()
	dis.deputyNodes = make(map[common.Hash]*RawNode, 20)
	dis.foundNodes = make(map[common.Hash]*RawNode, 20)
	dis.whiteNodes = make(map[common.Hash]*RawNode, 20)

	// 使用table里的node赋值
	for i := 0; i <= 5; i++ {
		if i%3 == 0 { // i == 0,3
			// 设置白名单
			setupRawNode(dis.whiteNodes, i)
		} else if i%3 == 1 { // i == 1,4
			// 设置deputynodes
			setupRawNode(dis.deputyNodes, i)
		} else { // i == 2,5
			// 设置节点发现nodes
			setupRawNode(dis.foundNodes, i)
		}
	}

	testFindNodes := make([]string, 0, 10)  // 随机生成用于节点发现的nodes
	testBlackNodes := make([]string, 0, 10) // 黑名单nodes
	nodesFromTable := make([]string, 0, 5)  // 存储table中的nodes

	for i := 0; i <= 5; i++ {
		s := table[i].n.String() + "@" + table[i].v
		nodesFromTable = append(nodesFromTable, s)
	}

	for i := 0; i < 20; i++ {
		prv, _ := crypto.GenerateKey()
		nodeid := PubKeyToNodeID(&prv.PublicKey)
		hex := common.Bytes2Hex(nodeid[:])
		if i%2 == 0 {
			testFindNodes = append(testFindNodes, hex+"@127.0.0.1:123"+strconv.Itoa(i))
		} else {
			// 设置port为基数的为黑名单
			dis.blackNodes[nodeid.Hash()] = newRawNode(&nodeid, "@127.0.0.1:123"+strconv.Itoa(i))
			testBlackNodes = append(testBlackNodes, hex+"@127.0.0.1:123"+strconv.Itoa(i))
		}
	}

	lists := make([]string, 0, 20)
	lists = append(append(testFindNodes, testBlackNodes...), nodesFromTable...)

	dis.AddNewList(lists)

	for _, v := range dis.foundNodes {
		// lists里包含了节点发现的node
		assert.Contains(t, lists, v.NodeID.String()+"@"+v.Endpoint)
		// AddNewList节点发现里一定没有黑名单中的node
		assert.NotContains(t, testBlackNodes, v.NodeID.String()+"@"+v.Endpoint)
	}
	// white nodes
	for _, v := range dis.whiteNodes {
		// AddNewList节点发现后更新的最初白名单节点属性
		assert.Equal(t, 0, int(v.ConnCounter))
		assert.Equal(t, false, v.IsReconnect)
	}
	// deputy nodes
	for _, v := range dis.deputyNodes {
		// AddNewList节点发现后更新的最初的deputynode属性
		assert.Equal(t, uint32(0), uint32(v.Sequence))
		assert.Equal(t, uint32(0), uint32(v.ConnCounter))
		assert.Equal(t, false, v.IsReconnect)
	}
	// 更新的初始的节点发现的node属性
	assert.Equal(t, 0, int(dis.foundNodes[table[2].k].ConnCounter))
	assert.Equal(t, 0, int(dis.foundNodes[table[2].k].Sequence))
	assert.Equal(t, 0, int(dis.foundNodes[table[5].k].ConnCounter))
	assert.Equal(t, 0, int(dis.foundNodes[table[5].k].Sequence))
	assert.Equal(t, false, dis.foundNodes[table[2].k].IsReconnect)
	assert.Equal(t, false, dis.foundNodes[table[5].k].IsReconnect)

	removeFile(FoundFile)
}

func TestDiscoverManager_PutBlackNode_IsBlackNode(t *testing.T) {
	dis := newDiscover()
	assert.Empty(t, dis.blackNodes)
	n := 100
	keyList := make([]common.Hash, 0, n)
	nodes := make([]string, 0, n)
	for i := 0; i < n; i++ {
		prv, _ := crypto.GenerateKey()
		nodeID := PubKeyToNodeID(&prv.PublicKey)
		endpoint := "127.0.0.1:700" + strconv.Itoa(i%10)
		dis.PutBlackNode(&nodeID, endpoint)
		key := nodeID.Hash()
		keyList = append(keyList, key)
		nodes = append(nodes, nodeID.String()+"@"+endpoint)
	}
	// is black node
	for i := 0; i < n; i++ {
		k := dis.getBlackNode(keyList[i])
		assert.NotEmpty(t, k)
		nodeID, _ := ParseNodeString(nodes[i])
		b := dis.IsBlackNode(nodeID)
		assert.Equal(t, true, b)
	}
	// is not black node
	for i := 0; i <= 5; i++ {
		assert.Equal(t, false, dis.IsBlackNode(table[i].n))
	}
}

func Test_Start_err(t *testing.T) {
	dis := newDiscover()
	assert.NoError(t, dis.Start())
	assert.Equal(t, ErrHasStared, dis.Start())
}

func Test_Stop_err(t *testing.T) {
	dis := newDiscover()
	assert.Equal(t, ErrNotStart, dis.Stop())
	removeFile(FoundFile)
}

func Test_Start_restart(t *testing.T) {
	dis := newDiscover()
	assert.NoError(t, dis.Start())
	assert.NoError(t, dis.Stop())
	assert.NoError(t, dis.Start())

	removeFile(FoundFile)
}

func Test_SetConnectResult(t *testing.T) {
	dis := newDiscover()

	dis.foundNodes[table[0].k] = newRawNode(table[0].n, table[0].v)
	dis.foundNodes[table[0].k].Sequence = -1

	dis.foundNodes[table[1].k] = newRawNode(table[1].n, table[1].v)
	dis.foundNodes[table[1].k].Sequence = 1

	dis.foundNodes[table[2].k] = newRawNode(table[2].n, table[2].v)
	dis.foundNodes[table[2].k].Sequence = 0

	dis.foundNodes[table[3].k] = newRawNode(table[3].n, table[3].v)
	dis.foundNodes[table[3].k].Sequence = 0

	assert.NoError(t, dis.SetConnectResult(dis.foundNodes[table[2].k].NodeID, true))
	assert.Equal(t, dis.foundNodes[table[2].k].Sequence, int32(1))

	assert.NoError(t, dis.SetConnectResult(dis.foundNodes[table[3].k].NodeID, false))
	assert.Equal(t, dis.foundNodes[table[3].k].Sequence, int32(-1))

	dis.SetReconnect(table[3].n)
	assert.NoError(t, dis.SetConnectResult(dis.foundNodes[table[3].k].NodeID, false))
	assert.Equal(t, dis.foundNodes[table[3].k].Sequence, int32(0))

	assert.Equal(t, ErrNoSpecialNode, dis.SetConnectResult(new(NodeID), false))
}
