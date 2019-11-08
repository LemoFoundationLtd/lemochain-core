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

func PKCS5UnPadding(originData []byte) []byte {
	length := len(originData)
	if length == 0 {
		return nil
	}

	padding := originData[length-1]
	index := length - int(padding)
	if index < 0 || padding > aes.BlockSize || padding == 0 {
		return nil
	}
	for i := length - 1; i >= index; i-- {
		if originData[i] != padding {
			return nil
		}
	}
	return originData[:index]
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
	plaintext := PKCS5UnPadding(origData)
	if plaintext == nil {
		return nil, ErrPKCS5UnPadding
	}
	return plaintext, nil
}
