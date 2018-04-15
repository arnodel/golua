package ast

import "github.com/arnodel/golua/ir"

type Bool bool

var True Bool = true
var False Bool = false

func (b Bool) HWrite(w HWriter) {
	w.Writef("%t", bool(b))
}

func (b Bool) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	ir.EmitConstant(c, ir.Bool(b), dst)
	return dst
}
