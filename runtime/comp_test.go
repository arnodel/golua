package runtime

import (
	"math"
	"testing"
)

func TestComparisons(t *testing.T) {
	// Constants for edge cases
	const (
		SafeInt   = int64(1<<53 + 1)
		SafeFloat = 1 << 53
	)

	tests := []struct {
		name     string
		n        int64
		f        float64
		wantLtIF bool // ltIntAndFloat (n < f)
		wantLtFI bool // ltFloatAndInt (f < n)
		wantLeIF bool // leIntAndFloat (n <= f)
		wantLeFI bool // leFloatAndInt (f <= n)
	}{
		// Standard Integer Logic
		{"Equal", 10, 10.0, false, false, true, true},
		{"n Less", 10, 11.0, true, false, true, false},
		{"f Less", 11, 10.0, false, true, false, true},

		// Fractions & Signs
		{"Positive Fraction (n < f)", 5, 5.1, true, false, true, false},
		{"Positive Fraction (f < n)", 5, 4.9, false, true, false, true},
		{"Negative Fraction (n > f)", -5, -5.1, false, true, false, true},
		{"Negative Fraction (n < f)", -5, -4.9, true, false, true, false},

		// Precision Trap (2^53)
		// n = 2^53 + 1 (odd), f = 2^53 (even).
		// n > f is TRUE.
		{"Precision Trap", SafeInt, SafeFloat, false, true, false, true},

		// Overflow / Infinite
		{"MaxInt64 vs Inf", math.MaxInt64, math.Inf(1), true, false, true, false},
		{"MaxInt64 vs -Inf", math.MaxInt64, math.Inf(-1), false, true, false, true},
		{"MaxInt64 vs Huge Float", math.MaxInt64, 1e20, true, false, true, false},

		// NaN
		{"NaN Check", 123, math.NaN(), false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ltIntAndFloat(tt.n, tt.f); got != tt.wantLtIF {
				t.Errorf("ltIntAndFloat(%d, %g) = %v, want %v", tt.n, tt.f, got, tt.wantLtIF)
			}
			if got := ltFloatAndInt(tt.f, tt.n); got != tt.wantLtFI {
				t.Errorf("ltFloatAndInt(%g, %d) = %v, want %v", tt.f, tt.n, got, tt.wantLtFI)
			}
			if got := leIntAndFloat(tt.n, tt.f); got != tt.wantLeIF {
				t.Errorf("leIntAndFloat(%d, %g) = %v, want %v", tt.n, tt.f, got, tt.wantLeIF)
			}
			if got := leFloatAndInt(tt.f, tt.n); got != tt.wantLeFI {
				t.Errorf("leFloatAndInt(%g, %d) = %v, want %v", tt.f, tt.n, got, tt.wantLeFI)
			}
		})
	}
}
