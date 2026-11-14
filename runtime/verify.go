package runtime

import (
	"fmt"

	"github.com/arnodel/golua/code"
)

// Register operands are 8 bits wide, so a frame cannot have more registers
// than this, nor a closure more upvalues.
const maxRegCount = 256

// verify checks that running c cannot touch anything outside its frame, its
// code or its constants: the register counts make sense, every register
// operand of every opcode is within them, every constant an opcode names is
// in the constant table, every jump lands inside the code on an opcode (not
// on one of the operand words that list a call's arguments), and execution
// cannot run past the end of the code, whichever way it goes.  Without
// quotas, the run loop relies on this to access registers, opcodes and
// constants without checking their index (luacont_access_noquotas.go).
//
// Code produced by the compiler should always pass; code loaded from a binary
// chunk cannot be trusted to.
func (c *Code) verify() error {
	switch {
	case c.RegCount < 1 || c.RegCount > maxRegCount:
		return fmt.Errorf("invalid register count %d", c.RegCount)
	case c.UpvalueCount < 0 || c.UpvalueCount > maxRegCount:
		return fmt.Errorf("invalid upvalue count %d", c.UpvalueCount)
	case len(c.UpNames) != int(c.UpvalueCount):
		return fmt.Errorf("%d upvalue names for %d upvalues", len(c.UpNames), c.UpvalueCount)
	}
	// The frame fills the entry loads before running anything; r0 holds the
	// continuation.
	for _, l := range c.entryLoads {
		switch {
		case int(l.Up) >= int(c.UpvalueCount):
			return fmt.Errorf("entry load from upvalue %d of %d", l.Up, c.UpvalueCount)
		case l.Reg == 0 || int(l.Reg) >= int(c.RegCount):
			return fmt.Errorf("entry load into register %d outside the frame (%d registers)", l.Reg, c.RegCount)
		}
	}
	// Execution starts at the first opcode.
	if len(c.code) == 0 {
		return fmt.Errorf("no code")
	}
	// First the opcodes and the operand words of calls, which are not
	// opcodes and must be known before jump targets can be checked.
	operand := make([]bool, len(c.code))
	for pc := 0; pc < len(c.code); pc++ {
		op := c.code[pc]
		if err := c.verifyOpcode(pc, op); err != nil {
			return fmt.Errorf("opcode %d: %w", pc, err)
		}
		if n := callArgCount(op); n > 0 {
			words := code.ArgWordCount(n)
			if pc+words >= len(c.code) {
				return fmt.Errorf("opcode %d: call with %d arguments runs past the end of the code", pc, n)
			}
			for _, r := range code.ArgRegs(c.code[pc+1:pc+1+words], n) {
				if err := c.verifyRegisters(r); err != nil {
					return fmt.Errorf("opcode %d: argument %w", pc, err)
				}
			}
			for i := pc + 1; i <= pc+words; i++ {
				operand[i] = true
			}
			pc += words
		}
	}
	for pc, op := range c.code {
		if operand[pc] {
			continue
		}
		if target, ok := jumpTarget(pc, op); ok && operand[target] {
			return fmt.Errorf("opcode %d: jump to %d, inside the arguments of a call", pc, target)
		}
	}
	return c.verifyEnd()
}

// verifyEnd checks that execution cannot run past the end of the code: it
// follows every way execution can go from the first opcode, and each opcode
// it goes on to must be there.  Code that nothing reaches is not looked at:
// the compiler leaves some after a return (the closing of a scope that the
// return has left).  The opcodes and jump targets must have been verified.
func (c *Code) verifyEnd() error {
	reached := make([]bool, len(c.code))
	reached[0] = true
	todo := []int{0}
	for len(todo) > 0 {
		pc := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		op := c.code[pc]
		// Where execution can go from pc: at most two places.
		var next [2]int
		n := 0
		switch target, isJump := jumpTarget(pc, op); {
		case !op.HasType1() && op.TypePfx() == code.Type7Pfx:
			// It takes the jump after it (which goes to its target and
			// nowhere else, so is not looked at itself) or steps over it.
			target, _ = jumpTarget(pc+1, c.code[pc+1])
			next, n = [2]int{target, pc + 2}, 2
		case isJump && op.GetJ() == code.OpJump:
			next, n = [2]int{target}, 1
		case isJump:
			next, n = [2]int{target, pc + 1}, 2
		case goesOn(op):
			next, n = [2]int{pc + 1 + code.ArgWordCount(callArgCount(op))}, 1
		}
		for _, to := range next[:n] {
			if to >= len(c.code) {
				return fmt.Errorf("opcode %d: execution runs past the end of the code", pc)
			}
			if !reached[to] {
				reached[to] = true
				todo = append(todo, to)
			}
		}
	}
	return nil
}

// goesOn reports whether execution goes on to the opcode after op (after
// its operand words, for a call that returns here), op not being a jump.
// It doesn't after a tail call, which leaves the frame for good, or a call
// of r0, which is a return.
func goesOn(op code.Opcode) bool {
	if op.HasType1() || op.TypePfx() != code.Type5Pfx || op.GetJ() != code.OpCall {
		return true
	}
	return !op.GetF() && op.GetA() != code.ValueReg(0)
}

// callArgCount returns the number of arguments listed after op if it is a
// call, else 0.
func callArgCount(op code.Opcode) int {
	if op.HasType1() || op.TypePfx() != code.Type5Pfx || op.GetJ() != code.OpCall {
		return 0
	}
	return op.GetArgCount()
}

// verifyNextJump checks that the opcode after pc is a jump: the run loop
// takes or steps over it after a for loop opcode or a predicate, without
// looking at it.
func (c *Code) verifyNextJump(pc int) error {
	if pc+1 >= len(c.code) || !isJump(c.code[pc+1]) {
		return fmt.Errorf("opcode that decides a jump not followed by one")
	}
	return nil
}

// isJump reports whether op is an unconditional jump.
func isJump(op code.Opcode) bool {
	return !op.HasType1() && op.TypePfx() == code.Type5Pfx && op.GetJ() == code.OpJump
}

// jumpTarget returns the target of op if it is a jump.
func jumpTarget(pc int, op code.Opcode) (int, bool) {
	if op.HasType1() || op.TypePfx() != code.Type5Pfx {
		return 0, false
	}
	switch op.GetJ() {
	case code.OpJump, code.OpJumpIf:
		return pc + int(op.GetOffset()), true
	}
	return 0, false
}

// verifyOpcode checks the register operands of op, its jump target if it is
// a jump, and its constant index if it has one.  Which of the fields A, B and
// C name registers depends on the type of the opcode; the others hold
// literals or operators.
func (c *Code) verifyOpcode(pc int, op code.Opcode) error {
	if op.HasType1() {
		return c.verifyRegisters(op.GetA(), op.GetB(), op.GetC())
	}
	switch op.TypePfx() {
	case code.Type2Pfx:
		return c.verifyRegisters(op.GetA(), op.GetB(), op.GetC())
	case code.Type7Pfx:
		if err := c.verifyNextJump(pc); err != nil {
			return err
		}
		switch p := op.GetP(); p {
		case code.OpPrepFor, code.OpAdvFor:
			return c.verifyRegisters(op.GetA(), op.GetB(), op.GetC())
		case code.OpIfEq, code.OpIfLt, code.OpIfLeq:
			if a := op.GetA(); a > 1 {
				return fmt.Errorf("predicate with A = %d, not 0 or 1", a)
			}
			return c.verifyRegisters(op.GetB(), op.GetC())
		default:
			return fmt.Errorf("unknown jump-deciding operation %d", p)
		}
	case code.Type8Pfx:
		if err := c.verifyConstant(op.GetK8().ToKIndex()); err != nil {
			return err
		}
		return c.verifyRegisters(op.GetA(), op.GetB())
	case code.Type6Pfx:
		return c.verifyRegisters(op.GetA(), op.GetB())
	case code.Type4Pfx:
		if !op.HasType4a() {
			return c.verifyRegisters(op.GetA())
		}
		return c.verifyRegisters(op.GetA(), op.GetB())
	case code.Type5Pfx:
		switch op.GetJ() {
		case code.OpJump, code.OpJumpIf:
			target := pc + int(op.GetOffset())
			if target < 0 || target >= len(c.code) {
				return fmt.Errorf("jump to %d, outside the code", target)
			}
		}
		return c.verifyRegisters(op.GetA())
	case code.Type3Pfx:
		if op.GetY().LoadsK() {
			if err := c.verifyConstant(op.GetKIndex()); err != nil {
				return err
			}
		}
		return c.verifyRegisters(op.GetA())
	default: // Type0Pfx
		return c.verifyRegisters(op.GetA())
	}
}

// verifyConstant checks that k is an index into the constant table.
func (c *Code) verifyConstant(k code.KIndex) error {
	if int(k) >= len(c.consts) {
		return fmt.Errorf("constant K%d outside the constant table (%d constants)", k, len(c.consts))
	}
	return nil
}

// verifyRegisters checks that each register is within the frame.
func (c *Code) verifyRegisters(regs ...code.Reg) error {
	for _, r := range regs {
		if int(r) >= int(c.RegCount) {
			return fmt.Errorf("register %s outside the frame (%d registers)", r, c.RegCount)
		}
	}
	return nil
}
