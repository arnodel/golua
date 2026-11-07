package ir

import (
	"testing"

	"github.com/arnodel/golua/ops"
)

func TestHoistConstants(t *testing.T) {
	add := func(dst, l, r Register) Instruction { return Combine{Op: ops.OpAdd, Dst: dst, Lsrc: l, Rsrc: r} }
	k := func(dst Register, kidx uint) Instruction { return LoadConst{Dst: dst, Kidx: kidx} }

	// Locals x=r0 and c=r1; temporaries from r2; the code has 6 registers,
	// so hoisted ones are r6 and r7.
	for _, tc := range []struct {
		name     string
		locals   int
		in, want []Instruction
	}{
		// while c do x = x + 1 end, as the compiler lays it out: the test
		// at the bottom, entered by a jump.  The load goes before that jump.
		{"while", 2,
			[]Instruction{jmp(2), lbl(1), k(2, 0), add(0, 0, 2), lbl(2), jif(1, 1), ret(0)},
			[]Instruction{k(6, 0), jmp(2), lbl(1), add(0, 0, 6), lbl(2), jif(1, 1), ret(0)}},
		// A loop entered by falling into it: the load goes at the end of the
		// block before.  The same constant twice shares one register, and a
		// different one gets another.
		{"fall in, shared", 2,
			[]Instruction{set(0), lbl(1), k(2, 0), add(0, 0, 2), k(3, 0), add(0, 0, 3), k(4, 1), add(0, 0, 4), jif(1, 1), ret(0)},
			[]Instruction{set(0), k(6, 0), k(7, 1), lbl(1), add(0, 0, 6), add(0, 0, 6), add(0, 0, 7), jif(1, 1), ret(0)}},
		// Nested loops: the load in the inner loop goes before the outer.
		{"nested", 2,
			[]Instruction{set(0), lbl(1), lbl(2), k(2, 0), add(0, 0, 2), jif(2, 0), jif(1, 1), ret(0)},
			[]Instruction{set(0), k(6, 0), lbl(1), lbl(2), add(0, 0, 6), jif(2, 0), jif(1, 1), ret(0)}},
		// The whole function is the loop: the load goes first.
		{"entry", 2,
			[]Instruction{lbl(1), k(2, 0), add(0, 0, 2), jif(1, 1), ret(0)},
			[]Instruction{k(6, 0), lbl(1), add(0, 0, 6), jif(1, 1), ret(0)}},
		// A loop entered by a conditional jump: the load goes at the top of
		// the entry under a new label, L2, and the jump back is moved to it.
		{"conditional entry", 2,
			[]Instruction{jif(1, 1), ret(0), lbl(1), k(2, 0), add(0, 0, 2), jif(1, 1), ret(0)},
			[]Instruction{jif(1, 1), ret(0), lbl(1), k(6, 0), lbl(2), add(0, 0, 6), jif(2, 1), ret(0)}},
		// A loop that goto gives two entries, L1 and L2, reached by jumps:
		// both get the load, and the jumps back are moved past it.
		{"two entries", 2,
			[]Instruction{jif(2, 1), lbl(1), set(0), jmp(2), lbl(2), k(2, 0), add(0, 0, 2), jif(1, 1), ret(0)},
			[]Instruction{jif(2, 1), lbl(1), k(6, 0), lbl(3), set(0), jmp(4), lbl(2), k(6, 0), lbl(4), add(0, 0, 6), jif(3, 1), ret(0)}},
		// With two entries and a block falling from one into the other, the
		// loads can't go at the top of the second: left alone.
		{"two entries, fall-through", 2,
			[]Instruction{jif(2, 1), lbl(1), set(0), lbl(2), k(2, 0), add(0, 0, 2), jif(1, 1), ret(0)},
			nil},
		// A load into a named local, or into a register written again, stays.
		{"local", 3,
			[]Instruction{lbl(1), k(2, 0), add(0, 0, 2), jif(1, 1), ret(0)},
			nil},
		{"written twice", 2,
			[]Instruction{lbl(1), k(2, 0), add(0, 0, 2), set(2), jif(1, 1), ret(0)},
			nil},
	} {
		c := testCode(tc.locals, 6-tc.locals, nil, tc.in...)
		got := HoistConstants(c)
		want := tc.want
		if want == nil {
			want = tc.in
		}
		check(t, tc.name, got, want...)
	}
}
