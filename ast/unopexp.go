package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
	"github.com/arnodel/golua/token"
)

type UnOp struct {
	Location
	Op      ops.Op
	Operand ExpNode
}

func NewUnOp(opTok *token.Token, op ops.Op, exp ExpNode) *UnOp {
	return &UnOp{
		Location: MergeLocations(LocFromToken(opTok), exp),
		Op:       op,
		Operand:  exp,
	}
}

func (u *UnOp) HWrite(w HWriter) {
	w.Writef("unop: %s", u.Op)
	w.Indent()
	w.Next()
	u.Operand.HWrite(w)
	w.Dedent()
}

func (u *UnOp) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	EmitInstr(c, u, ir.Transform{
		Op:  u.Op,
		Dst: dst,
		Src: CompileExp(c, u.Operand),
	})
	return dst
}
