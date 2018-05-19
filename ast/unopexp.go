package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

type UnOp struct {
	Location
	op      ops.Op
	operand ExpNode
}

func NewUnOp(op ops.Op, exp ExpNode) (*UnOp, error) {
	return &UnOp{
		op:      op,
		operand: exp,
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
	c.Emit(ir.Transform{
		Op:  u.op,
		Dst: dst,
		Src: CompileExp(c, u.operand),
	})
	return dst
}
