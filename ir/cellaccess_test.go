package ir

import (
	"testing"

	"github.com/arnodel/golua/ops"
)

func TestLowerCells(t *testing.T) {
	add := func(dst, l, r Register) Instruction { return Combine{Op: ops.OpAdd, Dst: dst, Lsrc: l, Rsrc: r} }
	load := func(dst, box Register) Instruction { return Load{Dst: dst, Box: box} }
	store := func(box, src Register) Instruction { return Store{Box: box, Src: src} }
	// A cell local u0 (r0) and a temporary r1; new registers start at r2.
	cell := func(instrs ...Instruction) Code { return testCode(0, 2, []Register{0}, instrs...) }
	fixedCell := func(instrs ...Instruction) Code {
		c := cell(instrs...)
		c.Registers[0].IsFixed = true
		return c
	}

	check(t, "read", LowerCells(cell(add(1, 0, 1))),
		load(2, 0), add(1, 2, 1))

	check(t, "write", LowerCells(cell(add(0, 1, 1))),
		add(2, 1, 1), store(0, 2))

	// A second read after an operation that may run a metamethod loads
	// again, unless the cell is fixed.
	check(t, "reread", LowerCells(cell(add(1, 0, 1), add(1, 0, 1))),
		load(2, 0), add(1, 2, 1), load(3, 0), add(1, 3, 1))
	check(t, "reread fixed", LowerCells(fixedCell(add(1, 0, 1), add(1, 0, 1))),
		load(2, 0), add(1, 2, 1), add(1, 2, 1))

	// After a store, the stored register stands for the cell.
	check(t, "store then read", LowerCells(cell(move(0, 1), add(1, 0, 1))),
		store(0, 1), add(1, 1, 1))
	// The result written back stands for the cell too.
	check(t, "write then read", LowerCells(cell(add(0, 1, 1), move(1, 0))),
		add(2, 1, 1), store(0, 2), move(1, 2))

	// Not across a label, nor once the register is written.
	check(t, "label", LowerCells(cell(move(0, 1), DeclareLabel{Label: 0}, move(1, 0))),
		store(0, 1), DeclareLabel{Label: 0}, load(1, 0))
	check(t, "register written", LowerCells(cell(move(0, 1), LoadConst{Dst: 1, Kidx: 0}, add(1, 0, 1))),
		store(0, 1), LoadConst{Dst: 1, Kidx: 0}, load(2, 0), add(1, 2, 1))

	// A cell an instruction both reads and writes gets one register for
	// both, loaded and stored.
	check(t, "read-write", LowerCells(cell(MkVarargTable{Dst: 1, Etc: 0})),
		load(2, 0), MkVarargTable{Dst: 1, Etc: 2}, store(0, 2))

	// A closure stored in the cell it captures: the cell stays the upvalue.
	check(t, "closure in cell", LowerCells(cell(MkClosure{Dst: 0, Code: 0, Upvalues: []Register{0}})),
		MkClosure{Dst: 2, Code: 0, Upvalues: []Register{0}}, store(0, 2))

	// A fixed upvalue is loaded once at entry, before the parameters.
	up := testCode(0, 2, []Register{0}, Receive{Dst: []Register{1}}, add(1, 0, 1), DeclareLabel{Label: 0}, add(1, 0, 1))
	up.UpvalueDests = []Register{0}
	up.Registers[0].IsFixed = true
	check(t, "fixed upvalue", LowerCells(up),
		load(2, 0), Receive{Dst: []Register{1}}, add(1, 2, 1), DeclareLabel{Label: 0}, add(1, 2, 1))

	// A clear of a cell makes a new box; a move between two cells is a load
	// and a store.
	two := testCode(0, 2, []Register{0, 1}, ClearReg{Dst: 0}, move(1, 0))
	check(t, "clear and cell to cell", LowerCells(two),
		NewBox{Dst: 0}, load(2, 0), store(1, 2))

	// Not when only a nested closure captures it.
	captured := testCode(0, 2, []Register{0}, MkClosure{Dst: 1, Code: 0, Upvalues: []Register{0}})
	captured.UpvalueDests = []Register{0}
	captured.Registers[0].IsFixed = true
	check(t, "captured only", LowerCells(captured),
		MkClosure{Dst: 1, Code: 0, Upvalues: []Register{0}})
}
