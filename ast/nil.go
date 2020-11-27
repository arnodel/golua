package ast

import (
	"github.com/arnodel/golua/token"
)

type NilType struct {
	Location
}

var _ ExpNode = NilType{}

func (n NilType) HWrite(w HWriter) {
	w.Writef("nil")
}

func (n NilType) ProcessExp(p ExpProcessor) {
	p.ProcessNilExp(n)
}

func Nil(tok *token.Token) NilType {
	return NilType{Location: LocFromToken(tok)}
}
