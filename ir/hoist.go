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
// The work goes in stages, each a method of hoister: choose a loop for each
// load, place the loops' loads, then write the code out.
func HoistConstants(c Code) Code {
	h := newHoister(c)
	if len(h.g.Loops) == 0 {
		return c
	}
	for i, instr := range h.instrs {
		if load, ok := instr.(LoadConst); ok {
			h.choose(i, load)
		}
	}
	if len(h.rename) == 0 {
		return c
	}
	h.place()
	return h.emit()
}

// A hoister holds the state of HoistConstants on one function.
type hoister struct {
	code    Code
	instrs  []Instruction
	regs    []RegData
	g       *Graph
	writes  []int // how many instructions write each register
	blockOf []int // the block of each instruction

	plans map[int]*hoistPlan // by loop, made on demand

	// What choose decides.
	hoisted map[int]map[uint]Register // loop -> constant -> its register
	rename  map[Register]Register     // a load's destination -> the register read instead
	deleted []bool                    // the loads that go

	// What place decides.
	inserts  map[int][]Instruction // position -> instructions to put before it
	retarget map[int]Label         // jump instruction -> its new target
}

func newHoister(c Code) *hoister {
	h := &hoister{
		code:     c,
		instrs:   c.Instructions,
		regs:     slices.Clone(c.Registers),
		g:        NewGraph(c.Instructions),
		writes:   make([]int, len(c.Registers)),
		blockOf:  make([]int, len(c.Instructions)),
		plans:    map[int]*hoistPlan{},
		hoisted:  map[int]map[uint]Register{},
		rename:   map[Register]Register{},
		deleted:  make([]bool, len(c.Instructions)),
		inserts:  map[int][]Instruction{},
		retarget: map[int]Label{},
	}
	for _, instr := range h.instrs {
		for _, r := range instr.Writes() {
			h.writes[r]++
		}
	}
	for b, blk := range h.g.Blocks {
		for i := blk.Start; i < blk.End; i++ {
			h.blockOf[i] = b
		}
	}
	return h
}

// A hoistPlan says how the loads for a loop get in front of it: on the edges
// into it, or by splitting its entries.  See HoistConstants.
type hoistPlan struct {
	ok     bool
	edges  []int // positions to insert before
	splits []int // entry blocks to split
}

// plan returns the plan for loop l, making it the first time.
func (h *hoister) plan(l int) *hoistPlan {
	if p, ok := h.plans[l]; ok {
		return p
	}
	p := &hoistPlan{}
	h.plans[l] = p
	loop := h.g.Loops[l]
	if h.loopLen(l) > hoistMaxLoop {
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
		for _, q := range h.g.Blocks[e].Preds {
			if inside(q) || !h.g.Reachable(q) {
				continue
			}
			blk := h.g.Blocks[q]
			lbl, jumps, next := controlFlow(h.instrs[blk.End-1])
			switch {
			case jumps && h.g.labelBlock(lbl, h.instrs) == e && !next:
				p.edges = append(p.edges, blk.End-1) // before the jump in
			case !jumps || h.g.labelBlock(lbl, h.instrs) != e:
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
		for _, q := range h.g.Blocks[e].Preds {
			if inside(q) && q+1 == e {
				if _, _, next := controlFlow(h.instrs[h.g.Blocks[q].End-1]); next {
					return p
				}
			}
		}
	}
	p.ok = true
	p.splits = loop.Entries
	return p
}

// loopLen is the number of instructions in loop l.
func (h *hoister) loopLen(l int) int {
	n := 0
	for _, b := range h.g.Loops[l].Blocks {
		n += h.g.Blocks[b].End - h.g.Blocks[b].Start
	}
	return n
}

// choose decides whether the load at instruction i is hoisted, and into
// which loop: the outermost loop around it that has a plan and room for the
// constant, or already has it.
func (h *hoister) choose(i int, load LoadConst) {
	if h.regs[load.Dst].IsCell || h.regs[load.Dst].IsLocal || h.writes[load.Dst] != 1 {
		return
	}
	target := -1
	for l := h.g.Blocks[h.blockOf[i]].Loop; l >= 0; l = h.g.Loops[l].Parent {
		if !h.plan(l).ok {
			continue
		}
		if _, done := h.hoisted[l][load.Kidx]; done || len(h.hoisted[l]) < hoistMaxConsts {
			target = l
		}
	}
	if target < 0 {
		return
	}
	if h.hoisted[target] == nil {
		h.hoisted[target] = map[uint]Register{}
	}
	r, ok := h.hoisted[target][load.Kidx]
	if !ok {
		r = Register(len(h.regs))
		h.regs = append(h.regs, RegData{})
		h.hoisted[target][load.Kidx] = r
	}
	h.rename[load.Dst] = r
	h.deleted[i] = true
}

// place decides where the loads of every loop go, according to its plan.
// Loops that split the same entry put their loads under one new label, which
// every jump back to that entry from inside any of them is moved to: a
// hoisted register is never written again, so no jump back needs to pass
// the loads.
func (h *hoister) place() {
	splitLoops := map[int][]int{} // entry block -> loops split there
	for l := range h.g.Loops {
		if h.hoisted[l] == nil {
			continue
		}
		p := h.plan(l)
		for _, pos := range p.edges {
			for _, in := range h.loads(l) {
				if !slices.Contains(h.inserts[pos], in) {
					h.inserts[pos] = append(h.inserts[pos], in)
				}
			}
		}
		for _, e := range p.splits {
			splitLoops[e] = append(splitLoops[e], l)
		}
	}
	nextLabel := h.freeLabel()
	for _, e := range slices.Sorted(maps.Keys(splitLoops)) {
		h.split(e, splitLoops[e], nextLabel)
		nextLabel++
	}
}

// loads returns the loads for loop l's hoisted constants, in register order.
func (h *hoister) loads(l int) []Instruction {
	var loads []Instruction
	for kidx, r := range h.hoisted[l] {
		loads = append(loads, LoadConst{Dst: r, Kidx: kidx})
	}
	slices.SortFunc(loads, func(a, b Instruction) int {
		return int(a.(LoadConst).Dst) - int(b.(LoadConst).Dst)
	})
	return loads
}

// freeLabel returns a label the code doesn't use.
func (h *hoister) freeLabel() Label {
	next := Label(0)
	for _, instr := range h.instrs {
		if l, ok := instr.(DeclareLabel); ok && l.Label >= next {
			next = l.Label + 1
		}
	}
	return next
}

// split puts the loads of loops at the top of entry block e, after its
// labels, under the new label lbl, and moves the jumps to e's labels from
// inside those loops to lbl.
func (h *hoister) split(e int, loops []int, lbl Label) {
	blk := h.g.Blocks[e]
	pos := blk.Start
	var labels []Label
	for pos < blk.End && isLabel(h.instrs[pos]) {
		labels = append(labels, h.instrs[pos].(DeclareLabel).Label)
		pos++
	}
	for _, l := range loops {
		h.inserts[pos] = append(h.inserts[pos], h.loads(l)...)
	}
	h.inserts[pos] = append(h.inserts[pos], DeclareLabel{Label: lbl})
	for _, l := range loops {
		for _, b := range h.g.Loops[l].Blocks {
			last := h.g.Blocks[b].End - 1
			if to, jumps, _ := controlFlow(h.instrs[last]); jumps && slices.Contains(labels, to) {
				h.retarget[last] = lbl
			}
		}
	}
}

// emit writes the code out: the inserted loads and labels, the kept
// instructions with their reads renamed and their jumps retargeted, and the
// registers with the new ones.
func (h *hoister) emit() Code {
	c := h.code
	out := make([]Instruction, 0, len(h.instrs)+len(h.inserts))
	lines := make([]int, 0, len(h.instrs)+len(h.inserts))
	for i := 0; i <= len(h.instrs); i++ {
		for _, in := range h.inserts[i] {
			out = append(out, in)
			lines = append(lines, 0)
		}
		if i == len(h.instrs) || h.deleted[i] {
			continue
		}
		instr := h.instrs[i]
		for _, r := range instr.Reads() {
			if to, ok := h.rename[r]; ok {
				instr = instr.ReplaceRead(r, to)
			}
		}
		if to, ok := h.retarget[i]; ok {
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
	c.Registers = h.regs
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
