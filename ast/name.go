package ast

import (
	"github.com/arnodel/golua/token"
)

// Name is an expression token representing a name (identifier).
type Name struct {
	Location
	Val string
}

var _ Var = Name{}

// NewName returns a Name instance with value taken from the given token.
func NewName(id *token.Token) Name {
	return Name{
		Location: LocFromToken(id),
		Val:      string(id.Lit),
	}
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (n Name) ProcessExp(p ExpProcessor) {
	p.ProcessNameExp(n)
}

// ProcessVar process the receiver using the given VarProcessor.
func (n Name) ProcessVar(p VarProcessor) {
	p.ProcessNameVar(n)
}

// HWrite prints a tree representation of the node.
func (n Name) HWrite(w HWriter) {
	w.Writef(n.Val)
}

// FunctionName returns the string associated with the name.
func (n Name) FunctionName() string {
	return n.Val
}

// AstString returns a String with the same value and location as the receiver.
func (n Name) AstString() String {
	return String{Location: n.Location, Val: []byte(n.Val)}
}
