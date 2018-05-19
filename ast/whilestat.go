package ast

import "github.com/arnodel/golua/ir"

type WhileStat struct {
	Location
	CondStat
}

func NewWhileStat(cond ExpNode, body BlockStat) (WhileStat, error) {
	return WhileStat{CondStat: CondStat{cond: cond, body: body}}, nil
}

func (s WhileStat) HWrite(w HWriter) {
	w.Writef("while: ")
	s.CondStat.HWrite(w)
}

func (s WhileStat) CompileStat(c *ir.Compiler) {
	c.PushContext()
	stopLbl := c.DeclareGotoLabel(breakLblName)

	loopLbl := c.GetNewLabel()
	c.EmitLabel(loopLbl)

	s.CondStat.CompileCond(c, stopLbl)

	c.Emit(ir.Jump{Label: loopLbl})

	c.EmitGotoLabel(breakLblName)
	c.PopContext()
}
