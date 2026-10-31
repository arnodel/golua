package runtime

import (
	"io"
	"testing"
)

// The array fast path of the index opcodes must agree with the general path:
// a store through it keeps the length bookkeeping right (a nil store takes the
// general path), a key outside the array part or a nil slot falls back to the
// general path, and metamethods still apply where the raw lookup misses.
func TestArrayFastPath(t *testing.T) {
	r := New(io.Discard)
	class := NewTable()
	class.Set(StringValue("__index"), runChunk(t, r, `return function(_, k) return -k end`))
	o := NewTable()
	o.Set(IntValue(1), IntValue(10))
	o.Set(IntValue(2), IntValue(20))
	o.Set(IntValue(3), IntValue(30))
	o.SetMetatable(class)

	f := runChunk(t, r, `
		return function(t)
			t[2] = 200          -- fast path store
			t[3] = nil          -- general path: removes, #t becomes 2
			local a = t[2]      -- fast path load
			local b = t[3]      -- nil slot: __index gives -3
			local c = t[7]      -- outside the array part: __index gives -7
			local d = t[0]      -- not an array index: __index gives 0
			return a, b, c, d, #t
		end
	`)
	term := NewTerminationWith(nil, 5, false)
	if err := Call(r.MainThread(), f, []Value{TableValue(o)}, term); err != nil {
		t.Fatal(err)
	}
	want := []Value{IntValue(200), IntValue(-3), IntValue(-7), IntValue(0), IntValue(2)}
	for i, w := range want {
		if got := term.Get(i); got != w {
			t.Errorf("result %d: got %v, want %v", i, got, w)
		}
	}
}
