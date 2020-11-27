package ast

import (
	"github.com/arnodel/golua/token"
)

type Name struct {
	Location
	Val string
}

var _ Var = Name{}

func NewName(id *token.Token) Name {
	return Name{
		Location: LocFromToken(id),
		Val:      string(id.Lit),
	}
}

func (n Name) ProcessExp(p ExpProcessor) {
	p.ProcessNameExp(n)
}

func (n Name) ProcessVar(p VarProcessor) {
	p.ProcessNameVar(n)
}

func (n Name) HWrite(w HWriter) {
	w.Writef(n.Val)
}

func (n Name) FunctionName() string {
	return n.Val
}

func (n Name) AstString() String {
	return String{Location: n.Location, Val: []byte(n.Val)}
}
