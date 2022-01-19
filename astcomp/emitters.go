package astcomp

import (
	"fmt"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/ir"
)

// These methods offer the convenience of not having to calculate the line
// number when emitting instructions.

func (c *compiler) emitInstr(l ast.Locator, instr ir.Instruction) {
	c.CodeBuilder.Emit(instr, getLine(l))
}

func (c *compiler) emitJump(l ast.Locator, lbl ir.Name) {
	if !c.CodeBuilder.EmitJump(lbl, getLine(l)) {
		panic(Error{
			Where:   l,
			Message: fmt.Sprintf("no visible label '%s'", lbl),
		})
	}
}

func (c *compiler) emitLoadConst(l ast.Locator, k ir.Constant, reg ir.Register) {
	ir.EmitConstant(c.CodeBuilder, k, reg, getLine(l))
}

func (c *compiler) emitMove(l ast.Locator, dst, src ir.Register) {
	ir.EmitMove(c.CodeBuilder, dst, src, getLine(l))
}

func getLine(l ast.Locator) int {
	if l != nil {
		locStart := l.Locate().StartPos()
		if locStart != nil {
			return locStart.Line
		}
	}
	return 0
}
