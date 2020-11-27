package ast

import (
	"github.com/arnodel/golua/token"
)

// RepeatStat represents a repeat / until statement.
type RepeatStat struct {
	Location
	CondStat
}

var _ Stat = RepeatStat{}

func NewRepeatStat(repTok *token.Token, body BlockStat, cond ExpNode) RepeatStat {
	return RepeatStat{
		Location: MergeLocations(LocFromToken(repTok), cond),
		CondStat: CondStat{Body: body, Cond: cond},
	}
}

func (s RepeatStat) ProcessStat(p StatProcessor) {
	p.ProcessRepeatStat(s)
}

func (s RepeatStat) HWrite(w HWriter) {
	w.Writef("repeat if: ")
	s.CondStat.HWrite(w)
}
