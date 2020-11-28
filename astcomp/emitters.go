package astcomp

import (
	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/ir"
)

func (c *Compiler) EmitInstr(l ast.Locator, instr ir.Instruction) {
	c.Builder.Emit(instr, getLine(l))
}

func (c *Compiler) EmitJump(l ast.Locator, lbl ir.Name) {
	c.Builder.EmitJump(lbl, getLine(l))
}

func (c *Compiler) EmitLoadConst(l ast.Locator, k ir.Constant, reg ir.Register) {
	ir.EmitConstant(c.Builder, k, reg, getLine(l))
}

func (c *Compiler) EmitMove(l ast.Locator, dst, src ir.Register) {
	ir.EmitMove(c.Builder, dst, src, getLine(l))
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
