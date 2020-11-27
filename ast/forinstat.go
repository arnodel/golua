package ast

import (
	"github.com/arnodel/golua/token"
)

type ForInStat struct {
	Location
	Vars   []Name
	Params []ExpNode
	Body   BlockStat
}

var _ Stat = ForInStat{}

func NewForInStat(startTok, endTok *token.Token, itervars []Name, params []ExpNode, body BlockStat) *ForInStat {
	return &ForInStat{
		Location: LocFromTokens(startTok, endTok),
		Vars:     itervars,
		Params:   params,
		Body:     body,
	}
}

func (s ForInStat) ProcessStat(p StatProcessor) {
	p.ProcessForInStat(s)
}

func (s ForInStat) HWrite(w HWriter) {
	w.Writef("for in")
	w.Indent()
	for i, v := range s.Vars {
		w.Next()
		w.Writef("var_%d: ", i)
		v.HWrite(w)
	}
	for i, p := range s.Params {
		w.Next()
		w.Writef("param_%d", i)
		p.HWrite(w)
	}
	w.Next()
	w.Writef("Body: ")
	s.Body.HWrite(w)
	w.Dedent()
}
