package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type NilType struct {
	Location
}

func (n NilType) HWrite(w HWriter) {
	w.Writef("nil")
}

func (n NilType) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	EmitLoadConst(c, n, ir.NilType{}, dst)
	return dst
}

func Nil(tok *token.Token) NilType {
	return NilType{Location: LocFromToken(tok)}
}
