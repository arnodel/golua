package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type Name struct {
	Location
	Val string
}

func NewName(id *token.Token) Name {
	return Name{
		Location: LocFromToken(id),
		Val:      string(id.Lit),
	}
}

func (n Name) HWrite(w HWriter) {
	w.Writef(n.Val)
}

func (n Name) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	reg, ok := c.GetRegister(ir.Name(n.Val))
	if ok {
		return reg
	}
	return IndexExp{
		Location: n.Location,
		Coll:     Name{Location: n.Location, Val: "_ENV"},
		Idx:      n.AstString(),
	}.CompileExp(c, dst)
}

func (n Name) CompileAssign(c *ir.Compiler) Assign {
	reg, ok := c.GetRegister(ir.Name(n.Val))
	if ok {
		return func(src ir.Register) {
			EmitMove(c, n, reg, src)
		}
	}
	return IndexExp{
		Location: n.Location,
		Coll:     Name{Location: n.Location, Val: "_ENV"},
		Idx:      n.AstString(),
	}.CompileAssign(c)
}

func (n Name) FunctionName() string {
	return n.Val
}

func (n Name) AstString() String {
	return String{Location: n.Location, Val: []byte(n.Val)}
}
