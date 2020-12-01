package ast

import (
	"github.com/arnodel/golua/token"
)

// Nil is an expression node representing "nil".
type Nil struct {
	Location
}

var _ ExpNode = Nil{}

// HWrite prints a tree representation of the node.
func (n Nil) HWrite(w HWriter) {
	w.Writef("nil")
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (n Nil) ProcessExp(p ExpProcessor) {
	p.ProcessNilExp(n)
}

// NewNil returns a Nil instance located where the given token is.
func NewNil(tok *token.Token) Nil {
	return Nil{Location: LocFromToken(tok)}
}
