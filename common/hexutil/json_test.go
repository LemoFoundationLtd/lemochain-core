// Copyright 2016 The lemochain-go Authors
// This file is part of the lemochain-go library.
//
// The lemochain-go library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The lemochain-go library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the lemochain-go library. If not, see <http://www.gnu.org/licenses/>.

package hexutil

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"math/big"
	"net"
	"testing"
)

func referenceBig(s string) *big.Int {
	b, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("invalid")
	}
	return b
}

func referenceBytes(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func referenceIP(s string) net.IP {
	b := net.ParseIP(s)
	if b == nil {
		panic("invalid")
	}
	return b
}

var errJSONEOF = errors.New("unexpected end of JSON input")
var errHexSyntax = errors.New("invalid character 'x' after top-level value")
var errIPSyntax = errors.New("invalid character '.' after top-level value")

func TestUnmarshalBytes(t *testing.T) {
	var tests = []unmarshalTest{
		// invalid encoding
		{input: "", wantErr: errJSONEOF},
		{input: "null", wantErr: errNonString(bytesT)},
		{input: "10", wantErr: errNonString(bytesT)},
		{input: `"0"`, wantErr: wrapTypeError(ErrMissingPrefix, bytesT)},
		{input: `"0x0"`, wantErr: wrapTypeError(ErrOddLength, bytesT)},
		{input: `"0xxx"`, wantErr: wrapTypeError(ErrSyntax, bytesT)},
		{input: `"0x01zz01"`, wantErr: wrapTypeError(ErrSyntax, bytesT)},

		// valid encoding
		{input: `""`, want: referenceBytes("")},
		{input: `"0x"`, want: referenceBytes("")},
		{input: `"0x02"`, want: referenceBytes("02")},
		{input: `"0X02"`, want: referenceBytes("02")},
		{input: `"0xffffffffff"`, want: referenceBytes("ffffffffff")},
		{
			input: `"0xffffffffffffffffffffffffffffffffffff"`,
			want:  referenceBytes("ffffffffffffffffffffffffffffffffffff"),
		},
	}
	for _, test := range tests {
		var v Bytes
		err := json.Unmarshal([]byte(test.input), &v)
		if test.wantErr != nil {
			assert.EqualError(t, err, test.wantErr.Error(), "input %s", test.input)
		} else {
			assert.NoError(t, err, "input %s", test.input)
			assert.Equal(t, test.want, []byte(v), "input %s", test.input)
		}
	}
}

func BenchmarkUnmarshalBytes(b *testing.B) {
	input := []byte(`"0x123456789abcdef123456789abcdef"`)
	for i := 0; i < b.N; i++ {
		var v Bytes
		if err := v.UnmarshalJSON(input); err != nil {
			b.Fatal(err)
		}
	}
}

func TestMarshalBytes(t *testing.T) {
	tests := []marshalTest{
		{[]byte{}, "0x"},
		{[]byte{0}, "0x00"},
		{[]byte{0, 0, 1, 2}, "0x00000102"},
	}
	for _, test := range tests {
		in := test.input.([]byte)
		assert.Equal(t, test.want, Bytes(in).String(), "input %s", test.input)

		out, err := json.Marshal(Bytes(in))
		assert.NoError(t, err, "input %s", test.input)
		assert.Equal(t, `"`+test.want+`"`, string(out), "input %s", test.input)
	}
}

func TestUnmarshalBig(t *testing.T) {
	var tests = []unmarshalTest{
		// invalid encoding
		{input: "", wantErr: errJSONEOF},
		{input: "null", wantErr: errNonString(bigT)},
		{input: "10", wantErr: errNonString(bigT)},
		{input: `"0"`, wantErr: wrapTypeError(ErrMissingPrefix, bigT)},
		{input: `"0x"`, wantErr: wrapTypeError(ErrEmptyNumber, bigT)},
		{input: `"0xx"`, wantErr: wrapTypeError(ErrSyntax, bigT)},
		{input: `"0x1zz01"`, wantErr: wrapTypeError(ErrSyntax, bigT)},
		{
			input:   `"0x10000000000000000000000000000000000000000000000000000000000000000"`,
			wantErr: wrapTypeError(Err256Range, bigT),
		},

		// valid encoding
		{input: `""`, want: big.NewInt(0)},
		{input: `"0x0"`, want: big.NewInt(0)},
		{input: `"0x01"`, want: big.NewInt(0x1)},
		{input: `"0x2"`, want: big.NewInt(0x2)},
		{input: `"0x2F2"`, want: big.NewInt(0x2f2)},
		{input: `"0X2F2"`, want: big.NewInt(0x2f2)},
		{input: `"0x1122aaff"`, want: big.NewInt(0x1122aaff)},
		{input: `"0xbBb"`, want: big.NewInt(0xbbb)},
		{input: `"0xfffffffff"`, want: big.NewInt(0xfffffffff)},
		{
			input: `"0x112233445566778899aabbccddeeff"`,
			want:  referenceBig("112233445566778899aabbccddeeff"),
		},
		{
			input: `"0xffffffffffffffffffffffffffffffffffff"`,
			want:  referenceBig("ffffffffffffffffffffffffffffffffffff"),
		},
		{
			input: `"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"`,
			want:  referenceBig("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
		},
	}
	for _, test := range tests {
		var v Big
		err := json.Unmarshal([]byte(test.input), &v)
		if test.wantErr != nil {
			assert.EqualError(t, err, test.wantErr.Error(), "input %s", test.input)
		} else {
			assert.NoError(t, err, "input %s", test.input)
			assert.Equal(t, test.want.(*big.Int), (*big.Int)(&v), "input %s", test.input)
		}
	}
}

func BenchmarkUnmarshalBig(b *testing.B) {
	input := []byte(`"0x123456789abcdef123456789abcdef"`)
	for i := 0; i < b.N; i++ {
		var v Big
		if err := v.UnmarshalJSON(input); err != nil {
			b.Fatal(err)
		}
	}
}

func TestMarshalBig(t *testing.T) {
	tests := []marshalTest{
		{referenceBig("0"), "0x0"},
		{referenceBig("1"), "0x1"},
		{referenceBig("ff"), "0xff"},
		{referenceBig("112233445566778899aabbccddeeff"), "0x112233445566778899aabbccddeeff"},
		{referenceBig("80a7f2c1bcc396c00"), "0x80a7f2c1bcc396c00"},
		{referenceBig("-80a7f2c1bcc396c00"), "-0x80a7f2c1bcc396c00"},
	}
	for _, test := range tests {
		in := test.input.(*big.Int)
		assert.Equal(t, test.want, (*Big)(in).String(), "input %s", test.input)

		out, err := json.Marshal((*Big)(in))
		assert.NoError(t, err, "input %s", test.input)
		assert.Equal(t, `"`+test.want+`"`, string(out), "input %s", test.input)
	}
}

func TestUnmarshalUint64(t *testing.T) {
	var tests = []unmarshalTest{
		// invalid encoding
		{input: "", wantErr: errJSONEOF},
		{input: "null", wantErr: wrapTypeError(ErrSyntax, uint64T)},
		{input: `0x01`, wantErr: errHexSyntax},
		{input: `"0x"`, wantErr: wrapTypeError(ErrEmptyNumber, uint64T)},
		{input: `"ffffff"`, wantErr: wrapTypeError(ErrSyntax, uint64T)},
		{input: `"0xfffffffffffffffff"`, wantErr: wrapTypeError(Err256Range, uint64T)},
		{input: `"0xx"`, wantErr: wrapTypeError(ErrSyntax, uint64T)},
		{input: `"0x1zz01"`, wantErr: wrapTypeError(ErrSyntax, uint64T)},

		// valid encoding
		{input: `""`, want: uint64(0)},
		{input: `"0"`, want: uint64(0)},
		{input: `"0x0"`, want: uint64(0)},
		{input: "0", want: uint64(0)},
		{input: "10", want: uint64(10)},
		{input: `"0x2"`, want: uint64(0x2)},
		{input: `"0x2F2"`, want: uint64(0x2f2)},
		{input: `"0X2F2"`, want: uint64(0x2f2)},
		{input: `"0x1122aaff"`, want: uint64(0x1122aaff)},
		{input: `"0xbbb"`, want: uint64(0xbbb)},
		{input: `"0xffffffffffffffff"`, want: uint64(0xffffffffffffffff)},
	}
	for _, test := range tests {
		var v Uint64
		err := json.Unmarshal([]byte(test.input), &v)
		if test.wantErr != nil {
			assert.EqualError(t, err, test.wantErr.Error(), "input %s", test.input)
		} else {
			assert.NoError(t, err, "input %s", test.input)
			assert.Equal(t, test.want.(uint64), uint64(v), "input %s", test.input)
		}
	}
}

func BenchmarkUnmarshalUint64(b *testing.B) {
	input := []byte(`"0x123456789abcdf"`)
	for i := 0; i < b.N; i++ {
		var v Uint64
		v.UnmarshalJSON(input)
	}
}

func TestMarshalUint64(t *testing.T) {
	tests := []marshalTest{
		{uint64(0), "0"},
		{uint64(1), "1"},
		{uint64(0xff), "255"},
		{uint64(0x1122334455667788), "1234605616436508552"},
	}
	for _, test := range tests {
		in := test.input.(uint64)
		out, err := json.Marshal((Uint64)(in))
		assert.NoError(t, err, "input %s", test.input)
		assert.Equal(t, `"`+test.want+`"`, string(out), "input %s", test.input)
	}
}

func TestUnmarshalUint32(t *testing.T) {
	var tests = []unmarshalTest{
		// invalid encoding
		{input: "", wantErr: errJSONEOF},
		{input: "null", wantErr: wrapTypeError(ErrSyntax, uint32T)},
		{input: `0x01`, wantErr: errHexSyntax},
		{input: `"0x"`, wantErr: wrapTypeError(ErrEmptyNumber, uint32T)},
		{input: `"ffffff"`, wantErr: wrapTypeError(ErrSyntax, uint32T)},
		{input: `"0xffffffffff"`, wantErr: wrapTypeError(Err256Range, uint32T)},
		{input: `"0xx"`, wantErr: wrapTypeError(ErrSyntax, uint32T)},
		{input: `"0x1zz01"`, wantErr: wrapTypeError(ErrSyntax, uint32T)},

		// valid encoding
		{input: `""`, want: uint32(0)},
		{input: `"0"`, want: uint32(0)},
		{input: `"0x0"`, want: uint32(0)},
		{input: "0", want: uint32(0)},
		{input: "10", want: uint32(10)},
		{input: `"0x2"`, want: uint32(0x2)},
		{input: `"0x2F2"`, want: uint32(0x2f2)},
		{input: `"0X2F2"`, want: uint32(0x2f2)},
		{input: `"0x1122aaff"`, want: uint32(0x1122aaff)},
		{input: `"0xbbb"`, want: uint32(0xbbb)},
		{input: `"0xffffffff"`, want: uint32(0xffffffff)},
	}
	for _, test := range tests {
		var v Uint32
		err := json.Unmarshal([]byte(test.input), &v)
		if test.wantErr != nil {
			assert.EqualError(t, err, test.wantErr.Error(), "input %s", test.input)
		} else {
			assert.NoError(t, err, "input %s", test.input)
			assert.Equal(t, test.want.(uint32), uint32(v), "input %s", test.input)
		}
	}
}

func BenchmarkUnmarshalUint32(b *testing.B) {
	input := []byte(`"0x12345678"`)
	for i := 0; i < b.N; i++ {
		var v Uint32
		v.UnmarshalJSON(input)
	}
}

func TestMarshalUint32(t *testing.T) {
	tests := []marshalTest{
		{uint32(0), "0"},
		{uint32(1), "1"},
		{uint32(0xff), "255"},
		{uint32(0x11223344), "287454020"},
	}
	for _, test := range tests {
		in := test.input.(uint32)
		out, err := json.Marshal((Uint32)(in))
		assert.NoError(t, err, "input %s", test.input)
		assert.Equal(t, `"`+test.want+`"`, string(out), "input %s", test.input)
	}
}

func TestUnmarshalIP(t *testing.T) {
	var tests = []unmarshalTest{
		// invalid encoding
		{input: "", wantErr: errJSONEOF},
		{input: `1.2.3.4`, wantErr: errIPSyntax},
		{input: `"1"`, wantErr: wrapTypeError(ErrSyntax, IPT)},
		{input: `"1.2.3.4."`, wantErr: wrapTypeError(ErrSyntax, IPT)},
		{input: `"256.1.1.1"`, wantErr: wrapTypeError(ErrSyntax, IPT)},
		{input: `"...1"`, wantErr: wrapTypeError(ErrSyntax, IPT)},

		// valid encoding
		{input: `"0.0.0.0"`, want: referenceIP("0.0.0.0")},
		{input: `"1.1.1.1"`, want: referenceIP("1.1.1.1")},
		{input: `"255.255.255.255"`, want: referenceIP("255.255.255.255")},
	}
	for _, test := range tests {
		var v IP
		err := json.Unmarshal([]byte(test.input), &v)
		if test.wantErr != nil {
			assert.EqualError(t, err, test.wantErr.Error(), "input %s", test.input)
		} else {
			assert.NoError(t, err, "input %s", test.input)
			assert.Equal(t, test.want.(net.IP), (net.IP)(v), "input %s", test.input)
		}
	}
}

func BenchmarkUnmarshalIP(b *testing.B) {
	input := []byte(`"0x12345678"`)
	for i := 0; i < b.N; i++ {
		var v IP
		v.UnmarshalJSON(input)
	}
}

func TestMarshalIP(t *testing.T) {
	tests := []marshalTest{
		{referenceIP("0.0.0.0"), "0.0.0.0"},
		{referenceIP("1.1.1.1"), "1.1.1.1"},
		{referenceIP("255.255.255.255"), "255.255.255.255"},
	}
	for _, test := range tests {
		in := test.input.(net.IP)
		assert.Equal(t, test.want, (*IP)(&in).String(), "input %s", test.input)

		out, err := json.Marshal((*IP)(&in))
		assert.NoError(t, err, "input %s", test.input)
		assert.Equal(t, `"`+test.want+`"`, string(out), "input %s", test.input)
	}
}
