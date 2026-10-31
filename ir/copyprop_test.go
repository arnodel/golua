package ir

import (
	"strings"
	"testing"

	"github.com/arnodel/golua/ops"
)

// A code with the given registers (named locals first, then temporaries) and
// instructions; cells are the registers listed in cells.
func testCode(locals, temps int, cells []Register, instrs ...Instruction) Code {
	regs := make([]RegData, locals+temps)
	for i := 0; i < locals; i++ {
		regs[i].IsLocal = true
	}
	for _, c := range cells {
		regs[c].IsCell = true
	}
	return Code{Instructions: instrs, Lines: make([]int, len(instrs)), Registers: regs}
}

func move(dst, src Register) Instruction { return Transform{Op: ops.OpId, Dst: dst, Src: src} }

func render(c Code) string {
	var b strings.Builder
	for _, in := range c.Instructions {
		b.WriteString(in.String())
		b.WriteString("\n")
	}
	return b.String()
}

func check(t *testing.T, name string, got Code, want ...Instruction) {
	t.Helper()
	w := render(Code{Instructions: want})
	if g := render(got); g != w {
		t.Errorf("%s:\ngot\n%swant\n%s", name, g, w)
	}
	if len(got.Lines) != len(got.Instructions) {
		t.Errorf("%s: %d lines for %d instructions", name, len(got.Lines), len(got.Instructions))
	}
}

func TestPropagateCopies(t *testing.T) {
	// Locals: q=r0, i=r1, j=r2.  Temporaries from r3.
	q, i, j := Register(0), Register(1), Register(2)
	taken := func(rs ...Register) []Instruction {
		var out []Instruction
		for _, r := range rs {
			out = append(out, TakeRegister{Reg: r})
		}
		return out
	}

	// q[i], q[j] = q[j], q[i]: the copies of the locals go, the stores read
	// the locals.  The compiler releases operands just before the store.
	swap := append(taken(q, i, j),
		Lookup{Dst: 3, Table: q, Index: j}, TakeRegister{Reg: 3},
		Lookup{Dst: 4, Table: q, Index: i}, TakeRegister{Reg: 4},
		move(5, q), TakeRegister{Reg: 5}, move(6, i), TakeRegister{Reg: 6},
		move(7, q), TakeRegister{Reg: 7}, move(8, j), TakeRegister{Reg: 8},
		ReleaseRegister{Reg: 3}, ReleaseRegister{Reg: 5}, ReleaseRegister{Reg: 6},
		SetIndex{Table: 5, Index: 6, Src: 3},
		ReleaseRegister{Reg: 4}, ReleaseRegister{Reg: 7}, ReleaseRegister{Reg: 8},
		SetIndex{Table: 7, Index: 8, Src: 4},
	)
	check(t, "swap", PropagateCopies(testCode(3, 6, nil, swap...)), append(taken(q, i, j),
		Lookup{Dst: 3, Table: q, Index: j}, TakeRegister{Reg: 3},
		Lookup{Dst: 4, Table: q, Index: i}, TakeRegister{Reg: 4},
		TakeRegister{Reg: 5}, TakeRegister{Reg: 6}, TakeRegister{Reg: 7}, TakeRegister{Reg: 8},
		ReleaseRegister{Reg: 3}, ReleaseRegister{Reg: 5}, ReleaseRegister{Reg: 6},
		SetIndex{Table: q, Index: i, Src: 3},
		ReleaseRegister{Reg: 4}, ReleaseRegister{Reg: 7}, ReleaseRegister{Reg: 8},
		SetIndex{Table: q, Index: j, Src: 4},
	)...)

	// i = i + 1: the result is written to i directly.
	inc := append(taken(i),
		LoadConst{Dst: 3, Kidx: 0},
		Combine{Op: ops.OpAdd, Dst: 4, Lsrc: i, Rsrc: 3},
		TakeRegister{Reg: 4}, ReleaseRegister{Reg: 4},
		move(i, 4),
	)
	check(t, "increment", PropagateCopies(testCode(3, 2, nil, inc...)), append(taken(i),
		LoadConst{Dst: 3, Kidx: 0},
		Combine{Op: ops.OpAdd, Dst: i, Lsrc: i, Rsrc: 3},
		TakeRegister{Reg: 4}, ReleaseRegister{Reg: 4},
	)...)

	// A copy is not written straight to a register touched since the
	// definition: here j is read between the two.
	touched := append(taken(i, j),
		Combine{Op: ops.OpAdd, Dst: 3, Lsrc: i, Rsrc: i},
		Push{Cont: 4, Item: j},
		move(j, 3),
	)
	check(t, "destination touched", PropagateCopies(testCode(3, 2, nil, touched...)), touched...)

	// A copy of a cell is left alone, and so is a copy into a cell.
	cell := append(taken(q, i),
		move(3, q), SetIndexK{Table: 3, Kidx: 0, Src: i},
		Combine{Op: ops.OpAdd, Dst: 4, Lsrc: i, Rsrc: i}, move(q, 4),
	)
	check(t, "cells", PropagateCopies(testCode(3, 2, []Register{q}, cell...)), cell...)

	// A copy from a register that is not taken is not read through: its slot
	// may be reused.  The copy into a named local stays.
	untaken := []Instruction{
		TakeRegister{Reg: i},
		Receive{Dst: []Register{3}},
		move(i, 3),
		Push{Cont: 4, Item: i},
	}
	check(t, "untaken source", PropagateCopies(testCode(3, 2, nil, untaken...)), untaken...)

	// Nothing is known across a label; a copy before the label is kept and
	// not read through after it.
	label := append(taken(q, i),
		move(3, q), TakeRegister{Reg: 3},
		DeclareLabel{Label: 1},
		SetIndexK{Table: 3, Kidx: 0, Src: i},
		ReleaseRegister{Reg: 3},
	)
	check(t, "label", PropagateCopies(testCode(3, 1, nil, label...)), label...)

	// A register the instruction also writes is not substituted.
	forloop := append(taken(q, i, j),
		move(3, q), move(4, i), move(5, j),
		PrepForLoop{Start: 3, Stop: 4, Step: 5},
	)
	check(t, "read-write operands", PropagateCopies(testCode(3, 3, nil, forloop...)), forloop...)

	// A copy made after its source was released is not read through.
	released := append(taken(q),
		move(3, q), TakeRegister{Reg: 3}, ReleaseRegister{Reg: 3},
		move(4, 3), TakeRegister{Reg: 4},
		Push{Cont: 5, Item: 4},
	)
	check(t, "released source", PropagateCopies(testCode(3, 3, nil, released...)), append(taken(q),
		// r3 <- q is read through (q is taken) into r4 <- q, then r4 is
		// read through into the push; both copies are dead temporaries.
		TakeRegister{Reg: 3}, ReleaseRegister{Reg: 3},
		TakeRegister{Reg: 4},
		Push{Cont: 5, Item: q},
	)...)
}
