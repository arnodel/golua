package ast

import "github.com/arnodel/golua/ir"

type NilType struct{}

var Nil NilType

func (n NilType) HWrite(w HWriter) {
	w.Writef("nil")
}

func (n NilType) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	ir.EmitConstant(c, ir.NilType{}, dst)
	return dst
}
