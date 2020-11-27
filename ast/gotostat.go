package ast

import (
	"github.com/arnodel/golua/token"
)

type GotoStat struct {
	Location
	Label Name
}

var _ Stat = GotoStat{}

func NewGotoStat(gotoTok *token.Token, lbl Name) GotoStat {
	return GotoStat{
		Location: MergeLocations(LocFromToken(gotoTok), lbl),
		Label:    lbl,
	}
}

func (s GotoStat) ProcessStat(p StatProcessor) {
	p.ProcessGotoStat(s)
}

func (s GotoStat) HWrite(w HWriter) {
	w.Writef("goto %s", s.Label)
}
