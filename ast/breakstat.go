package ast

import (
	"github.com/arnodel/golua/token"
)

// BreakStat is a statement node representing the "break" statement.
type BreakStat struct {
	Location
}

var _ Stat = BreakStat{}

// NewBreakStat returns a BreakStat instance (the token is needed to record the
// location of the statement).
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
