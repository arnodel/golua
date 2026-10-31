package runtime

import (
	"errors"
	"fmt"

	"github.com/arnodel/golua/code"
)

// Register operands are 8 bits wide, so a frame cannot have more registers
// than this of either kind.
const maxRegCount = 256

// verify checks that running c cannot touch anything outside its frame: the
// register counts make sense, every register operand of every opcode is
// within them, and every jump lands inside the code on an opcode (not on one
// of the operand words that list a call's arguments).  The run loop relies on
// this to access registers without checking their index.
//
// Code produced by the compiler should always pass; code loaded from a binary
// chunk cannot be trusted to.
func (c *Code) verify() error {
	switch {
	case c.RegCount < 1 || c.RegCount > maxRegCount:
		return fmt.Errorf("invalid register count %d", c.RegCount)
	case c.CellCount < 0 || c.CellCount > maxRegCount:
		return fmt.Errorf("invalid cell count %d", c.CellCount)
	case c.UpvalueCount < 0 || c.UpvalueCount > c.CellCount:
		return fmt.Errorf("invalid upvalue count %d for %d cells", c.UpvalueCount, c.CellCount)
	case len(c.UpNames) != int(c.UpvalueCount):
		return fmt.Errorf("%d upvalue names for %d upvalues", len(c.UpNames), c.UpvalueCount)
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
	return nil
}

// callArgCount returns the number of arguments listed after op if it is a
// call, else 0.
func callArgCount(op code.Opcode) int {
	if op.HasType1() || op.TypePfx() != code.Type5Pfx || op.GetJ() != code.OpCall {
		return 0
	}
	return op.GetArgCount()
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
	case code.Type2Pfx, code.Type7Pfx:
		return c.verifyRegisters(op.GetA(), op.GetB(), op.GetC())
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
		if op.GetUnOp() == code.OpUpvalue && !op.GetB().IsCell() {
			return errors.New("upval needs a cell register")
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
		switch {
		case !r.IsCell() && int(r.Idx()) >= int(c.RegCount):
			return fmt.Errorf("register %s outside the frame (%d registers)", r, c.RegCount)
		case r.IsCell() && int(r.Idx()) >= int(c.CellCount):
			return fmt.Errorf("register %s outside the frame (%d cells)", r, c.CellCount)
		}
	}
	return nil
}
