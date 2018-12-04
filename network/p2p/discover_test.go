package p2p

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
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
	ioutil.WriteFile(file, []byte(context), 777)
}

func removeFile(file string) {
	datadir := filepath.Join(os.TempDir(), "datadir")
	file = filepath.Join(datadir, file)
	os.Remove(file)
}

func newDiscover() *DiscoverManager {
	datadir := filepath.Join(os.TempDir(), "datadir")
	if _, err := os.Stat(datadir); err != nil {
		os.MkdirAll(datadir, 777)
	}
	return NewDiscoverManager(datadir)
}

var nodeIDs = []*NodeID{
	ToNodeID(common.Hex2Bytes("adb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0")),
	ToNodeID(common.Hex2Bytes("bdb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0")),
	ToNodeID(common.Hex2Bytes("cdb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0")),
	ToNodeID(common.Hex2Bytes("ddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0")),
	ToNodeID(common.Hex2Bytes("edb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0")),
	ToNodeID(common.Hex2Bytes("fdb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0")),
}

var table = []struct {
	k common.Hash
	v string
	n *NodeID
}{
	{
		k: crypto.Keccak256Hash(nodeIDs[0][:]),
		v: "127.0.0.1:7001",
		n: nodeIDs[0],
	},
	{
		k: crypto.Keccak256Hash(nodeIDs[1][:]),
		v: "127.0.0.1:7002",
		n: nodeIDs[1],
	},
	{
		k: crypto.Keccak256Hash(nodeIDs[2][:]),
		v: "127.0.0.1:7003",
		n: nodeIDs[2],
	},
	{
		k: crypto.Keccak256Hash(nodeIDs[3][:]),
		v: "127.0.0.1:7004",
		n: nodeIDs[3],
	},
	{
		k: crypto.Keccak256Hash(nodeIDs[4][:]),
		v: "127.0.0.1:7005",
		n: nodeIDs[4],
	},
	{
		k: crypto.Keccak256Hash(nodeIDs[5][:]),
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

		n := ToNodeID(common.Hex2Bytes(s))
		v := fmt.Sprintf("160.0.0.1:70%d", i)
		k := crypto.Keccak256Hash(n[:])

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

		n := ToNodeID(common.Hex2Bytes(s))
		v := fmt.Sprintf("170.0.0.1:70%d", i)
		k := crypto.Keccak256Hash(n[:])

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

		n := ToNodeID(common.Hex2Bytes(s))
		v := fmt.Sprintf("180.0.0.1:70%d", i)
		k := crypto.Keccak256Hash(n[:])

		dis.foundNodes[k] = newRawNode(n, v)
		if i%3 == 0 {
			dis.foundNodes[k].Sequence = -1
		}
	}

	list := dis.getAvailableNodes()
	assert.Len(t, list, 200)
	assert.Contains(t, list, "adb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb1@160.0.0.1:701")
	assert.Contains(t, list, "adb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74af10@160.0.0.1:7010")
	assert.Contains(t, list, "adb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74af17@160.0.0.1:7017")
	assert.Contains(t, list, "bdb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb1@170.0.0.1:701")
	assert.Contains(t, list, "bdb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74af10@170.0.0.1:7010")
	assert.Contains(t, list, "bdb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74a100@170.0.0.1:70100")
	assert.Contains(t, list, "cdb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb1@180.0.0.1:701")
	assert.Contains(t, list, "cdb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74af10@180.0.0.1:7010")
}

func Test_setWhiteList_ok(t *testing.T) {
	list := make([]string, 0)

	for i := 0; i < 5; i++ {
		prv, _ := crypto.GenerateKey()
		nodeid := PubkeyID(&prv.PublicKey)
		hex := common.Bytes2Hex(nodeid[:])
		list = append(list, hex+"@127.0.0.1:1234"+strconv.Itoa(i))
	}

	content := strings.Join(list, "\r\n")
	writeFile(WhiteFile, content)
	defer removeFile(WhiteFile)

	dis := newDiscover()
	dis.setWhiteList()

	assert.Len(t, dis.whiteNodes, 5)

	for _, v := range dis.whiteNodes {
		assert.Contains(t, list, v.NodeID.String()+"@"+v.Endpoint)
	}
}

func Test_setWhiteList_err(t *testing.T) {
	dis := newDiscover()
	dis.setWhiteList()

	assert.Len(t, dis.whiteNodes, 0)
}

func Test_writeFindFile(t *testing.T) {
	dis := newDiscover()

	for i := 1; i < 18; i++ {
		prv, _ := crypto.GenerateKey()
		n := PubkeyID(&prv.PublicKey)
		v := fmt.Sprintf("160.0.0.1:110%d", i)
		k := crypto.Keccak256Hash(n[:])

		dis.deputyNodes[k] = newRawNode(&n, v)

		if i%3 == 0 {
			dis.deputyNodes[k].Sequence = -1
		}
	}

	for i := 1; i < 100; i++ {
		prv, _ := crypto.GenerateKey()
		n := PubkeyID(&prv.PublicKey)
		v := fmt.Sprintf("170.0.0.1:110%d", i)
		k := crypto.Keccak256Hash(n[:])

		dis.whiteNodes[k] = newRawNode(&n, v)
		if i%3 == 0 {
			dis.whiteNodes[k].Sequence = -1
		}
	}

	for i := 1; i < 200; i++ {
		prv, _ := crypto.GenerateKey()
		n := PubkeyID(&prv.PublicKey)
		v := fmt.Sprintf("180.0.0.1:110%d", i%100)
		k := crypto.Keccak256Hash(n[:])

		dis.foundNodes[k] = newRawNode(&n, v)
		if i%3 == 0 {
			dis.foundNodes[k].Sequence = -1
		}
	}

	dis.writeFindFile()

	dis = newDiscover()
	dis.initDiscoverList()

	list := dis.getAvailableNodes()
	assert.Len(t, list, 200)
	removeFile(FindFile)
}

func Test_SetDeputyNodes(t *testing.T) {
	list := make([]string, 0)

	for i := 0; i < 5; i++ {
		prv, _ := crypto.GenerateKey()
		nodeid := PubkeyID(&prv.PublicKey)
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

	assert.Error(t, dis.SetReconnect(&NodeID{}), ErrNoSpecialNode)
}

func Test_GetNodesForDiscover(t *testing.T) {
	dis := newDiscover()

	for i := 1; i < 18; i++ {
		prv, _ := crypto.GenerateKey()
		n := PubkeyID(&prv.PublicKey)
		v := fmt.Sprintf("160.0.0.1:110%d", i)
		k := crypto.Keccak256Hash(n[:])

		dis.deputyNodes[k] = newRawNode(&n, v)

		if i%3 == 0 {
			dis.deputyNodes[k].Sequence = -1
		} else if i%3 == 1 {
			dis.deputyNodes[k].Sequence = 1
		}
	}

	for i := 1; i < 100; i++ {
		prv, _ := crypto.GenerateKey()
		n := PubkeyID(&prv.PublicKey)
		v := fmt.Sprintf("170.0.0.1:110%d", i%100)
		k := crypto.Keccak256Hash(n[:])

		dis.whiteNodes[k] = newRawNode(&n, v)
		if i%3 == 0 {
			dis.whiteNodes[k].Sequence = -1
		} else if i%3 == 1 {
			dis.whiteNodes[k].Sequence = 1
		}
	}

	for i := 1; i < 200; i++ {
		prv, _ := crypto.GenerateKey()
		n := PubkeyID(&prv.PublicKey)
		v := fmt.Sprintf("180.0.0.1:110%d", i%100)
		k := crypto.Keccak256Hash(n[:])

		dis.foundNodes[k] = newRawNode(&n, v)
		if i%3 == 0 {
			dis.foundNodes[k].Sequence = -1
		} else if i%3 == 1 {
			dis.foundNodes[k].Sequence = 1
		}
	}

	nodes := dis.GetNodesForDiscover(1)

	assert.Len(t, nodes, 200)
}

func Test_AddNewList(t *testing.T) {
	dis := newDiscover()
	assert.NoError(t, dis.Start())

	dis.whiteNodes[table[2].k] = newRawNode(table[2].n, table[2].v)
	dis.whiteNodes[table[2].k].Sequence = -1

	dis.whiteNodes[table[3].k] = newRawNode(table[3].n, table[3].v)
	dis.whiteNodes[table[3].k].Sequence = 0

	dis.deputyNodes[table[4].k] = newRawNode(table[4].n, table[4].v)
	dis.deputyNodes[table[4].k].Sequence = 0

	dis.deputyNodes[table[5].k] = newRawNode(table[5].n, table[5].v)
	dis.deputyNodes[table[5].k].Sequence = -1

	list := make([]string, 0)

	for i := 0; i < 5; i++ {
		prv, _ := crypto.GenerateKey()
		nodeid := PubkeyID(&prv.PublicKey)
		hex := common.Bytes2Hex(nodeid[:])
		list = append(list, hex+"@127.0.0.1:123"+strconv.Itoa(i))
	}
	dis.AddNewList(list)

	for _, v := range dis.foundNodes {
		assert.Contains(t, list, v.NodeID.String()+"@"+v.Endpoint)
	}
	assert.NoError(t, dis.Stop())

	removeFile(FindFile)
}

func Test_Start_err(t *testing.T) {
	dis := newDiscover()
	assert.NoError(t, dis.Start())
	assert.Error(t, dis.Start(), ErrHasStared)
}

func Test_Stop_err(t *testing.T) {
	dis := newDiscover()
	assert.Error(t, dis.Stop(), ErrNotStart)
	removeFile(FindFile)
}

func Test_Start_restart(t *testing.T) {
	dis := newDiscover()
	assert.NoError(t, dis.Start())
	assert.NoError(t, dis.Stop())
	assert.NoError(t, dis.Start())

	removeFile(FindFile)
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

	assert.Error(t, dis.SetConnectResult(new(NodeID), false), ErrNoSpecialNode)
}
