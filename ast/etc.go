package ast

import (
	"github.com/arnodel/golua/ir"
)

type EtcType struct{}

var Etc EtcType

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
