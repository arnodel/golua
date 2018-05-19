package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type Bool struct {
	Location
	val bool
}

func (b Bool) HWrite(w HWriter) {
	w.Writef("%t", b.val)
}

func (b Bool) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	ir.EmitConstant(c, ir.Bool(b.val), dst)
	return dst
}

func True(tok *token.Token) Bool {
	return Bool{val: true}
}

func False(tok *token.Token) Bool {
	return Bool{val: false}
}
