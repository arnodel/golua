package astcomp

import (
	"strings"
	"testing"

	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
	"github.com/arnodel/golua/parsing"
	"github.com/arnodel/golua/scanner"
)

// controlFlow compiles src and renders the main chunk's labels, jumps and
// "not" operations, one per line, and its moves if moves is true.
func controlFlow(t *testing.T, src string, moves bool) string {
	t.Helper()
	stat, err := parsing.ParseChunk(scanner.New("test", []byte(src)))
	if err != nil {
		t.Fatal(err)
	}
	kidx, consts, err := CompileLuaChunk("test", stat)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, instr := range consts[kidx].(*ir.Code).Instructions {
		switch x := instr.(type) {
		case ir.DeclareLabel, ir.Jump, ir.JumpIf, ir.JumpIfCmp:
			b.WriteString(instr.String() + "\n")
		case ir.Transform:
			if x.Op == ops.OpNot || moves && x.Op == ops.OpId {
				b.WriteString(instr.String() + "\n")
			}
		}
	}
	return b.String()
}

// Conditions compile to one jump per operand, with no value to build and no
// "not" to compute.  Locals a, b and c are in r2, r3 and r4.
func TestCondJumps(t *testing.T) {
	for _, tc := range []struct{ src, want string }{
		{"if a or b and c then a = 1 end", `
jump L2 if r2 is not false
jump L1 if r3 is not true
jump L1 if r4 is not true
L2:
L1:
L0:
`},
		{"if not (a and b) then a = 1 end", `
jump L2 if r2 is not true
jump L1 if r3 is not false
L2:
L1:
L0:
`},
		// A comparison is one instruction with its jump: no register holds
		// the boolean.  "~=" is "==" with the jump's direction swapped, ">"
		// and ">=" are "<" and "<=" with the operands swapped.
		{"if a ~= b then a = 1 end", `
if OpEq(r2, r3) jump L1
L1:
L0:
`},
		{"if a < b then a = 1 end", `
if not OpLt(r2, r3) jump L1
L1:
L0:
`},
		{"if a >= b then a = 1 end", `
if not OpLeq(r3, r2) jump L1
L1:
L0:
`},
		{"if not (a > b) then a = 1 end", `
if OpLt(r3, r2) jump L1
L1:
L0:
`},
		{"while a <= b do a = 1 end", `
L1:
if not OpLeq(r2, r3) jump L0
jump L1
L0:
`},
		{"if a < b and b == c then a = 1 end", `
if not OpLt(r2, r3) jump L1
if not OpEq(r3, r4) jump L1
L1:
L0:
`},
		{"while true do a = 1 end", `
L1:
jump L1
L0:
`},
		{"while a and b do a = 1 end", `
L1:
jump L0 if r2 is not true
jump L0 if r3 is not true
jump L1
L0:
`},
		// The condition of a repeat loop is compiled as a condition like any
		// other, as long as closing the body's scope emits nothing.
		{"repeat a = 1 until not a", `
L1:
jump L1 if r2 is not false
L0:
`},
		{"repeat local d = a until d", `
L1:
jump L1 if r5 is not true
L0:
`},
		{"repeat a = 1 until a == b", `
L1:
if not OpEq(r2, r3) jump L1
L0:
`},
		// When closing the scope emits code (a to-be-closed variable, a local
		// captured by a closure), the condition is a value computed before,
		// and tested after.
		{"repeat local d <close> = a until d", `
L1:
jump L1 if r6 is not true
L0:
`},
		{"repeat local d = a; local function f() return d end until d == b", `
L1:
jump L1 if r7 is not true
L0:
`},
		// A generic for loop tests its first variable against a nil loaded
		// once, before the loop.
		{"for k in a do a = 1 end", `
L0:
if OpEq(r10, r9) jump L1
jump L0
L1:
`},
		// A constant condition needs no test.
		{"repeat a = 1 until false", `
L1:
jump L1
L0:
`},
		{"repeat a = 1 until not nil", `
L1:
L0:
`},
		{"repeat a = 1 until -1", `
L1:
L0:
`},
		// So do conditions whose value is known, not only literals.
		{"if -1 then a = 1 end", `
L1:
L0:
`},
		{"if a and -1 then a = 1 end", `
jump L1 if r2 is not true
L1:
L0:
`},
		{"while -1 do a = 1 end", `
L1:
jump L1
L0:
`},
	} {
		if got := controlFlow(t, "local a, b, c = ...\n"+tc.src, false); "\n"+got != tc.want {
			t.Errorf("%s:\ngot%s\nwant%s", tc.src, "\n"+got, tc.want)
		}
	}
}

// An operand of "and" or "or" computed as a value only puts its value in the
// result when it can be the result.  Locals a, b and c are in r2, r3 and r4,
// and the result goes into r5.
func TestValueJumps(t *testing.T) {
	for _, tc := range []struct{ src, want string }{
		// Unchanged: every operand can be the result.
		{"local x = a or b", `
r5 := OpId(r2)
jump L0 if r5 is not false
r5 := OpId(r3)
L0:
`},
		// A false a can't be the result, so it jumps to c.
		{"local x = a and b or c", `
jump L1 if r2 is not true
r5 := OpId(r3)
jump L0 if r5 is not false
L1:
r5 := OpId(r4)
L0:
`},
		{"local x = (a or b) and c", `
jump L1 if r2 is not false
r5 := OpId(r3)
jump L0 if r5 is not true
L1:
r5 := OpId(r4)
L0:
`},
		// A constant that can't be the result needs no code.
		{"local x = a and nil or c", `
jump L1 if r2 is not true
L1:
r5 := OpId(r4)
L0:
`},
		// "not a" can't be the result if it is false, so it jumps as a
		// condition does.
		{"local x = not a and b or c", `
jump L1 if r2 is not false
r5 := OpId(r3)
jump L0 if r5 is not false
L1:
r5 := OpId(r4)
L0:
`},
		// Here it can, so its value is computed.
		{"local x = not a or b", `
r5 := OpNot(r2)
jump L0 if r5 is not false
r5 := OpId(r3)
L0:
`},
	} {
		if got := controlFlow(t, "local a, b, c = ...\n"+tc.src, true); "\n"+got != tc.want {
			t.Errorf("%s:\ngot%s\nwant%s", tc.src, "\n"+got, tc.want)
		}
	}
}

// A negated number literal is a constant, except a float zero (-0.0 is
// computed at run time).
func TestNegatedLiterals(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want ir.Constant // nil if the negation stays
	}{
		{"local x = -1", ir.Int(-1)},
		{"local x = - -2", ir.Int(2)},
		{"local x = -2.5", ir.Float(-2.5)},
		{"local x = -0x8000000000000000", ir.Int(-0x8000000000000000)},
		{"local x = -0.0", nil},
		{"local x = -(-0.0)", nil},
	} {
		stat, err := parsing.ParseChunk(scanner.New("test", []byte(tc.src)))
		if err != nil {
			t.Fatal(err)
		}
		kidx, consts, err := CompileLuaChunk("test", stat)
		if err != nil {
			t.Fatal(err)
		}
		var loads []ir.Constant
		negs := 0
		for _, instr := range consts[kidx].(*ir.Code).Instructions {
			switch x := instr.(type) {
			case ir.LoadConst:
				loads = append(loads, consts[x.Kidx])
			case ir.Transform:
				if x.Op == ops.OpNeg {
					negs++
				}
			}
		}
		switch {
		case tc.want == nil && negs == 0:
			t.Errorf("%s: no negation left", tc.src)
		case tc.want != nil && (negs != 0 || len(loads) != 1 || loads[0] != tc.want):
			t.Errorf("%s: loads %v and %d negations, want a load of %v", tc.src, loads, negs, tc.want)
		}
	}
}
