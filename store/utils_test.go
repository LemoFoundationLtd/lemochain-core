package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCheckSum(t *testing.T) {
	val := []byte{'x', 'g', 'q'}
	crc := CheckSum(val)
	assert.Equal(t, crc, uint16(52507))
}

func TestByte2Uint32(t *testing.T) {
	val := []byte{'x', 'g', 'q'}
	hash := Byte2Uint32(val)
	assert.Equal(t, hash, uint32(3143121155))
}
