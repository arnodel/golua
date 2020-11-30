package ast

import (
	"github.com/arnodel/golua/token"
)

// Bool is an expression node representing a boolean literal.
type Bool struct {
	Location
	Val bool
}

var _ ExpNode = Bool{}

// HWrite prints a tree representation of the node.
func (b Bool) HWrite(w HWriter) {
	w.Writef("%t", b.Val)
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (b Bool) ProcessExp(p ExpProcessor) {
	p.ProcesBoolExp(b)
}

// True is a true boolean literal.
func True(tok *token.Token) Bool {
	return Bool{Location: LocFromToken(tok), Val: true}
}

// False is a false boolean literal.
func False(tok *token.Token) Bool {
	return Bool{Location: LocFromToken(tok), Val: false}
}
