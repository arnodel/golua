package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type Name struct {
	Location
	string
}

func NewName(id *token.Token) (Name, error) {
	return Name{string: string(id.Lit)}, nil
}

func (n Name) HWrite(w HWriter) {
	w.Writef(n.string)
}

func (n Name) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	reg, ok := c.GetRegister(ir.Name(n.string))
	if ok {
		return reg
	}
	return IndexExp{
		collection: Name{string: "_ENV"},
		index:      n.AstString(),
	}.CompileExp(c, dst)
}

func (n Name) CompileAssign(c *ir.Compiler, src ir.Register) {
	reg, ok := c.GetRegister(ir.Name(n.string))
	if ok {
		ir.EmitMove(c, reg, src)
		return
	}
	IndexExp{
		collection: Name{string: "_ENV"},
		index:      n.AstString(),
	}.CompileAssign(c, src)
}

func (n Name) AstString() String {
	return String{val: []byte(n.string)}
}
