package ast

import (
	"github.com/arnodel/golua/token"
)

// WhileStat represents a while / end statement
type WhileStat struct {
	Location
	CondStat
}

var _ Stat = WhileStat{}

func NewWhileStat(whileTok, endTok *token.Token, cond ExpNode, body BlockStat) WhileStat {
	return WhileStat{
		Location: LocFromTokens(whileTok, endTok),
		CondStat: CondStat{Cond: cond, Body: body},
	}
}

func (s WhileStat) ProcessStat(p StatProcessor) {
	p.ProcessWhileStat(s)
}

func (s WhileStat) HWrite(w HWriter) {
	w.Writef("while: ")
	s.CondStat.HWrite(w)
}
