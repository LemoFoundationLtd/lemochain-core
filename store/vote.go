package store

import (
	"bytes"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
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
	return candidate.total
}

func (candidate *Candidate) Clone() *Candidate {
	return &Candidate{
		address: candidate.address,
		total:   new(big.Int).Set(candidate.total),
	}
}

type VoteTop struct {
	TopCnt int
	TopCap int
	Top    []*Candidate
}

func NewVoteTop() *VoteTop {
	return &VoteTop{
		TopCnt: 0,
		TopCap: 30,
		Top:    make([]*Candidate, 0, 30),
	}
}

func (top *VoteTop) Clone() *VoteTop {
	copy := &VoteTop{
		TopCnt: top.TopCnt,
		TopCap: top.TopCap,
	}

	copy.Top = make([]*Candidate, top.TopCnt)
	for index := 0; index < top.TopCnt; index++ {
		copy.Top[index] = top.Top[index].Clone()
	}

	return copy
}

func (top *VoteTop) Rank() {
	for i := 0; i < top.TopCnt; i++ {
		for j := i + 1; j < top.TopCnt; j++ {
			if top.Top[i].total.Cmp(top.Top[j].total) < 0 {
				top.Top[i], top.Top[j] = top.Top[j], top.Top[i]
			}

			if (top.Top[i].total.Cmp(top.Top[j].total) == 0) &&
				(bytes.Compare(top.Top[i].address[:], top.Top[j].address[:]) < 0) {
				top.Top[i], top.Top[j] = top.Top[j], top.Top[i]
			}
		}
	}
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
			break
		}
	}
}

func (top *VoteTop) Clear() {
	top.TopCnt = 0
}

func (top *VoteTop) Count() int {
	return top.TopCnt
}

func (top *VoteTop) ToHashMap() map[common.Address]*Candidate {
	result := make(map[common.Address]*Candidate)
	for index := 0; index < len(top.Top); index++ {
		result[top.Top[index].address] = top.Top[index]
	}
	return result
}

func (top *VoteTop) Reset(candidates []*Candidate) {
	if len(candidates) <= 0 {
		top.TopCnt = 0
	} else {
		//top.Top = append(top.Top[0:], candidates...)
		top.Top = candidates
		top.TopCnt = len(candidates)
	}
}

type VoteRank struct {
	TopCnt int
	TopCap int
	Top    []*Candidate

	CacheCnt  int
	RankCache []*Candidate
}

func NewVoteRank() *VoteRank {
	vote := new(VoteRank)
	vote.TopCnt = 0
	vote.TopCap = 30
	vote.Top = make([]*Candidate, vote.TopCap)

	vote.CacheCnt = 0
	vote.RankCache = make([]*Candidate, 1024)
	return vote
}

func (vote *VoteRank) GetTop() []*Candidate {
	if vote.TopCnt <= 0 {
		return make([]*Candidate, 0)
	} else {
		result := make([]*Candidate, vote.TopCnt)
		for index := 0; index < vote.TopCnt; index++ {
			result[index] = vote.Top[index]
		}

		return result
	}
}

func (vote *VoteRank) rank() {
	if vote.CacheCnt <= 0 {
		return
	}

	if vote.CacheCnt == 1 {
		vote.Top[0] = vote.RankCache[0]
		vote.TopCnt = 1
		return
	}

	minCnt := min(vote.TopCap, vote.CacheCnt)
	vote.TopCnt = 0
	for i := 0; i < minCnt; i++ {
		for j := i + 1; j < vote.CacheCnt; j++ {
			if vote.RankCache[i].total.Cmp(vote.RankCache[j].total) < 0 {
				vote.RankCache[i], vote.RankCache[j] = vote.RankCache[j], vote.RankCache[i]
			}

			if (vote.RankCache[i].total.Cmp(vote.RankCache[j].total) == 0) &&
				(bytes.Compare(vote.RankCache[i].address[:], vote.RankCache[j].address[:]) < 0) {
				vote.RankCache[i], vote.RankCache[j] = vote.RankCache[j], vote.RankCache[i]
			}
		}
		vote.Top[vote.TopCnt] = vote.RankCache[i]
		vote.TopCnt = vote.TopCnt + 1
	}
}

func (vote *VoteRank) puts(candidates []*Candidate) {
	if len(candidates) <= 0 {
		return
	}

	total := vote.TopCnt + len(candidates)
	if cap(vote.RankCache) < total {
		vote.RankCache = make([]*Candidate, total)
	}

	if vote.TopCnt <= 0 {
		for index := 0; index < len(candidates); index++ {
			vote.RankCache[index] = candidates[index]
		}
		vote.CacheCnt = len(candidates)
		return
	}

	tmp := make(map[common.Address]*Candidate)
	for index := 0; index < vote.TopCnt; index++ {
		tmp[vote.Top[index].address] = vote.Top[index]
	}

	if vote.TopCnt < vote.TopCap {
		for index := 0; index < len(candidates); index++ {
			tmp[candidates[index].address] = candidates[index]
		}
	} else {
		lastMin := vote.Top[vote.TopCnt-1]
		for index := 0; index < len(candidates); index++ {
			_, ok := tmp[candidates[index].address]
			if !ok && lastMin.total.Cmp(candidates[index].total) > 0 {
				continue
			}

			if !ok &&
				(lastMin.total.Cmp(candidates[index].total) == 0) &&
				(bytes.Compare(lastMin.address[:], candidates[index].address[:]) < 0) {
				continue
			}

			tmp[candidates[index].address] = candidates[index]
		}
	}

	index := 0
	for _, v := range tmp {
		vote.RankCache[index] = v
		index = index + 1
	}
	vote.CacheCnt = len(tmp)
}

func (vote *VoteRank) Rank(candidates []*Candidate) {
	if len(candidates) <= 0 {
		return
	} else {
		vote.puts(candidates)
		vote.rank()
	}
}

func (vote *VoteRank) RankAll(candidates []*Candidate) {
	vote.TopCnt = 0

	if len(candidates) <= 0 {
		return
	} else {
		minCnt := min(vote.TopCap, len(candidates))
		for i := 0; i < minCnt; i++ {
			for j := i + 1; j < len(candidates); j++ {
				log.Warnf("zh candidates[i].total: %v", candidates[i])
				if candidates[i].total.Cmp(candidates[j].total) < 0 {
					candidates[i], candidates[j] = candidates[j], candidates[i]
				}

				if (candidates[i].total.Cmp(candidates[j].total) == 0) &&
					(bytes.Compare(candidates[i].address[:], candidates[j].address[:]) < 0) {
					candidates[i], candidates[j] = candidates[j], candidates[i]
				}
			}
			vote.Top[vote.TopCnt] = candidates[i]
			vote.TopCnt = vote.TopCnt + 1
		}
	}
}
