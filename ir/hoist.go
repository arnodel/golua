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
// The code generator computes register lifetimes, so the new registers need
// no take or release here.  The new loads carry no line: they stand for no
// statement, and line hooks should not see one.  A hoisted register is live
// over its whole loop, so a function that runs out of registers with
// hoisting is compiled again without it (runtime/lib.go).
//
// The work is a pipeline: planLoops says how each loop's loads can get in
// front of it, chooseLoads picks a loop for each load, placeLoads turns the
// choices into edits to the code, which are then applied.
func HoistConstants(c Code) Code {
	a := analyse(c)
	if len(a.g.Loops) == 0 {
		return c
	}
	plans := planLoops(a)
	choices := chooseLoads(a, plans)
	if len(choices.loads) == 0 {
		return c
	}
	return placeLoads(a, plans, choices).Apply(c)
}

// A hoistPlan says how the loads for a loop get in front of it: on the edges
// into it, or by splitting its entries; or that they can't.
type hoistPlan struct {
	ok     bool
	edges  []int // positions to insert before
	splits []int // entry blocks to split
}

// planLoops makes the plan of every loop.
func planLoops(a *analysis) []hoistPlan {
	plans := make([]hoistPlan, len(a.g.Loops))
	for l := range plans {
		plans[l] = planLoop(a, l)
	}
	return plans
}

func planLoop(a *analysis, l int) (p hoistPlan) {
	loop := a.g.Loops[l]
	if a.loopLen(l) > hoistMaxLoop {
		return p
	}
	inside := func(b int) bool { return slices.Contains(loop.Blocks, b) }

	// On the edges: the loop has one entry, and every block outside that
	// leads to it either jumps to it unconditionally or falls into it.
	if len(loop.Entries) == 1 {
		e := loop.Entries[0]
		if e == 0 {
			p.edges = append(p.edges, 0)
		}
		p.ok = true
		for _, q := range a.g.Blocks[e].Preds {
			if inside(q) || !a.g.Reachable(q) {
				continue
			}
			blk := a.g.Blocks[q]
			lbl, jumps, next := controlFlow(a.instrs[blk.End-1])
			switch {
			case jumps && a.g.labelBlock(lbl, a.instrs) == e && !next:
				p.edges = append(p.edges, blk.End-1) // before the jump in
			case !jumps || a.g.labelBlock(lbl, a.instrs) != e:
				p.edges = append(p.edges, blk.End) // falls in
			default:
				p.ok = false // a conditional jump into the loop
			}
		}
		if p.ok {
			return p
		}
		p.edges = nil
	}

	// By splitting the entries, unless a block inside falls into one.
	for _, e := range loop.Entries {
		for _, q := range a.g.Blocks[e].Preds {
			if inside(q) && a.fallsInto(q, e) {
				return p
			}
		}
	}
	p.ok = true
	p.splits = loop.Entries
	return p
}

// hoistChoices says which loads are hoisted and where.
type hoistChoices struct {
	loads   map[int]Register          // position of a hoisted load -> the register read instead
	byLoop  map[int]map[uint]Register // loop -> constant -> its register
	nregs   int                       // registers the code has
	newRegs int                       // registers added, numbered from nregs
}

// chooseLoads picks, for each load that can be hoisted, the outermost loop
// around it that has a plan and room for its constant, or already has it.
func chooseLoads(a *analysis, plans []hoistPlan) hoistChoices {
	ch := hoistChoices{
		loads:  map[int]Register{},
		byLoop: map[int]map[uint]Register{},
		nregs:  len(a.code.Registers),
	}
	for i, instr := range a.instrs {
		load, ok := instr.(LoadConst)
		if !ok {
			continue
		}
		if data := a.code.Registers[load.Dst]; data.IsCell || data.IsLocal || a.writes[load.Dst] != 1 {
			continue
		}
		target := -1
		for l := a.g.Blocks[a.blockOf[i]].Loop; l >= 0; l = a.g.Loops[l].Parent {
			if !plans[l].ok {
				continue
			}
			if _, done := ch.byLoop[l][load.Kidx]; done || len(ch.byLoop[l]) < hoistMaxConsts {
				target = l
			}
		}
		if target < 0 {
			continue
		}
		if ch.byLoop[target] == nil {
			ch.byLoop[target] = map[uint]Register{}
		}
		r, ok := ch.byLoop[target][load.Kidx]
		if !ok {
			r = Register(ch.nregs + ch.newRegs)
			ch.newRegs++
			ch.byLoop[target][load.Kidx] = r
		}
		ch.loads[i] = r
	}
	return ch
}

// loadsFor returns the loads of loop l's hoisted constants, in register
// order.
func (ch hoistChoices) loadsFor(l int) []Instruction {
	var loads []Instruction
	for kidx, r := range ch.byLoop[l] {
		loads = append(loads, LoadConst{Dst: r, Kidx: kidx})
	}
	slices.SortFunc(loads, func(a, b Instruction) int {
		return int(a.(LoadConst).Dst) - int(b.(LoadConst).Dst)
	})
	return loads
}

// placeLoads turns the choices into edits: the hoisted loads deleted and
// their destinations renamed, the new registers, and each loop's loads put
// where its plan says.  Loops that split the same entry put their loads
// under one new label, which every jump back to that entry from inside any
// of them is moved to: a hoisted register is never written again, so no
// jump back needs to pass the loads.
func placeLoads(a *analysis, plans []hoistPlan, ch hoistChoices) *Edits {
	e := NewEdits(a.code)
	for range ch.newRegs {
		e.NewRegister(RegData{})
	}
	for pos, r := range ch.loads {
		e.Delete(pos)
		e.RenameReads(a.instrs[pos].(LoadConst).Dst, r)
	}
	splitLoops := map[int][]int{} // entry block -> loops split there
	for l := range a.g.Loops {
		if ch.byLoop[l] == nil {
			continue
		}
		for _, pos := range plans[l].edges {
			for _, in := range ch.loadsFor(l) {
				if !slices.Contains(e.Inserted(pos), in) {
					e.InsertBefore(pos, in)
				}
			}
		}
		for _, entry := range plans[l].splits {
			splitLoops[entry] = append(splitLoops[entry], l)
		}
	}
	lbl := a.freeLabel()
	for _, entry := range slices.Sorted(maps.Keys(splitLoops)) {
		labels, pos := a.leadingLabels(entry)
		for _, l := range splitLoops[entry] {
			e.InsertBefore(pos, ch.loadsFor(l)...)
		}
		e.InsertBefore(pos, DeclareLabel{Label: lbl})
		for _, l := range splitLoops[entry] {
			for _, b := range a.g.Loops[l].Blocks {
				last := a.g.Blocks[b].End - 1
				if to, jumps, _ := controlFlow(a.instrs[last]); jumps && slices.Contains(labels, to) {
					e.Retarget(last, lbl)
				}
			}
		}
		lbl++
	}
	return e
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
