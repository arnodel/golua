package ir

import "slices"

// Limits on hoisting, so that a big function still gets its inner loops
// done without the hoisted registers piling up: a loop longer than
// hoistMaxLoop instructions is left alone, and so is a constant once a loop
// has hoistMaxConsts of them.
const (
	hoistMaxLoop   = 300
	hoistMaxConsts = 24
)

// HoistConstants moves the loading of constants out of loops.  A constant
// used inside a loop is loaded each time round:
//
//	for i = 1, n do x = x + 1 end        L1: r6 <- 1; x <- x + r6; ...; jump L1
//
// After hoisting it is loaded once, before the loop, into a register of its
// own that the loop reads:
//
//	r7 <- 1; L1: x <- x + r7; ...; jump L1
//
// A load is hoisted if its destination is a temporary that nothing else
// writes, so that every read of the temporary reads this constant.  It goes
// before the outermost loop around it that fits the limits above and has a
// single entry: just before the jump into the loop, or at the end of the
// block that falls into it, for each block outside the loop that leads to
// the entry.  A loop with two entries (goto can make one) is left alone,
// and so is one entered by a conditional jump: there is no place before it
// that only the loop's paths go through.  The constants of one loop share a
// register when they are the same.
//
// Register lifetimes are computed afterwards (ComputeLifetimes), so the new
// registers need no take or release here.  The new loads carry no line:
// they stand for no statement, and line hooks should not see one.
func HoistConstants(c Code) Code {
	instrs := c.Instructions
	g := NewGraph(instrs)
	if len(g.Loops) == 0 {
		return c
	}
	writes := make([]int, len(c.Registers))
	for _, instr := range instrs {
		for _, r := range instr.Writes() {
			writes[r]++
		}
	}
	blockOf := make([]int, len(instrs))
	for b, blk := range g.Blocks {
		for i := blk.Start; i < blk.End; i++ {
			blockOf[i] = b
		}
	}
	loopLen := func(l int) int {
		n := 0
		for _, b := range g.Loops[l].Blocks {
			n += g.Blocks[b].End - g.Blocks[b].Start
		}
		return n
	}

	// Where the loads for a loop go: the positions (instruction indices) to
	// insert before, or nil if the loop can't be hoisted into.
	entryPoints := map[int][]int{}
	points := func(l int) []int {
		if ps, ok := entryPoints[l]; ok {
			return ps
		}
		var ps []int
		loop := g.Loops[l]
		if len(loop.Entries) != 1 || loopLen(l) > hoistMaxLoop {
			entryPoints[l] = nil
			return nil
		}
		e := loop.Entries[0]
		if e == 0 {
			ps = append(ps, 0)
		}
		for _, p := range g.Blocks[e].Preds {
			if slices.Contains(loop.Blocks, p) || !g.Reachable(p) {
				continue
			}
			blk := g.Blocks[p]
			lbl, jumps, next := controlFlow(instrs[blk.End-1])
			switch {
			case jumps && g.labelBlock(lbl, instrs) == e && !next:
				ps = append(ps, blk.End-1) // before the jump into the loop
			case !jumps || g.labelBlock(lbl, instrs) != e:
				ps = append(ps, blk.End) // falls into the loop
			default:
				ps = nil // a conditional jump into the loop
				entryPoints[l] = nil
				return nil
			}
		}
		entryPoints[l] = ps
		return ps
	}

	regs := slices.Clone(c.Registers)
	hoisted := map[int]map[uint]Register{} // loop -> constant -> register
	inserts := map[int][]Instruction{}     // position -> loads to insert before it
	deleted := make([]bool, len(instrs))
	rename := map[Register]Register{}
	for i, instr := range instrs {
		load, ok := instr.(LoadConst)
		if !ok || regs[load.Dst].IsCell || regs[load.Dst].IsLocal || writes[load.Dst] != 1 {
			continue
		}
		// The outermost loop around the load that can be hoisted into.
		target := -1
		for l := g.Blocks[blockOf[i]].Loop; l >= 0; l = g.Loops[l].Parent {
			if points(l) == nil {
				continue
			}
			if _, done := hoisted[l][load.Kidx]; done || len(hoisted[l]) < hoistMaxConsts {
				target = l
			}
		}
		if target < 0 {
			continue
		}
		if hoisted[target] == nil {
			hoisted[target] = map[uint]Register{}
		}
		r, ok := hoisted[target][load.Kidx]
		if !ok {
			r = Register(len(regs))
			regs = append(regs, RegData{})
			hoisted[target][load.Kidx] = r
			for _, p := range points(target) {
				in := LoadConst{Dst: r, Kidx: load.Kidx}
				if !slices.Contains(inserts[p], Instruction(in)) {
					inserts[p] = append(inserts[p], in)
				}
			}
		}
		rename[load.Dst] = r
		deleted[i] = true
	}
	if len(rename) == 0 {
		return c
	}

	out := make([]Instruction, 0, len(instrs)+len(inserts))
	lines := make([]int, 0, len(instrs)+len(inserts))
	for i := 0; i <= len(instrs); i++ {
		for _, in := range inserts[i] {
			out = append(out, in)
			lines = append(lines, 0)
		}
		if i == len(instrs) || deleted[i] {
			continue
		}
		instr := instrs[i]
		for _, r := range instr.Reads() {
			if to, ok := rename[r]; ok {
				instr = instr.ReplaceRead(r, to)
			}
		}
		out = append(out, instr)
		lines = append(lines, c.Lines[i])
	}
	c.Instructions = out
	c.Lines = lines
	c.Registers = regs
	return c
}

// labelBlock returns the block that starts at lbl.
func (g *Graph) labelBlock(lbl Label, instrs []Instruction) int {
	for b, blk := range g.Blocks {
		for i := blk.Start; i < blk.End; i++ {
			if l, ok := instrs[i].(DeclareLabel); ok && l.Label == lbl {
				return b
			}
			if !isLabel(instrs[i]) {
				break
			}
		}
	}
	return -1
}
