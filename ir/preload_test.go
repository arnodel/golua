package ir

import (
	"testing"

	"github.com/arnodel/golua/ops"
)

func TestPreloadConstants(t *testing.T) {
	// Locals: x=r0, y=r1.  Temporaries from r2.  The code has 6 registers,
	// so preloaded ones are r6, r7, ...
	x, y := Register(0), Register(1)
	add := func(dst, l, r Register) Instruction { return Combine{Op: ops.OpAdd, Dst: dst, Lsrc: l, Rsrc: r} }
	k := func(dst Register, kidx uint) Instruction { return LoadConst{Dst: dst, Kidx: kidx} }

	// Temporaries loaded once: their reads read the preloaded register, the
	// same constant twice shares one, and a read far from the load counts.
	c := PreloadConstants(testCode(2, 4, nil,
		k(2, 0), add(y, x, 2),
		k(3, 1), k(4, 0), SetIndex{Table: x, Index: 3, Src: 4},
		Push{Cont: 5, Item: 3}))
	check(t, "temporaries", c,
		add(y, x, 6),
		SetIndex{Table: x, Index: 7, Src: 6},
		Push{Cont: 5, Item: 7})
	if len(c.Registers) != 8 {
		t.Errorf("%d registers, want 8", len(c.Registers))
	}
	for r, want := range map[Register]uint{6: 0, 7: 1} {
		if data := c.Registers[r]; !data.Preloaded || data.Kidx != want {
			t.Errorf("r%d is %+v, want preloaded K%d", r, data, want)
		}
	}

	// Loads that stay: into a named local, into a cell, into a register
	// written again.
	for _, tc := range []struct {
		name   string
		cells  []Register
		instrs []Instruction
	}{
		{"local", nil, []Instruction{k(y, 0), add(2, x, y)}},
		{"cell", []Register{y}, []Instruction{k(y, 0), add(y, x, y)}},
		{"written twice", nil, []Instruction{k(2, 0), add(y, x, 2), set(2), Push{Cont: 3, Item: 2}}},
	} {
		c := PreloadConstants(testCode(2, 4, tc.cells, tc.instrs...))
		check(t, tc.name, c, tc.instrs...)
		if len(c.Registers) != 6 {
			t.Errorf("%s: %d registers, want 6", tc.name, len(c.Registers))
		}
	}

	// PreloadSmall leaves a function above its size alone.
	small := []Instruction{k(2, 0), add(y, x, 2), ret(0)}
	check(t, "small", PreloadSmall(3)(testCode(2, 4, nil, small...)), add(y, x, 6), ret(0))
	check(t, "too big", PreloadSmall(2)(testCode(2, 4, nil, small...)), small...)
}
