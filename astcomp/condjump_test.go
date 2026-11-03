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
// "not" operations, one per line.
func controlFlow(t *testing.T, src string) string {
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
			if x.Op == ops.OpNot {
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
		if got := controlFlow(t, "local a, b, c = ...\n"+tc.src); "\n"+got != tc.want {
			t.Errorf("%s:\ngot%s\nwant%s", tc.src, "\n"+got, tc.want)
		}
	}
}
