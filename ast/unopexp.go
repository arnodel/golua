package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
	"github.com/arnodel/golua/token"
)

type UnOp struct {
	Location
	op      ops.Op
	operand ExpNode
}

func NewUnOp(opTok *token.Token, op ops.Op, exp ExpNode) (*UnOp, error) {
	return &UnOp{
		Location: MergeLocations(LocFromToken(opTok), exp),
		op:       op,
		operand:  exp,
	}, nil
}

func (u *UnOp) HWrite(w HWriter) {
	w.Writef("unop: %s", u.op)
	w.Indent()
	w.Next()
	u.operand.HWrite(w)
	w.Dedent()
}

func (u *UnOp) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	EmitInstr(c, u, ir.Transform{
		Op:  u.op,
		Dst: dst,
		Src: CompileExp(c, u.operand),
	})
	return dst
}
