package ircomp

import (
	"math"
	"slices"

	"github.com/arnodel/golua/ir"
)

// computeLifetimes gives the code generator the lifetime of every register of
// a function, as TakeRegister and ReleaseRegister instructions placed from
// liveness: the generator gives a register a slot in the frame only between
// its take and its release.  Lifetimes are the generator's business, so any
// markers already in the code are discarded.  Passes over the IR can
// therefore read a register later than before, or make a value live longer,
// without caring for them.
//
// A register's lifetime is a range of instructions, in the order of the
// code: from the first instruction that mentions it to the last, widened to
// every block it is live into or out of.  A value that is live where a loop
// jumps back is live over the whole loop, so its register stays taken from
// before the loop's first block to after its last, whatever the order of the
// jumps.  A register that is written but never read still needs its slot at
// the write.  A register read on some path before any write is live from the
// entry.
//
// An instruction reads its operands before it writes its destination, so a
// register read for the last time by an instruction is released before it,
// and the destination can take its slot: "r2 <- r1 == r2".  A register that
// is live when a block starts holds a value already, so it is taken before
// the block's first instruction releases anything.  In terms of positions, a
// read at instruction i is at 2i, a write at 2i+1, and a register is taken
// over the positions from its first to its last.
//
// Register 0 holds the caller's continuation: the frame puts it there before
// the function starts, and a tail call or tail continuation (Call or MkCont
// with Tail) reads it from the frame without naming it, so it is taken over
// the whole function,
// and first, which gives it slot 0.  Registers that receive upvalues are
// taken by ProcessCode before anything else and left alone here.
func computeLifetimes(c ir.Code) ir.Code {
	instrs := make([]ir.Instruction, 0, len(c.Instructions))
	lines := make([]int, 0, len(c.Lines))
	for i, instr := range c.Instructions {
		switch instr.(type) {
		case ir.TakeRegister, ir.ReleaseRegister:
			continue
		}
		instrs = append(instrs, instr)
		lines = append(lines, c.Lines[i])
	}

	// first and last are positions: 2i for a read by instruction i, 2i+1
	// for a write, 2·Start for a value live into a block, 2·End-1 for one
	// live out of it.
	n := len(c.Registers)
	first := make([]int, n)
	last := make([]int, n)
	for r := range first {
		first[r], last[r] = math.MaxInt, -1
	}
	at := func(r ir.Register, pos int) {
		first[r] = min(first[r], pos)
		last[r] = max(last[r], pos)
	}
	for i, instr := range instrs {
		for _, r := range instr.Reads() {
			at(r, 2*i)
		}
		for _, r := range instr.Writes() {
			at(r, 2*i+1)
		}
	}
	g := ir.NewGraph(instrs)
	liveIn, liveOut := g.Liveness(instrs, c.Registers)
	for b, blk := range g.Blocks {
		for r := ir.Register(0); r < ir.Register(n); r++ {
			if liveIn[b].Has(r) {
				at(r, 2*blk.Start)
			}
			if liveOut[b].Has(r) {
				at(r, 2*blk.End-1)
			}
		}
	}
	first[0], last[0] = 0, 2*len(instrs)-1
	for _, r := range c.UpvalueDests {
		last[r] = -1
	}

	out := make([]ir.Instruction, 0, len(instrs)+2*n)
	outLines := make([]int, 0, len(instrs)+2*n)
	emit := func(instr ir.Instruction, line int) {
		out = append(out, instr)
		outLines = append(outLines, line)
	}
	takes := func(pos int) {
		for r := ir.Register(0); r < ir.Register(n); r++ {
			if first[r] == pos && last[r] >= pos {
				emit(ir.TakeRegister{Reg: r}, 0)
			}
		}
	}
	releases := func(pos int) {
		for r := ir.Register(0); r < ir.Register(n); r++ {
			if last[r] == pos && first[r] <= pos {
				emit(ir.ReleaseRegister{Reg: r}, 0)
			}
		}
	}
	for i, instr := range instrs {
		takes(2 * i)    // live into the block, or read first
		releases(2 * i) // read for the last time
		takes(2*i + 1)  // written first
		emit(instr, lines[i])
		releases(2*i + 1) // written for the last time, or live out
	}
	c.Instructions = slices.Clip(out)
	c.Lines = slices.Clip(outLines)
	return c
}
