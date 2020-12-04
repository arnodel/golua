package ir

import (
	"github.com/arnodel/golua/ops"
)

// FoldConstants uses the given FoldFunc to fold the code items in the given
// constant slice.
func FoldConstants(consts []Constant, f FoldFunc) []Constant {
	fConsts := make([]Constant, len(consts))
	for i, k := range consts {
		if c, ok := k.(*Code); ok {
			fc := FoldCode(*c, f)
			fConsts[i] = &fc
		} else {
			fConsts[i] = k
		}
	}
	return fConsts
}

// FoldCode uses the given FoldFunc to fold the instructions in the given code.
// Returns a new Code instande with folded code.
func FoldCode(c Code, f FoldFunc) Code {
	if f == nil {
		return c
	}
	var s foldStack
	var i1 Instruction
	var l1 int
	for i, i2 := range c.Instructions {
		l2 := c.Lines[i]
		if i1 != nil {
			i1, i2 = f(i1, i2, c.Registers)
			switch {
			case i1 == nil && i2 == nil:
				// Folded to nothing, pop from the stack to be able to fold the
				// next instruction.
				l2, i2 = s.pop()
			case i1 == nil:
				// Folded to i2
				l2 = mergeLines(l1, l2)
			case i2 == nil:
				// Folded to i1
				i1, i2 = nil, i1
				l2 = mergeLines(l1, l2)
			default:
				// Not folded
			}
		}
		if i1 != nil {
			s.push(l1, i1)
		}
		i1 = i2
		l1 = l2
	}
	if i1 != nil {
		s.push(l1, i1)
	}
	c.Lines = s.lines
	c.Instructions = s.instructions
	return c
}

// A FoldFunc can turn 2 instructions into fewer instructions (in which case
// some of the returned instructions are nil).
type FoldFunc func(i1, i2 Instruction, regs []RegData) (Instruction, Instruction)

// DefaultFold applies a few simple folds
var DefaultFold FoldFunc = nil

// FoldTakeRelease folds the following
//
// (take r1; release r1) ==> ()
func FoldTakeRelease(i1, i2 Instruction, regs []RegData) (Instruction, Instruction) {
	t1, ok := i1.(TakeRegister)
	if !ok {
		return i1, i2
	}
	r2, ok := i2.(ReleaseRegister)
	if !ok {
		return i1, i2
	}
	if t1.Reg != r2.Reg {
		return i1, i2
	}
	return nil, nil
}

// FoldMoveReg folds the following
//
// (r1 <- X; r2 <- r1) ==> r2 <- X
func FoldMoveReg(i1, i2 Instruction, regs []RegData) (Instruction, Instruction) {
	sr1, ok := i1.(SetRegInstruction)
	if !ok {
		return i1, i2
	}
	dst := sr1.DestReg()
	if regs[dst].IsCell {
		return i1, i2
	}
	tr2, ok := i2.(Transform)
	if !ok {
		return i1, i2
	}
	if tr2.Op != ops.OpId || dst != tr2.Src {
		return i1, i2
	}
	return nil, sr1.WithDestReg(tr2.Dst)
}

// ComposeFolds takes several fold functions are composes them into one single
// function applying all the folds.
func ComposeFolds(fs ...FoldFunc) FoldFunc {
	return func(i1, i2 Instruction, regs []RegData) (Instruction, Instruction) {
		for _, f := range fs {
			i1, i2 = f(i1, i2, regs)
			if i1 == nil || i2 == nil {
				break
			}
		}
		return i1, i2
	}
}

type foldStack struct {
	lines        []int
	instructions []Instruction
}

func (s *foldStack) push(l int, i Instruction) {
	s.lines = append(s.lines, l)
	s.instructions = append(s.instructions, i)
}

func (s *foldStack) empty() bool {
	return len(s.instructions) == 0
}

func (s *foldStack) pop() (l int, i Instruction) {
	last := len(s.instructions) - 1
	if last < 0 {
		return
	}
	l = s.lines[last]
	i = s.instructions[last]
	s.lines = s.lines[:last]
	s.instructions = s.instructions[:last]
	return
}

func mergeLines(l1, l2 int) int {
	if l1 != 0 {
		return l1
	}
	return l2
}
