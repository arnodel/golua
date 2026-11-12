package ir

import (
	"slices"

	"github.com/arnodel/golua/ops"
)

// PropagateCopies removes register copies from a function's code.  The AST
// compiler copies values into fresh registers freely, so that it never has to
// reason about what else may change them; this pass reasons about it
// afterwards.  Two rewrites:
//
//  1. Reading through a copy.  After "r2 <- r1", a read of r2 becomes a read
//     of r1, as long as neither has been written since.  If r2 is then read
//     nowhere, the copy is removed.  With q, i and j locals,
//     "q[i], q[j] = q[j], q[i]" compiles to
//
//     r7 <- q[j]; r8 <- q[i]; r9 <- q; r10 <- i; r11 <- q; r12 <- j; r9[r10] <- r7; r11[r12] <- r8
//
//     and becomes
//
//     r7 <- q[j]; r8 <- q[i]; q[i] <- r7; q[j] <- r8
//
//  2. Writing straight to the destination.  "rT <- X; rL <- rT", where rT is
//     a temporary written and read nowhere else and rL is untouched between
//     the two, becomes "rL <- X".  So "i = i + 1" goes from
//
//     r6 <- 1; r6 <- i + r6; i <- r6
//
//     to
//
//     r6 <- 1; i <- i + r6
//
// Within a block (a straight-line stretch of code, between labels and
// jumps) the pass sees every read and write.  At the start of a block it
// knows the copies that are available: made on every path into the block and
// with neither register written since (availableCopies).  So the copy of
// the counter that a numeric for loop makes into its variable is read
// through all over the body, past the branches in it, and removed:
//
//	L1: i <- r1; r3 <- t[i]; if not r3 jump L2; r4 <- f(i); L2: advfor r1, r2, r5; if r1 jump L1
//
// becomes
//
//	L1: r3 <- t[r1]; if not r3 jump L2; r4 <- f(r1); L2: advfor r1, r2, r5; if r1 jump L1
//
// since advfor, the only other write to r1, comes after the body.  Writing
// straight to the destination stays within a block.
//
// Cells (locals captured by closures, and upvalues) are left alone: a function
// called in between could change them.  The exception is a fixed cell (see
// markFixed), which nothing changes once its value can be seen: a copy of it
// is read through like a copy of a named local.  A call reads its callee and
// its listed arguments at the call, so the AST compiler copies a captured
// one before evaluating the arguments, which could change it; with f and n
// fixed, "f(n, g() + 1)" goes from
//
//	r1 <- u0; r2 <- u1; ...g() + 1 into r3...; call r1 (r2, r3)
//
// to
//
//	...g() + 1 into r1...; call u0 (u1, r1)
//
// Only copies of named locals and of fixed cells are read through.  A local is
// live for its whole scope, so reading it instead of its copy costs nothing
// and lets the copy die early; reading a temporary instead of its copy would
// keep the temporary alive for longer.  Reading a register later than before
// is always safe: the code generator computes register lifetimes itself.
//
// A copy into a named local goes like any other once nothing reads the local
// (a for loop's variable, typically).  Error messages don't name locals and
// there is no debug.getlocal, so nothing observes the difference.
//
// Registers are never reused for another purpose in a function, so a register
// means the same thing throughout, and the counts of its reads and writes
// over the whole function are exact.
func PropagateCopies(c Code) Code {
	instrs := c.Instructions
	regs := c.Registers
	if len(instrs) == 0 {
		return c
	}
	reads := make([]int, len(regs))
	writes := make([]int, len(regs))
	for _, instr := range instrs {
		for _, r := range instr.Reads() {
			reads[r]++
		}
		for _, r := range instr.Writes() {
			writes[r]++
		}
	}
	plain := func(r Register) bool { return !regs[r].IsCell }
	temp := func(r Register) bool { return !regs[r].IsCell && !regs[r].IsLocal }
	e := NewEdits(c)

	a := analyse(c)
	avail := availableCopies(a)

	// State for the current block.
	var (
		block     = -1
		copies    map[Register]Register // destination -> source of a copy still valid
		lastDef   map[Register]int      // last instruction writing the register (alone)
		lastTouch map[Register]int      // last instruction reading or writing the register
	)
	invalidate := func(w Register) {
		delete(copies, w)
		for d, s := range copies {
			if s == w {
				delete(copies, d)
			}
		}
	}

	for i, instr := range instrs {
		if b := a.blockOf[i]; b != block {
			block = b
			copies = avail.at(b)
			lastDef = map[Register]int{}
			lastTouch = map[Register]int{}
		}
		if _, ok := instr.(DeclareLabel); ok {
			continue
		}

		// 1. Read through the copies still valid.  A register the instruction
		// also writes (a for loop's control registers) is left alone: the
		// write must go to the same register as the read.
		writes_ := instr.Writes()
		for _, r := range instr.Reads() {
			if s, ok := copies[r]; ok && !slices.Contains(writes_, r) {
				e.ReplaceRead(i, r, s)
				reads[r]--
				reads[s]++
			}
		}
		instr = e.Current(i)

		// 2. A move of a temporary with a single definition in this block
		// into a register untouched since: write there directly.
		if mv, ok := asMove(instr); ok && temp(mv.Src) && plain(mv.Dst) && writes[mv.Src] == 1 && reads[mv.Src] == 1 {
			if d, ok := lastDef[mv.Src]; ok {
				if t, touched := lastTouch[mv.Dst]; !touched || t <= d {
					if def, ok := e.Current(d).(SetRegInstruction); ok {
						e.Replace(d, def.WithDestReg(mv.Dst))
						e.Delete(i)
						writes[mv.Src]--
						reads[mv.Src]--
						delete(lastDef, mv.Src)
						invalidate(mv.Dst)
						lastDef[mv.Dst] = d
						lastTouch[mv.Dst] = i
						continue
					}
				}
			}
		}

		// Writes end what was known about the register.
		for _, w := range instr.Writes() {
			invalidate(w)
			lastTouch[w] = i
			delete(lastDef, w)
		}
		if ws := instr.Writes(); len(ws) == 1 {
			if _, ok := instr.(SetRegInstruction); ok {
				lastDef[ws[0]] = i
			}
		}
		for _, r := range instr.Reads() {
			lastTouch[r] = i
		}

		// 3. Remember a copy of a named local or of a fixed cell.
		if mv, ok := asMove(instr); ok && isCopy(regs, mv) {
			copies[mv.Dst] = mv.Src
		}
	}

	// 4. A copy into a register that is never read does nothing.
	for i := range instrs {
		if mv, ok := asMove(e.Current(i)); ok && !e.Deleted(i) && plain(mv.Dst) && reads[mv.Dst] == 0 {
			e.Delete(i)
			reads[mv.Src]--
		}
	}
	return e.Apply(c)
}

// isCopy reports whether the move is a copy that may be read through: of a
// named local or a fixed cell, into a plain register.
func isCopy(regs []RegData, mv Transform) bool {
	src, dst := regs[mv.Src], regs[mv.Dst]
	return mv.Src != mv.Dst && !dst.IsCell && (src.IsCell && src.IsFixed || !src.IsCell && src.IsLocal)
}

// copies are the copies (as pairs of registers) a function's code makes that
// may be read through, and for each block the ones available at its start:
// made on every path into the block, with neither register written since.
type copies struct {
	pairs []Transform
	in    []RegSet // indexed by block, then by pair
}

// availableCopies works out the copies available at the start of each block,
// as a forward dataflow over the control-flow graph: each block kills the
// copies of a register it writes and makes those it copies, and a block has
// what all its predecessors have at their end.
func availableCopies(a *analysis) copies {
	regs := a.code.Registers
	var cs copies
	index := map[Transform]int{}
	involving := map[Register][]int{} // register -> pairs it is one side of
	for _, instr := range a.instrs {
		if mv, ok := asMove(instr); ok && isCopy(regs, mv) {
			if _, seen := index[mv]; !seen {
				index[mv] = len(cs.pairs)
				involving[mv.Src] = append(involving[mv.Src], len(cs.pairs))
				involving[mv.Dst] = append(involving[mv.Dst], len(cs.pairs))
				cs.pairs = append(cs.pairs, mv)
			}
		}
	}
	n := len(cs.pairs)
	g := a.g
	cs.in = make([]RegSet, len(g.Blocks))
	if n == 0 {
		return cs
	}

	// What each block does to the copies it is entered with.
	gen := make([]RegSet, len(g.Blocks))
	kill := make([]RegSet, len(g.Blocks))
	for b, blk := range g.Blocks {
		gen[b], kill[b] = NewRegSet(n), NewRegSet(n)
		for _, instr := range a.instrs[blk.Start:blk.End] {
			for _, w := range instr.Writes() {
				for _, p := range involving[w] {
					kill[b].Add(Register(p))
					gen[b].Remove(Register(p))
				}
			}
			if mv, ok := asMove(instr); ok {
				if p, ok := index[mv]; ok {
					gen[b].Add(Register(p))
				}
			}
		}
	}

	// Until nothing changes: what comes in is what every predecessor that
	// can run sends out.  A block starts with everything, so that a loop's
	// back edge doesn't take away what the loop makes on every path.
	all := NewRegSet(n)
	for p := range n {
		all.Add(Register(p))
	}
	out := func(b int) RegSet {
		s := slices.Clone(cs.in[b])
		for i := range s {
			s[i] = s[i]&^kill[b][i] | gen[b][i]
		}
		return s
	}
	for b := range cs.in {
		cs.in[b] = NewRegSet(n)
		if b != 0 && g.Reachable(b) {
			copy(cs.in[b], all)
		}
	}
	for again := true; again; {
		again = false
		for _, b := range g.Order[1:] {
			in := slices.Clone(all)
			for _, p := range g.Blocks[b].Preds {
				if !g.Reachable(p) {
					continue
				}
				for i, w := range out(p) {
					in[i] &= w
				}
			}
			if !slices.Equal(in, cs.in[b]) {
				cs.in[b] = in
				again = true
			}
		}
	}
	return cs
}

// at returns the copies available at the start of block b, as a map from
// destination to source.
func (cs copies) at(b int) map[Register]Register {
	m := map[Register]Register{}
	for p, mv := range cs.pairs {
		if cs.in[b].Has(Register(p)) {
			m[mv.Dst] = mv.Src
		}
	}
	return m
}

// asMove returns the instruction as a register copy, if it is one.
func asMove(instr Instruction) (Transform, bool) {
	t, ok := instr.(Transform)
	return t, ok && t.Op == ops.OpId
}
