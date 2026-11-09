package ir

import (
	"testing"

	"github.com/arnodel/golua/ops"
)

func TestMarkFixed(t *testing.T) {
	k := LoadConst{Kidx: 0}
	load := func(r Register) Instruction { k.Dst = r; return k }
	use := func(r Register) Instruction { return Push{Cont: 9, Item: r} }
	for _, x := range []struct {
		name   string
		instrs []Instruction
		reg    Register
		want   bool
	}{
		{"written once", []Instruction{load(1), use(1)}, 1, true},
		{"never seen before the second write", []Instruction{load(1), load(1), use(1)}, 1, true},
		{"written after being seen", []Instruction{load(1), use(1), load(1)}, 1, false},
		{"read and written by one instruction", []Instruction{
			load(1), Combine{Op: ops.OpAdd, Dst: 1, Lsrc: 1, Rsrc: 2},
		}, 1, false},
		{"one of two branches", []Instruction{
			load(1), JumpIf{Cond: 2, Label: 1},
			load(1), Jump{Label: 2},
			DeclareLabel{Label: 1}, load(1),
			DeclareLabel{Label: 2}, use(1),
		}, 1, true},
		{"written in a loop, seen the next time round", []Instruction{
			DeclareLabel{Label: 1}, load(1), use(1), JumpIf{Cond: 2, Label: 1},
		}, 1, false},
		{"written in a loop, cleared each time round", []Instruction{
			DeclareLabel{Label: 1}, load(1), use(1), ClearReg{Dst: 1}, JumpIf{Cond: 2, Label: 1},
		}, 1, true},
		{"a closure stored in the cell it captures", []Instruction{
			MkClosure{Dst: 1, Code: 1, Upvalues: []Register{1}}, use(1),
		}, 1, true},
		{"captured, then written", []Instruction{
			load(1), MkClosure{Dst: 3, Code: 1, Upvalues: []Register{1}}, load(1),
		}, 1, false},
	} {
		main := testCode(0, 10, []Register{1}, x.instrs...)
		child := testCode(0, 1, []Register{0})
		child.UpvalueDests = []Register{0}
		out := markFixed([]Constant{&main, &child})
		if got := out[0].(*Code).Registers[x.reg].IsFixed; got != x.want {
			t.Errorf("%s: IsFixed = %t, want %t", x.name, got, x.want)
		}
	}

	// A cell is shared: a closure writing its upvalue changes it in its
	// maker too, and a change in the maker reaches the closure.
	main := testCode(0, 4, []Register{1, 2},
		load(1), load(2),
		MkClosure{Dst: 3, Code: 1, Upvalues: []Register{1, 2}},
		use(1), use(2),
	)
	main.UpvalueDests = []Register{0}
	main.Registers[0].IsCell = true
	child := testCode(0, 2, []Register{0, 1}, load(0), use(1))
	child.UpvalueDests = []Register{0, 1}
	out := markFixed([]Constant{&main, &child})
	m, c := out[0].(*Code), out[1].(*Code)
	for _, y := range []struct {
		name string
		code *Code
		reg  Register
		want bool
	}{
		{"_ENV, never written", m, 0, true},
		{"written by the closure", m, 1, false},
		{"...as the closure sees it", c, 0, false},
		{"fixed in the maker", m, 2, true},
		{"...as the closure sees it", c, 1, true},
	} {
		if got := y.code.Registers[y.reg].IsFixed; got != y.want {
			t.Errorf("%s: IsFixed = %t, want %t", y.name, got, y.want)
		}
	}
	if main.Registers[2].IsFixed {
		t.Error("markFixed changed its input")
	}

	// An upvalue has its value from the start: any write changes it.
	assigned := testCode(0, 1, []Register{0}, load(0))
	assigned.UpvalueDests = []Register{0}
	if markFixed([]Constant{&assigned})[0].(*Code).Registers[0].IsFixed {
		t.Error("assigned _ENV: IsFixed")
	}
}
