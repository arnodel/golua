package ast

import (
	"github.com/arnodel/golua/token"
)

type EmptyStat struct {
	Location
}

var _ Stat = EmptyStat{}

func NewEmptyStat(tok *token.Token) EmptyStat {
	return EmptyStat{Location: LocFromToken(tok)}
}

func (s EmptyStat) ProcessStat(p StatProcessor) {
	p.ProcessEmptyStat(s)
}

func (s EmptyStat) HWrite(w HWriter) {
	w.Writef("empty stat")
}
