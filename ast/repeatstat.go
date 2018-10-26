package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
	"github.com/arnodel/golua/token"
)

// RepeatStat represents a repeat / until statement.
type RepeatStat struct {
	Location
	CondStat
}

func NewRepeatStat(repTok *token.Token, body BlockStat, cond ExpNode) RepeatStat {
	return RepeatStat{
		Location: MergeLocations(LocFromToken(repTok), cond),
		CondStat: CondStat{Body: body, Cond: cond},
	}
}

func (s RepeatStat) HWrite(w HWriter) {
	w.Writef("repeat if: ")
	s.CondStat.HWrite(w)
}

// CompileStat implements Stat.CompileStat.
func (s RepeatStat) CompileStat(c *ir.Compiler) {
	c.PushContext()
	c.DeclareGotoLabel(breakLblName)

	loopLbl := c.GetNewLabel()
	c.EmitLabel(loopLbl)
	pop := s.Body.CompileBlockNoPop(c)
	condReg := CompileExp(c, s.Cond)
	negReg := c.GetFreeRegister()
	EmitInstr(c, s.Cond, ir.Transform{Op: ops.OpNot, Dst: negReg, Src: condReg})
	EmitInstr(c, s.Cond, ir.JumpIf{Cond: negReg, Label: loopLbl})
	pop()

	c.EmitGotoLabel(breakLblName)
	c.PopContext()
}
