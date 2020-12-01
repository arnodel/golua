package ast

import (
	"github.com/arnodel/golua/token"
)

// RepeatStat ia a statement expression that represents a repeat / until statement.
type RepeatStat struct {
	Location
	CondStat
}

var _ Stat = RepeatStat{}

// NewRepeatStat returns a RepeatStat instance representing the statement
// "repeat <body> until <cond>".
func NewRepeatStat(repTok *token.Token, body BlockStat, cond ExpNode) RepeatStat {
	return RepeatStat{
		Location: MergeLocations(LocFromToken(repTok), cond),
		CondStat: CondStat{Body: body, Cond: cond},
	}
}

// ProcessStat uses the given StatProcessor to process the receiver.
func (s RepeatStat) ProcessStat(p StatProcessor) {
	p.ProcessRepeatStat(s)
}

// HWrite prints a tree representation of the node.
func (s RepeatStat) HWrite(w HWriter) {
	w.Writef("repeat if: ")
	s.CondStat.HWrite(w)
}
