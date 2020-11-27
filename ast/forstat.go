package ast

import (
	"github.com/arnodel/golua/token"
)

type ForStat struct {
	Location
	Var   Name
	Start ExpNode
	Stop  ExpNode
	Step  ExpNode
	Body  BlockStat
}

var _ Stat = ForStat{}

func NewForStat(startTok, endTok *token.Token, itervar Name, params []ExpNode, body BlockStat) *ForStat {
	return &ForStat{
		Location: LocFromTokens(startTok, endTok),
		Var:      itervar,
		Start:    params[0],
		Stop:     params[1],
		Step:     params[2],
		Body:     body,
	}
}

func (s ForStat) HWrite(w HWriter) {
	w.Writef("for %s", s.Var)
	w.Indent()
	if s.Start != nil {
		w.Next()
		w.Writef("Start: ")
		s.Start.HWrite(w)
	}
	if s.Stop != nil {
		w.Next()
		w.Writef("Stop: ")
		s.Stop.HWrite(w)
	}
	if s.Step != nil {
		w.Next()
		w.Writef("Step: ")
		s.Step.HWrite(w)
	}
	w.Next()
	s.Body.HWrite(w)
	w.Dedent()
}

func (s ForStat) ProcessStat(p StatProcessor) {
	p.ProcessForStat(s)
}
