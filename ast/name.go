package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type Name string

func NewName(id *token.Token) (Name, error) {
	return Name(id.Lit), nil
}

func (n Name) HWrite(w HWriter) {
	w.Writef(string(n))
}

func (n Name) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	reg, ok := c.GetRegister(ir.Name(n))
	if ok {
		return reg
	}
	return IndexExp{Name("_ENV"), String(n)}.CompileExp(c, dst)
}

func (n Name) CompileAssign(c *ir.Compiler, src ir.Register) {
	reg, ok := c.GetRegister(ir.Name(n))
	if ok {
		ir.EmitMove(c, reg, src)
		return
	}
	IndexExp{Name("_ENV"), String(n)}.CompileAssign(c, src)
}

func (n Name) AstString() String {
	return String(n)
}
