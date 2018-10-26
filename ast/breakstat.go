package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type BreakStat struct {
	Location
}

func NewBreakStat(tok *token.Token) BreakStat {
	return BreakStat{Location: LocFromToken(tok)}
}

func (s BreakStat) HWrite(w HWriter) {
	w.Writef("break")
}

func (s BreakStat) CompileStat(c *ir.Compiler) {
	lbl, ok := c.GetGotoLabel(breakLblName)
	if !ok {
		panic("Cannot break from here")
	}
	EmitInstr(c, s, ir.Jump{Label: lbl})
}

var breakLblName = ir.Name("<break>")
