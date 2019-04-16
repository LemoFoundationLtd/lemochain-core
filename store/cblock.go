package store

import (
	"bytes"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"math/big"
	"strconv"
)

type CBlock struct {
	Block           *types.Block
	AccountTrieDB   *AccountTrieDB
	CandidateTrieDB *CandidateTrieDB
	Top             *VoteTop
	Parent          *CBlock
	Children        []*CBlock
}

func NewGenesisBlock(block *types.Block, reader DatabaseReader) *CBlock {
	return &CBlock{
		Block:           block,
		AccountTrieDB:   NewEmptyAccountTrieDB(reader),
		CandidateTrieDB: NewEmptyCandidateTrieDB(),
		Top:             NewEmptyVoteTop(),
	}
}

func NewNormalBlock(block *types.Block, accountTrieDB *AccountTrieDB, candidateTrieDB *CandidateTrieDB, top *VoteTop) *CBlock {
	return &CBlock{
		Block:           block,
		AccountTrieDB:   accountTrieDB.Clone(),
		CandidateTrieDB: candidateTrieDB.Clone(),
		Top:             top.Clone(),
	}
}

func (block *CBlock) toHashMap(src []*Candidate) map[common.Address]*Candidate {
	result := make(map[common.Address]*Candidate)
	for index := 0; index < len(src); index++ {
		result[src[index].Address] = src[index]
	}
	return result
}

func (block *CBlock) toSlice(src map[common.Address]*Candidate) []*Candidate {
	if len(src) <= 0 {
		return make([]*Candidate, 0)
	} else {
		dst := make([]*Candidate, 0, len(src))
		for _, v := range src {
			dst = append(dst, v.Copy())
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

func (block *CBlock) filterCandidates(accounts []*types.AccountData) []*Candidate {
	candidates := make([]*Candidate, 0)
	for index := 0; index < len(accounts); index++ {
		account := accounts[index]
		if block.isCandidate(account) {
			candidates = append(candidates, &Candidate{
				Address: account.Address,
				Total:   new(big.Int).Set(account.Candidate.Votes),
			})
		}
	}
	return candidates
}

func (block *CBlock) data(candidates []*Candidate) ([]*Candidate, []*Candidate) {
	in := block.Top.GetTop()
	out := make([]*Candidate, 0)

	if len(candidates) <= 0 {
		return in, out
	}

	inMap := block.toHashMap(in)
	for index := 0; index < len(candidates); index++ {
		candidate := candidates[index]
		_, ok := inMap[candidate.Address]
		if ok {
			inMap[candidate.Address] = candidate
		} else {
			out = append(out, candidate)
		}
	}

	return block.toSlice(inMap), out
}

func (block *CBlock) dye(candidates []*Candidate) {
	if len(candidates) <= 0 {
		return
	}

	for index := 0; index < len(candidates); index++ {
		candidate := candidates[index]
		block.CandidateTrieDB.Put(candidate, block.Block.Height())
	}
}

func (block *CBlock) less30(in []*Candidate, out []*Candidate) {
	block.Top.Rank(max_candidate_count, append(in, out...))
}

func (block *CBlock) minIsReduce(oldMin *Candidate, newMin *Candidate) bool {
	if (oldMin != nil) &&
		(newMin != nil) &&
		(newMin.Total.Cmp(oldMin.Total) >= 0) {
		return false
	} else {
		return true
	}
}

func (block *CBlock) canPick(src *Candidate, dst *Candidate) bool {
	if (src.Total.Cmp(dst.Total) < 0) ||
		((src.Total.Cmp(dst.Total) == 0) && (bytes.Compare(src.Address[:], dst.Address[:]) < 0)) {
		return true
	} else {
		return false
	}
}

func (block *CBlock) greater30(in []*Candidate, out []*Candidate) {
	top := NewEmptyVoteTop()
	top.Rank(max_candidate_count, in)
	newMin := top.Min()
	oldMin := block.Top.Min()
	if !block.minIsReduce(oldMin, newMin) {
		for index := 0; index < len(out); index++ {
			candidate := out[index]
			if block.canPick(newMin, candidate) {
				in = append(in, candidate)
			}
		}
		block.Top.Rank(max_candidate_count, in)
	} else {
		candidates := block.CandidateTrieDB.GetAll()
		block.Top.Rank(max_candidate_count, candidates)
	}
}

func (block *CBlock) Ranking() {
	height := block.Block.Height()
	accounts := block.AccountTrieDB.Collect(height)
	if len(accounts) <= 0 {
		return
	}

	candidates := block.filterCandidates(accounts)
	block.dye(candidates)
	in, out := block.data(candidates)
	if block.Top.Count() < max_candidate_count {
		block.less30(in, out)
	} else {
		block.greater30(in, out)
	}
}

func (block *CBlock) BeChildOf(parent *CBlock) {
	block.Parent = parent

	if parent != nil {
		// check if exist
		for _, child := range parent.Children {
			if child == block {
				return
			}
		}

		parent.Children = append(parent.Children, block)
	}
}

func (block *CBlock) IsSameBlock(b *CBlock) bool {
	if block == b {
		return true
	}
	if block == nil || b == nil {
		return false
	}
	return block.Block.Hash() == b.Block.Hash()
}

// CollectToParent collect blocks from parent to parent, include itself and exclude the end block
func (block *CBlock) CollectToParent(end *CBlock) []*CBlock {
	blocks := make([]*CBlock, 0)
	for iter := block; iter != end && iter != nil; iter = iter.Parent {
		blocks = append(blocks, iter)
	}
	return blocks
}

// Walk iterate every child recursively. Not include itself
func (block *CBlock) Walk(fn func(*CBlock), exclude *CBlock) {
	for _, child := range block.Children {
		if exclude == nil || !child.IsSameBlock(exclude) {
			fn(child)
			child.Walk(fn, exclude)
		}
	}
}

// ChooseBlock compare children and choose one of them, then choose its children recursively
func (block *CBlock) ChooseChild(compareFn func(*CBlock, *CBlock) *CBlock) *CBlock {
	count := len(block.Children)
	if count == 0 {
		return block
	}

	best := 0
	for i := 1; i < count; i++ {
		betterOne := compareFn(block.Children[best], block.Children[i])
		if betterOne == block.Children[i] {
			best = i
		} else if betterOne != block.Children[best] { // for debug
			panic("must choose one from the parameters")
		}
	}
	return block.Children[best].ChooseChild(compareFn)
}
