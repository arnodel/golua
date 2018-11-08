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
	EmitJump(c, s, ir.Name(s.Label.Val))
}
