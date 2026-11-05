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
		case ir.DeclareLabel, ir.Jump, ir.JumpIf:
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
		// r5 is a == b.
		{"if a ~= b then a = 1 end", `
jump L1 if r5 is not false
L1:
L0:
`},
		// A while loop tests its condition after the body, and jumps back to
		// the body if it holds.  It starts with a jump to the test (L2)...
		{"while a and b do a = 1 end", `
jump L2
L1:
L2:
jump L3 if r2 is not true
jump L1 if r3 is not false
L3:
L0:
`},
		// ... unless the condition always holds.
		{"while true do a = 1 end", `
L1:
L2:
jump L1
L0:
`},
		// r6 holds the condition, computed before the body's scope closes.
		{"repeat a = 1 until not a", `
L1:
jump L1 if r6 is not false
L0:
`},
		{"repeat local d = a until d", `
L1:
jump L1 if r6 is not true
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
