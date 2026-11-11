package ir

import "slices"

// markFixed returns consts with IsFixed set on each register whose value is
// fixed by the time anything can see it: no write to the register can come
// after its value is seen, unless a clear comes in between.  Reading such a
// register later reads the same value as reading it earlier, so a pass may
// read it in place of a copy of it.  A clear ends what the register held (it
// starts a new cell, or drops a value no longer needed), so a read must not
// be moved past a clear.
//
// For a plain register, what sees it is its function's instructions that
// read it.  So in
//
//	r1 <- nil; r1 <- K1; push r2, r1
//
// r1 is fixed: nothing sees the nil.  In
//
//	r1 <- K1; push r2, r1; r1 <- K2
//
// it is not.  Neither is r1 in "r1 <- r1 + r2", nor a register written in a
// loop and read in it without a clear between the two, as the read sees the
// value of the previous time round.
//
// A cell (a local captured by a closure, or an upvalue) is shared by the
// function that makes it and every closure that captures it, so what sees
// or writes it there counts too:
//
//   - a closure that captures the cell may read it whenever it runs, so the
//     capture sees it from then on;
//   - a write by a closure to its upvalue changes the cell;
//   - an upvalue has its value when the function starts, so the function's
//     entry sees it.
//
// A closure stored in a cell it captures (a local function, recursive or
// not) cannot run before it is stored, so making it sees the cell after
// writing it.  So the cells of
//
//	local band, bxor
//	if bit then band, bxor = bit.band, bit.bxor else band, bxor = ... end
//	local function rol(a, b) ... end
//
// are all fixed, in the main chunk and as upvalues of the functions that
// use them.  Not so for n in
//
//	local n = 0
//	local function inc() n = n + 1 end
//
// as inc writes it.  The debug library can change any local or upvalue
// (debug.setlocal, debug.setupvalue, debug.upvaluejoin), which the manual
// allows to break such assumptions (CONFORMANCE.md, GOLUA-046).
//
// Copy propagation alone would need less: that no closure writes the cell.
// The rule here is stronger so that a fixed cell can also be read later than
// where the source reads it, with nothing able to change it in between: a
// call reads a fixed callee at the call rather than before its arguments
// (notes/call-protocol.md, step 3), and a frame can load a fixed upvalue
// once, when it is made (notes/cells.md).
func markFixed(consts []Constant) []Constant {
	// Changes found in each function, then shared between the registers of
	// a cell: a capture links the cell in the function that makes the
	// closure to the upvalue in the closure, and union-find gathers them.
	type reg struct {
		code *Code
		r    Register
	}
	ids := map[reg]int{}
	var parent []int
	id := func(x reg) int {
		i, ok := ids[x]
		if !ok {
			i = len(parent)
			ids[x] = i
			parent = append(parent, i)
		}
		return i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	changes := map[*Code]RegSet{}
	for _, k := range consts {
		c, ok := k.(*Code)
		if !ok {
			continue
		}
		changes[c] = changedAfterSeen(c)
		for _, instr := range c.Instructions {
			if m, ok := instr.(MkClosure); ok {
				child := consts[m.Code].(*Code)
				for i, r := range m.Upvalues {
					// Both ids may grow parent, so neither side of the
					// assignment may index it before they are done.
					a := find(id(reg{c, r}))
					b := find(id(reg{child, child.UpvalueDests[i]}))
					parent[a] = b
				}
			}
		}
	}
	changedCell := map[int]bool{}
	for x := range ids {
		if changes[x.code].Has(x.r) {
			changedCell[find(id(x))] = true
		}
	}

	out := make([]Constant, len(consts))
	for i, k := range consts {
		c, ok := k.(*Code)
		if !ok {
			out[i] = k
			continue
		}
		oc := *c
		oc.Registers = slices.Clone(c.Registers)
		for r := range oc.Registers {
			fixed := !changes[c].Has(Register(r))
			if i, linked := ids[reg{c, Register(r)}]; linked && changedCell[find(i)] {
				fixed = false
			}
			oc.Registers[r].IsFixed = fixed
		}
		out[i] = &oc
	}
	return out
}

// changedAfterSeen returns the registers of c that some instruction of c
// writes after their value can be seen (see markFixed).  It is a forward
// dataflow over the control-flow graph: the state is the set of registers
// whose value may have been seen since they were last cleared, and a write
// to one of them is a change.  Upvalues are seen from the start.
func changedAfterSeen(c *Code) RegSet {
	n := len(c.Registers)
	changed := NewRegSet(n)
	instrs := c.Instructions
	step := func(seen RegSet, instr Instruction) {
		if cl, ok := instr.(ClearReg); ok {
			seen.Remove(cl.Dst)
			return
		}
		see := func() {
			for _, r := range instr.Reads() {
				seen.Add(r)
			}
		}
		// An instruction reads its operands before it writes, except that a
		// closure is stored before it can run and see what it captures.
		_, closure := instr.(MkClosure)
		if !closure {
			see()
		}
		for _, w := range instr.Writes() {
			if seen.Has(w) {
				changed.Add(w)
			}
		}
		if closure {
			see()
		}
	}
	if len(instrs) == 0 {
		return changed
	}
	g := NewGraph(instrs)
	in := make([]RegSet, len(g.Blocks))
	for b := range in {
		in[b] = NewRegSet(n)
	}
	for _, r := range c.UpvalueDests {
		in[0].Add(r)
	}
	seen := NewRegSet(n)
	for again := true; again; {
		again = false
		for _, b := range g.Order {
			copy(seen, in[b])
			for _, instr := range instrs[g.Blocks[b].Start:g.Blocks[b].End] {
				step(seen, instr)
			}
			for _, s := range g.Blocks[b].Succs {
				for i, w := range seen {
					if in[s][i]|w != in[s][i] {
						in[s][i] |= w
						again = true
					}
				}
			}
		}
	}
	return changed
}
