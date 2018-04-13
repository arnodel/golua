package ast

import (
	"github.com/arnodel/golua/ir"
)

type FunctionName struct {
	name   Var
	method Name
}

func (n FunctionName) HWrite(w HWriter) {
	n.name.HWrite(w)
	if n.method != "" {
		w.Writef(":%s", n.method)
	}
}

type Function struct {
	ParList
	body BlockStat
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

func (f Function) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	fc := c.NewChild()
	recvRegs := make([]ir.Register, 1+len(f.params))
	callerReg := fc.GetFreeRegister()
	fc.DeclareLocal(ir.Name(Name("<caller>")), callerReg)
	recvRegs[0] = callerReg
	for i, p := range f.params {
		reg := fc.GetFreeRegister()
		fc.DeclareLocal(ir.Name(p), reg)
		recvRegs[i+1] = reg
	}
	if !f.hasDots {
		fc.Emit(ir.Receive{Dst: recvRegs})
	} else {
		reg := fc.GetFreeRegister()
		fc.DeclareLocal(ir.Name(Name("...")), reg)
		fc.Emit(ir.ReceiveEtc{Dst: recvRegs, Etc: reg})
	}

	// Need to make sure there is a return instruction emitted at the
	// end.
	body := f.body
	if body.returnValues == nil {
		body.returnValues = []ExpNode{}
	}
	body.CompileBlock(fc)

	kidx := c.GetConstant(fc.GetCode())
	c.Emit(ir.MkClosure{
		Dst:      dst,
		Code:     kidx,
		Upvalues: fc.Upvalues(),
	})
	return dst
}

type LocalFunctionStat struct {
	Function
	name Name
}

func (s LocalFunctionStat) HWrite(w HWriter) {
	w.Writef("local function ")
	s.name.HWrite(w)
	s.Function.HWrite(w)
}

func (s LocalFunctionStat) CompileStat(c *ir.Compiler) {
	fReg := c.GetFreeRegister()
	c.DeclareLocal(ir.Name(s.name), fReg)
	reg := s.Function.CompileExp(c, fReg)
	ir.EmitMove(c, fReg, reg)

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

func NewFunctionName(name Var, method Name) (FunctionName, error) {
	return FunctionName{
		name:   name,
		method: method,
	}, nil
}

func NewFunction(parList ParList, body BlockStat) (Function, error) {
	// Make sure we return at the end of the function
	if body.returnValues == nil {
		body.returnValues = []ExpNode{}
	}
	return Function{ParList: parList, body: body}, nil
}

func NewFunctionStat(name FunctionName, fx Function) (AssignStat, error) {
	fName := name.name
	if name.method != "" {
		fx, _ = NewFunction(
			ParList{append([]Name{"self"}, fx.params...), fx.hasDots},
			fx.body,
		)
		fName, _ = NewIndexExp(name.name, String(name.method))
	}
	return NewAssignStat([]Var{fName}, []ExpNode{fx})
}

func NewLocalFunctionStat(name Name, fx Function) (LocalFunctionStat, error) {
	return LocalFunctionStat{Function: fx, name: name}, nil
}
