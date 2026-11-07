package ir

import (
	"maps"
	"slices"
)

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
// before the outermost loop around it that fits the limits above, in one of
// two ways.  When the loop has one entry and nothing jumps into it
// conditionally, on the edges: just before the jump into the loop, or at
// the end of the block that falls into it, for each block outside the loop
// that leads to the entry.  Otherwise, at the top of each entry block under
// a new label, which the jumps back from inside the loop are moved to; that
// covers a loop entered by a conditional jump (a for loop with no iteration
// skips to the statement after it, which may be the next loop) and a loop
// that goto gives two entries, but not a loop whose entry a block inside
// falls into, as a while loop's body falls into its test.  The constants of
// one loop share a register when they are the same.
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

	// How the loads for a loop get in front of it.  On the edges when the
	// loop has one entry and nothing outside jumps into it conditionally: just
	// before the jump into the loop, or at the end of the block that falls
	// into it.  Otherwise by splitting each entry: the loads at its top, under
	// a new label that the jumps back from inside the loop are moved to, so
	// that every way in from outside passes the loads once and the loop
	// itself skips them.  Splitting is impossible when a block inside the
	// loop falls into the entry (a while loop's body falls into its test).
	type plan struct {
		ok     bool
		edges  []int // positions to insert before
		splits []int // entry blocks to split
	}
	plans := map[int]*plan{}
	planFor := func(l int) *plan {
		if p, ok := plans[l]; ok {
			return p
		}
		p := &plan{}
		plans[l] = p
		loop := g.Loops[l]
		if loopLen(l) > hoistMaxLoop {
			return p
		}
		inside := func(b int) bool { return slices.Contains(loop.Blocks, b) }
		if len(loop.Entries) == 1 {
			e := loop.Entries[0]
			if e == 0 {
				p.edges = append(p.edges, 0)
			}
			p.ok = true
			for _, q := range g.Blocks[e].Preds {
				if inside(q) || !g.Reachable(q) {
					continue
				}
				blk := g.Blocks[q]
				lbl, jumps, next := controlFlow(instrs[blk.End-1])
				switch {
				case jumps && g.labelBlock(lbl, instrs) == e && !next:
					p.edges = append(p.edges, blk.End-1)
				case !jumps || g.labelBlock(lbl, instrs) != e:
					p.edges = append(p.edges, blk.End)
				default:
					p.ok = false // a conditional jump into the loop
				}
			}
			if p.ok {
				return p
			}
			p.edges = nil
		}
		for _, e := range loop.Entries {
			for _, q := range g.Blocks[e].Preds {
				if inside(q) && q+1 == e {
					if _, _, next := controlFlow(instrs[g.Blocks[q].End-1]); next {
						return p // falls into the entry from inside
					}
				}
			}
		}
		p.ok = true
		p.splits = loop.Entries
		return p
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
			if !planFor(l).ok {
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
		}
		rename[load.Dst] = r
		deleted[i] = true
	}
	if len(rename) == 0 {
		return c
	}

	// Place the loads.  Loops that split the same entry put their loads
	// under one new label, which every jump back to that entry from inside
	// any of them is moved to: a hoisted register is never written again, so
	// no jump back needs to pass the loads.
	nextLabel := Label(0)
	for _, instr := range instrs {
		if l, ok := instr.(DeclareLabel); ok && l.Label >= nextLabel {
			nextLabel = l.Label + 1
		}
	}
	loadsOf := func(l int) []Instruction {
		var loads []Instruction
		for kidx, r := range hoisted[l] {
			loads = append(loads, LoadConst{Dst: r, Kidx: kidx})
		}
		slices.SortFunc(loads, func(a, b Instruction) int {
			return int(a.(LoadConst).Dst) - int(b.(LoadConst).Dst)
		})
		return loads
	}
	splitLoops := map[int][]int{} // entry block -> loops split there
	for l := range g.Loops {
		if hoisted[l] == nil {
			continue
		}
		p := planFor(l)
		for _, pos := range p.edges {
			for _, in := range loadsOf(l) {
				if !slices.Contains(inserts[pos], in) {
					inserts[pos] = append(inserts[pos], in)
				}
			}
		}
		for _, e := range p.splits {
			splitLoops[e] = append(splitLoops[e], l)
		}
	}
	retarget := map[int]Label{} // jump instruction -> its new target
	for _, e := range slices.Sorted(maps.Keys(splitLoops)) {
		blk := g.Blocks[e]
		pos := blk.Start
		var labels []Label
		for pos < blk.End && isLabel(instrs[pos]) {
			labels = append(labels, instrs[pos].(DeclareLabel).Label)
			pos++
		}
		for _, l := range splitLoops[e] {
			inserts[pos] = append(inserts[pos], loadsOf(l)...)
		}
		inserts[pos] = append(inserts[pos], DeclareLabel{Label: nextLabel})
		for _, l := range splitLoops[e] {
			for _, b := range g.Loops[l].Blocks {
				last := g.Blocks[b].End - 1
				if lbl, jumps, _ := controlFlow(instrs[last]); jumps && slices.Contains(labels, lbl) {
					retarget[last] = nextLabel
				}
			}
		}
		nextLabel++
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
		if to, ok := retarget[i]; ok {
			switch x := instr.(type) {
			case Jump:
				x.Label = to
				instr = x
			case JumpIf:
				x.Label = to
				instr = x
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
