package ast

import (
	"github.com/arnodel/golua/token"
)

type NilType struct {
	Location
}

var _ ExpNode = NilType{}

// HWrite prints a tree representation of the node.
func (n NilType) HWrite(w HWriter) {
	w.Writef("nil")
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (n NilType) ProcessExp(p ExpProcessor) {
	p.ProcessNilExp(n)
}

func Nil(tok *token.Token) NilType {
	return NilType{Location: LocFromToken(tok)}
}
