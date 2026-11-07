package ir

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"

	"github.com/arnodel/golua/ops"
)

func lbl(n int) Instruction             { return DeclareLabel{Label: Label(n)} }
func jmp(n int) Instruction             { return Jump{Label: Label(n)} }
func jif(n int, r Register) Instruction { return JumpIf{Cond: r, Label: Label(n)} }
func set(r Register) Instruction        { return LoadConst{Dst: r} }
func ret(r Register) Instruction        { return Call{Cont: r, Tail: true} }

// renderGraph shows the blocks with their instruction ranges, successors and
// immediate dominators, then the loops.
func renderGraph(g *Graph) string {
	var b strings.Builder
	for i, blk := range g.Blocks {
		fmt.Fprintf(&b, "b%d %d-%d ->%v idom %d\n", i, blk.Start, blk.End, blk.Succs, g.Idom[i])
	}
	for i, l := range g.Loops {
		fmt.Fprintf(&b, "loop%d %v entries %v exits %v parent %d\n", i, l.Blocks, l.Entries, l.Exits, l.Parent)
	}
	return b.String()
}

func TestGraphShapes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		instrs []Instruction
		want   string
	}{
		{"straight line", []Instruction{set(1), set(2), ret(0)}, `
b0 0-3 ->[] idom 0
`},
		{"if else", []Instruction{
			set(1), jif(1, 0),
			set(2), jmp(2),
			lbl(1), set(2),
			lbl(2), ret(0),
		}, `
b0 0-2 ->[1 2] idom 0
b1 2-4 ->[3] idom 0
b2 4-6 ->[3] idom 0
b3 6-8 ->[] idom 0
`},
		// The shape of a while loop since #196: the test is at the bottom and
		// the loop is entered by jumping to it.
		{"while", []Instruction{
			jmp(2),
			lbl(1), set(1),
			lbl(2), jif(1, 0),
			ret(0),
		}, `
b0 0-1 ->[2] idom 0
b1 1-3 ->[2] idom 2
b2 3-5 ->[1 3] idom 0
b3 5-6 ->[] idom 2
loop0 [1 2] entries [2] exits [3] parent -1
`},
		// A: ...; B: ...; C: jump A; D: jump B.  From D execution goes back
		// to B, then C, then A: one loop from A to D, with B to D inside it.
		{"overlapping jumps", []Instruction{
			set(1),
			lbl(1), set(2),
			lbl(2), set(3), jif(1, 0),
			set(4), jif(2, 0),
			ret(0),
		}, `
b0 0-1 ->[1] idom 0
b1 1-3 ->[2] idom 0
b2 3-6 ->[1 3] idom 1
b3 6-8 ->[2 4] idom 2
b4 8-9 ->[] idom 3
loop0 [1 2 3] entries [1] exits [4] parent -1
loop1 [2 3] entries [2] exits [1 4] parent 0
`},
		// "if c then goto b end ::a:: ... ::b:: ... if x then goto a end":
		// the loop from a is entered at a and at b, so nothing inside it is
		// a loop of its own.
		{"two entries", []Instruction{
			jif(0, 0),
			jmp(1),
			lbl(0), set(1),
			lbl(1), set(1), jif(0, 2),
			ret(1),
		}, `
b0 0-1 ->[1 2] idom 0
b1 1-2 ->[3] idom 0
b2 2-4 ->[3] idom 0
b3 4-7 ->[2 4] idom 0
b4 7-8 ->[] idom 3
loop0 [2 3] entries [2 3] exits [4] parent -1
`},
		// The function's entry starts the outer loop; the inner loop is a
		// single block jumping to itself.
		{"nested", []Instruction{
			lbl(1), set(1),
			lbl(2), set(2), jif(2, 0),
			jif(1, 3),
			ret(1),
		}, `
b0 0-2 ->[1] idom 0
b1 2-5 ->[1 2] idom 0
b2 5-6 ->[0 3] idom 1
b3 6-7 ->[] idom 2
loop0 [0 1 2] entries [0] exits [3] parent -1
loop1 [1] entries [1] exits [2] parent 0
`},
		{"unreachable", []Instruction{
			jmp(1),
			set(1),
			lbl(1), lbl(2), ret(0),
		}, `
b0 0-1 ->[2] idom 0
b1 1-2 ->[2] idom -1
b2 2-5 ->[] idom 0
`},
	} {
		g := NewGraph(tc.instrs)
		if got := "\n" + renderGraph(g); got != tc.want {
			t.Errorf("%s:\ngot%swant%s", tc.name, got, tc.want)
		}
	}

	g := NewGraph([]Instruction{jmp(1), set(1), lbl(1), ret(0)})
	if g.Reachable(1) || !g.Reachable(2) || !slices.Equal(g.Order, []int{0, 2}) {
		t.Errorf("unreachable: Order %v", g.Order)
	}
	if g.LabelBlock(1) != 2 || g.LabelBlock(7) != -1 {
		t.Errorf("LabelBlock: L1 in b%d, L7 in b%d", g.LabelBlock(1), g.LabelBlock(7))
	}
	if !slices.Equal(g.Blocks[2].Preds, []int{0, 1}) {
		t.Errorf("unreachable: preds %v", g.Blocks[2].Preds)
	}
}

func TestLiveness(t *testing.T) {
	add := func(dst, l, r Register) Instruction { return Combine{Op: ops.OpAdd, Dst: dst, Lsrc: l, Rsrc: r} }
	regSet := func(s RegSet, n int) []Register {
		var out []Register
		for r := Register(0); r < Register(n); r++ {
			if s.Has(r) {
				out = append(out, r)
			}
		}
		return out
	}

	// x = 0; while c do x = x + y end; return x, with c, x, y in r0, r1, r2.
	instrs := []Instruction{
		set(1), jmp(2),
		lbl(1), add(1, 1, 2),
		lbl(2), jif(1, 0),
		ret(1),
	}
	g := NewGraph(instrs)
	in, out := g.Liveness(instrs, make([]RegData, 3))
	for b, want := range [][]Register{{0, 2}, {0, 1, 2}, {0, 1, 2}, {1}} {
		if got := regSet(in[b], 3); !slices.Equal(got, want) {
			t.Errorf("live into b%d: %v, want %v", b, got, want)
		}
	}
	if got := regSet(out[3], 3); len(got) != 0 {
		t.Errorf("live out of the last block: %v", got)
	}

	// Writing a cell stores into the cell, so the cell must be there; only
	// clearing it ends what it held.
	cells := []RegData{{}, {IsCell: true}}
	instrs = []Instruction{set(1), ClearReg{Dst: 1}, set(1), ret(1)}
	g = NewGraph(instrs)
	in, _ = g.Liveness(instrs, cells)
	if got := regSet(in[0], 2); !slices.Equal(got, []Register{1}) {
		t.Errorf("cell: live into b0: %v, want [r1]", got)
	}
}

// TestGraphRandom checks the graph of random code against brute force over
// the instructions.
func TestGraphRandom(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	const nregs = 6
	regs := make([]RegData, nregs)
	regs[4].IsCell, regs[5].IsCell = true, true
	loops, nested, multi := 0, 0, 0
	defer func() { t.Logf("%d loops, %d nested, %d with several entries", loops, nested, multi) }()
	for iter := 0; iter < 2000; iter++ {
		instrs := randomCode(rnd, nregs)
		g := NewGraph(instrs)
		in, _ := g.Liveness(instrs, regs)
		checkGraph(t, instrs, g, in, regs)
		for _, l := range g.Loops {
			loops++
			if l.Parent >= 0 {
				nested++
			}
			if len(l.Entries) > 1 {
				multi++
			}
		}
		if t.Failed() {
			for i, instr := range instrs {
				t.Logf("%d: %s", i, instr)
			}
			t.Logf("\n%s", renderGraph(g))
			return
		}
	}
}

func randomCode(rnd *rand.Rand, nregs int) []Instruction {
	n := 1 + rnd.Intn(25)
	nlabels := 1 + rnd.Intn(4)
	reg := func() Register { return Register(rnd.Intn(nregs)) }
	label := func() int { return rnd.Intn(nlabels) }
	instrs := make([]Instruction, n)
	for i := range instrs {
		switch rnd.Intn(8) {
		case 0:
			instrs[i] = jmp(label())
		case 1, 2:
			instrs[i] = jif(label(), reg())
		case 3:
			instrs[i] = ret(reg())
		case 4:
			instrs[i] = ClearReg{Dst: reg()}
		case 5:
			instrs[i] = Combine{Op: ops.OpAdd, Dst: reg(), Lsrc: reg(), Rsrc: reg()}
		default:
			instrs[i] = set(reg())
		}
	}
	// Declare every label once, at random places.
	for l := 0; l < nlabels; l++ {
		i := rnd.Intn(len(instrs) + 1)
		instrs = slices.Insert(instrs, i, lbl(l))
	}
	return instrs
}

func checkGraph(t *testing.T, instrs []Instruction, g *Graph, liveIn []RegSet, regs []RegData) {
	t.Helper()
	labelAt := map[Label]int{}
	for i, instr := range instrs {
		if l, ok := instr.(DeclareLabel); ok {
			labelAt[l.Label] = i
		}
	}
	succs := func(i int) []int {
		var out []int
		next := i+1 < len(instrs)
		switch x := instrs[i].(type) {
		case Jump:
			return []int{labelAt[x.Label]}
		case JumpIf:
			out = append(out, labelAt[x.Label])
		case Call:
			next = next && !x.Tail
		}
		if next {
			out = append(out, i+1)
		}
		return out
	}
	blockOf := make([]int, len(instrs))
	for b, blk := range g.Blocks {
		for i := blk.Start; i < blk.End; i++ {
			blockOf[i] = b
		}
	}

	// Blocks: entered only at their start, left only after their end, with
	// the successors of their last instruction.
	for b, blk := range g.Blocks {
		for i := blk.Start; i < blk.End-1; i++ {
			if s := succs(i); !slices.Equal(s, []int{i + 1}) {
				t.Errorf("b%d: instruction %d goes to %v", b, i, s)
			}
		}
		for i := blk.Start + 1; i < blk.End; i++ {
			if isLabel(instrs[i]) && !isLabel(instrs[i-1]) {
				t.Errorf("b%d: label inside, at %d", b, i)
			}
		}
		var want []int
		if blk.End > blk.Start {
			for _, s := range succs(blk.End - 1) {
				want = append(want, blockOf[s])
			}
		} else if b+1 < len(g.Blocks) {
			want = []int{b + 1}
		}
		slices.Sort(want)
		want = slices.Compact(want)
		if !slices.Equal(blk.Succs, want) {
			t.Errorf("b%d: successors %v, want %v", b, blk.Succs, want)
		}
	}

	// Reachability between blocks, by paths of at least one edge.
	nb := len(g.Blocks)
	reach := make([][]bool, nb)
	for a := range reach {
		reach[a] = make([]bool, nb)
		stack := slices.Clone(g.Blocks[a].Succs)
		for len(stack) > 0 {
			x := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if !reach[a][x] {
				reach[a][x] = true
				stack = append(stack, g.Blocks[x].Succs...)
			}
		}
	}
	reachable := func(b int) bool { return b == 0 || reach[0][b] }
	for b := range g.Blocks {
		if g.Reachable(b) != reachable(b) {
			t.Errorf("b%d: Reachable %v", b, g.Reachable(b))
		}
	}

	// Dominators: a dominates b if b can't be reached from the entry
	// without going through a.
	reachableAvoiding := func(avoid, target int) bool {
		if avoid == 0 {
			return false
		}
		seen := map[int]bool{0: true}
		stack := []int{0}
		for len(stack) > 0 {
			x := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if x == target {
				return true
			}
			for _, s := range g.Blocks[x].Succs {
				if s != avoid && !seen[s] {
					seen[s] = true
					stack = append(stack, s)
				}
			}
		}
		return false
	}
	for a := range g.Blocks {
		for b := range g.Blocks {
			want := reachable(a) && reachable(b) && (a == b || !reachableAvoiding(a, b))
			if g.Dominates(a, b) != want {
				t.Errorf("Dominates(b%d, b%d) = %v", a, b, !want)
			}
		}
	}

	// Loops: two reachable blocks are in the same outermost loop if each
	// can reach the other; a block is in a loop if it can reach itself.
	outer := func(b int) int {
		l := g.Blocks[b].Loop
		for l >= 0 && g.Loops[l].Parent >= 0 {
			l = g.Loops[l].Parent
		}
		return l
	}
	for a := range g.Blocks {
		if !reachable(a) {
			if g.Blocks[a].Loop != -1 {
				t.Errorf("unreachable b%d in loop %d", a, g.Blocks[a].Loop)
			}
			continue
		}
		if (outer(a) >= 0) != reach[a][a] {
			t.Errorf("b%d: in loop %d, reaches itself %v", a, outer(a), reach[a][a])
		}
		for b := range g.Blocks {
			if reachable(b) && a != b && (outer(a) >= 0 && outer(a) == outer(b)) != (reach[a][b] && reach[b][a]) {
				t.Errorf("b%d, b%d: loops %d, %d", a, b, outer(a), outer(b))
			}
		}
	}
	for i, l := range g.Loops {
		in := map[int]bool{}
		for _, b := range l.Blocks {
			in[b] = true
		}
		if l.Parent >= 0 {
			for _, b := range l.Blocks {
				if !slices.Contains(g.Loops[l.Parent].Blocks, b) {
					t.Errorf("loop%d: b%d not in parent loop%d", i, b, l.Parent)
				}
			}
		}
		for _, b := range l.Blocks {
			entered := b == 0
			for _, p := range g.Blocks[b].Preds {
				entered = entered || reachable(p) && !in[p]
			}
			if entered != slices.Contains(l.Entries, b) {
				t.Errorf("loop%d: b%d entry %v", i, b, !entered)
			}
		}
	}

	// Nested loops: inside a loop, without the jumps back to its entries,
	// two blocks are in the same nested loop if each can reach the other.
	for p, pl := range g.Loops {
		in := map[int]bool{}
		for _, b := range pl.Blocks {
			in[b] = true
		}
		inner := map[int][]int{}
		for _, a := range pl.Blocks {
			seen := map[int]bool{}
			stack := []int{a}
			for len(stack) > 0 {
				x := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				for _, s := range g.Blocks[x].Succs {
					if in[s] && !slices.Contains(pl.Entries, s) && !seen[s] {
						seen[s] = true
						stack = append(stack, s)
					}
				}
			}
			for s := range seen {
				inner[a] = append(inner[a], s)
			}
		}
		child := func(b int) int {
			for l := g.Blocks[b].Loop; l >= 0; l = g.Loops[l].Parent {
				if g.Loops[l].Parent == p {
					return l
				}
			}
			return -1
		}
		for _, a := range pl.Blocks {
			if (child(a) >= 0) != slices.Contains(inner[a], a) {
				t.Errorf("loop%d: b%d in nested loop %d", p, a, child(a))
			}
			for _, b := range pl.Blocks {
				mutual := slices.Contains(inner[a], b) && slices.Contains(inner[b], a)
				if a != b && (child(a) >= 0 && child(a) == child(b)) != mutual {
					t.Errorf("loop%d: b%d, b%d in nested loops %d, %d", p, a, b, child(a), child(b))
				}
			}
		}
	}

	// Liveness: r is live before instruction i if some path from i reaches
	// an instruction that needs r without first passing one that sets it.
	uses := func(i int, r Register) bool {
		_, clear := instrs[i].(ClearReg)
		return slices.Contains(instrs[i].Reads(), r) ||
			regs[r].IsCell && !clear && slices.Contains(instrs[i].Writes(), r)
	}
	defines := func(i int, r Register) bool {
		_, clear := instrs[i].(ClearReg)
		return (!regs[r].IsCell || clear) && slices.Contains(instrs[i].Writes(), r)
	}
	liveBefore := func(i int, r Register) bool {
		seen := map[int]bool{i: true}
		stack := []int{i}
		for len(stack) > 0 {
			j := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if uses(j, r) {
				return true
			}
			if defines(j, r) {
				continue
			}
			for _, s := range succs(j) {
				if !seen[s] {
					seen[s] = true
					stack = append(stack, s)
				}
			}
		}
		return false
	}
	for b, blk := range g.Blocks {
		if blk.End == blk.Start {
			continue
		}
		for r := range regs {
			if want := liveBefore(blk.Start, Register(r)); liveIn[b].Has(Register(r)) != want {
				t.Errorf("b%d: r%d live in %v, want %v", b, r, !want, want)
			}
		}
	}
}
