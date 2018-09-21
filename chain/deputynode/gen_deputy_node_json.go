// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package deputynode

import (
	"encoding/json"
	"errors"
	"net"

	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/math"
)

var _ = (*Marshaling)(nil)

// MarshalJSON marshals as JSON.
func (d DeputyNode) MarshalJSON() ([]byte, error) {
	type DeputyNode struct {
		LemoBase common.Address      `json:"lemoBase"   gencodec:"required"`
		NodeID   hexutil.Bytes       `json:"nodeID"     gencodec:"required"`
		IP       hexutil.IP          `json:"ip"         gencodec:"required"`
		Port     math.HexOrDecimal64 `json:"port"       gencodec:"required"`
		Rank     math.HexOrDecimal64 `json:"rank"       gencodec:"required"`
		Votes    math.HexOrDecimal64 `json:"votes"      gencodec:"required"`
	}
	var enc DeputyNode
	enc.LemoBase = d.LemoBase
	enc.NodeID = d.NodeID
	enc.IP = hexutil.IP(d.IP)
	enc.Port = math.HexOrDecimal64(d.Port)
	enc.Rank = math.HexOrDecimal64(d.Rank)
	enc.Votes = math.HexOrDecimal64(d.Votes)
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (d *DeputyNode) UnmarshalJSON(input []byte) error {
	type DeputyNode struct {
		LemoBase *common.Address      `json:"lemoBase"   gencodec:"required"`
		NodeID   *hexutil.Bytes       `json:"nodeID"     gencodec:"required"`
		IP       *hexutil.IP          `json:"ip"         gencodec:"required"`
		Port     *math.HexOrDecimal64 `json:"port"       gencodec:"required"`
		Rank     *math.HexOrDecimal64 `json:"rank"       gencodec:"required"`
		Votes    *math.HexOrDecimal64 `json:"votes"      gencodec:"required"`
	}
	var dec DeputyNode
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.LemoBase == nil {
		return errors.New("missing required field 'lemoBase' for DeputyNode")
	}
	d.LemoBase = *dec.LemoBase
	if dec.NodeID == nil {
		return errors.New("missing required field 'nodeID' for DeputyNode")
	}
	d.NodeID = *dec.NodeID
	if dec.IP == nil {
		return errors.New("missing required field 'ip' for DeputyNode")
	}
	d.IP = net.IP(*dec.IP)
	if dec.Port == nil {
		return errors.New("missing required field 'port' for DeputyNode")
	}
	d.Port = uint(*dec.Port)
	if dec.Rank == nil {
		return errors.New("missing required field 'rank' for DeputyNode")
	}
	d.Rank = uint(*dec.Rank)
	if dec.Votes == nil {
		return errors.New("missing required field 'votes' for DeputyNode")
	}
	d.Votes = uint64(*dec.Votes)
	return nil
}
