package ircomp

import (
	"strings"
	"testing"

	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

type (
	Instruction = ir.Instruction
	Register    = ir.Register
)

func lbl(n int) Instruction             { return ir.DeclareLabel{Label: ir.Label(n)} }
func jmp(n int) Instruction             { return ir.Jump{Label: ir.Label(n)} }
func jif(n int, r Register) Instruction { return ir.JumpIf{Cond: r, Label: ir.Label(n)} }
func set(r Register) Instruction        { return ir.LoadConst{Dst: r} }
func ret(r Register) Instruction        { return ir.Call{Cont: r, Tail: true} }
func take(r Register) Instruction       { return ir.TakeRegister{Reg: r} }
func release(r Register) Instruction    { return ir.ReleaseRegister{Reg: r} }
func add(dst, l, r Register) Instruction {
	return ir.Combine{Op: ops.OpAdd, Dst: dst, Lsrc: l, Rsrc: r}
}

// A code with n registers, of which cells are cells, and upvals receive
// upvalues.
func testCode(n int, cells, upvals []Register, instrs ...Instruction) ir.Code {
	regs := make([]ir.RegData, n)
	for _, c := range cells {
		regs[c].IsCell = true
	}
	return ir.Code{Instructions: instrs, Lines: make([]int, len(instrs)), Registers: regs, UpvalueDests: upvals}
}

func render(instrs []Instruction) string {
	var b strings.Builder
	for _, in := range instrs {
		b.WriteString(in.String())
		b.WriteString("\n")
	}
	return b.String()
}

func TestComputeLifetimes(t *testing.T) {
	for _, tc := range []struct {
		name          string
		cells, upvals []Register
		in, want      []Instruction
	}{
		// Register 0 holds the caller's continuation and is taken over the
		// whole function.  The others live from their first write to their
		// last read; the compiler's own markers are replaced.  r1 is read for
		// the last time by the addition, so it is released before it; r2 is
		// written by it, so after.
		{"straight line", nil, nil,
			[]Instruction{take(1), set(1), take(2), set(2), add(2, 1, 2), release(1), release(2), ret(0)},
			[]Instruction{take(0), take(1), set(1), take(2), set(2), release(1), add(2, 1, 2), release(2), ret(0), release(0)}},
		// A register written and never read keeps its slot at the write.
		{"never read", nil, nil,
			[]Instruction{set(1), ret(0)},
			[]Instruction{take(0), take(1), set(1), release(1), ret(0), release(0)}},
		// x = 0; while c do x = x + y end; return x, with c, x, y in r1, r2,
		// r3.  c and y are read before any write: live from the entry.  All
		// three are live over the whole loop, so they stay taken until after
		// the jump back, which is the last instruction in code order.
		{"loop", nil, nil,
			[]Instruction{set(2), jmp(2), lbl(1), add(2, 2, 3), lbl(2), jif(1, 1), ret(2)},
			[]Instruction{take(0), take(1), take(3), take(2), set(2), jmp(2), lbl(1), add(2, 2, 3), lbl(2), jif(1, 1), release(1), release(3), release(2), ret(2), release(0)}},
		// A: ...; B: ...; C: jump A; D: jump B.  r1 is set in A and read
		// in B, and D jumps back to B, so r1 stays taken until after D.  r2
		// is set in B and read at C only.
		{"overlapping jumps", nil, nil,
			[]Instruction{lbl(1), set(1), lbl(2), add(2, 1, 1), jif(1, 2), jif(2, 3), ret(0)},
			[]Instruction{take(0), take(3), lbl(1), take(1), set(1), lbl(2), take(2), add(2, 1, 1), release(2), jif(1, 2), jif(2, 3), release(1), release(3), ret(0), release(0)}},
		// Writing a cell stores into it, so a cell is live from the entry
		// until its last use; only clearing it starts afresh.
		{"cell", []Register{1}, nil,
			[]Instruction{set(1), ir.ClearReg{Dst: 1}, set(1), ret(0)},
			[]Instruction{take(0), take(1), set(1), ir.ClearReg{Dst: 1}, set(1), release(1), ret(0), release(0)}},
		// Upvalues are the code generator's business.
		{"upvalue", []Register{1}, []Register{1},
			[]Instruction{set(1), ret(0)},
			[]Instruction{take(0), set(1), ret(0), release(0)}},
	} {
		got := computeLifetimes(testCode(4, tc.cells, tc.upvals, tc.in...))
		if g, w := render(got.Instructions), render(tc.want); g != w {
			t.Errorf("%s:\ngot\n%swant\n%s", tc.name, g, w)
		}
		if len(got.Lines) != len(got.Instructions) {
			t.Errorf("%s: %d lines for %d instructions", tc.name, len(got.Lines), len(got.Instructions))
		}
	}
}
