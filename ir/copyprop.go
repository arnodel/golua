package ir

import (
	"slices"

	"github.com/arnodel/golua/ops"
)

// PropagateCopies removes register copies from a function's code.  The AST
// compiler copies values into fresh registers freely, so that it never has to
// reason about what else may change them; this pass reasons about it
// afterwards, within each straight-line stretch of code (between labels and
// jumps), where it can see every read and write.  Two rewrites:
//
//  1. Reading through a copy.  After "r2 <- r1", a read of r2 becomes a read
//     of r1, as long as neither has been written since.  If r2 is a
//     temporary that nobody then reads, the copy is removed.  With q, i and
//     j locals, "q[i], q[j] = q[j], q[i]" compiles to
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
// Cells (locals captured by closures, and upvalues) are left alone: a function
// called in between could change them.  The exception is a fixed cell (see
// markFixed), which nothing changes once its value can be seen: a copy of it
// is read through like a copy of a named local.  An argument listed in a
// call is read at the call, so the AST compiler copies a cell argument before
// evaluating the next ones, which could change it; with n fixed,
// "f(n, g() + 1)" goes from
//
//	r1 <- cont(u0); r2 <- u1; ...g() + 1 into r3...; call r1 (r2, r3)
//
// to
//
//	r1 <- cont(u0); ...g() + 1 into r2...; call r1 (u1, r2)
//
// Named locals keep their copies when they are the destination, so that
// their value stays observable.
//
// Only copies of named locals and of fixed cells are read through.  A local is live for its
// whole scope, so reading it instead of its copy costs nothing and lets the
// copy die early; reading a temporary instead of its copy would keep the
// temporary alive for longer.  Reading a register later than before is
// always safe: the code generator computes register lifetimes itself.
//
// Registers are never reused for another purpose in a function, so a register
// means the same thing throughout, and the counts of its reads and writes
// over the whole function are exact.
func PropagateCopies(c Code) Code {
	instrs := c.Instructions
	regs := c.Registers
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
	local := func(r Register) bool { return !regs[r].IsCell && regs[r].IsLocal }
	fixedCell := func(r Register) bool { return regs[r].IsCell && regs[r].IsFixed }
	e := NewEdits(c)

	// State for the current straight-line stretch.
	var (
		copies    map[Register]Register // destination -> source of a copy still valid
		lastDef   map[Register]int      // last instruction writing the register (alone)
		lastTouch map[Register]int      // last instruction reading or writing the register
	)
	reset := func() {
		copies = map[Register]Register{}
		lastDef = map[Register]int{}
		lastTouch = map[Register]int{}
	}
	reset()
	invalidate := func(w Register) {
		delete(copies, w)
		for d, s := range copies {
			if s == w {
				delete(copies, d)
			}
		}
	}

	for i, instr := range instrs {
		if _, ok := instr.(DeclareLabel); ok {
			reset() // a join point: nothing known about what came before
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

		// 2. A move of a temporary with a single definition in this stretch
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
		if mv, ok := asMove(instr); ok && (local(mv.Src) || fixedCell(mv.Src)) && plain(mv.Dst) && mv.Src != mv.Dst {
			copies[mv.Dst] = mv.Src
		}

		// The end of a straight-line stretch.
		switch x := instr.(type) {
		case Jump, JumpIf:
			reset()
		case Call:
			if x.Tail {
				reset()
			}
		}
	}

	// 4. A copy into a temporary that is never read does nothing.
	for i := range instrs {
		if mv, ok := asMove(e.Current(i)); ok && !e.Deleted(i) && temp(mv.Dst) && reads[mv.Dst] == 0 {
			e.Delete(i)
			reads[mv.Src]--
		}
	}
	return e.Apply(c)
}

// asMove returns the instruction as a register copy, if it is one.
func asMove(instr Instruction) (Transform, bool) {
	t, ok := instr.(Transform)
	return t, ok && t.Op == ops.OpId
}
