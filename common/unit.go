package common

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"math/big"
	"strings"
)

var (
	OneLEMO          = big.NewInt(1000000000000000000)
	ErrParseLemoFail = errors.New("Lemo2Mo parse input lemo failed, please make sure it is a number")
)

// Lemo2Mo change LEMO to mo. 1 LEMO = 1e18 mo
func Lemo2Mo(lemo string) *big.Int {
	if len(lemo) == 0 {
		return new(big.Int)
	}

	// If we use big.Float to multiply 1000000000000.1 LEMO with 1e18, and convert the result to big.Int, then the lowest bit of the result is random. So we have to split the string to integer part and decimal part
	parts := strings.Split(lemo, ".")
	integer := parts[0]
	intB, ok := new(big.Int).SetString(integer, 10)
	if !ok {
		log.Error("Lemo2Mo parse input lemo failed", "input", lemo)
		panic(ErrParseLemoFail)
	}
	result := new(big.Int).Mul(intB, OneLEMO)

	// lemo is a decimal
	if len(parts) == 2 {
		decimal := parts[1]
		decB, ok := new(big.Int).SetString(setWith(decimal, 18), 10)
		if !ok {
			log.Error("Lemo2Mo parse input lemo failed", "input", lemo)
			panic(ErrParseLemoFail)
		}
		result.Add(result, decB)
	}
	return result
}

func setWith(str string, totalLength int) string {
	if totalLength > len(str) {
		return str + strings.Repeat("0", totalLength-len(str))
	} else {
		return str[:totalLength]
	}
}
