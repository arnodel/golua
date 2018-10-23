package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type Bool struct {
	Location
	Val bool
}

func (b Bool) HWrite(w HWriter) {
	w.Writef("%t", b.Val)
}

func (b Bool) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	EmitLoadConst(c, b, ir.Bool(b.Val), dst)
	return dst
}

func True(tok *token.Token) Bool {
	return Bool{Location: LocFromToken(tok), Val: true}
}

func False(tok *token.Token) Bool {
	return Bool{Location: LocFromToken(tok), Val: false}
}
