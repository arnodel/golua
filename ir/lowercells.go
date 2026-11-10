package ir

import (
	"fmt"
	"slices"

	"github.com/arnodel/golua/ops"
)

// LowerCells makes a cell register a register holding a box, whose contents
// only Load, Store and NewBox touch (notes/cells.md).  Every instruction
// with a cell operand gets a plain register in its place: one loaded from
// the box before it for a read, one stored into the box after it for a
// write.  MkClosure captures boxes themselves, not their contents, so its
// upvalue operands stay.
//
// A local of the function's own gets its box where it is declared: a NewBox
// before the local's first mention, which is the write that declares it (a
// local without an initialiser is set to nil), and which every use of the
// local comes after.  A local declared in a loop gets a new box each time
// round, so the clears the compiler emits when a scope ends are dropped.
// An upvalue's box is put in its register when the frame is made.
//
// After this pass every other register access is a plain index, and the
// run loop needs to know about boxes only in the opcodes these three
// instructions compile to.
//
// Two rules keep the loads and stores down:
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
func LowerCells(c Code) Code {
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
	upvalue := map[Register]bool{}
	for _, u := range c.UpvalueDests {
		upvalue[u] = true
		if fixed(u) && readsContents[u] {
			t := e.NewRegister(RegData{})
			e.InsertBefore(0, Load{Dst: t, Box: u})
			entry[u] = t
		}
	}

	// The boxes of the function's own cells.  A captured parameter's first
	// mention is the recv, which must stay the first opcode (a call stores
	// an argument straight into the register the callee's recv names), so
	// its box is made right after the parameters are received, before the
	// store into it; so is the box of a cell whose first mention is not a
	// write, which comes before everything else.
	afterParams := 0
	for afterParams < len(c.Instructions) && receives(c.Instructions[afterParams]) {
		afterParams++
	}
	boxed := map[Register]bool{}
	for i, instr := range c.Instructions {
		writes := instr.Writes()
		for _, r := range slices.Concat(writes, instr.Reads()) {
			if !isCell(r) || upvalue[r] || boxed[r] {
				continue
			}
			boxed[r] = true
			if slices.Contains(writes, r) && !receives(instr) {
				e.InsertBefore(i, NewBox{Dst: r})
			} else {
				e.InsertBefore(afterParams, NewBox{Dst: r})
			}
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
			if isCell(x.Dst) {
				e.Delete(i)
			}
			continue
		case MkClosure:
			if !isCell(x.Dst) {
				written(x.Dst)
				continue
			}
			t := e.NewRegister(RegData{})
			e.Replace(i, x.WithDestReg(t))
			e.InsertBefore(i+1, Store{Box: x.Dst, Src: t})
			known[x.Dst] = t
			continue
		}
		if m, ok := asMove(instr); ok {
			src, dst := m.Src, m.Dst
			switch {
			case isCell(src) && isCell(dst):
				t, ok := holding(src)
				if !ok {
					t = e.NewRegister(RegData{})
					e.InsertBefore(i, Load{Dst: t, Box: src})
					known[src] = t
				}
				e.Replace(i, Store{Box: dst, Src: t})
				known[dst] = t
				continue
			case isCell(src): // a load
				written(dst)
				if t, ok := holding(src); ok {
					e.Replace(i, mv(dst, t))
				} else {
					e.Replace(i, Load{Dst: dst, Box: src})
					known[src] = dst
				}
				continue
			case isCell(dst): // a store
				e.Replace(i, Store{Box: dst, Src: src})
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
				e.InsertBefore(i, Load{Dst: t, Box: r})
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
			e.InsertBefore(i+1, Store{Box: w, Src: t})
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

func receives(instr Instruction) bool {
	switch instr.(type) {
	case Receive, ReceiveEtc:
		return true
	}
	return false
}

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
