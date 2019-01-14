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
