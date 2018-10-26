package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type GotoStat struct {
	Location
	Label Name
}

func NewGotoStat(gotoTok *token.Token, lbl Name) GotoStat {
	return GotoStat{
		Location: MergeLocations(LocFromToken(gotoTok), lbl),
		Label:    lbl,
	}
}

func (s GotoStat) HWrite(w HWriter) {
	w.Writef("goto %s", s.Label)
}

func (s GotoStat) CompileStat(c *ir.Compiler) {
	lbl, ok := c.GetGotoLabel(ir.Name(s.Label.Val))
	if !ok {
		panic("Undefined label for goto")
	}
	EmitInstr(c, s, ir.Jump{Label: lbl})
}
