package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type GotoStat struct {
	Location
	label Name
}

func NewGotoStat(gotoTok *token.Token, lbl Name) (GotoStat, error) {
	return GotoStat{
		Location: MergeLocations(LocFromToken(gotoTok), lbl),
		label:    lbl,
	}, nil
}

func (s GotoStat) HWrite(w HWriter) {
	w.Writef("goto %s", s.label)
}

func (s GotoStat) CompileStat(c *ir.Compiler) {
	lbl, ok := c.GetGotoLabel(ir.Name(s.label.string))
	if !ok {
		panic("Undefined label for goto")
	}
	EmitInstr(c, s, ir.Jump{Label: lbl})
}
