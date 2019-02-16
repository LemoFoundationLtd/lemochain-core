package store

import (
	"bytes"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"math/big"
	"strconv"
)

type CBlock struct {
	Block           *types.Block
	AccountTrieDB   *AccountTrieDB
	CandidateTrieDB *CandidateTrieDB
	Top30           []*Candidate
}

func NewGenesisBlock(block *types.Block, reader DatabaseReader) *CBlock {
	return &CBlock{
		Block:           block,
		AccountTrieDB:   NewEmptyAccountTrieDB(reader),
		CandidateTrieDB: NewEmptyCandidateTrieDB(),
		Top30:           make([]*Candidate, 0),
	}
}

func NewNormalBlock(block *types.Block, accountTrieDB *AccountTrieDB, candidateTrieDB *CandidateTrieDB, top30 []*Candidate) *CBlock {
	cblock := &CBlock{
		Block:           block,
		AccountTrieDB:   accountTrieDB.Clone(),
		CandidateTrieDB: candidateTrieDB.Clone(),
	}

	if len(top30) <= 0 {
		cblock.Top30 = make([]*Candidate, 0)
	} else {
		cblock.Top30 = make([]*Candidate, len(top30))
		for index := 0; index < len(top30); index++ {
			cblock.Top30[index] = top30[index].Copy()
		}
	}

	return cblock
}

func (block *CBlock) toHashMap(src []*Candidate) map[common.Address]*Candidate {
	result := make(map[common.Address]*Candidate)
	for index := 0; index < len(src); index++ {
		result[src[index].address] = src[index]
	}
	return result
}

func (block *CBlock) toSlice(src map[common.Address]*Candidate) []*Candidate {
	if len(src) <= 0 {
		return make([]*Candidate, 0)
	} else {
		dst := make([]*Candidate, 0, len(src))
		for _, v := range src {
			dst = append(dst, v)
		}
		return dst
	}
}

func (block *CBlock) isCandidate(account *types.AccountData) bool {
	if (account == nil) ||
		(len(account.Candidate.Profile) <= 0) {
		return false
	}

	result, ok := account.Candidate.Profile[types.CandidateKeyIsCandidate]
	if !ok {
		return false
	}

	val, err := strconv.ParseBool(result)
	if err != nil {
		panic("to bool err : " + err.Error())
	} else {
		return val
	}
}

func (block *CBlock) data(accounts []*types.AccountData, lastCandidatesMap map[common.Address]*Candidate) (map[common.Address]*Candidate, []*Candidate) {
	nextCandidates := make([]*Candidate, 0)

	for index := 0; index < len(accounts); index++ {
		account := accounts[index]
		isCandidate := block.isCandidate(account)
		if isCandidate {
			candidate := &Candidate{
				address: account.Address,
				total:   new(big.Int).Set(account.Candidate.Votes),
			}
			nextCandidates = append(nextCandidates, candidate)
			block.CandidateTrieDB.Put(candidate, block.Block.Height())
		}

		_, ok := lastCandidatesMap[account.Address]
		if ok {
			if !isCandidate {
				panic("can't cancel candidate: " + account.Address.Hex())
			} else {
				lastCandidatesMap[account.Address].total.Set(account.Candidate.Votes)
			}
		}
	}

	return lastCandidatesMap, nextCandidates
}

func (block *CBlock) less30(lastCandidatesMap map[common.Address]*Candidate, nextCandidates []*Candidate) {
	for index := 0; index < len(nextCandidates); index++ {
		lastCandidatesMap[nextCandidates[index].address] = nextCandidates[index]
	}
	voteTop := NewVoteTop(block.Top30)
	voteTop.Rank(max_candidate_count, block.toSlice(lastCandidatesMap))
	block.Top30 = voteTop.GetTop()
}

func (block *CBlock) greater30(lastCandidatesMap map[common.Address]*Candidate, nextCandidates []*Candidate) {
	voteTop := NewVoteTop(block.Top30)
	lastMinCandidate := voteTop.Min()
	lastCount := voteTop.Count()

	voteTop.Rank(max_candidate_count, block.toSlice(lastCandidatesMap))
	if (lastMinCandidate != nil) &&
		(lastCount == voteTop.Count()) &&
		(lastMinCandidate.total.Cmp(voteTop.Min().total) <= 0) {

		lastMinCandidate = voteTop.Min()
		for index := 0; index < len(nextCandidates); index++ {
			if (lastMinCandidate.total.Cmp(nextCandidates[index].total) < 0) ||
				((lastMinCandidate.total.Cmp(nextCandidates[index].total) == 0) && (bytes.Compare(lastMinCandidate.address[:], nextCandidates[index].address[:]) < 0)) {
				_, ok := lastCandidatesMap[nextCandidates[index].address]
				if !ok {
					lastCandidatesMap[nextCandidates[index].address] = nextCandidates[index]
				}
			}
		}

		voteTop.Rank(max_candidate_count, block.toSlice(lastCandidatesMap))
		block.Top30 = voteTop.GetTop()
	} else {
		candidates := block.CandidateTrieDB.GetAll()
		voteTop.Rank(max_candidate_count, candidates)
		block.Top30 = voteTop.GetTop()
	}
}

func (block *CBlock) Ranking() {
	height := block.Block.Height()
	accounts := block.AccountTrieDB.Collect(height)
	if len(accounts) <= 0 {
		return
	}

	lastCandidatesMap, nextCandidates := block.data(accounts, block.toHashMap(block.Top30))
	if len(block.Top30) < max_candidate_count {
		block.less30(lastCandidatesMap, nextCandidates)
		return
	} else {
		block.greater30(lastCandidatesMap, nextCandidates)
		return
	}
}
