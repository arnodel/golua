package runtime

import (
	"io"
	"testing"

	"github.com/arnodel/golua/code"
)

// Calls list their arguments in operand words; the values arrive in order,
// self first for a method, for any number of arguments, and a missing
// argument is nil.  A trailing vararg or call still pushes one by one.  (The
// bare runtime has no standard library, so the results are checked from Go.)
func TestCallArgs(t *testing.T) {
	r := New(io.Discard)
	f := runChunk(t, r, `
		local o = {}
		function o:m(a, b) return self == o, a, b end
		local function many(a, b, c, d, e, f, g) return a + b + c + d + e + f + g end
		local function two() return 2, 3 end
		local function id(...) return ... end
		return function(x)
			local s1, a1, b1 = o:m(5)
			return x, many(1, 2, 3, 4, 5, 6, 7), s1, a1, b1, id(1, two()), id(two(), 4)
		end
	`)
	term := NewTerminationWith(nil, 10, false)
	if err := Call(r.MainThread(), f, []Value{IntValue(9)}, term); err != nil {
		t.Fatal(err)
	}
	// id(1, two()) is not last in the list, so it yields one value; id(two(),
	// 4) truncates two() to one value and is last, so it yields both.
	want := []Value{IntValue(9), IntValue(28), BoolValue(true), IntValue(5), NilValue, IntValue(1), IntValue(2), IntValue(4), NilValue, NilValue}
	for i, w := range want {
		if got := term.Get(i); got != w {
			t.Errorf("result %d: got %v, want %v", i, got, w)
		}
	}
}

// A temporary passed as argument is cleared by the call, so the value can be
// collected; a local is not.
func TestCallArgsClearTemporaries(t *testing.T) {
	r := New(io.Discard)
	ops := compileMain(t, r, `
		local x = {}
		local function f(a, b) end
		f(x, {})
	`)
	for i := 0; i < len(ops); i++ {
		op := ops[i]
		if n := callArgCount(op); n == 2 {
			clears := ops[i+1].ArgClears()
			if clears[0] || !clears[1] {
				t.Errorf("clear flags %v, want the local kept and the temporary cleared", clears[:2])
			}
			return
		}
	}
	t.Error("no call with two arguments found")
}

// A call with fixed arguments compiles to no push opcodes.
func TestCallArgsNoPush(t *testing.T) {
	r := New(io.Discard)
	ops := compileMain(t, r, `
		local function f(a, b) return a, b end
		local x = f(1, 2)
		return x
	`)
	calls, pushes := 0, 0
	for i := 0; i < len(ops); i++ {
		op := ops[i]
		if !op.HasType1() && op.TypePfx() == code.Type5Pfx && op.GetJ() == code.OpCall {
			if n := op.GetArgCount(); n > 0 {
				calls++
				i += code.ArgWordCount(n)
			}
		}
		if !op.HasType1() && op.TypePfx() == code.Type4Pfx && op.HasType4a() && op.GetUnOp() == code.OpId && op.GetF() {
			pushes++
		}
	}
	if calls != 2 || pushes != 0 {
		t.Errorf("got %d calls with arguments and %d pushes, want 2 and 0", calls, pushes)
	}
}

// string.dump keeps the argument words of a call intact.
func TestCallArgsDump(t *testing.T) {
	r := New(io.Discard)
	f := runChunk(t, r, `return function(a, b) local function g(x, y) return x * y end return g(a, b) end`)
	slim := r.RefactorCodeConsts(f.AsClosure().Code)
	g := NewClosure(r, slim)
	term := NewTerminationWith(nil, 1, false)
	if err := Call(r.MainThread(), FunctionValue(g), []Value{IntValue(6), IntValue(7)}, term); err != nil {
		t.Fatal(err)
	}
	if got := term.Get(0); got != IntValue(42) {
		t.Errorf("got %v, want 42", got)
	}
}
