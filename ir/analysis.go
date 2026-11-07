package ir

// An analysis is a function's code with what passes over it keep asking:
// its control-flow graph, which block each instruction is in, and how many
// instructions write each register.  It is built once and read only.
type analysis struct {
	code    Code
	instrs  []Instruction
	g       *Graph
	blockOf []int // the block of each instruction
	writes  []int // how many instructions write each register
}

func analyse(c Code) *analysis {
	a := &analysis{
		code:    c,
		instrs:  c.Instructions,
		g:       NewGraph(c.Instructions),
		blockOf: make([]int, len(c.Instructions)),
		writes:  make([]int, len(c.Registers)),
	}
	for b, blk := range a.g.Blocks {
		for i := blk.Start; i < blk.End; i++ {
			a.blockOf[i] = b
		}
	}
	for _, instr := range a.instrs {
		for _, r := range instr.Writes() {
			a.writes[r]++
		}
	}
	return a
}

// loopLen is the number of instructions in loop l.
func (a *analysis) loopLen(l int) int {
	n := 0
	for _, b := range a.g.Loops[l].Blocks {
		n += a.g.Blocks[b].End - a.g.Blocks[b].Start
	}
	return n
}

// fallsInto reports whether block q is the one before block e in the code
// and goes on to it.
func (a *analysis) fallsInto(q, e int) bool {
	if q+1 != e {
		return false
	}
	_, _, next := controlFlow(a.instrs[a.g.Blocks[q].End-1])
	return next
}

// freeLabel returns a label the code doesn't use.
func (a *analysis) freeLabel() Label {
	next := Label(0)
	for _, instr := range a.instrs {
		if l, ok := instr.(DeclareLabel); ok && l.Label >= next {
			next = l.Label + 1
		}
	}
	return next
}

// leadingLabels returns the labels at the start of block b, and the position
// of its first other instruction.
func (a *analysis) leadingLabels(b int) (labels []Label, pos int) {
	blk := a.g.Blocks[b]
	pos = blk.Start
	for pos < blk.End && isLabel(a.instrs[pos]) {
		labels = append(labels, a.instrs[pos].(DeclareLabel).Label)
		pos++
	}
	return labels, pos
}
