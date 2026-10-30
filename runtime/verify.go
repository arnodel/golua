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
// within them, and every jump lands inside the code.  The run loop relies on
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
	for pc, op := range c.code {
		if err := c.verifyOpcode(pc, op); err != nil {
			return fmt.Errorf("opcode %d: %w", pc, err)
		}
	}
	return nil
}

// verifyOpcode checks the register operands of op, and its jump target if it
// is a jump.  Which of the fields A, B and C name registers depends on the
// type of the opcode; the others hold literals or operators.
func (c *Code) verifyOpcode(pc int, op code.Opcode) error {
	regs := [3]code.Reg{op.GetA(), op.GetB(), op.GetC()}
	n := 1 // Every type has a register in A
	switch {
	case op.HasType1():
		n = 3
	default:
		switch op.TypePfx() {
		case code.Type2Pfx, code.Type7Pfx:
			n = 3
		case code.Type6Pfx:
			n = 2
		case code.Type4Pfx:
			if op.HasType4a() {
				n = 2
				if op.GetUnOp() == code.OpUpvalue && !op.GetB().IsCell() {
					return errors.New("upval needs a cell register")
				}
			}
		case code.Type5Pfx:
			switch op.GetJ() {
			case code.OpJump, code.OpJumpIf:
				target := pc + int(op.GetOffset())
				if target < 0 || target >= len(c.code) {
					return fmt.Errorf("jump to %d, outside the code", target)
				}
			}
		case code.Type3Pfx, code.Type0Pfx:
		default:
			return errors.New("unknown opcode type")
		}
	}
	for _, r := range regs[:n] {
		if err := c.verifyReg(r); err != nil {
			return err
		}
	}
	return nil
}

func (c *Code) verifyReg(r code.Reg) error {
	if r.IsCell() {
		if int(r.Idx()) >= int(c.CellCount) {
			return fmt.Errorf("register %s outside the frame (%d cells)", r, c.CellCount)
		}
	} else if int(r.Idx()) >= int(c.RegCount) {
		return fmt.Errorf("register %s outside the frame (%d registers)", r, c.RegCount)
	}
	return nil
}
