package ast

import (
	"github.com/arnodel/golua/token"
)

type BreakStat struct {
	Location
}

var _ Stat = BreakStat{}

func NewBreakStat(tok *token.Token) BreakStat {
	return BreakStat{Location: LocFromToken(tok)}
}

func (s BreakStat) HWrite(w HWriter) {
	w.Writef("break")
}

func (s BreakStat) ProcessStat(p StatProcessor) {
	p.ProcessBreakStat(s)
}
