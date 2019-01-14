package store

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"strconv"
	"testing"
)

func TestVoteRank_Rank(t *testing.T) {
	vote := NewVoteRank()
	result := vote.GetTop()
	assert.Equal(t, 0, len(result))
}

func isSort(src []*Candidate, dst []*Candidate, size int) bool {
	for index := 0; index < size; index++ {
		if src[index].total.Cmp(dst[index].total) != 0 {
			return false
		}
	}

	return true
}

func get0(size int) ([]*Candidate, []*Candidate) {
	src := make([]*Candidate, size)
	dst := make([]*Candidate, size)

	for index := 0; index < size; index++ {
		src[index] = &Candidate{
			address: common.HexToAddress(strconv.Itoa(index)),
			total:   big.NewInt(int64(index)),
		}
	}

	for index := size - 1; index >= 0; index-- {
		dst[(size-1)-index] = &Candidate{
			address: common.HexToAddress(strconv.Itoa(index)),
			total:   big.NewInt(int64(index)),
		}
	}

	return src, dst
}

func TestVoteRank_Rank0Put1(t *testing.T) {
	src, dst := get0(1)
	vote := NewVoteRank()
	vote.Rank(src)

	result := vote.GetTop()
	assert.Equal(t, 1, len(result))
	assert.Equal(t, true, isSort(result, dst, 1))
}

func TestVoteRank_Rank0Put29(t *testing.T) {
	src, dst := get0(29)
	vote := NewVoteRank()
	vote.Rank(src)

	result := vote.GetTop()
	assert.Equal(t, 29, len(result))
	assert.Equal(t, true, isSort(result, dst, 29))
}

func TestVoteRank_Rank0Put30(t *testing.T) {
	src, dst := get0(30)
	vote := NewVoteRank()
	vote.Rank(src)

	result := vote.GetTop()
	assert.Equal(t, 30, len(result))
	assert.Equal(t, true, isSort(result, dst, 30))
}

func TestVoteRank_Rank0Put31(t *testing.T) {
	src, dst := get0(31)
	vote := NewVoteRank()
	vote.Rank(src)

	result := vote.GetTop()
	assert.Equal(t, 30, len(result))
	assert.Equal(t, true, isSort(result, dst, 30))
}

func TestVoteRank_Rank0Put100(t *testing.T) {
	src, dst := get0(100)
	vote := NewVoteRank()
	vote.Rank(src)

	result := vote.GetTop()
	assert.Equal(t, 30, len(result))
	assert.Equal(t, true, isSort(result, dst, 30))
}

// total : 50w candidates
func TestVoteRank_Rank0Put500000(t *testing.T) {
	src, dst := get0(500000)
	vote := NewVoteRank()
	vote.Rank(src)

	result := vote.GetTop()
	assert.Equal(t, 30, len(result))
	assert.Equal(t, true, isSort(result, dst, 30))
}

func get15(vote *VoteRank, size int) ([]*Candidate, []*Candidate) {
	src1 := make([]*Candidate, 0)
	dst := make([]*Candidate, 15+size)

	index := 0
	for ; index < 15; index++ {
		candidate := &Candidate{
			address: common.HexToAddress(strconv.Itoa(index)),
			total:   big.NewInt(int64(index)),
		}
		src1 = append(src1, candidate)
	}
	vote.Rank(src1)

	src2 := make([]*Candidate, 0)
	for ; index < 15+size; index++ {
		candidate := &Candidate{
			address: common.HexToAddress(strconv.Itoa(index)),
			total:   big.NewInt(int64(index)),
		}

		src2 = append(src2, candidate)
	}

	for index := 15 + size - 1; index >= 0; index-- {
		dst[(15+size-1)-index] = &Candidate{
			address: common.HexToAddress(strconv.Itoa(index)),
			total:   big.NewInt(int64(index)),
		}
	}

	return src2, dst
}

func TestVoteRank_Rank15Put1(t *testing.T) {
	vote := NewVoteRank()
	src, dst := get15(vote, 1)

	vote.Rank(src)
	result := vote.GetTop()
	assert.Equal(t, 16, len(result))
	assert.Equal(t, true, isSort(result, dst, 16))
}

func TestVoteRank_Rank15Put15(t *testing.T) {
	vote := NewVoteRank()
	src, dst := get15(vote, 15)

	vote.Rank(src)
	result := vote.GetTop()
	assert.Equal(t, 30, len(result))
	assert.Equal(t, true, isSort(result, dst, 30))
}

func TestVoteRank_Rank15Put30(t *testing.T) {
	vote := NewVoteRank()
	src, dst := get15(vote, 30)

	vote.Rank(src)
	result := vote.GetTop()
	assert.Equal(t, 30, len(result))
	assert.Equal(t, true, isSort(result, dst, 30))
}

func get30(vote *VoteRank, size int) ([]*Candidate, []*Candidate) {
	src1 := make([]*Candidate, 0)
	dst := make([]*Candidate, 30+size)

	index := 0
	for ; index < 30; index++ {
		candidate := &Candidate{
			address: common.HexToAddress(strconv.Itoa(index)),
			total:   big.NewInt(int64(index)),
		}
		src1 = append(src1, candidate)
	}
	vote.Rank(src1)

	src2 := make([]*Candidate, 0)
	for ; index < 30+size; index++ {
		candidate := &Candidate{
			address: common.HexToAddress(strconv.Itoa(index)),
			total:   big.NewInt(int64(index)),
		}

		src2 = append(src2, candidate)
	}

	for index := 30 + size - 1; index >= 0; index-- {
		dst[(30+size-1)-index] = &Candidate{
			address: common.HexToAddress(strconv.Itoa(index)),
			total:   big.NewInt(int64(index)),
		}
	}

	return src2, dst
}

func TestVoteRank_Rank30Put1(t *testing.T) {
	vote := NewVoteRank()
	src, dst := get30(vote, 1)

	vote.Rank(src)
	result := vote.GetTop()
	assert.Equal(t, 30, len(result))
	assert.Equal(t, true, isSort(result, dst, 30))
}

func TestVoteRank_Rank30Put30(t *testing.T) {
	vote := NewVoteRank()
	src, dst := get30(vote, 30)

	vote.Rank(src)
	result := vote.GetTop()
	assert.Equal(t, 30, len(result))
	assert.Equal(t, true, isSort(result, dst, 30))
}

func TestVoteRank_Rank30Put4000(t *testing.T) {
	vote := NewVoteRank()
	src, dst := get30(vote, 4000)

	vote.Rank(src)
	result := vote.GetTop()
	assert.Equal(t, 30, len(result))
	assert.Equal(t, true, isSort(result, dst, 30))
}

func TestVoteRank_RankOverlapping(t *testing.T) {
	vote := NewVoteRank()

	src1 := make([]*Candidate, 0)
	index := 0
	for ; index < 30; index++ {
		src1 = append(src1, &Candidate{
			address: common.HexToAddress(strconv.Itoa(index)),
			total:   big.NewInt(int64(index + 100)),
		})
	}
	vote.Rank(src1)

	//
	index = 0
	tmp := make([]*Candidate, 0)
	for ; index < 10; index++ {
		tmp = append(tmp, &Candidate{
			address: common.HexToAddress(strconv.Itoa(index)),
			total:   big.NewInt(int64(index)),
		})
	}
	tmp[0].total = big.NewInt(int64(60000))
	tmp[9].total = big.NewInt(int64(100000))

	//
	index = 30
	src2 := make([]*Candidate, 0)
	for ; index < 5000; index++ {
		src2 = append(src2, &Candidate{
			address: common.HexToAddress(strconv.Itoa(index)),
			total:   big.NewInt(int64(index)),
		})
	}

	index = 0
	for ; index < len(tmp); index++ {
		src2 = append(src2, tmp[index])
	}

	vote.Rank(src2)
	result := vote.GetTop()
	assert.Equal(t, 30, len(result))

	dst := make([]*Candidate, 30)
	dst[0] = &Candidate{
		address: common.HexToAddress(strconv.Itoa(9)),
		total:   big.NewInt(int64(100000)),
	}
	dst[1] = &Candidate{
		address: common.HexToAddress(strconv.Itoa(0)),
		total:   big.NewInt(int64(60000)),
	}

	index = 2
	for i := 5000 - 1; i > 0 && index < 30; i-- {
		dst[index] = &Candidate{
			address: common.HexToAddress(strconv.Itoa(i)),
			total:   big.NewInt(int64(i)),
		}
		index = index + 1
	}

	assert.Equal(t, true, isSort(result, dst, 30))
}
