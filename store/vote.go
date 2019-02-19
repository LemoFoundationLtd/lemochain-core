package store

import (
	"bytes"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"math/big"
)

type Candidate struct {
	Address common.Address
	Total   *big.Int
}

func (candidate *Candidate) GetAddress() common.Address {
	return candidate.Address
}

func (candidate *Candidate) GetTotal() *big.Int {
	return new(big.Int).Set(candidate.Total)
}

func (candidate *Candidate) Copy() *Candidate {
	return &Candidate{
		Address: candidate.GetAddress(),
		Total:   candidate.GetTotal(),
	}
}

func (candidate *Candidate) Clone() types.NodeData {
	return candidate.Copy()
}

type VoteTop struct {
	Top []*Candidate
}

func NewEmptyVoteTop() *VoteTop {
	return &VoteTop{Top: make([]*Candidate, 0)}
}

func NewVoteTop(top []*Candidate) *VoteTop {
	voteTop := &VoteTop{}
	if len(top) <= 0 {
		voteTop.Top = make([]*Candidate, 0)
	} else {
		voteTop.Top = make([]*Candidate, len(top))
		for index := 0; index < len(top); index++ {
			voteTop.Top[index] = top[index].Copy()
		}
	}
	return voteTop
}

func (top *VoteTop) Clone() *VoteTop {
	return NewVoteTop(top.Top)
}

func (top *VoteTop) Max() *Candidate {
	if len(top.Top) <= 0 {
		return nil
	} else {
		return top.Top[0].Copy()
	}
}

func (top *VoteTop) Min() *Candidate {
	if len(top.Top) <= 0 {
		return nil
	} else {
		return top.Top[len(top.Top)-1].Copy()
	}
}

func (top *VoteTop) Clear() {
	top.Top = make([]*Candidate, 0)
}

func (top *VoteTop) Count() int {
	return len(top.Top)
}

func (top *VoteTop) GetTop() []*Candidate {
	result := make([]*Candidate, len(top.Top))
	for index := 0; index < len(top.Top); index++ {
		result[index] = top.Top[index].Copy()
	}
	return result
}

func (top *VoteTop) Reset(candidates []*Candidate) {
	if len(candidates) <= 0 {
		top.Top = make([]*Candidate, 0)
	} else {
		top.Top = make([]*Candidate, len(candidates))
		for index := 0; index < len(candidates); index++ {
			top.Top[index] = candidates[index].Copy()
		}
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
				val := candidates[i].Total.Cmp(candidates[j].Total)

				if val < 0 {
					candidates[i], candidates[j] = candidates[j], candidates[i]
				} else {
					if (val == 0) &&
						(bytes.Compare(candidates[i].Address[:], candidates[j].Address[:]) > 0) {
						candidates[i], candidates[j] = candidates[j], candidates[i]
					}
				}
			}
			result[i] = candidates[i]
		}
		return result
	}
}
