package base26

import (
	"github.com/testify/assert"
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
		{[]byte{'2', '2', '2'}},
		{[]byte{'5', 'Q', 'A', 'G'}},
		{[]byte{'9', 'G', '2', 'Y'}},
		{[]byte{'P', 'Q', '2', 'D'}},
		{[]byte{'3', '5', 'Z', '2', '6'}},
	}

	for Index, test := range tests {
		encode := Encode(test.data)
		assert.Equal(t, results[Index].data, encode)
	}
}

// TestDecode 解码功能测试
func TestDecode(t *testing.T) {
	tests := []struct {
		data []byte
	}{
		{[]byte{'0', '2', '2'}},
		{[]byte{'5', 'Q', 'A', 'G'}},
		{[]byte{'9', 'G', '2', 'Y'}},
		{[]byte{'P', 'Q', '2', 'D'}},
		{[]byte{0x08, 0x09, 0x10}},
		{[]byte{'3', '5', 'Z', '2', '6'}},
		{[]byte{0x06, 0x03, 0x04}},
	}

	for _, test := range tests {
		encode := Encode(test.data)
		Decode := Decode(encode)
		assert.Equal(t, test.data, Decode)
	}
}
