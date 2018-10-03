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
	return Name{
		Location: LocFromToken(id),
		string:   string(id.Lit),
	}, nil
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
		Location:   n.Location,
		collection: Name{Location: n.Location, string: "_ENV"},
		index:      n.AstString(),
	}.CompileExp(c, dst)
}

func (n Name) CompileAssign(c *ir.Compiler) Assign {
	reg, ok := c.GetRegister(ir.Name(n.string))
	if ok {
		return func(src ir.Register) {
			EmitMove(c, n, reg, src)
		}
	}
	return IndexExp{
		Location:   n.Location,
		collection: Name{Location: n.Location, string: "_ENV"},
		index:      n.AstString(),
	}.CompileAssign(c)
}

func (n Name) FunctionName() string {
	return n.string
}

func (n Name) AstString() String {
	return String{Location: n.Location, val: []byte(n.string)}
}
