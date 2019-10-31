package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

var (
	ErrPKCS5UnPadding = errors.New("PKCS5UnPadding error")
)

func PKCS5Padding(cipherText []byte, blockSize int) []byte {
	padding := blockSize - len(cipherText)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(cipherText, padText...)
}

func PKCS5UnPadding(originData []byte) ([]byte, error) {
	length := len(originData)
	unPadding := int(originData[length-1])
	index := length - unPadding
	if index < 0 {
		return nil, ErrPKCS5UnPadding
	} else {
		return originData[:index], nil
	}
}

func AesEncrypt(originData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	originData = PKCS5Padding(originData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	encResult := make([]byte, len(originData))
	blockMode.CryptBlocks(encResult, originData)
	return encResult, nil
}

func AesDecrypt(encResult, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(encResult))
	blockMode.CryptBlocks(origData, encResult)
	return PKCS5UnPadding(origData)
}
