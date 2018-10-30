package base26

import (
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

// TestEncode 编码功能测试
func TestEncode(t *testing.T) {
	tests := []struct {
		data []byte
	}{
		{[]byte{0x00, 0x00, 0x00}},
		{[]byte{0x01, 0x01, 0x01}},
		{[]byte{0x02, 0x03, 0x04}},
		{[]byte{0x05, 0x06, 0x07}},
		{[]byte{0x08, 0x09, 0x10}},
	}
	// 期望值
	results := []struct {
		data []byte
	}{
		{[]byte{0x32, 0x32, 0x32}},
		{[]byte{'5', 'Q', 'A', 'G'}},
		{[]byte{'9', 'G', '2', 'Y'}},
		{[]byte{'P', 'Q', '2', 'D'}},
		{[]byte{'3', '5', 'Z', '2', '6'}},
	}

	for Index, test := range tests {
		encode := Encode(test.data)
		assert.Equal(t, results[Index].data, encode)
	}

	// input01 := []byte{0, 0, 0}
	// x := big.NewInt(0).SetBytes(input01)
	// t.Log(x.Bytes())
	// t.Log(string(Encode(input01)))
	// t.Log(Decode(Encode(input01)))

	inputs := []byte{0, 0, 0, 0, 2, 185, 74, 64}
	t.Log(big.NewInt(0).SetBytes(inputs))
	t.Log(inputs)
	enc := Encode(inputs)
	t.Log(string(enc))
	t.Log(Decode(enc))

}

// TestDecode 解码功能测试
func TestDecode(t *testing.T) {
	tests := []struct {
		data []byte
	}{
		{[]byte("0x01fffffffffffffffffff")},
		{[]byte("0x0ffffffffffffffffffff")},
		{[]byte("0x011111111111111111111")},
		{[]byte("0x010000000000000000000")},
		{[]byte{' ', 0x02, 0x03, 'a'}},
	}
	for _, test := range tests {
		encode := Encode(test.data)
		t.Log(string(encode))
		t.Log(len(encode))
		decode := Decode(encode)
		assert.Equal(t, test.data, decode)
	}
}
