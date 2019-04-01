package deputynode

import (
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto/sha3"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"math/big"
	"net"
)

//go:generate gencodec -type DeputyNode --field-override deputyNodeMarshaling -out gen_deputy_node_json.go

// DeputyNode
type DeputyNode struct {
	MinerAddress common.Address `json:"minerAddress"   gencodec:"required"`
	NodeID       []byte         `json:"nodeID"         gencodec:"required"`
	IP           net.IP         `json:"ip"             gencodec:"required"` // ip
	Port         uint32         `json:"port"           gencodec:"required"` // 端口
	Rank         uint32         `json:"rank"           gencodec:"required"` // 排名 从0开始
	Votes        *big.Int       `json:"votes"          gencodec:"required"` // 得票数
}

type deputyNodeMarshaling struct {
	NodeID hexutil.Bytes
	IP     hexutil.IP
	Port   hexutil.Uint32
	Rank   hexutil.Uint32
	Votes  *hexutil.Big10
}

func (d *DeputyNode) Hash() (h common.Hash) {
	data := []interface{}{
		d.MinerAddress,
		d.NodeID,
		d.IP,
		d.Port,
		d.Rank,
		d.Votes,
	}
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, data)
	hw.Sum(h[:0])
	return h
}

func (d *DeputyNode) Check() error {
	if len(d.NodeID) != 64 {
		log.Errorf("incorrect field: 'NodeID'. value: %s", common.ToHex(d.NodeID))
		return errors.New("incorrect field: 'NodeID'")
	}
	if d.Port > 65535 {
		log.Errorf("incorrect field: 'port'. value: %d", d.Port)
		return errors.New("max deputy node's port is 65535")
	}
	if d.Rank > 65535 {
		log.Errorf("incorrect field: 'rank'. value: %d", d.Rank)
		return errors.New("max deputy node's rank is 65535")
	}
	return nil
}

type DeputyNodes []*DeputyNode

func (nodes DeputyNodes) String() string {
	if buf, err := json.Marshal(nodes); err == nil {
		return string(buf)
	}
	return ""
}
