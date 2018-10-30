/*
base26的字符集为 23456789ABCDFGHJKNPQRSTWYZ
*/
package base26

import (
	"bytes"
	"math/big"
)

var b26AIphabet = []byte("83456729ABCDFGHJKNPQRSTWYZ")

// Encode 将字节数组编码为Base26
func Encode(input []byte) []byte {
	var result []byte
	x := big.NewInt(0).SetBytes(input)
	base := big.NewInt(int64(len(b26AIphabet)))
	zero := big.NewInt(0)
	mod := &big.Int{}
	for x.Cmp(zero) != 0 {
		x.DivMod(x, base, mod)
		result = append(result, b26AIphabet[mod.Int64()])
	}
	ReverseBytes(result) // 反转字节数组

	if len(result) < 36 {
		for len(result) != 36 {
			result = append([]byte{b26AIphabet[0]}, result...)
		}
	}
	return result
}

// Decode 解码Base26所编码的数据
func Decode(input []byte) []byte {
	result := big.NewInt(0)
	zeroBytes := 0
	for _, b := range input {
		if b == b26AIphabet[0] {
			zeroBytes++
		} else {
			break
		}
	}
	payload := input[zeroBytes:]
	for _, b := range payload {
		charIndex := bytes.IndexByte(b26AIphabet, b)
		result.Mul(result, big.NewInt(int64(len(b26AIphabet))))
		result.Add(result, big.NewInt(int64(charIndex)))
	}
	decoded := result.Bytes()

	return decoded
}

// ReverseBytes 反转字节数组
func ReverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}
