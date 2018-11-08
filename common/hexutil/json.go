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
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"math/big"
	"net"
	"reflect"
)

var (
	bytesT  = reflect.TypeOf(Bytes(nil))
	bigT    = reflect.TypeOf((*Big)(nil))
	big10T  = reflect.TypeOf((*Big10)(nil))
	uint64T = reflect.TypeOf(Uint64(0))
	uint32T = reflect.TypeOf(Uint32(0))
	IPT     = reflect.TypeOf(IP(nil))
)

// Bytes marshals/unmarshals as a JSON string with 0x prefix.
// The empty slice marshals as "0x".
type Bytes []byte

// MarshalText implements encoding.TextMarshaler
func (b Bytes) MarshalText() ([]byte, error) {
	result := make([]byte, len(b)*2+2)
	copy(result, `0x`)
	hex.Encode(result[2:], b)
	return result, nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *Bytes) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return errNonString(bytesT)
	}
	return wrapTypeError(b.UnmarshalText(input[1:len(input)-1]), bytesT)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (b *Bytes) UnmarshalText(input []byte) error {
	raw, err := checkText(input, true)
	if err != nil {
		return err
	}
	dec := make([]byte, len(raw)/2)
	if _, err = hex.Decode(dec, raw); err != nil {
		err = mapError(err)
	} else {
		*b = dec
	}
	return err
}

// String returns the hex encoding of b.
func (b Bytes) String() string {
	return Encode(b)
}

// Big marshals/unmarshals as a JSON string with 0x prefix.
// The zero value marshals as "0x0".
//
// Negative integers are not supported at this time. Attempting to marshal them will
// return an error. Values larger than 256bits are rejected by Unmarshal but will be
// marshaled without error.
type Big big.Int

// MarshalText implements encoding.TextMarshaler
func (b Big) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *Big) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return errNonString(bigT)
	}
	return wrapTypeError(b.UnmarshalText(input[1:len(input)-1]), bigT)
}

// UnmarshalText implements encoding.TextUnmarshaler
func (b *Big) UnmarshalText(input []byte) error {
	dec, err := decodeBig(input, 16)
	if err == nil {
		*b = (Big)(*dec)
	}
	return err
}

// ToInt converts b to a big.Int.
func (b *Big) ToInt() *big.Int {
	return (*big.Int)(b)
}

// String returns the hex encoding of b.
func (b *Big) String() string {
	if b.ToInt().BitLen() == 0 {
		return "0x0"
	}
	return fmt.Sprintf("%#x", b.ToInt())
}

// Big10 marshals/unmarshals as a JSON decimal string.
// The zero value marshals as "0".
//
// Negative integers are not supported at this time. Attempting to marshal them will
// return an error. Values larger than 256bits are rejected by Unmarshal but will be
// marshaled without error.
type Big10 big.Int

// MarshalText implements encoding.TextMarshaler
func (b Big10) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *Big10) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return errNonString(big10T)
	}
	return wrapTypeError(b.UnmarshalText(input[1:len(input)-1]), big10T)
}

// UnmarshalText implements encoding.TextUnmarshaler
func (b *Big10) UnmarshalText(input []byte) error {
	dec, err := decodeBig(input, 10)
	if err == nil {
		*b = (Big10)(*dec)
	}
	return err
}

// ToInt converts b to a big.Int.
func (b *Big10) ToInt() *big.Int {
	return (*big.Int)(b)
}

// String returns the hex encoding of b.
func (b *Big10) String() string {
	return b.ToInt().String()
}

// Uint64 marshals uint64 as decimal, and unmarshals string and number as decimal or hex.
type Uint64 uint64

// MarshalText implements encoding.TextMarshaler.
func (i Uint64) MarshalText() ([]byte, error) {
	buf := fmt.Sprintf("%d", uint64(i))
	return []byte(buf), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (i *Uint64) UnmarshalText(input []byte) error {
	dec, err := ParseUint(string(input), 64)
	if err != nil {
		log.Warnf("invalid hex or decimal integer %q", input)
		return err
	}
	*i = Uint64(dec)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (i *Uint64) UnmarshalJSON(input []byte) error {
	if isString(input) {
		input = input[1 : len(input)-1]
	}
	return wrapTypeError(i.UnmarshalText(input), uint64T)
}

// Uint32 marshals uint32 as decimal, and unmarshals string and number as decimal or hex.
type Uint32 uint32

// MarshalText implements encoding.TextMarshaler.
func (i Uint32) MarshalText() ([]byte, error) {
	buf := fmt.Sprintf("%d", uint32(i))
	return []byte(buf), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (i *Uint32) UnmarshalText(input []byte) error {
	dec, err := ParseUint(string(input), 32)
	if err != nil {
		log.Warnf("invalid hex or decimal integer %q", input)
		return err
	}
	*i = Uint32(dec)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (i *Uint32) UnmarshalJSON(input []byte) error {
	if isString(input) {
		input = input[1 : len(input)-1]
	}
	return wrapTypeError(i.UnmarshalText(input), uint32T)
}

func isString(input []byte) bool {
	return len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"'
}

func bytesHave0xPrefix(input []byte) bool {
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}

func checkText(input []byte, want0xPrefix bool) ([]byte, error) {
	if len(input) == 0 {
		return nil, nil // empty strings are allowed
	}
	if bytesHave0xPrefix(input) {
		input = input[2:]
	} else if want0xPrefix {
		return nil, ErrMissingPrefix
	}
	if len(input)%2 != 0 {
		return nil, ErrOddLength
	}
	return input, nil
}

func checkNumberText(input []byte, want0xPrefix bool) (raw []byte, err error) {
	if len(input) == 0 {
		return nil, nil // empty strings are allowed
	}
	if bytesHave0xPrefix(input) {
		input = input[2:]
	} else if want0xPrefix {
		return nil, ErrMissingPrefix
	}
	if len(input) == 0 {
		return nil, ErrEmptyNumber
	}
	return input, nil
}

// decodeBig decodes a decimal string as a quantity.
// Numbers larger than 256 bits are not accepted.
func decodeBig(input []byte, base int) (*big.Int, error) {
	raw, err := checkNumberText(input, base == 16)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return big.NewInt(0), nil
	}
	if base == 16 && len(raw) > 64 {
		return nil, Err256Range
	}
	dec := new(big.Int)
	_, ok := dec.SetString(string(raw), base)
	if !ok {
		return nil, ErrSyntax
	}
	return dec, nil
}

func wrapTypeError(err error, typ reflect.Type) error {
	if _, ok := err.(*decError); ok {
		return &json.UnmarshalTypeError{Value: err.Error(), Type: typ}
	}
	return err
}

func errNonString(typ reflect.Type) error {
	return &json.UnmarshalTypeError{Value: "non-string", Type: typ}
}

type IP net.IP

func (ip *IP) String() string {
	t := net.IP(*ip)
	return t.String()
}

// MarshalText implements encoding.TextMarshaler.
func (ip *IP) MarshalText() ([]byte, error) {
	return []byte(ip.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (ip *IP) UnmarshalText(input []byte) error {
	dec := net.ParseIP(string(input))
	if dec == nil {
		log.Warnf("invalid hex or decimal integer %q", input)
		return ErrSyntax
	}
	*ip = IP(dec)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (ip *IP) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return errNonString(IPT)
	}
	return wrapTypeError(ip.UnmarshalText(input[1:len(input)-1]), IPT)
}
