package ast

import (
	"github.com/arnodel/golua/ir"
)

type Function struct {
	ParList
	body BlockStat
}

func NewFunction(parList ParList, body BlockStat) (Function, error) {
	// Make sure we return at the end of the function
	if body.returnValues == nil {
		body.returnValues = []ExpNode{}
	}
	return Function{ParList: parList, body: body}, nil
}

func (f Function) HWrite(w HWriter) {
	w.Writef("(")
	for i, param := range f.params {
		w.Writef(string(param))
		if i < len(f.params)-1 || f.hasDots {
			w.Writef(", ")
		}
	}
	if f.hasDots {
		w.Writef("...")
	}
	w.Writef(")")
	w.Indent()
	w.Next()
	f.body.HWrite(w)
	w.Dedent()
}

func (f Function) CompileBody(c *ir.Compiler) {
	recvRegs := make([]ir.Register, 1+len(f.params))
	callerReg := c.GetFreeRegister()
	c.DeclareLocal("<caller>", callerReg)
	recvRegs[0] = callerReg
	for i, p := range f.params {
		reg := c.GetFreeRegister()
		c.DeclareLocal(ir.Name(p), reg)
		recvRegs[i+1] = reg
	}
	if !f.hasDots {
		c.Emit(ir.Receive{Dst: recvRegs})
	} else {
		reg := c.GetFreeRegister()
		c.DeclareLocal("...", reg)
		c.Emit(ir.ReceiveEtc{Dst: recvRegs, Etc: reg})
	}

	// Need to make sure there is a return instruction emitted at the
	// end.
	body := f.body
	if body.returnValues == nil {
		body.returnValues = []ExpNode{}
	}
	body.CompileBlock(c)
}

func (f Function) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	fc := c.NewChild()
	f.CompileBody(fc)
	kidx := c.GetConstant(fc.GetCode())
	c.Emit(ir.MkClosure{
		Dst:      dst,
		Code:     kidx,
		Upvalues: fc.Upvalues(),
	})
	return dst
}

type ParList struct {
	params  []Name
	hasDots bool
}

func NewParList(params []Name, hasDots bool) (ParList, error) {
	return ParList{
		params:  params,
		hasDots: hasDots,
	}, nil
}
