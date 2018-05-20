package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type EtcType struct {
	Location
}

func Etc(tok *token.Token) EtcType {
	return EtcType{Location: LocFromToken(tok)}
}

func (e EtcType) HWrite(w HWriter) {
	w.Writef("...")
}

func (e EtcType) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	reg, ok := c.GetRegister(ir.Name("..."))
	if ok {
		return reg
	}
	panic("... not defined")
}
