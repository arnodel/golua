package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

type RepeatStat struct {
	Location
	CondStat
}

func NewRepeatStat(body BlockStat, cond ExpNode) (RepeatStat, error) {
	return RepeatStat{CondStat: CondStat{body: body, cond: cond}}, nil
}

func (s RepeatStat) HWrite(w HWriter) {
	w.Writef("repeat if: ")
	s.CondStat.HWrite(w)
}

func (s RepeatStat) CompileStat(c *ir.Compiler) {
	c.PushContext()
	c.DeclareGotoLabel(breakLblName)

	loopLbl := c.GetNewLabel()
	c.EmitLabel(loopLbl)
	s.body.CompileBlock(c)
	condReg := CompileExp(c, s.cond)
	negReg := c.GetFreeRegister()
	c.Emit(ir.Transform{Op: ops.OpNot, Dst: negReg, Src: condReg})
	c.Emit(ir.JumpIf{Cond: negReg, Label: loopLbl})

	c.EmitGotoLabel(breakLblName)
	c.PopContext()
}
