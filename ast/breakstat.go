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

// HWrite prints a tree representation of the node.
func (s BreakStat) HWrite(w HWriter) {
	w.Writef("break")
}

// ProcessStat uses the given StatProcessor to process the receiver.
func (s BreakStat) ProcessStat(p StatProcessor) {
	p.ProcessBreakStat(s)
}
