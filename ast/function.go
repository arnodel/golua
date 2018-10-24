package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type Function struct {
	Location
	ParList
	Body BlockStat
	Name string
}

func NewFunction(startTok, endTok *token.Token, parList ParList, body BlockStat) Function {
	// Make sure we return at the end of the function
	if body.returnValues == nil {
		body.returnValues = []ExpNode{}
	}
	return Function{
		Location: LocFromTokens(startTok, endTok),
		ParList:  parList,
		Body:     body,
	}
}

func (f Function) HWrite(w HWriter) {
	w.Writef("(")
	for i, param := range f.Params {
		w.Writef(param.Val)
		if i < len(f.Params)-1 || f.HasDots {
			w.Writef(", ")
		}
	}
	if f.HasDots {
		w.Writef("...")
	}
	w.Writef(")")
	w.Indent()
	w.Next()
	f.Body.HWrite(w)
	w.Dedent()
}

func (f Function) CompileBody(c *ir.Compiler) {
	recvRegs := make([]ir.Register, len(f.Params))
	callerReg := c.GetFreeRegister()
	c.DeclareLocal("<caller>", callerReg)
	for i, p := range f.Params {
		reg := c.GetFreeRegister()
		c.DeclareLocal(ir.Name(p.Val), reg)
		recvRegs[i] = reg
	}
	if !f.HasDots {
		EmitInstr(c, f, ir.Receive{Dst: recvRegs})
	} else {
		reg := c.GetFreeRegister()
		c.DeclareLocal("...", reg)
		EmitInstr(c, f, ir.ReceiveEtc{Dst: recvRegs, Etc: reg})
	}

	// Need to make sure there is a return instruction emitted at the
	// end.
	body := f.Body
	if body.returnValues == nil {
		body.returnValues = []ExpNode{}
	}
	body.CompileBlock(c)
}

func (f Function) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	fc := c.NewChild()
	f.CompileBody(fc)
	kidx := c.GetConstant(fc.GetCode(f.Name))
	EmitInstr(c, f, ir.MkClosure{
		Dst:      dst,
		Code:     kidx,
		Upvalues: fc.Upvalues(),
	})
	return dst
}

type ParList struct {
	Params  []Name
	HasDots bool
}

func NewParList(params []Name, hasDots bool) ParList {
	return ParList{
		Params:  params,
		HasDots: hasDots,
	}
}
