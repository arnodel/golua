package ast

import (
	"github.com/arnodel/golua/ops"
	"github.com/arnodel/golua/token"
)

type UnOp struct {
	Location
	Op      ops.Op
	Operand ExpNode
}

var _ ExpNode = UnOp{}

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
