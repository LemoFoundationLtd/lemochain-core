package p2p

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
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

var table = []struct {
	k common.Hash
	v string
}{
	{
		k: crypto.Keccak256Hash([]byte("127.0.0.1:7001")),
		v: "127.0.0.1:7001",
	},
	{
		k: crypto.Keccak256Hash([]byte("127.0.0.1:7002")),
		v: "127.0.0.1:7002",
	},
	{
		k: crypto.Keccak256Hash([]byte("127.0.0.1:7003")),
		v: "127.0.0.1:7003",
	},
	{
		k: crypto.Keccak256Hash([]byte("127.0.0.1:7004")),
		v: "127.0.0.1:7004",
	},
	{
		k: crypto.Keccak256Hash([]byte("127.0.0.1:7005")),
		v: "127.0.0.1:7005",
	},
	{
		k: crypto.Keccak256Hash([]byte("127.0.0.1:7006")),
		v: "127.0.0.1:7006",
	},
}

func Test_connectedNodes_1(t *testing.T) {
	dis := newDiscover()

	correct := make(map[string]struct{})
	correct[table[1].v] = struct{}{}
	correct[table[2].v] = struct{}{}

	dis.foundNodes[table[0].k] = newRawNode(table[0].v)
	dis.foundNodes[table[0].k].Sequence = 0

	dis.foundNodes[table[1].k] = newRawNode(table[1].v)
	dis.foundNodes[table[1].k].Sequence = 1

	dis.foundNodes[table[2].k] = newRawNode(table[2].v)
	dis.foundNodes[table[2].k].Sequence = 2

	list := dis.connectedNodes()
	assert.Len(t, list, len(correct))

	for i := 0; i < len(correct); i++ {
		assert.Contains(t, correct, list[i])
	}
}

func Test_connectedNodes_2(t *testing.T) {
	correct := make(map[string]struct{})
	correct[table[1].v] = struct{}{}
	correct[table[3].v] = struct{}{}
	correct[table[4].v] = struct{}{}

	dis := newDiscover()

	dis.foundNodes[table[0].k] = newRawNode(table[0].v)
	dis.foundNodes[table[0].k].Sequence = -1

	dis.foundNodes[table[1].k] = newRawNode(table[1].v)
	dis.foundNodes[table[1].k].Sequence = 1

	dis.whiteNodes[table[2].k] = newRawNode(table[2].v)
	dis.whiteNodes[table[2].k].Sequence = -1

	dis.whiteNodes[table[3].k] = newRawNode(table[3].v)
	dis.whiteNodes[table[3].k].Sequence = 3

	dis.deputyNodes[table[4].k] = newRawNode(table[4].v)
	dis.deputyNodes[table[4].k].Sequence = 4

	dis.deputyNodes[table[5].k] = newRawNode(table[5].v)
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
	correct[table[0].v] = struct{}{}
	correct[table[3].v] = struct{}{}

	dis.foundNodes[table[0].k] = newRawNode(table[0].v)
	dis.foundNodes[table[0].k].Sequence = 0

	dis.foundNodes[table[1].k] = newRawNode(table[1].v)
	dis.foundNodes[table[1].k].Sequence = 1

	dis.foundNodes[table[2].k] = newRawNode(table[2].v)
	dis.foundNodes[table[2].k].Sequence = -1

	dis.foundNodes[table[3].k] = newRawNode(table[3].v)
	dis.foundNodes[table[3].k].Sequence = 0

	list := dis.connectingNodes()
	assert.Len(t, list, len(correct))

	for i := 0; i < len(correct); i++ {
		assert.Contains(t, correct, list[i])
	}
}

func Test_connectingNodes_2(t *testing.T) {
	correct := make(map[string]struct{})
	correct[table[0].v] = struct{}{}
	correct[table[3].v] = struct{}{}

	dis := newDiscover()

	dis.foundNodes[table[0].k] = newRawNode(table[0].v)
	dis.foundNodes[table[0].k].Sequence = 0

	dis.foundNodes[table[1].k] = newRawNode(table[1].v)
	dis.foundNodes[table[1].k].Sequence = 1

	dis.whiteNodes[table[2].k] = newRawNode(table[2].v)
	dis.whiteNodes[table[2].k].Sequence = -1

	dis.whiteNodes[table[3].k] = newRawNode(table[3].v)
	dis.whiteNodes[table[3].k].Sequence = 0

	dis.deputyNodes[table[4].k] = newRawNode(table[4].v)
	dis.deputyNodes[table[4].k].Sequence = 2

	dis.deputyNodes[table[5].k] = newRawNode(table[5].v)
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
	correct[table[2].v] = struct{}{}

	dis.foundNodes[table[0].k] = newRawNode(table[0].v)
	dis.foundNodes[table[0].k].Sequence = 0

	dis.foundNodes[table[1].k] = newRawNode(table[1].v)
	dis.foundNodes[table[1].k].Sequence = 1

	dis.foundNodes[table[2].k] = newRawNode(table[2].v)
	dis.foundNodes[table[2].k].Sequence = -1

	dis.foundNodes[table[3].k] = newRawNode(table[3].v)
	dis.foundNodes[table[3].k].Sequence = 0

	list := dis.staleNodes()
	assert.Len(t, list, len(correct))

	for i := 0; i < len(correct); i++ {
		assert.Contains(t, correct, list[i])
	}
}

func Test_staleNodes_2(t *testing.T) {
	correct := make(map[string]struct{})
	correct[table[0].v] = struct{}{}
	correct[table[1].v] = struct{}{}
	correct[table[2].v] = struct{}{}
	correct[table[3].v] = struct{}{}
	correct[table[4].v] = struct{}{}
	correct[table[5].v] = struct{}{}

	dis := newDiscover()

	dis.foundNodes[table[0].k] = newRawNode(table[0].v)
	dis.foundNodes[table[0].k].Sequence = -1

	dis.foundNodes[table[1].k] = newRawNode(table[1].v)
	dis.foundNodes[table[1].k].Sequence = -1

	dis.whiteNodes[table[2].k] = newRawNode(table[2].v)
	dis.whiteNodes[table[2].k].Sequence = -1

	dis.whiteNodes[table[3].k] = newRawNode(table[3].v)
	dis.whiteNodes[table[3].k].Sequence = -1

	dis.deputyNodes[table[4].k] = newRawNode(table[4].v)
	dis.deputyNodes[table[4].k].Sequence = -1

	dis.deputyNodes[table[5].k] = newRawNode(table[5].v)
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
		v := fmt.Sprintf("160.0.0.1:70%d", i)
		k := crypto.Keccak256Hash([]byte(v))
		dis.deputyNodes[k] = newRawNode(v)
		if i%3 == 0 {
			dis.deputyNodes[k].Sequence = -1
		}
	}

	for i := 1; i < 101; i++ {
		v := fmt.Sprintf("170.0.0.1:70%d", i)
		k := crypto.Keccak256Hash([]byte(v))
		dis.whiteNodes[k] = newRawNode(v)
		if i%3 == 0 {
			dis.whiteNodes[k].Sequence = -1
		}
	}

	for i := 1; i < 201; i++ {
		v := fmt.Sprintf("180.0.0.1:70%d", i)
		k := crypto.Keccak256Hash([]byte(v))
		dis.foundNodes[k] = newRawNode(v)
		if i%3 == 0 {
			dis.foundNodes[k].Sequence = -1
		}
	}

	list := dis.getAvailableNodes()
	assert.Len(t, list, 200)
	assert.Contains(t, list, "160.0.0.1:701")
	assert.Contains(t, list, "160.0.0.1:7010")
	assert.Contains(t, list, "160.0.0.1:7017")
	assert.Contains(t, list, "170.0.0.1:701")
	assert.Contains(t, list, "170.0.0.1:7010")
	assert.Contains(t, list, "170.0.0.1:70100")
	assert.Contains(t, list, "180.0.0.1:701")
	assert.Contains(t, list, "180.0.0.1:7010")
	assert.Contains(t, list, "180.0.0.1:70100")
	assert.Contains(t, list, "180.0.0.1:70182")
}

func Test_setWhiteList(t *testing.T) {
	list := []string{
		"127.0.0.1:12343",
		"127.0.0.1:12344",
		"127.0.0.1:12345",
		"127.0.0.1:12346",
		"127.0.0.1:12346",
		"127.0.0.1:12347",
	}

	content := strings.Join(list, "\r\n")
	writeFile(WhiteFile, content)
	defer removeFile(WhiteFile)

	dis := newDiscover()
	dis.setWhiteList()

	assert.Len(t, dis.whiteNodes, 5)

	for _, v := range dis.whiteNodes {
		assert.Contains(t, list, v.Endpoint)
	}
}

func Test_writeFindFile(t *testing.T) {
	dis := newDiscover()

	for i := 1; i < 18; i++ {
		v := fmt.Sprintf("160.0.0.1:70%d", i)
		k := crypto.Keccak256Hash([]byte(v))
		dis.deputyNodes[k] = newRawNode(v)
		if i%3 == 0 {
			dis.deputyNodes[k].Sequence = -1
		}
	}

	for i := 1; i < 101; i++ {
		v := fmt.Sprintf("170.0.0.1:70%d", i)
		k := crypto.Keccak256Hash([]byte(v))
		dis.whiteNodes[k] = newRawNode(v)
		if i%3 == 0 {
			dis.whiteNodes[k].Sequence = -1
		}
	}

	for i := 1; i < 201; i++ {
		v := fmt.Sprintf("180.0.0.1:70%d", i)
		k := crypto.Keccak256Hash([]byte(v))
		dis.foundNodes[k] = newRawNode(v)
		if i%3 == 0 {
			dis.foundNodes[k].Sequence = -1
		}
	}

	dis.writeFindFile()

	dis = newDiscover()
	dis.initDiscoverList()

	list := dis.getAvailableNodes()
	assert.Len(t, list, 200)
}

func Test_readFile(t *testing.T) {

}

func Test_initDiscoverList(t *testing.T) {

}

func Test_addDiscoverNodes(t *testing.T) {

}
