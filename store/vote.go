package store

import (
	"bytes"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"math/big"
)

type Candidate struct {
	address common.Address
	nodeID  []byte
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

func (candidate *Candidate) Clone() *Candidate {
	return &Candidate{
		address: candidate.GetAddress(),
		total:   candidate.GetTotal(),
	}
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
		copy.Top[index] = top.Top[index].Clone()
	}

	return copy
}

func (top *VoteTop) Max() *Candidate {
	if top.TopCnt <= 0 {
		return nil
	} else {
		return top.Top[0].Clone()
	}
}

func (top *VoteTop) Min() *Candidate {
	if top.TopCnt <= 0 {
		return nil
	} else {
		return top.Top[top.TopCnt-1].Clone()
	}
}

func (top *VoteTop) Del(address common.Address) {
	for index := 0; index < top.TopCnt; index++ {
		if bytes.Compare(top.Top[index].address[:], address[:]) != 0 {
			continue
		} else {
			copy(top.Top[0:index], top.Top[index+1:])
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
		result[index] = top.Top[index].Clone()
	}
	return result
}

func (top *VoteTop) ToHashMap() map[common.Address]*Candidate {
	result := make(map[common.Address]*Candidate)
	for index := 0; index < top.TopCnt; index++ {
		result[top.Top[index].address] = top.Top[index]
	}
	return result
}

func (top *VoteTop) ToSlice(src map[common.Address]*Candidate) {
	if len(src) <= 0 {
		top.Clear()
	} else {
		top.TopCnt = len(src)
		top.Top = make([]*Candidate, 0, len(src))
		for _, v := range src {
			top.Top = append(top.Top, v)
		}
	}
}

func (top *VoteTop) Reset(candidates []*Candidate) {
	if len(candidates) <= 0 {
		top.Clear()
	} else {
		top.Top = candidates
		top.TopCnt = len(candidates)
	}
}

func (top *VoteTop) Rank(topSize int) {
	result := top.ranking(topSize, top.Top)
	top.Reset(result)
}

func (top *VoteTop) RankAll(topSize int, candidates []*Candidate) {
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
				if candidates[i].total.Cmp(candidates[j].total) < 0 {
					candidates[i], candidates[j] = candidates[j], candidates[i]
				}

				if (candidates[i].total.Cmp(candidates[j].total) == 0) &&
					(bytes.Compare(candidates[i].address[:], candidates[j].address[:]) > 0) {
					candidates[i], candidates[j] = candidates[j], candidates[i]
				}
			}
			result[i] = candidates[i]
		}
		return result
	}
}
