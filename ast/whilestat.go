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

// NewWhileStat returns a WhileStat instance representing the statement "while
// <cond> do <body> done".
func NewWhileStat(whileTok, endTok *token.Token, cond ExpNode, body BlockStat) WhileStat {
	return WhileStat{
		Location: LocFromTokens(whileTok, endTok),
		CondStat: CondStat{Cond: cond, Body: body},
	}
}

// ProcessStat uses the given StatProcessor to process the receiver.
func (s WhileStat) ProcessStat(p StatProcessor) {
	p.ProcessWhileStat(s)
}

// HWrite prints a tree representation of the node.
func (s WhileStat) HWrite(w HWriter) {
	w.Writef("while: ")
	s.CondStat.HWrite(w)
}
