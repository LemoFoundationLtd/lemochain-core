package common

import (
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func s2b(str string) *big.Int {
	result, _ := new(big.Int).SetString(str, 10)
	return result
}

func TestLemo2Mo(t *testing.T) {
	assert.Equal(t, s2b("0"), Lemo2Mo(""))
	assert.Equal(t, s2b("0"), Lemo2Mo("0"))
	assert.Equal(t, s2b("0"), Lemo2Mo("0000.000"))
	assert.Equal(t, s2b("1000000000000000000"), Lemo2Mo("1"))
	assert.Equal(t, s2b("-1000000000000000000"), Lemo2Mo("-1"))
	assert.Equal(t, s2b("100000000000000000000"), Lemo2Mo("100"))
	assert.Equal(t, s2b("1000000000000000000000000000000"), Lemo2Mo("1000000000000"))
	assert.Equal(t, s2b("1000000000000100000000000000000"), Lemo2Mo("1000000000000.1"))
	assert.Equal(t, s2b("1000000000000000000000000000001"), Lemo2Mo("1000000000000.000000000000000001"))
	assert.Equal(t, s2b("10000000000000000"), Lemo2Mo("0.01"))
	assert.Equal(t, s2b("1"), Lemo2Mo("0.000000000000000001"))
	assert.Equal(t, s2b("0"), Lemo2Mo("0.0000000000000000001"))
	assert.Equal(t, s2b("100000000000000001"), Lemo2Mo("0.100000000000000001"))
	assert.PanicsWithValue(t, ErrParseLemoFail, func() {
		Lemo2Mo("abc")
	})
	assert.PanicsWithValue(t, ErrParseLemoFail, func() {
		Lemo2Mo("0xabc")
	})
	assert.PanicsWithValue(t, ErrParseLemoFail, func() {
		Lemo2Mo("1-1")
	})
	assert.PanicsWithValue(t, ErrParseLemoFail, func() {
		Lemo2Mo("1LEMO")
	})
}
