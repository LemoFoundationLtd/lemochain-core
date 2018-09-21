package dpovp

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"io"
	"os"
	"strings"
	"sync"
)

type AddrNodeIDMapping struct {
	Addr   common.Address
	Pubkey []byte
}

var (
	dataDir         string
	starList        []AddrNodeIDMapping
	readStarListMux sync.Mutex
)

// 设置 datadir路径
func SetDataDir(path string) {
	if dataDir == "" {
		dataDir = path
	}
}

// 读取主节点列表
func readStarList() {
	readStarListMux.Lock()
	defer readStarListMux.Unlock()

	starList = make([]AddrNodeIDMapping, 0)
	fileName := dataDir + "/starlist"
	fd, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0)
	if err != nil {
		log.Crit("Can't read file stalist")
		return
	}
	defer fd.Close()
	buff := bufio.NewReader(fd)
	log.Info("Star node list")
	for {
		line, err := buff.ReadString('\n')
		if err != nil {
			if io.EOF == err && strings.Compare(line, "") != 0 {

			} else {
				break
			}
		}
		line = strings.TrimSpace(line)
		tmp := strings.Split(line, " ")
		if len(tmp) != 2 {
			continue
		}
		addrStr := tmp[0]
		if strings.Index(tmp[0], "0x") == 0 || strings.Index(tmp[0], "0X") == 0 {
			addrStr = addrStr[2:]
		}
		var addr = common.HexToAddress(addrStr)
		var pubKey = common.Hex2Bytes(tmp[1])
		starList = append(starList, AddrNodeIDMapping{addr, pubKey})
		log.Info(fmt.Sprintf("addr:%s pubkey:%s", tmp[0], tmp[1]))
	}
}

// Get all sorted nodes that who can produce blocks
func GetAllSortedCoreNodes() []AddrNodeIDMapping {
	// TODO from合约
	if starList == nil {
		readStarList()
	}
	return starList
}

// 获取主节点数量
func GetCoreNodesCount() int {
	nodes := GetAllSortedCoreNodes()
	return len(nodes)
}

// 获取节点索引 后期可优化下
func GetCoreNodeIndex(address *common.Address) int {
	nodes := GetAllSortedCoreNodes()
	for i := 0; i < len(nodes); i++ {
		if bytes.Compare(nodes[i].Addr[:], address[:]) == 0 {
			return i
		}
	}
	return -1
}

// 根据pubkey获取节点索引
func GetCoreNodeIndexByPubkey(pubKey []byte) int {
	nodes := GetAllSortedCoreNodes()
	for i := 0; i < len(nodes); i++ {
		if bytes.Compare(nodes[i].Pubkey, pubKey[1:]) == 0 {
			return i
		}
	}
	return -1
}

// 通过出块者地址获取节点公钥
func GetPubkeyByAddress(address *common.Address) []byte {
	nodes := GetAllSortedCoreNodes()
	if len(nodes) == 0 {
		log.Error("GetAllSortedCoreNodes count=0")
	}
	for i := 0; i < len(nodes); i++ {
		if bytes.Compare(nodes[i].Addr[:], address[:]) == 0 {
			res := make([]byte, len(nodes[i].Pubkey))
			copy(res, nodes[i].Pubkey)
			return res
		}
	}
	log.Info("GetPubkeyByAddress is nil addr:", common.ToHex(address[:]))
	for _, node := range nodes {
		log.Info(fmt.Sprintf("addr:%s pubkey:%s", common.ToHex(node.Addr[:]), common.ToHex(node.Pubkey[:])))
	}
	return nil
}

// 根据publick key 获取地址
func GetAddressByPubkey(pubKey []byte) common.Address {
	if len(pubKey) == 65 {
		pubKey = pubKey[1:]
	}
	nodes := GetAllSortedCoreNodes()
	for i := 0; i < len(nodes); i++ {
		if bytes.Compare(nodes[i].Pubkey, pubKey) == 0 {
			return nodes[i].Addr
		}
	}
	return common.Address{}
}

// 获取最新块的出块者序号与本节点序号差
func GetSlot(firstAddress, nextAddress *common.Address) int {
	firstIndex := GetCoreNodeIndex(firstAddress)
	nextIndex := GetCoreNodeIndex(nextAddress)
	// 与创世块比较
	var emptyAddr [20]byte
	if bytes.Compare((*firstAddress)[:], emptyAddr[:]) == 0 {
		log.Debug("getSlot: firstAddress is empty")
		return nextIndex + 1
	}
	nodeCount := GetCoreNodesCount()
	// 只有一个主节点
	if nodeCount == 1 {
		log.Debug("getSlot: only one star node")
		return 1
	}
	return (nextIndex - firstIndex + nodeCount) % nodeCount
}
