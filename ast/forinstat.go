package ast

import (
	"github.com/arnodel/golua/token"
)

// ForInStat is a statement node representing the "for Vars in Params do Body end"
// statement.
type ForInStat struct {
	Location
	Vars   []Name
	Params []ExpNode
	Body   BlockStat
}

var _ Stat = ForInStat{}

// NewForInStat returns a ForInStat instance from the given parts.
func NewForInStat(startTok, endTok *token.Token, itervars []Name, params []ExpNode, body BlockStat) *ForInStat {
	return &ForInStat{
		Location: LocFromTokens(startTok, endTok),
		Vars:     itervars,
		Params:   params,
		Body:     body,
	}
}

// ProcessStat uses the given StatProcessor to process the receiver.
func (s ForInStat) ProcessStat(p StatProcessor) {
	p.ProcessForInStat(s)
}

// HWrite prints a tree representation of the node.
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
