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

	// q[i], q[j] = q[j], q[i]: the copies of the locals go, the stores read
	// the locals.
	swap := []Instruction{
		Lookup{Dst: 3, Table: q, Index: j},
		Lookup{Dst: 4, Table: q, Index: i},
		move(5, q), move(6, i), move(7, q), move(8, j),
		SetIndex{Table: 5, Index: 6, Src: 3},
		SetIndex{Table: 7, Index: 8, Src: 4},
	}
	check(t, "swap", PropagateCopies(testCode(3, 6, nil, swap...)),
		Lookup{Dst: 3, Table: q, Index: j},
		Lookup{Dst: 4, Table: q, Index: i},
		SetIndex{Table: q, Index: i, Src: 3},
		SetIndex{Table: q, Index: j, Src: 4},
	)

	// i = i + 1: the result is written to i directly.
	inc := []Instruction{
		LoadConst{Dst: 3, Kidx: 0},
		Combine{Op: ops.OpAdd, Dst: 4, Lsrc: i, Rsrc: 3},
		move(i, 4),
	}
	check(t, "increment", PropagateCopies(testCode(3, 2, nil, inc...)),
		LoadConst{Dst: 3, Kidx: 0},
		Combine{Op: ops.OpAdd, Dst: i, Lsrc: i, Rsrc: 3},
	)

	// A copy is not written straight to a register touched since the
	// definition: here j is read between the two.
	touched := []Instruction{
		Combine{Op: ops.OpAdd, Dst: 3, Lsrc: i, Rsrc: i},
		Push{Cont: 4, Item: j},
		move(j, 3),
	}
	check(t, "destination touched", PropagateCopies(testCode(3, 2, nil, touched...)), touched...)

	// A copy of a cell is left alone, and so is a copy into a cell.
	cell := []Instruction{
		move(3, q), SetIndexK{Table: 3, Kidx: 0, Src: i},
		Combine{Op: ops.OpAdd, Dst: 4, Lsrc: i, Rsrc: i}, move(q, 4),
	}
	check(t, "cells", PropagateCopies(testCode(3, 2, []Register{q}, cell...)), cell...)

	// A copy of a fixed cell is read through, like a copy of a local,
	// even across a call: nothing can change the cell.
	once := []Instruction{
		move(3, q),
		Call{Cont: 4},
		Push{Cont: 5, Item: 3},
	}
	fixed := testCode(3, 3, []Register{q}, once...)
	fixed.Registers[q].IsFixed = true
	check(t, "fixed cell", PropagateCopies(fixed),
		Call{Cont: 4},
		Push{Cont: 5, Item: q},
	)

	// A clear starts a new cell, so the copy is not read through after it.
	cleared := []Instruction{
		move(3, q),
		ClearReg{Dst: q},
		Push{Cont: 4, Item: 3},
	}
	fixed = testCode(3, 2, []Register{q}, cleared...)
	fixed.Registers[q].IsFixed = true
	check(t, "cell cleared", PropagateCopies(fixed), cleared...)

	// A copy of a temporary is not read through: that would only keep the
	// temporary alive.  Here the copy into a named local stays as well, so
	// that the local keeps its value.
	local := []Instruction{
		Receive{Dst: []Register{3}},
		move(i, 3),
		Push{Cont: 4, Item: i},
	}
	check(t, "copy of a temporary", PropagateCopies(testCode(3, 2, nil, local...)), local...)

	// A copy is read through past a label when it is available on every
	// path to it: here the label is only reached from the copy.
	label := []Instruction{
		move(3, q),
		DeclareLabel{Label: 1},
		SetIndexK{Table: 3, Kidx: 0, Src: i},
	}
	check(t, "label", PropagateCopies(testCode(3, 1, nil, label...)),
		DeclareLabel{Label: 1},
		SetIndexK{Table: q, Kidx: 0, Src: i},
	)

	// Not when a path to the label writes one of the registers: here the
	// copy is made on both paths but q is written on one of them.
	join := []Instruction{
		move(3, q),
		JumpIf{Cond: i, Label: 1},
		Lookup{Dst: q, Table: q, Index: i},
		DeclareLabel{Label: 1},
		SetIndexK{Table: 3, Kidx: 0, Src: i},
	}
	check(t, "join", PropagateCopies(testCode(3, 1, nil, join...)), join...)

	// A path that doesn't make the copy stops it too.
	missing := []Instruction{
		JumpIf{Cond: i, Label: 1},
		move(3, q),
		DeclareLabel{Label: 1},
		SetIndexK{Table: 3, Kidx: 0, Src: i},
	}
	check(t, "missing on a path", PropagateCopies(testCode(3, 1, nil, missing...)), missing...)

	// A register the instruction also writes is not substituted.
	forloop := []Instruction{
		move(3, q), move(4, i), move(5, j),
		PrepForLoop{Start: 3, Stop: 4, Step: 5, Label: 1},
		DeclareLabel{Label: 1},
	}
	check(t, "read-write operands", PropagateCopies(testCode(3, 3, nil, forloop...)), forloop...)

	// A chain of copies of a local: r3 <- q is read through into r4 <- q,
	// then r4 into the push; both copies are dead temporaries.
	chain := []Instruction{
		move(3, q),
		move(4, 3),
		Push{Cont: 5, Item: 4},
	}
	check(t, "chain", PropagateCopies(testCode(3, 3, nil, chain...)),
		Push{Cont: 5, Item: q},
	)
}

// A numeric for loop: the loop variable is a copy of the counter made at the
// top of every iteration, and the counter is only written by advfor at the
// bottom, which jumps back to the top.  The body reads the counter, past its branches, and the copy goes;
// unless the variable is a cell, when each iteration needs its own.
func TestForLoopVariable(t *testing.T) {
	// The counter is r0, the loop variable r1, temporaries r2 and r3.
	counter, v := Register(0), Register(1)
	loop := []Instruction{
		DeclareLabel{Label: 1},
		move(v, counter),
		JumpIf{Cond: v, Label: 2},
		Combine{Op: ops.OpAdd, Dst: 2, Lsrc: v, Rsrc: v},
		DeclareLabel{Label: 2},
		Combine{Op: ops.OpMul, Dst: 3, Lsrc: v, Rsrc: 2},
		AdvForLoop{Start: counter, Stop: 2, Step: 3, Label: 1},
	}
	check(t, "plain", PropagateCopies(testCode(2, 2, nil, loop...)),
		DeclareLabel{Label: 1},
		JumpIf{Cond: counter, Label: 2},
		Combine{Op: ops.OpAdd, Dst: 2, Lsrc: counter, Rsrc: counter},
		DeclareLabel{Label: 2},
		Combine{Op: ops.OpMul, Dst: 3, Lsrc: counter, Rsrc: 2},
		AdvForLoop{Start: counter, Stop: 2, Step: 3, Label: 1},
	)
	check(t, "cell", PropagateCopies(testCode(2, 2, []Register{v}, loop...)), loop...)

	// The counter written in the body: the copy stays where the body reads
	// the variable after the write.
	written := []Instruction{
		DeclareLabel{Label: 1},
		move(v, counter),
		Combine{Op: ops.OpAdd, Dst: 2, Lsrc: v, Rsrc: v},
		Combine{Op: ops.OpAdd, Dst: counter, Lsrc: counter, Rsrc: 2},
		Combine{Op: ops.OpMul, Dst: 3, Lsrc: v, Rsrc: 2},
		JumpIf{Cond: counter, Label: 1},
	}
	check(t, "counter written", PropagateCopies(testCode(2, 2, nil, written...)),
		DeclareLabel{Label: 1},
		move(v, counter),
		Combine{Op: ops.OpAdd, Dst: 2, Lsrc: counter, Rsrc: counter},
		Combine{Op: ops.OpAdd, Dst: counter, Lsrc: counter, Rsrc: 2},
		Combine{Op: ops.OpMul, Dst: 3, Lsrc: v, Rsrc: 2},
		JumpIf{Cond: counter, Label: 1},
	)
}
