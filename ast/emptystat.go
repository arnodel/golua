package ast

import (
	"github.com/arnodel/golua/token"
)

// EmptyStat is a statement expression containing an empty statement.
type EmptyStat struct {
	Location
}

var _ Stat = EmptyStat{}

// NewEmptyStat returns an EmptyStat instance (located from the given token).
func NewEmptyStat(tok *token.Token) EmptyStat {
	return EmptyStat{Location: LocFromToken(tok)}
}

// ProcessStat uses the given StatProcessor to process the receiver.
func (s EmptyStat) ProcessStat(p StatProcessor) {
	p.ProcessEmptyStat(s)
}

// HWrite prints a tree representation of the node.
func (s EmptyStat) HWrite(w HWriter) {
	w.Writef("empty stat")
}
