package ir

import "slices"

// ThreadJumps shortens the paths that jumps take, so that fewer jumps run.
// It looks at the first instruction that runs at a jump's destination (labels
// and register bookkeeping emit no code, so it looks past them) and applies
// three rewrites until none applies:
//
//  1. A jump to a jump goes straight to the second jump's destination.
//
//  2. A conditional jump to a test of the same value goes where that test
//     sends it.  The test either jumps, and the first jump can go to the
//     same place, or falls through, and the first jump can go just after it.
//     A numeric for loop is the common case: after advancing the loop, "if
//     the counter is not nil, jump back" lands on the loop's entry test "if
//     the counter is nil, jump to the end", which then never jumps.  So the
//     jump back now goes past the entry test, and each iteration runs one
//     opcode fewer.  The value tested is the same when the register is the
//     same, or when the instruction just before the first jump copies one
//     register into the other (an "a and b" condition does that).
//
//  3. A jump to the instruction that comes next anyway is removed.
//
// A rewrite never changes the order of the other instructions, so the
// register allocation, which follows that order, is not affected.
func ThreadJumps(c Code) Code {
	instrs := slices.Clone(c.Instructions)
	n := len(instrs)

	// Where each label is: the index of the instruction it precedes.
	pos := map[Label]int{}
	nextLabel := Label(0)
	for i, instr := range instrs {
		if l, ok := instr.(DeclareLabel); ok {
			pos[l.Label] = i
			if l.Label >= nextLabel {
				nextLabel = l.Label + 1
			}
		}
	}

	removed := make([]bool, n)
	newLabels := map[int]Label{} // labels to declare just before an index

	// emitsNothing reports whether instruction i emits no opcode.
	emitsNothing := func(i int) bool {
		if removed[i] {
			return true
		}
		switch instrs[i].(type) {
		case DeclareLabel, TakeRegister, ReleaseRegister:
			return true
		}
		return false
	}
	// isBookkeeping reports whether instruction i emits no opcode and is not
	// a label.
	isBookkeeping := func(i int) bool {
		if _, ok := instrs[i].(DeclareLabel); ok {
			return false
		}
		return emitsNothing(i)
	}
	// runsFrom returns the index of the first instruction that emits an
	// opcode, from index i on (n if there is none).
	runsFrom := func(i int) int {
		for i < n && emitsNothing(i) {
			i++
		}
		return i
	}
	// labelAfter returns a label for the position just after instruction i.
	labelAfter := func(i int) Label {
		if l, ok := newLabels[i+1]; ok {
			return l
		}
		l := nextLabel
		nextLabel++
		newLabels[i+1] = l
		pos[l] = i + 1
		return l
	}
	// sameValue returns the register that instruction i copies into or from
	// r, if instruction i is a copy involving r with no label in between
	// (another path could reach the jump through a label).
	sameValue := func(i int, r Register) (Register, bool) {
		i--
		for i >= 0 && isBookkeeping(i) {
			i--
		}
		if i < 0 {
			return 0, false
		}
		if m, ok := asMove(instrs[i]); ok {
			if m.Dst == r {
				return m.Src, true
			}
			if m.Src == r {
				return m.Dst, true
			}
		}
		return 0, false
	}

	// Each round can only shorten paths, but jumps that form a cycle would
	// keep moving, so the number of rounds is capped.
	for round, changed := 0, true; changed && round < 8; round++ {
		changed = false
		for i, instr := range instrs {
			if removed[i] {
				continue
			}
			var lbl Label
			switch j := instr.(type) {
			case Jump:
				lbl = j.Label
			case JumpIf:
				lbl = j.Label
			default:
				continue
			}
			t := runsFrom(pos[lbl])
			if t == runsFrom(i+1) {
				removed[i] = true
				changed = true
				continue
			}
			if t == n || t == i {
				continue
			}
			newLbl := lbl
			switch to := instrs[t].(type) {
			case Jump:
				newLbl = to.Label
			case JumpIf:
				j, ok := instr.(JumpIf)
				if !ok {
					break
				}
				other, copied := sameValue(i, j.Cond)
				if to.Cond != j.Cond && !(copied && to.Cond == other) {
					break
				}
				if to.Not == j.Not {
					newLbl = to.Label
				} else {
					newLbl = labelAfter(t)
				}
			}
			if newLbl == lbl {
				continue
			}
			switch j := instr.(type) {
			case Jump:
				j.Label = newLbl
				instrs[i] = j
			case JumpIf:
				j.Label = newLbl
				instrs[i] = j
			}
			changed = true
		}
	}

	outInstrs := make([]Instruction, 0, n+len(newLabels))
	outLines := make([]int, 0, n+len(newLabels))
	for i := 0; i <= n; i++ {
		if l, ok := newLabels[i]; ok {
			outInstrs = append(outInstrs, DeclareLabel{Label: l})
			outLines = append(outLines, 0)
		}
		if i < n && !removed[i] {
			outInstrs = append(outInstrs, instrs[i])
			outLines = append(outLines, c.Lines[i])
		}
	}
	c.Instructions = outInstrs
	c.Lines = outLines
	return c
}
