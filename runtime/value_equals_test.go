package runtime

import (
	"math"
	"testing"
)

// Equals rejects values with different scalars without looking further,
// except when the scalars differ only by the sign bit.  These cases exercise
// the edges of that shortcut.
func TestValueEquals(t *testing.T) {
	nan := math.NaN()
	tbl := NewTable()
	cases := []struct {
		name string
		v, w Value
		eq   bool
	}{
		{"same int", IntValue(42), IntValue(42), true},
		{"different ints", IntValue(42), IntValue(43), false},
		{"zero and min int", IntValue(0), IntValue(math.MinInt64), false},
		{"same float", FloatValue(1.5), FloatValue(1.5), true},
		{"different floats", FloatValue(1.5), FloatValue(2.5), false},
		{"positive and negative zero", FloatValue(0), FloatValue(math.Copysign(0, -1)), true},
		{"negative zero and min int", FloatValue(math.Copysign(0, -1)), IntValue(math.MinInt64), false},
		{"nan", FloatValue(nan), FloatValue(nan), false},
		{"int and float with same value", IntValue(1), FloatValue(1), false},
		{"int zero and float zero", IntValue(0), FloatValue(0), false},
		{"false and int zero", BoolValue(false), IntValue(0), false},
		{"true and int one", BoolValue(true), IntValue(1), false},
		{"same bool", BoolValue(true), BoolValue(true), true},
		{"empty string and int zero", StringValue(""), IntValue(0), false},
		{"empty string and nil", StringValue(""), NilValue, false},
		{"same short string", StringValue("abc"), StringValue("abc"), true},
		{"different short strings", StringValue("abc"), StringValue("abd"), false},
		{"same long string", StringValue("a long string"), StringValue("a long string"), true},
		{"different long strings", StringValue("a long string"), StringValue("b long string"), false},
		{"same table", TableValue(tbl), TableValue(tbl), true},
		{"different tables", TableValue(tbl), TableValue(NewTable()), false},
		{"table and nil", TableValue(tbl), NilValue, false},
		{"nil and nil", NilValue, NilValue, true},
	}
	for _, c := range cases {
		if got := c.v.Equals(c.w); got != c.eq {
			t.Errorf("%s: Equals(%v, %v) = %v, want %v", c.name, c.v, c.w, got, c.eq)
		}
		if got := c.w.Equals(c.v); got != c.eq {
			t.Errorf("%s (swapped): Equals(%v, %v) = %v, want %v", c.name, c.w, c.v, got, c.eq)
		}
	}
}
