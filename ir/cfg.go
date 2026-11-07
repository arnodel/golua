package ir

import (
	"fmt"
	"math/bits"
	"slices"
)

// Graph is the control-flow graph of a function's instructions: which code
// can run after which.  See notes/control-flow-graph.md.
//
// The instructions are cut into blocks, runs entered only at their first
// instruction and left only after their last.  A block starts at a label
// (consecutive labels share a block) or after an instruction that ends one,
// and ends after a Jump, a JumpIf or a tail Call, or just before a label.
// The blocks are the nodes of a directed graph whose edges are the jumps and
// the fall-throughs from one block to the next; a loop is a cycle in it.
//
// A non-tail Call is not an edge: with continuations, the code resumes at
// the next instruction.  Neither is an error, which leaves the whole
// function, nor TruncateCloseStack, whose __close handlers come back.  A
// tail Call leaves the function, and so does falling off the end.
//
// Two examples.  First a while loop as the compiler lays it out, with the
// test after the body and the loop entered by jumping to the test:
//
//	b0: jump L2             b0 → b2
//	b1: L1: body            b1 → b2
//	b2: L2: if c jump L1    b2 → b1, b3
//	b3: after
//
// Order is b0 b2 b3 b1: execution reaches the test before the body.  Every
// edge goes forward in it except b1 → b2, the jump back that closes the
// loop.  b2 dominates b1, since the body is only entered from the test, but
// b1 does not dominate b2 although it comes first in the code: the first
// time round, b0 jumped straight to b2.  The one loop is {b1, b2}, entered
// at b2 and exiting to b3.  Code meant to run once before the loop belongs
// in b0, which dominates both.
//
// Then a loop that goto enters at two places, schematically:
//
//	if c then goto b end
//	::a:: x = x + 1
//	::b:: x = x + 10
//	if x < 50 then goto a end
//
//	b0: if c jump L1                     b0 → b1, b2
//	b1: L0: x = x + 1                    b1 → b2
//	b2: L1: x = x + 10; if x < 50 jump L0    b2 → b1, b3
//	b3: after
//
// Order is b0 b1 b2 b3, with b2 → b1 the edge that closes the cycle.  The
// loop is {b1, b2}, entered at both b1 (falling in) and b2 (the goto), so
// neither dominates the other: b0 is the immediate dominator of both.  No
// single block of the loop runs before all the others, which is why loops
// are found as strongly connected components rather than from a header.
type Graph struct {
	// Blocks holds the blocks in the order of the code.  Block 0 is the
	// entry.
	Blocks []Block

	// Order is a linearisation of the blocks that can run: every edge goes
	// from an earlier block to a later one, except the edges that close
	// cycles.  It is the reverse postorder of a depth-first walk from the
	// entry: a block is written down once everything it leads to has been
	// visited, and the list is reversed.  An edge met on the way to a block
	// the walk is still inside is one that closes a cycle; those are the
	// only ones that go backwards.  Blocks that cannot run are absent.  A
	// block's dominators all come before it.
	Order []int

	// Idom holds, for each block, its immediate dominator: the last block
	// that every path from the entry to it goes through.  The entry is its
	// own; an unreachable block has -1.  Following Idom from any reachable
	// block leads to the entry, so the immediate dominators form a tree,
	// which Dominates walks.
	Idom []int

	// Loops holds the loops, each before the loops nested in it.
	Loops []Loop

	order []int // position of each block in Order, -1 if unreachable
}

// Block is a run of instructions, Instructions[Start:End].
type Block struct {
	Start, End   int
	Succs, Preds []int // block indices, increasing, without repeats
	Loop         int   // innermost loop containing the block, or -1
}

// Loop is a set of blocks where each can reach all the others: a strongly
// connected component of the graph, or of a loop with the jumps back to its
// entries removed for a nested loop.  Unlike the "natural loops" of a single
// header, this handles a loop that goto enters at two places.
type Loop struct {
	Blocks  []int // in increasing order
	Entries []int // blocks of the loop entered from outside it (or the function's entry)
	Exits   []int // blocks outside the loop with a predecessor in it
	Parent  int   // the loop this one is nested in, or -1
}

// NewGraph builds the control-flow graph of instrs.
func NewGraph(instrs []Instruction) *Graph {
	g := &Graph{}
	g.cutBlocks(instrs)
	g.orderBlocks()
	g.findDominators()
	for i := range g.Blocks {
		g.Blocks[i].Loop = -1
	}
	g.findLoops(slices.Sorted(slices.Values(g.Order)), -1, map[[2]int]bool{})
	return g
}

// Reachable reports whether block b can run.
func (g *Graph) Reachable(b int) bool {
	return g.order[b] >= 0
}

// Dominates reports whether every path from the entry to block b goes
// through block a.  A block dominates itself.  Nothing dominates an
// unreachable block.
func (g *Graph) Dominates(a, b int) bool {
	if !g.Reachable(a) || !g.Reachable(b) {
		return false
	}
	for b != a && b != 0 {
		b = g.Idom[b]
	}
	return b == a
}

func (g *Graph) cutBlocks(instrs []Instruction) {
	labelBlock := map[Label]int{}
	start := 0
	cut := func(end int) {
		if end > start {
			g.Blocks = append(g.Blocks, Block{Start: start, End: end})
			start = end
		}
	}
	for i, instr := range instrs {
		switch x := instr.(type) {
		case DeclareLabel:
			if i > 0 && !isLabel(instrs[i-1]) {
				cut(i)
			}
			labelBlock[x.Label] = len(g.Blocks)
		default:
			if _, jumps, next := controlFlow(instr); jumps || !next {
				cut(i + 1)
			}
		}
	}
	cut(len(instrs))
	if len(g.Blocks) == 0 {
		g.Blocks = []Block{{}}
	}

	target := func(lbl Label) int {
		b, ok := labelBlock[lbl]
		if !ok {
			panic(fmt.Sprintf("jump to undeclared label %s", lbl))
		}
		return b
	}
	for b := range g.Blocks {
		blk := &g.Blocks[b]
		fallsThrough := true
		if blk.End > blk.Start {
			lbl, jumps, next := controlFlow(instrs[blk.End-1])
			if jumps {
				blk.Succs = []int{target(lbl)}
			}
			fallsThrough = next
		}
		if fallsThrough && b+1 < len(g.Blocks) {
			blk.Succs = append(blk.Succs, b+1)
		}
		slices.Sort(blk.Succs)
		blk.Succs = slices.Compact(blk.Succs)
	}
	for b, blk := range g.Blocks {
		for _, s := range blk.Succs {
			g.Blocks[s].Preds = append(g.Blocks[s].Preds, b)
		}
	}
}

// controlFlow says where execution can go after instr: to lbl if jumps is
// true, and to the next instruction if next is true.  A Jump only jumps, a
// JumpIf does both, a tail Call does neither: it leaves the function.  Every
// other instruction goes to the next one.
func controlFlow(instr Instruction) (lbl Label, jumps, next bool) {
	switch x := instr.(type) {
	case Jump:
		return x.Label, true, false
	case JumpIf:
		return x.Label, true, true
	case Call:
		return 0, false, !x.Tail
	}
	return 0, false, true
}

func isLabel(instr Instruction) bool {
	_, ok := instr.(DeclareLabel)
	return ok
}

func (g *Graph) orderBlocks() {
	g.order = make([]int, len(g.Blocks))
	for i := range g.order {
		g.order[i] = -1
	}
	visited := make([]bool, len(g.Blocks))
	var post []int
	var visit func(b int)
	visit = func(b int) {
		visited[b] = true
		for _, s := range g.Blocks[b].Succs {
			if !visited[s] {
				visit(s)
			}
		}
		post = append(post, b)
	}
	visit(0)
	for i := len(post) - 1; i >= 0; i-- {
		g.order[post[i]] = len(g.Order)
		g.Order = append(g.Order, post[i])
	}
}

// findDominators uses the iterative algorithm of Cooper, Harvey and Kennedy
// ("A Simple, Fast Dominance Algorithm"): a block's immediate dominator is
// where the dominator chains of its predecessors meet, repeated in reverse
// postorder until nothing changes.
func (g *Graph) findDominators() {
	g.Idom = make([]int, len(g.Blocks))
	for i := range g.Idom {
		g.Idom[i] = -1
	}
	g.Idom[0] = 0
	meet := func(a, b int) int {
		for a != b {
			for g.order[a] > g.order[b] {
				a = g.Idom[a]
			}
			for g.order[b] > g.order[a] {
				b = g.Idom[b]
			}
		}
		return a
	}
	for changed := true; changed; {
		changed = false
		for _, b := range g.Order[1:] {
			idom := -1
			for _, p := range g.Blocks[b].Preds {
				switch {
				case g.Idom[p] < 0:
				case idom < 0:
					idom = p
				default:
					idom = meet(p, idom)
				}
			}
			if idom != g.Idom[b] {
				g.Idom[b] = idom
				changed = true
			}
		}
	}
}

// findLoops finds the loops among blocks (in increasing order), nested in
// loop parent, ignoring the edges in removed.  Each loop is a strongly
// connected component; the loops nested in it are the components of its
// blocks once the edges back to its entries are removed.
func (g *Graph) findLoops(blocks []int, parent int, removed map[[2]int]bool) {
	in := map[int]bool{}
	for _, b := range blocks {
		in[b] = true
	}
	succs := func(b int) []int {
		var out []int
		for _, s := range g.Blocks[b].Succs {
			if in[s] && !removed[[2]int{b, s}] {
				out = append(out, s)
			}
		}
		return out
	}
	for _, scc := range components(blocks, succs) {
		if len(scc) == 1 && !slices.Contains(succs(scc[0]), scc[0]) {
			continue
		}
		slices.Sort(scc)
		l := Loop{Blocks: scc, Parent: parent}
		inLoop := map[int]bool{}
		for _, b := range scc {
			inLoop[b] = true
		}
		for _, b := range scc {
			outside := b == 0
			for _, p := range g.Blocks[b].Preds {
				outside = outside || g.Reachable(p) && !inLoop[p]
			}
			if outside {
				l.Entries = append(l.Entries, b)
			}
			for _, s := range g.Blocks[b].Succs {
				if !inLoop[s] {
					l.Exits = append(l.Exits, s)
				}
			}
			g.Blocks[b].Loop = len(g.Loops)
		}
		slices.Sort(l.Exits)
		l.Exits = slices.Compact(l.Exits)
		idx := len(g.Loops)
		g.Loops = append(g.Loops, l)
		for _, b := range scc {
			for _, e := range l.Entries {
				if slices.Contains(g.Blocks[b].Succs, e) {
					removed[[2]int{b, e}] = true
				}
			}
		}
		g.findLoops(scc, idx, removed)
	}
}

// components returns the strongly connected components of the graph on
// nodes given by succs, using Tarjan's algorithm.
func components(nodes []int, succs func(int) []int) [][]int {
	index := map[int]int{}
	low := map[int]int{}
	onStack := map[int]bool{}
	var stack []int
	var sccs [][]int
	var visit func(v int)
	visit = func(v int) {
		index[v] = len(index)
		low[v] = index[v]
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range succs(v) {
			if _, seen := index[w]; !seen {
				visit(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] == index[v] {
			var scc []int
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, scc)
		}
	}
	for _, v := range nodes {
		if _, seen := index[v]; !seen {
			visit(v)
		}
	}
	return sccs
}

// RegSet is a set of registers.
type RegSet []uint64

// NewRegSet returns an empty set for registers below n.
func NewRegSet(n int) RegSet {
	return make(RegSet, (n+63)/64)
}

// Has reports whether r is in the set.
func (s RegSet) Has(r Register) bool {
	return s[r/64]&(1<<(r%64)) != 0
}

// Add puts r in the set.
func (s RegSet) Add(r Register) {
	s[r/64] |= 1 << (r % 64)
}

// Remove takes r out of the set.
func (s RegSet) Remove(r Register) {
	s[r/64] &^= 1 << (r % 64)
}

// Len returns the number of registers in the set.
func (s RegSet) Len() int {
	n := 0
	for _, w := range s {
		n += bits.OnesCount64(w)
	}
	return n
}

// Liveness says, for each block of g, which registers are live when it
// starts (LiveIn) and when it ends (LiveOut): their value may still be read.
//
// A cell register holds a reference to its cell, and writing it stores into
// the cell, so for a cell a write needs the register as a read does.  Only
// ClearReg, which puts a fresh cell in the register, ends what it held.
func (g *Graph) Liveness(instrs []Instruction, regs []RegData) (liveIn, liveOut []RegSet) {
	n := len(regs)
	use := make([]RegSet, len(g.Blocks))
	def := make([]RegSet, len(g.Blocks))
	for b, blk := range g.Blocks {
		use[b], def[b] = NewRegSet(n), NewRegSet(n)
		for _, instr := range instrs[blk.Start:blk.End] {
			for _, r := range instr.Reads() {
				if !def[b].Has(r) {
					use[b].Add(r)
				}
			}
			_, clear := instr.(ClearReg)
			for _, r := range instr.Writes() {
				if regs[r].IsCell && !clear {
					if !def[b].Has(r) {
						use[b].Add(r)
					}
				} else {
					def[b].Add(r)
				}
			}
		}
	}
	liveIn = make([]RegSet, len(g.Blocks))
	liveOut = make([]RegSet, len(g.Blocks))
	for b := range g.Blocks {
		liveIn[b], liveOut[b] = NewRegSet(n), NewRegSet(n)
	}
	// Backwards: a block's live-out is what its successors need, its live-in
	// what it reads before writing, plus its live-out less what it writes.
	for changed := true; changed; {
		changed = false
		for b := len(g.Blocks) - 1; b >= 0; b-- {
			out := liveOut[b]
			for _, s := range g.Blocks[b].Succs {
				for i, w := range liveIn[s] {
					out[i] |= w
				}
			}
			for i := range liveIn[b] {
				w := use[b][i] | out[i]&^def[b][i]
				if w != liveIn[b][i] {
					liveIn[b][i] = w
					changed = true
				}
			}
		}
	}
	return liveIn, liveOut
}
