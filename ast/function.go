package ast

import (
	"github.com/arnodel/golua/token"
)

type Function struct {
	Location
	ParList
	Body BlockStat
	Name string
}

var _ ExpNode = Function{}

func NewFunction(startTok, endTok *token.Token, parList ParList, body BlockStat) Function {
	// Make sure we return at the end of the function
	if body.Return == nil {
		body.Return = []ExpNode{}
	}
	return Function{
		Location: LocFromTokens(startTok, endTok),
		ParList:  parList,
		Body:     body,
	}
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (f Function) ProcessExp(p ExpProcessor) {
	p.ProcessFunctionExp(f)
}

// HWrite prints a tree representation of the node.
func (f Function) HWrite(w HWriter) {
	w.Writef("(")
	for i, param := range f.Params {
		w.Writef(param.Val)
		if i < len(f.Params)-1 || f.HasDots {
			w.Writef(", ")
		}
	}
	if f.HasDots {
		w.Writef("...")
	}
	w.Writef(")")
	w.Indent()
	w.Next()
	f.Body.HWrite(w)
	w.Dedent()
}

type ParList struct {
	Params  []Name
	HasDots bool
}

func NewParList(params []Name, hasDots bool) ParList {
	return ParList{
		Params:  params,
		HasDots: hasDots,
	}
}
