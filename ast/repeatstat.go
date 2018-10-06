package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
	"github.com/arnodel/golua/token"
)

type RepeatStat struct {
	Location
	CondStat
}

func NewRepeatStat(repTok *token.Token, body BlockStat, cond ExpNode) (RepeatStat, error) {
	return RepeatStat{
		Location: MergeLocations(LocFromToken(repTok), cond),
		CondStat: CondStat{body: body, cond: cond},
	}, nil
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
	pop := s.body.CompileBlockNoPop(c)
	condReg := CompileExp(c, s.cond)
	negReg := c.GetFreeRegister()
	EmitInstr(c, s.cond, ir.Transform{Op: ops.OpNot, Dst: negReg, Src: condReg})
	EmitInstr(c, s.cond, ir.JumpIf{Cond: negReg, Label: loopLbl})
	pop()

	c.EmitGotoLabel(breakLblName)
	c.PopContext()
}
