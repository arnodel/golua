package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type WhileStat struct {
	Location
	CondStat
}

func NewWhileStat(whileTok, endTok *token.Token, cond ExpNode, body BlockStat) WhileStat {
	return WhileStat{
		Location: LocFromTokens(whileTok, endTok),
		CondStat: CondStat{cond: cond, body: body},
	}
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

	EmitInstr(c, s, ir.Jump{Label: loopLbl}) // TODO: better location

	c.EmitGotoLabel(breakLblName)
	c.PopContext()
}
