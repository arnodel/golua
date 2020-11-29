package ast

import (
	"github.com/arnodel/golua/token"
)

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

func True(tok *token.Token) Bool {
	return Bool{Location: LocFromToken(tok), Val: true}
}

func False(tok *token.Token) Bool {
	return Bool{Location: LocFromToken(tok), Val: false}
}
