// Copyright 2015 The lemochain-go Authors
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

package common

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"math/big"
	"strings"
	"testing"
)

func TestBytesConversion(t *testing.T) {
	bytes := []byte{5}
	hash := BytesToHash(bytes)

	var exp Hash
	exp[31] = 5

	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestIsHexAddress(t *testing.T) {
	tests := []struct {
		str string
		exp bool
	}{
		{"0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed", true},
		{"5aaeb6053f3e94c9b9a09f33669435e7ef1beaed", true},
		{"0X5aaeb6053f3e94c9b9a09f33669435e7ef1beaed", true},
		{"0XAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", true},
		{"0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", true},
		{"0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed1", false},
		{"0x5aaeb6053f3e94c9b9a09f33669435e7ef1beae", false},
		{"5aaeb6053f3e94c9b9a09f33669435e7ef1beaed11", false},
		{"0xxaaeb6053f3e94c9b9a09f33669435e7ef1beaed", false},
	}

	for _, test := range tests {
		if result := IsHexAddress(test.str); result != test.exp {
			t.Errorf("IsHexAddress(%s) == %v; expected %v",
				test.str, result, test.exp)
		}
	}
}

func TestHashJsonValidation(t *testing.T) {
	var tests = []struct {
		Prefix string
		Size   int
		Error  string
	}{
		{"", 62, "json: cannot unmarshal hex string without 0x prefix into Go value of type common.Hash"},
		{"0x", 66, "hex string has length 66, want 64 for common.Hash"},
		{"0x", 63, "json: cannot unmarshal hex string of odd length into Go value of type common.Hash"},
		{"0x", 0, "hex string has length 0, want 64 for common.Hash"},
		{"0x", 64, ""},
		{"0X", 64, ""},
	}
	for _, test := range tests {
		input := `"` + test.Prefix + strings.Repeat("0", test.Size) + `"`
		var v Hash
		err := json.Unmarshal([]byte(input), &v)
		if err == nil {
			if test.Error != "" {
				t.Errorf("%s: error mismatch: have nil, want %q", input, test.Error)
			}
		} else {
			if err.Error() != test.Error {
				t.Errorf("%s: error mismatch: have %q, want %q", input, err, test.Error)
			}
		}
	}
}

func TestAddressUnmarshalJSON(t *testing.T) {
	var tests = []struct {
		Input     string
		ShouldErr bool
		Output    *big.Int
	}{
		{"", true, nil},
		{`""`, true, nil},
		{`"0x"`, true, nil},
		{`"0x00"`, true, nil},
		{`"0xG000000000000000000000000000000000000000"`, true, nil},
		{`"0x0000000000000000000000000000000000000000"`, false, big.NewInt(0)},
		{`"0x0000000000000000000000000000000000000010"`, false, big.NewInt(16)},
	}
	for i, test := range tests {
		var v Address
		err := json.Unmarshal([]byte(test.Input), &v)
		if err != nil && !test.ShouldErr {
			t.Errorf("test #%d: unexpected error: %v", i, err)
		}
		if err == nil {
			if test.ShouldErr {
				t.Errorf("test #%d: expected error, got none", i)
			}
			if v.Big().Cmp(test.Output) != 0 {
				t.Errorf("test #%d: address mismatch: have %v, want %v", i, v.Big(), test.Output)
			}
		}
	}
}

func TestAddressEncode(t *testing.T) {
	var tests = []struct {
		Input  string
		Output string
	}{
		// Test cases from https://github.com/lemochain/EIPs/blob/master/EIPS/eip-55.md#specification
		{"0x01c96d852165a10915ffa9c2281ef430784840f0", "Lemo848S799HQ3KPTNSYGDF5TARS3Z8Z2ZCAH5S3"},
		{"0x01818e82ba1e28b9a104e4ae972f0a42d20941ea", "Lemo83PGDHYY2Y6TDKZN6DG9N8ZJ8FHRN8KBK5W6"},
		{"0x01ba688be96e0680cfa3310a88c87a8c2eb0547d", "Lemo83Z44ZFDJK5GPC4DBRABJSNBST7G23K44KD3"},
		{"0x019fce0c15a9ad419d3e36cc56c631906180145e", "Lemo83T82GCAT5QNKSR7824KD8F7ZTW6RRBDTG97"},
		// Ensure that non-standard length input values are handled correctly
		{"0xa", "Lemo8888888888888888888888888888888885RT"},
		{"0x0a", "Lemo8888888888888888888888888888888885RT"},
		{"0x00a", "Lemo8888888888888888888888888888888885RT"},
		{"0x000000000000000000000000000000000000000a", "Lemo8888888888888888888888888888888885RT"},
	}
	for i, test := range tests {
		output := HexToAddress(test.Input).String()
		assert.Equal(t, test.Output, output, "index=%d", i)
	}
	//
	// address := HexToAddress("0xffffffff")
	// sum := GetCheckSum(address.Bytes())
	// fullBytes := append(address.Bytes(), sum)
	// t.Log(fullBytes)
	// encode := base26.Encode(fullBytes)
	// BB := strings.Join([]string{logo, encode}, "")
	// t.Log(BB)
	// t.Log(len(BB))
	// t.Log(base26.Decode([]byte(encode)))

}

func BenchmarkAddressHex(b *testing.B) {
	testAddr := HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	for n := 0; n < b.N; n++ {
		testAddr.Hex()
	}
}

// RestoreOriginalAddress function test
func TestRestoreOriginalAddress(t *testing.T) {
	tests := []struct {
		LemoAddress string
		Native      string
	}{
		{"Lemo837QGPS3YNTYNF53CD88WA5DR3ABNA95W2DG", "0x0112fDDcF0C08132A5dcd9ED77e1a3348ff378D2"},
		{"Lemo842BJZ4DKCC764C63Y6A943775JH6NQ3Z33Y", "0x01f98855Be9ecc5c23A28Ce345D2Cc04686f2c61"},
		{"Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY", "0x016ad4Fc7e1608685Bf5fe5573973BF2B1Ef9B8A"},
		{"Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG", "0x015780F8456F9c1532645087a19DcF9a7e0c7F97"},
	}
	for _, test := range tests {
		nativeAddress, err := RestoreOriginalAddress(test.LemoAddress)
		assert.Nil(t, err)
		NativeAddress := nativeAddress.Hex()
		assert.Equal(t, strings.ToLower(test.Native), NativeAddress)
	}
}
