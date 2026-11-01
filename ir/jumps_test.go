package ir

import (
	"testing"

	"github.com/arnodel/golua/ops"
)

func TestThreadJumps(t *testing.T) {
	lbl := func(l Label) Instruction { return DeclareLabel{Label: l} }
	jump := func(l Label) Instruction { return Jump{Label: l} }
	jumpIf := func(r Register, l Label) Instruction { return JumpIf{Cond: r, Label: l} }
	jumpIfNot := func(r Register, l Label) Instruction { return JumpIf{Cond: r, Label: l, Not: true} }
	add := func(dst, x, y Register) Instruction { return Combine{Op: ops.OpAdd, Dst: dst, Lsrc: x, Rsrc: y} }

	// for i = r0, r1, r2 do r4 = r4 + i end: the jump back skips the entry
	// test, which a new label L2 now follows.
	forLoop := []Instruction{
		PrepForLoop{Start: 0, Stop: 1, Step: 2},
		lbl(0),
		jumpIfNot(0, 1),
		move(3, 0),
		add(4, 4, 3),
		AdvForLoop{Start: 0, Stop: 1, Step: 2},
		jumpIf(0, 0),
		lbl(1),
	}
	check(t, "for loop", ThreadJumps(testCode(0, 5, nil, forLoop...)),
		PrepForLoop{Start: 0, Stop: 1, Step: 2},
		lbl(0),
		jumpIfNot(0, 1),
		lbl(2),
		move(3, 0),
		add(4, 4, 3),
		AdvForLoop{Start: 0, Stop: 1, Step: 2},
		jumpIf(0, 2),
		lbl(1),
	)

	// A jump to a jump, past register bookkeeping.
	chain := []Instruction{jump(0), add(1, 1, 1), lbl(0), ReleaseRegister{Reg: 1}, jump(1), add(1, 1, 1), lbl(1)}
	check(t, "jump to a jump", ThreadJumps(testCode(0, 2, nil, chain...)),
		jump(1), add(1, 1, 1), lbl(0), ReleaseRegister{Reg: 1}, jump(1), add(1, 1, 1), lbl(1))

	// Jumps to what comes next anyway go.
	next := []Instruction{jumpIf(0, 0), TakeRegister{Reg: 1}, lbl(0), jump(1), lbl(1), add(1, 1, 1)}
	check(t, "jump to next", ThreadJumps(testCode(0, 2, nil, next...)),
		TakeRegister{Reg: 1}, lbl(0), lbl(1), add(1, 1, 1))

	// "while a and b do": the test of a copies it into r2, which the loop
	// condition tests at L0.  If a is false, so is r2, so the first jump can
	// go straight to the end.
	and := []Instruction{
		move(2, 0),
		jumpIfNot(0, 0),
		move(2, 1),
		lbl(0),
		jumpIfNot(2, 1),
		add(3, 3, 3),
		lbl(1),
	}
	check(t, "and", ThreadJumps(testCode(2, 2, nil, and...)),
		move(2, 0),
		jumpIfNot(0, 1),
		move(2, 1),
		lbl(0),
		jumpIfNot(2, 1),
		add(3, 3, 3),
		lbl(1),
	)

	// A test of another register tells nothing.
	other := []Instruction{jumpIfNot(0, 0), add(2, 2, 2), lbl(0), jumpIfNot(1, 1), add(2, 2, 2), lbl(1)}
	check(t, "other register", ThreadJumps(testCode(0, 3, nil, other...)), other...)

	// Nor does a copy seen through a label, which another path can reach.
	viaLabel := []Instruction{move(2, 0), lbl(2), jumpIfNot(0, 0), add(3, 3, 3), lbl(0), jumpIfNot(2, 1), add(3, 3, 3), lbl(1)}
	check(t, "copy before a label", ThreadJumps(testCode(2, 2, nil, viaLabel...)), viaLabel...)

	// Jumps in a cycle don't stop the pass.
	cycle := []Instruction{lbl(0), jump(1), add(0, 0, 0), lbl(1), jump(0)}
	ThreadJumps(testCode(0, 1, nil, cycle...))
}
