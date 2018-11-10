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
	"bytes"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

type marshalTest struct {
	input interface{}
	want  string
}

type unmarshalTest struct {
	input        string
	want         interface{}
	wantErr      error // if set, decoding must fail on any platform
	wantErr32bit error // if set, decoding must fail on 32bit platforms (used for Uint tests)
}

func TestDecode(t *testing.T) {
	tests := []unmarshalTest{
		// invalid
		{input: ``, wantErr: ErrEmptyString},
		{input: `0`, wantErr: ErrMissingPrefix},
		{input: `0x0`, wantErr: ErrOddLength},
		{input: `0x023`, wantErr: ErrOddLength},
		{input: `0xxx`, wantErr: ErrSyntax},
		{input: `0x01zz01`, wantErr: ErrSyntax},
		// valid
		{input: `0x`, want: []byte{}},
		{input: `0X`, want: []byte{}},
		{input: `0x02`, want: []byte{0x02}},
		{input: `0X02`, want: []byte{0x02}},
		{input: `0xffffffffff`, want: []byte{0xff, 0xff, 0xff, 0xff, 0xff}},
		{
			input: `0xffffffffffffffffffffffffffffffffffff`,
			want:  []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
	}
	for _, test := range tests {
		dec, err := Decode(test.input)
		if test.wantErr != nil {
			assert.EqualError(t, err, test.wantErr.Error(), "input %s", test.input)
		} else {
			assert.NoError(t, err, "input %s", test.input)
			assert.Equal(t, test.want.([]byte), dec, "input %s", test.input)
		}
	}
}

func TestEncode(t *testing.T) {
	tests := []marshalTest{
		{[]byte{}, "0x"},
		{[]byte{0}, "0x00"},
		{[]byte{0, 0, 1, 2}, "0x00000102"},
	}
	for _, test := range tests {
		enc := Encode(test.input.([]byte))
		if enc != test.want {
			t.Errorf("input %x: wrong encoding %s", test.input, enc)
		}
	}
}

func TestUnmarshalFixedText(t *testing.T) {
	tests := []struct {
		input   string
		want    []byte
		wantErr error
	}{
		{input: "0x2", wantErr: ErrOddLength},
		{input: "2", wantErr: ErrOddLength},
		{input: "4444", wantErr: errors.New("hex string has length 4, want 8 for x")},
		{input: "4444", wantErr: errors.New("hex string has length 4, want 8 for x")},
		// check that output is not modified for partially correct input
		{input: "444444gg", wantErr: ErrSyntax, want: []byte{0, 0, 0, 0}},
		{input: "0x444444gg", wantErr: ErrSyntax, want: []byte{0, 0, 0, 0}},
		// valid inputs
		{input: "44444444", want: []byte{0x44, 0x44, 0x44, 0x44}},
		{input: "0x44444444", want: []byte{0x44, 0x44, 0x44, 0x44}},
	}

	for _, test := range tests {
		out := make([]byte, 4)
		err := UnmarshalFixedText("x", []byte(test.input), out, false)
		switch {
		case err == nil && test.wantErr != nil:
			t.Errorf("%q: got no error, expected %q", test.input, test.wantErr)
		case err != nil && test.wantErr == nil:
			t.Errorf("%q: unexpected error %q", test.input, err)
		case err != nil && err.Error() != test.wantErr.Error():
			t.Errorf("%q: error mismatch: got %q, want %q", test.input, err, test.wantErr)
		}
		if test.want != nil && !bytes.Equal(out, test.want) {
			t.Errorf("%q: output mismatch: got %x, want %x", test.input, out, test.want)
		}
	}
}

func TestParseUint(t *testing.T) {
	tests := []unmarshalTest{
		// Invalid syntax:
		{input: "abcdef", wantErr: ErrSyntax},
		{input: "0xgg", wantErr: ErrSyntax},
		{input: "18446744073709551617", wantErr: Err256Range},
		// valid
		{input: "", want: uint64(0)},
		{input: "0", want: uint64(0)},
		{input: "0x0", want: uint64(0)},
		{input: "12345678", want: uint64(12345678)},
		{input: "0x12345678", want: uint64(0x12345678)},
		{input: "0X12345678", want: uint64(0x12345678)},
		// Tests for leading zero behaviour:
		{input: "0123456789", want: uint64(123456789)}, // note: not octal
		{input: "0x00", want: uint64(0)},
		{input: "0x012345678abc", want: uint64(0x12345678abc)},
	}
	for _, test := range tests {
		v, err := ParseUint(test.input, 64)
		if test.wantErr != nil {
			assert.EqualError(t, err, test.wantErr.Error(), "input %s", test.input)
		} else {
			assert.NoError(t, err, "input %s", test.input)
			assert.Equal(t, test.want, v, "input %s", test.input)
		}
	}

	_, err := ParseUint("0x123456789", 32)
	assert.EqualError(t, err, Err256Range.Error())
}

func TestMustParseUint64(t *testing.T) {
	if v := MustParseUint64("12345"); v != 12345 {
		t.Errorf(`MustParseUint64("12345") = %d, want 12345`, v)
	}
}

func TestMustParseUint64Panic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("MustParseBig should've panicked")
		}
	}()
	MustParseUint64("ggg")
}
