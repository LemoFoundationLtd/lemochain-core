package store

import (
	"bytes"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"strconv"
	"testing"
)

func isSort(src []*Candidate, dst []*Candidate, size int) bool {
	for index := 0; index < size; index++ {
		if src[index].Total.Cmp(dst[index].Total) != 0 {
			return false
		}
	}

	return true
}

func isEqual(src *Candidate, dst *Candidate) bool {
	if src == nil && dst == nil {
		return true
	}

	if src == nil || dst == nil {
		return false
	}

	return (bytes.Compare(src.Address[:], dst.Address[:]) == 0) && (src.Total.Cmp(dst.Total) == 0)
}

func TestNewVoteTop(t *testing.T) {

	data := []*Candidate{
		&Candidate{
			Address: common.HexToAddress("0x01"),
			Total:   new(big.Int).SetInt64(1),
		},
		&Candidate{
			Address: common.HexToAddress("0x03"),
			Total:   new(big.Int).SetInt64(3),
		},
		&Candidate{
			Address: common.HexToAddress("0x04"),
			Total:   new(big.Int).SetInt64(3),
		},
		&Candidate{
			Address: common.HexToAddress("0x02"),
			Total:   new(big.Int).SetInt64(2),
		},
		&Candidate{
			Address: common.HexToAddress("0x05"),
			Total:   new(big.Int).SetInt64(5),
		},
	}
	result := []*Candidate{
		&Candidate{
			Address: common.HexToAddress("0x05"),
			Total:   new(big.Int).SetInt64(5),
		},
		&Candidate{
			Address: common.HexToAddress("0x03"),
			Total:   new(big.Int).SetInt64(3),
		},
		&Candidate{
			Address: common.HexToAddress("0x04"),
			Total:   new(big.Int).SetInt64(3),
		},
		&Candidate{
			Address: common.HexToAddress("0x02"),
			Total:   new(big.Int).SetInt64(2),
		},
		&Candidate{
			Address: common.HexToAddress("0x01"),
			Total:   new(big.Int).SetInt64(1),
		},
	}
	max := &Candidate{
		Address: common.HexToAddress("0x05"),
		Total:   new(big.Int).SetInt64(5),
	}
	min := &Candidate{
		Address: common.HexToAddress("0x01"),
		Total:   new(big.Int).SetInt64(1),
	}

	vote := NewVoteTop(data)
	assert.Equal(t, 5, vote.Count())

	vote.Rank(5, data)
	assert.Equal(t, true, isEqual(max, vote.Max()))
	assert.Equal(t, true, isEqual(min, vote.Min()))
	assert.Equal(t, true, isSort(result, vote.GetTop(), 5))

	vote.Rank(4, data)
	assert.Equal(t, true, isEqual(result[3], vote.Min()))
	assert.Equal(t, true, isSort(result, vote.GetTop(), 4))

	// Clone
	vote.Rank(5, data)
	clone := vote.Clone()
	assert.Equal(t, true, isSort(result, clone.GetTop(), 5))

	// Clear
	vote.Clear()
	assert.Equal(t, 0, len(vote.GetTop()))

	// Del
	vote.Rank(5, data)
	vote.Del(common.HexToAddress("0x03"))

	result = []*Candidate{
		&Candidate{
			Address: common.HexToAddress("0x05"),
			Total:   new(big.Int).SetInt64(5),
		},
		&Candidate{
			Address: common.HexToAddress("0x04"),
			Total:   new(big.Int).SetInt64(3),
		},
		&Candidate{
			Address: common.HexToAddress("0x02"),
			Total:   new(big.Int).SetInt64(2),
		},
		&Candidate{
			Address: common.HexToAddress("0x01"),
			Total:   new(big.Int).SetInt64(1),
		},
	}
	assert.Equal(t, 4, vote.Count())
	assert.Equal(t, true, isSort(result, vote.GetTop(), 4))
}

func BenchmarkNewVoteT(b *testing.B) {
	b.ReportAllocs()
	result := make([]*Candidate, 500000)
	for index := 0; index < 500000; index++ {
		result[index] = &Candidate{
			Address: common.HexToAddress(strconv.Itoa(index)),
			Total:   new(big.Int).SetInt64(int64(index)),
		}
	}

	vote := NewVoteTop(result)
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		vote.Rank(30, result)
	}
}
