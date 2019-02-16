package store

import (
	"bytes"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"math/big"
)

type Candidate struct {
	address common.Address
	total   *big.Int
}

func (candidate *Candidate) GetAddress() common.Address {
	return candidate.address
}

func (candidate *Candidate) GetNodeID() []byte {
	// return candidate.nodeID
	panic("don't call the method.")
}

func (candidate *Candidate) GetTotal() *big.Int {
	return new(big.Int).Set(candidate.total)
}

func (candidate *Candidate) Copy() *Candidate {
	return &Candidate{
		address: candidate.GetAddress(),
		total:   candidate.GetTotal(),
	}
}

func (candidate *Candidate) Clone() types.NodeData {
	return candidate.Copy()
}

type VoteTop struct {
	TopCnt int
	Top    []*Candidate
}

func NewVoteTop(top []*Candidate) *VoteTop {
	return &VoteTop{TopCnt: len(top), Top: top}
}

func (top *VoteTop) Clone() *VoteTop {
	copy := &VoteTop{TopCnt: top.TopCnt}

	copy.Top = make([]*Candidate, top.TopCnt)
	for index := 0; index < top.TopCnt; index++ {
		copy.Top[index] = top.Top[index].Copy()
	}

	return copy
}

func (top *VoteTop) Max() *Candidate {
	if top.TopCnt <= 0 {
		return nil
	} else {
		return top.Top[0].Copy()
	}
}

func (top *VoteTop) Min() *Candidate {
	if top.TopCnt <= 0 {
		return nil
	} else {
		return top.Top[top.TopCnt-1].Copy()
	}
}

func (top *VoteTop) Del(address common.Address) {
	for index := 0; index < top.TopCnt; index++ {
		if bytes.Compare(top.Top[index].address[:], address[:]) != 0 {
			continue
		} else {
			top.Top = append(top.Top[0:index], top.Top[index+1:]...)
			top.TopCnt = top.TopCnt - 1
			break
		}
	}
}

func (top *VoteTop) Clear() {
	top.TopCnt = 0
	top.Top = make([]*Candidate, 0)
}

func (top *VoteTop) Count() int {
	return top.TopCnt
}

func (top *VoteTop) GetTop() []*Candidate {
	result := make([]*Candidate, top.TopCnt)
	for index := 0; index < top.TopCnt; index++ {
		result[index] = top.Top[index].Copy()
	}
	return result
}

func (top *VoteTop) Reset(candidates []*Candidate) {
	if len(candidates) <= 0 {
		top.Clear()
	} else {
		top.Top = candidates
		top.TopCnt = len(candidates)
	}
}

func (top *VoteTop) Rank(topSize int, candidates []*Candidate) {
	result := top.ranking(topSize, candidates)
	top.Reset(result)
}

func (top *VoteTop) ranking(topSize int, candidates []*Candidate) []*Candidate {
	length := len(candidates)

	if length <= 0 {
		return make([]*Candidate, 0)
	} else if length == 1 {
		result := make([]*Candidate, 1)
		result[0] = candidates[0]
		return result
	} else {
		minCnt := min(topSize, length)
		result := make([]*Candidate, minCnt)
		for i := 0; i < minCnt; i++ {
			for j := i + 1; j < length; j++ {
				val := candidates[i].total.Cmp(candidates[j].total)

				if val < 0 {
					candidates[i], candidates[j] = candidates[j], candidates[i]
				} else {
					if (val == 0) &&
						(bytes.Compare(candidates[i].address[:], candidates[j].address[:]) > 0) {
						candidates[i], candidates[j] = candidates[j], candidates[i]
					}
				}
			}
			result[i] = candidates[i]
		}
		return result
	}
}
