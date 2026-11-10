package ir

import (
	"fmt"
	"slices"

	"github.com/arnodel/golua/ops"
)

// LoadStoreCells makes the move the only instruction that reads or writes
// the contents of a cell.  Every other instruction with a cell operand gets
// a plain register in its place: one loaded from the cell before it for a
// read, one stored into the cell after it for a write.  MkClosure captures
// cells themselves, not their contents, and ClearReg makes a new one, so
// both keep their cell operands.
//
// With this pass, the run loop needs to know about cells only in the move
// and upval opcodes, and every other register access is a plain index.
// This is the run-time half of notes/cells.md.
//
// Two rules keep the moves down:
//
//   - A fixed upvalue (IsFixed: nothing changes its contents once a closure
//     can see them, so nothing changes them while the function runs) is
//     loaded once, at the function's entry, into a register that stands for
//     it throughout.  The code generator makes that load part of making the
//     frame (code.EntryLoad), so it costs no opcode.
//   - Within a straight-line stretch, a register known to hold a cell's
//     contents, from a load or a store, stands for the cell until that
//     register is written, or, for a cell that is not fixed, until an
//     instruction that may run code (a call, an operation with a possible
//     metamethod, a __close), which could change the cell.
func LoadStoreCells(c Code) Code {
	regs := c.Registers
	isCell := func(r Register) bool { return regs[r].IsCell }
	fixed := func(r Register) bool { return regs[r].IsFixed }
	e := NewEdits(c)

	// Fixed upvalues whose contents the function reads are loaded at entry,
	// before anything else, the parameters' recv included: the frame does
	// these loads when it is made, so their registers must be ones the recv
	// does not write, which this order ensures.
	readsContents := map[Register]bool{}
	for _, instr := range c.Instructions {
		if _, ok := instr.(MkClosure); ok {
			continue
		}
		for _, r := range instr.Reads() {
			readsContents[r] = true
		}
	}
	entry := map[Register]Register{}
	for _, u := range c.UpvalueDests {
		if fixed(u) && readsContents[u] {
			t := e.NewRegister(RegData{})
			e.InsertBefore(0, mv(t, u))
			entry[u] = t
		}
	}

	// known maps a cell to a register holding its contents, in the current
	// stretch.
	known := map[Register]Register{}
	holding := func(u Register) (Register, bool) {
		if t, ok := entry[u]; ok {
			return t, true
		}
		t, ok := known[u]
		return t, ok
	}
	written := func(w Register) { // w no longer holds what it held
		for u, t := range known {
			if t == w {
				delete(known, u)
			}
		}
	}
	ranCode := func() { // any cell but a fixed one may have changed
		for u := range known {
			if !fixed(u) {
				delete(known, u)
			}
		}
	}

	for i, instr := range c.Instructions {
		switch x := instr.(type) {
		case DeclareLabel:
			known = map[Register]Register{}
			continue
		case TakeRegister, ReleaseRegister:
			continue
		case ClearReg:
			delete(known, x.Dst)
			written(x.Dst)
			continue
		case MkClosure:
			if !isCell(x.Dst) {
				written(x.Dst)
				continue
			}
			t := e.NewRegister(RegData{})
			e.Replace(i, x.WithDestReg(t))
			e.InsertBefore(i+1, mv(x.Dst, t))
			known[x.Dst] = t
			continue
		}
		if m, ok := asMove(instr); ok {
			src, dst := m.Src, m.Dst
			switch {
			case isCell(src) && isCell(dst):
				if t, ok := holding(src); ok {
					e.Replace(i, mv(dst, t))
					known[dst] = t
				} else {
					delete(known, dst)
				}
				continue
			case isCell(src): // a load
				written(dst)
				if t, ok := holding(src); ok {
					e.Replace(i, mv(dst, t))
				} else {
					known[src] = dst
				}
				continue
			case isCell(dst): // a store
				known[dst] = src
				continue
			}
		}
		reads, writes := instr.Reads(), instr.Writes()
		temps := map[Register]Register{}
		changed := false
		for _, r := range reads {
			if _, done := temps[r]; done || !isCell(r) {
				continue
			}
			// A cell the instruction also writes gets a register of its own
			// for both, as the operand is one and the same.
			t, ok := holding(r)
			if !ok || slices.Contains(writes, r) {
				t = e.NewRegister(RegData{})
				e.InsertBefore(i, mv(t, r))
				if !slices.Contains(writes, r) {
					known[r] = t
				}
			}
			temps[r] = t
			instr = instr.ReplaceRead(r, t)
			changed = true
		}
		for _, w := range writes {
			if !isCell(w) {
				written(w)
			}
		}
		if !pure(instr) {
			ranCode()
		}
		stored := map[Register]bool{}
		for _, w := range writes {
			if stored[w] || !isCell(w) {
				continue
			}
			stored[w] = true
			t, ok := temps[w]
			if !ok {
				t = e.NewRegister(RegData{})
			}
			instr = renameWrite(instr, w, t)
			e.InsertBefore(i+1, mv(w, t))
			known[w] = t
			changed = true
		}
		if changed {
			e.Replace(i, instr)
		}
	}
	return e.Apply(c)
}

func mv(dst, src Register) Instruction { return Transform{Op: ops.OpId, Dst: dst, Src: src} }

// pure says whether instr cannot run Lua code: no call, no metamethod, no
// __close.  Only such an instruction leaves every cell as it was.
func pure(instr Instruction) bool {
	switch x := instr.(type) {
	case LoadConst, MkTable, MkClosure, MkCont, Push, Receive, ReceiveEtc,
		Jump, JumpIf, EtcLookup, FillTable, MkVarargTable, PushCloseStack,
		ClearReg, TakeRegister, ReleaseRegister, DeclareLabel:
		return true
	case Transform:
		return x.Op == ops.OpId || x.Op == ops.OpNot
	}
	return false
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
