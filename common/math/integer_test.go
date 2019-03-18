// Copyright 2017 The lemochain-core Authors
// This file is part of the lemochain-core library.
//
// The lemochain-core library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The lemochain-core library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the lemochain-core library. If not, see <http://www.gnu.org/licenses/>.

package math

import (
	"testing"
)

type operation byte

const (
	sub operation = iota
	add
	mul
)

func TestOverflow(t *testing.T) {
	for i, test := range []struct {
		x        uint64
		y        uint64
		overflow bool
		op       operation
	}{
		// add operations
		{MaxUint64, 1, true, add},
		{MaxUint64 - 1, 1, false, add},

		// sub operations
		{0, 1, true, sub},
		{0, 0, false, sub},

		// mul operations
		{0, 0, false, mul},
		{10, 10, false, mul},
		{MaxUint64, 2, true, mul},
		{MaxUint64, 1, false, mul},
	} {
		var overflows bool
		switch test.op {
		case sub:
			_, overflows = SafeSub(test.x, test.y)
		case add:
			_, overflows = SafeAdd(test.x, test.y)
		case mul:
			_, overflows = SafeMul(test.x, test.y)
		}

		if test.overflow != overflows {
			t.Errorf("%d failed. Expected test to be %v, got %v", i, test.overflow, overflows)
		}
	}
}
