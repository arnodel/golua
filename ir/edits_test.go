package ir

import (
	"testing"

	"github.com/arnodel/golua/ops"
)

func TestEdits(t *testing.T) {
	add := func(dst, l, r Register) Instruction { return Combine{Op: ops.OpAdd, Dst: dst, Lsrc: l, Rsrc: r} }
	c := testCode(1, 2, nil,
		lbl(1), set(1), add(0, 0, 1), jif(1, 2), ret(0))
	for i := range c.Lines {
		c.Lines[i] = 1
	}
	e := NewEdits(c)
	r := e.NewRegister(RegData{})
	if r != 3 {
		t.Fatalf("new register is r%d, want r3", r)
	}
	e.InsertBefore(1, LoadConst{Dst: r, Kidx: 7}, lbl(2))
	e.InsertBefore(1, set(2))
	e.Delete(1)
	e.RenameReads(1, r)
	e.Retarget(3, 2)
	e.InsertBefore(5, set(2)) // at the end
	got := e.Apply(c)
	check(t, "edits", got,
		lbl(1), LoadConst{Dst: r, Kidx: 7}, lbl(2), set(2), add(0, 0, r), jif(2, 2), ret(0), set(2))
	if len(got.Registers) != 4 || len(c.Registers) != 3 {
		t.Errorf("%d registers after, %d before; want 4 and 3", len(got.Registers), len(c.Registers))
	}
	for i, in := range got.Instructions {
		if _, inserted := map[int]bool{1: true, 2: true, 3: true, 7: true}[i]; inserted != (got.Lines[i] == 0) {
			t.Errorf("line %d for %s", got.Lines[i], in)
		}
	}
}
