package base26

import (
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

// TestEncode
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
	// expect
	results := []struct {
		data string
	}{
		{string("888888888888888888888888888888888888")},
		{string("888888888888888888888888888888885QAG")},
		{string("888888888888888888888888888888889G8Y")},
		{string("88888888888888888888888888888888PQ8D")},
		{string("888888888888888888888888888888835Z86")},
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
	t.Log(enc)
	t.Log(Decode([]byte(enc)))

}

// TestDecode
func TestDecode(t *testing.T) {
	tests := []struct {
		data []byte
	}{
		{[]byte("0x01fffffffffffffffffff")},
		{[]byte("0x0ffffffffffffffffffff")},
		{[]byte("0x011111111111111111111")},
		{[]byte("0x01000000000")},
		{[]byte{' ', 0x02, 0x03, 'a'}},
	}
	for _, test := range tests {
		encode := Encode(test.data)
		t.Log(encode)
		t.Log(len(encode))
		decode := Decode([]byte(encode))
		assert.Equal(t, test.data, decode)
	}
}
