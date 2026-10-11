package code

import (
	"math"
	"testing"
)

func TestLoadSmallInt(t *testing.T) {
	// Only values that fit in an int16 are inlined. The large cases must be
	// rejected on every platform; a conversion through int used to wrap them
	// to small values where int is 32 bits (1<<32 was inlined as 0).
	tests := []struct {
		n       int64
		inlined bool
	}{
		{n: 0, inlined: true},
		{n: 1, inlined: true},
		{n: -1, inlined: true},
		{n: math.MaxInt16, inlined: true},
		{n: math.MinInt16, inlined: true},
		{n: math.MaxInt16 + 1, inlined: false},
		{n: math.MinInt16 - 1, inlined: false},
		{n: 1 << 32, inlined: false},
		{n: 1<<32 + 5, inlined: false},
		{n: 1 << 62, inlined: false},
		{n: math.MaxInt64, inlined: false},
		{n: math.MinInt64, inlined: false},
	}
	for _, tt := range tests {
		op, ok := LoadSmallInt(Reg{}, tt.n)
		if ok != tt.inlined {
			t.Errorf("LoadSmallInt(%d) inlined = %t, want %t", tt.n, ok, tt.inlined)
			continue
		}
		if ok && op != LoadInt16(Reg{}, int16(tt.n)) {
			t.Errorf("LoadSmallInt(%d) = %v, want LoadInt16", tt.n, op)
		}
	}
}
