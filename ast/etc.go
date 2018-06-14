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
	if !ok {
		panic("... not defined")
	}
	EmitInstr(c, e, ir.EtcLookup{Dst: dst, Etc: reg})
	return dst
}

func (e EtcType) CompileEtcExp(c *ir.Compiler, dst ir.Register) ir.Register {
	reg, ok := c.GetRegister(ir.Name("..."))
	if !ok {
		panic("... not defined")
	}
	return reg
}

func (e EtcType) CompileTailExp(c *ir.Compiler, dstRegs []ir.Register) {
	reg, ok := c.GetRegister(ir.Name("..."))
	if !ok {
		panic("... not defined")
	}
	for i, dst := range dstRegs {
		EmitInstr(c, e, ir.EtcLookup{
			Dst: dst,
			Etc: reg,
			Idx: i,
		})
	}
}
