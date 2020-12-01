package ast

import (
	"github.com/arnodel/golua/ops"
	"github.com/arnodel/golua/token"
)

// An UnOp is an expression node representing the application of a unary
// operation on an expression.
type UnOp struct {
	Location
	Op      ops.Op
	Operand ExpNode
}

var _ ExpNode = UnOp{}

// NewUnOp returns a UnOp instance from the given operator an expression.
func NewUnOp(opTok *token.Token, op ops.Op, exp ExpNode) *UnOp {
	return &UnOp{
		Location: MergeLocations(LocFromToken(opTok), exp),
		Op:       op,
		Operand:  exp,
	}
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (u UnOp) ProcessExp(p ExpProcessor) {
	p.ProcessUnOpExp(u)
}

// HWrite prints a tree representation of the node.
func (u UnOp) HWrite(w HWriter) {
	w.Writef("unop: %s", u.Op)
	w.Indent()
	w.Next()
	u.Operand.HWrite(w)
	w.Dedent()
}

func (u UnOp) isNumber() bool {
	switch u.Op {
	case ops.OpNeg, ops.OpId:
		return IsNumber(u.Operand)
	}
	return false
}
