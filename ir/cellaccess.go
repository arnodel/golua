package ir

import (
	"fmt"

	"github.com/arnodel/golua/ops"
)

// LoadStoreCells makes the move the only instruction that reads or writes
// the contents of a cell.  Every other instruction with a cell operand gets
// a plain temporary in its place: a move from the cell before it for a read,
// a move into the cell after it for a write.  MkClosure captures cells
// themselves, not their contents, and ClearReg makes a new one, so both are
// left alone.
//
// This is the prototype of notes/cells.md: with it, the run loop needs to
// know about cells only in the move opcode, and every other register access
// is a plain index.  Nothing is folded, so a cell used by an instruction
// costs one more opcode than today; the suite measures that against the
// branches the run loop loses.
func LoadStoreCells(c Code) Code {
	regs := c.Registers
	e := NewEdits(c)
	for i, instr := range c.Instructions {
		switch x := instr.(type) {
		case ClearReg, TakeRegister, ReleaseRegister, DeclareLabel:
			continue
		case MkClosure:
			if regs[x.Dst].IsCell {
				t := e.NewRegister(RegData{})
				e.Replace(i, x.WithDestReg(t))
				e.InsertBefore(i+1, Transform{Op: ops.OpId, Dst: x.Dst, Src: t})
			}
			continue
		}
		if _, ok := asMove(instr); ok {
			continue
		}
		temps := map[Register]Register{}
		temp := func(r Register) Register {
			t, ok := temps[r]
			if !ok {
				t = e.NewRegister(RegData{})
				temps[r] = t
			}
			return t
		}
		for _, r := range instr.Reads() {
			if _, done := temps[r]; done || !regs[r].IsCell {
				continue
			}
			t := temp(r)
			e.InsertBefore(i, Transform{Op: ops.OpId, Dst: t, Src: r})
			instr = instr.ReplaceRead(r, t)
		}
		stored := map[Register]bool{}
		for _, w := range instr.Writes() {
			if stored[w] || !regs[w].IsCell {
				continue
			}
			stored[w] = true
			t := temp(w)
			instr = renameWrite(instr, w, t)
			e.InsertBefore(i+1, Transform{Op: ops.OpId, Dst: w, Src: t})
		}
		if len(temps) > 0 {
			e.Replace(i, instr)
		}
	}
	return e.Apply(c)
}

// renameWrite returns instr writing to instead of from.
func renameWrite(instr Instruction, from, to Register) Instruction {
	switch x := instr.(type) {
	case Receive:
		x.Dst = subAll(x.Dst, from, to)
		return x
	case ReceiveEtc:
		x.Dst = subAll(x.Dst, from, to)
		x.Etc = sub(x.Etc, from, to)
		return x
	case MkVarargTable:
		x.Dst = sub(x.Dst, from, to)
		x.Etc = sub(x.Etc, from, to)
		return x
	case MkTable:
		x.Dst = sub(x.Dst, from, to)
		return x
	case SetRegInstruction:
		if x.DestReg() == from {
			return x.WithDestReg(to)
		}
	}
	panic(fmt.Sprintf("cannot rename the write of %s in %s", from, instr))
}

func subAll(rs []Register, from, to Register) []Register {
	out := make([]Register, len(rs))
	for i, r := range rs {
		out[i] = sub(r, from, to)
	}
	return out
}
