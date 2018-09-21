/*
base26的字符集为 23456789ABCDFGHJKNPQRSTWYZ
*/
package base26

import (
	"bytes"
	"math/big"
)

var b26AIphabet = []byte("23456789ABCDFGHJKNPQRSTWYZ")

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
	for _, b := range input {
		if b == 0x00 {
			result = append([]byte{b26AIphabet[0]}, result...)
		} else {
			break
		}
	}
	return result
}

// Decode 解码Base26所编码的数据
func Decode(input []byte) []byte {
	result := big.NewInt(0)
	zeroBytes := 0
	for _, b := range input {
		if b == 0x00 {
			zeroBytes++
		}
	}
	payload := input[zeroBytes:]
	for _, b := range payload {
		charIndex := bytes.IndexByte(b26AIphabet, b)
		result.Mul(result, big.NewInt(26))
		result.Add(result, big.NewInt(int64(charIndex)))
	}
	decoded := result.Bytes()
	decoded = append(bytes.Repeat([]byte{byte(0x00)}, zeroBytes), decoded...)

	return decoded
}

// ReverseBytes 反转字节数组
func ReverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}
