package ast

import "github.com/arnodel/golua/ir"

type FunctionName struct {
	nameList []Name
	method   Name
}

func (n FunctionName) HWrite(w HWriter) {
	for i, name := range n.nameList {
		if i > 0 {
			w.Writef(".")
		}
		w.Writef(string(name))
	}
	if n.method != "" {
		w.Writef(":%s", n.method)
	}
}

type Function struct {
	ParList
	body Stat
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

func (f Function) CompileExp(c *Compiler, dst ir.Register) ir.Register {
	fc := NewCompiler(c)
	recvRegs := make([]ir.Register, 1+len(f.params))
	callerReg := fc.GetFreeRegister()
	fc.DeclareLocal(Name("<caller>"), callerReg)
	recvRegs[0] = callerReg
	for i, p := range f.params {
		reg := fc.GetFreeRegister()
		fc.DeclareLocal(p, reg)
		recvRegs[i+1] = reg
	}
	if !f.hasDots {
		fc.Emit(ir.Receive{Dst: recvRegs})
	} else {
		reg := fc.GetFreeRegister()
		fc.DeclareLocal(Name("..."), reg)
		fc.Emit(ir.ReceiveEtc{Dst: recvRegs, Etc: reg})
	}
	f.body.CompileStat(fc)
	kidx := c.GetConstant(fc.code)
	c.Emit(ir.MkClosure{
		Dst:      dst,
		Code:     kidx,
		Upvalues: fc.upvalues,
	})
	return dst
}

type FunctionStat struct {
	Function
	name FunctionName
}

func (s FunctionStat) HWrite(w HWriter) {
	w.Writef("function ")
	s.name.HWrite(w)
	s.Function.HWrite(w)
}

func (s FunctionStat) CompileStat(c *Compiler) {
	// TODO
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

func (s LocalFunctionStat) CompileStat(c *Compiler) {
	reg := CompileExp(c, s.Function)
	c.DeclareLocal(s.name, reg)
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

func NewFunctionName(names []Name, method Name) (FunctionName, error) {
	return FunctionName{
		nameList: names,
		method:   method,
	}, nil
}

func NewFunction(parList ParList, body Stat) (Function, error) {
	return Function{ParList: parList, body: body}, nil
}

func NewFunctionStat(name FunctionName, fx Function) (FunctionStat, error) {
	return FunctionStat{Function: fx, name: name}, nil
}

func NewLocalFunctionStat(name Name, fx Function) (LocalFunctionStat, error) {
	return LocalFunctionStat{Function: fx, name: name}, nil
}
