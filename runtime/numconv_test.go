package runtime

import (
	"math"
	"reflect"
	"testing"
)

func TestToNumber(t *testing.T) {
	tests := []struct {
		name string
		v    Value
		n    int64
		x    float64
		tp   NumberType
	}{
		{
			name: "int",
			v:    IntValue(23),
			n:    23,
			tp:   IsInt,
		},
		{
			name: "float",
			v:    FloatValue(1.1),
			x:    1.1,
			tp:   IsFloat,
		},
		{
			name: "int string",
			v:    StringValue("-12"),
			n:    -12,
			tp:   IsInt,
		},
		{
			name: "float string",
			v:    StringValue("1.45"),
			x:    1.45,
			tp:   IsFloat,
		},
		{
			name: "non numeric string",
			v:    StringValue("hello"),
			tp:   NaN,
		},
		{
			name: "string with numeric prefix",
			v:    StringValue("49ers"),
			tp:   NaN,
		},
		{
			name: "boolean",
			v:    BoolValue(false),
			tp:   NaN,
		},
		{
			name: "nil",
			v:    NilValue,
			tp:   NaN,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := ToNumber(tt.v)
			if got != tt.n {
				t.Errorf("ToNumber() got = %v, want %v", got, tt.n)
			}
			if got1 != tt.x {
				t.Errorf("ToNumber() got1 = %v, want %v", got1, tt.x)
			}
			if got2 != tt.tp {
				t.Errorf("ToNumber() got2 = %v, want %v", got2, tt.tp)
			}
		})
	}
}

func TestToNumberValue(t *testing.T) {

	tests := []struct {
		name string
		v    Value
		want Value
		tp   NumberType
	}{
		{
			name: "int",
			v:    IntValue(23),
			want: IntValue(23),
			tp:   IsInt,
		},
		{
			name: "float",
			v:    FloatValue(1.1),
			want: FloatValue(1.1),
			tp:   IsFloat,
		},
		{
			name: "int string",
			v:    StringValue("-12"),
			want: IntValue(-12),
			tp:   IsInt,
		},
		{
			name: "float string",
			v:    StringValue("1.45"),
			want: FloatValue(1.45),
			tp:   IsFloat,
		},
		{
			name: "non numeric string",
			v:    StringValue("hello"),
			want: StringValue("hello"),
			tp:   NaN,
		},
		{
			name: "string with numeric prefix",
			v:    StringValue("49ers"),
			want: StringValue("49ers"),
			tp:   NaN,
		},
		{
			name: "boolean",
			v:    BoolValue(false),
			want: BoolValue(false),
			tp:   NaN,
		},
		{
			name: "nil",
			v:    NilValue,
			want: NilValue,
			tp:   NaN,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ToNumberValue(tt.v)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToNumberValue() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.tp {
				t.Errorf("ToNumberValue() got1 = %v, want %v", got1, tt.tp)
			}
		})
	}
}

func TestToInt(t *testing.T) {

	tests := []struct {
		name string
		v    Value
		want int64
		ok   bool
	}{
		{
			name: "int",
			v:    IntValue(100),
			want: 100,
			ok:   true,
		},
		{
			name: "integral float",
			v:    FloatValue(53),
			want: 53,
			ok:   true,
		},
		{
			name: "int string",
			v:    StringValue("1e6"),
			want: 1e6,
			ok:   true,
		},
		{
			name: "decimal float",
			v:    FloatValue(2.3),
		},
		{
			name: "decimal string",
			v:    StringValue("55.5"),
		},
		{
			name: "nil",
		},
		{
			name: "boolean",
			v:    BoolValue(true),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ToInt(tt.v)
			if got != tt.want {
				t.Errorf("ToInt() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.ok {
				t.Errorf("ToInt() got1 = %v, want %v", got1, tt.ok)
			}
		})
	}
}

func TestToIntNoString(t *testing.T) {
	tests := []struct {
		name string
		v    Value
		want int64
		ok   bool
	}{
		{
			name: "int",
			v:    IntValue(100),
			want: 100,
			ok:   true,
		},
		{
			name: "integral float",
			v:    FloatValue(53),
			want: 53,
			ok:   true,
		},
		{
			name: "int string",
			v:    StringValue("1e6"),
		},
		{
			name: "decimal float",
			v:    FloatValue(2.3),
		},
		{
			name: "decimal string",
			v:    StringValue("55.5"),
		},
		{
			name: "nil",
		},
		{
			name: "boolean",
			v:    BoolValue(true),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ToIntNoString(tt.v)
			if got != tt.want {
				t.Errorf("ToIntNoString() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.ok {
				t.Errorf("ToIntNoString() got1 = %v, want %v", got1, tt.ok)
			}
		})
	}
}

func TestToFloat(t *testing.T) {
	tests := []struct {
		name string
		v    Value
		want float64
		ok   bool
	}{
		{
			name: "int",
			v:    IntValue(100),
			want: 100,
			ok:   true,
		},
		{
			name: "integral float",
			v:    FloatValue(53),
			want: 53,
			ok:   true,
		},
		{
			name: "int string",
			v:    StringValue("10"),
			want: 10,
			ok:   true,
		},
		{
			name: "decimal float",
			v:    FloatValue(2.3),
			want: 2.3,
			ok:   true,
		},
		{
			name: "decimal string",
			v:    StringValue("55.5"),
			want: 55.5,
			ok:   true,
		},
		{
			name: "nil",
		},
		{
			name: "boolean",
			v:    BoolValue(true),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ToFloat(tt.v)
			if got != tt.want {
				t.Errorf("ToFloat() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.ok {
				t.Errorf("ToFloat() got1 = %v, want %v", got1, tt.ok)
			}
		})
	}
}

func TestFloatToInt(t *testing.T) {
	tests := []struct {
		input   float64
		wantVal int64
		wantTyp NumberType
	}{
		{123.0, 123, IsInt},
		{-123.0, -123, IsInt},
		{0.0, 0, IsInt},
		{123.456, 0, NaI},                              // Fraction
		{math.NaN(), 0, NaI},                           // NaN
		{math.Inf(1), 0, NaI},                          // Infinity
		{9.23e18, 0, NaI},                              // Overflow (> MaxInt64)
		{-9.23e18, 0, NaI},                             // Underflow (< MinInt64)
		{9223372036854775807.0, 0, NaI},                // MaxInt64 rounds to 2^63 (Too big)
		{-9223372036854775808.0, math.MinInt64, IsInt}, // MinInt64 is exact
	}

	for _, tt := range tests {
		val, typ := FloatToInt(tt.input)
		if val != tt.wantVal || typ != tt.wantTyp {
			t.Errorf("FloatToInt(%g) = (%d, %d), want (%d, %d)",
				tt.input, val, typ, tt.wantVal, tt.wantTyp)
		}
	}
}

func TestClampToInt(t *testing.T) {
	// On 64-bit platforms every case is the identity; on 32-bit platforms the
	// values outside the int range saturate instead of wrapping.
	tests := []struct {
		name string
		n    int64
		want int
	}{
		{name: "zero", n: 0, want: 0},
		{name: "small positive", n: 42, want: 42},
		{name: "small negative", n: -42, want: -42},
		{name: "max int32", n: math.MaxInt32, want: math.MaxInt32},
		{name: "min int32", n: math.MinInt32, want: math.MinInt32},
		{name: "max int64", n: math.MaxInt64, want: math.MaxInt},
		{name: "min int64", n: math.MinInt64, want: math.MinInt},
		{name: "just above max int32", n: math.MaxInt32 + 1, want: clampWant(math.MaxInt32 + 1)},
		{name: "just below min int32", n: math.MinInt32 - 1, want: clampWant(math.MinInt32 - 1)},
		{name: "2^32+1 does not wrap to 1", n: 1<<32 + 1, want: clampWant(1<<32 + 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClampToInt(tt.n); got != tt.want {
				t.Errorf("ClampToInt(%d) = %d, want %d", tt.n, got, tt.want)
			}
		})
	}
}

// clampWant is the expected result for a value that fits in an int on 64-bit
// platforms but not on 32-bit ones.
func clampWant(n int64) int {
	if math.MaxInt == math.MaxInt32 {
		if n > 0 {
			return math.MaxInt
		}
		return math.MinInt
	}
	return int(n)
}
