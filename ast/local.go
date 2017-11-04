package ast

import "github.com/arnodel/golua/ir"

type LocalStat struct {
	names  []Name
	values []ExpNode
}

func (s LocalStat) HWrite(w HWriter) {
	w.Writef("local")
	w.Indent()
	for i, name := range s.names {
		w.Next()
		w.Writef("name_%d: %s", i, name)
	}
	for i, val := range s.values {
		w.Next()
		w.Writef("val_%d: ", i)
		val.HWrite(w)
	}
	w.Dedent()
}

func (s LocalStat) CompileStat(c *Compiler) {
	nameCount := len(s.names)
	valueCount := len(s.values)
	commonCount := nameCount
	if commonCount > valueCount {
		commonCount = valueCount
	}
	var fCall FunctionCall
	doFCall := false
	if nameCount < valueCount {
		fCall, doFCall = s.values[valueCount-1].(FunctionCall)
		if doFCall {
			commonCount--
		}
	}
	localRegs := make([]ir.Register, len(s.names))
	for i := 0; i < commonCount; i++ {
		localReg := c.GetFreeRegister()
		reg := s.values[i].CompileExp(c, localReg)
		EmmitMove(c, localReg, reg)
		c.TakeRegister(localReg)
		localRegs[i] = localReg
	}
	for i := commonCount; i < nameCount; i++ {
		localReg := c.GetFreeRegister()
		c.TakeRegister(localReg)
		localRegs[i] = localReg
	}
	if doFCall {
		fCall.CompileCall(c)
		c.Emit(ir.Receive{Dst: localRegs[commonCount:]})
	} else if nameCount > valueCount {
		nilK := c.GetConstant(ir.NilType{})
		for i := valueCount; i < nameCount; i++ {
			EmitConstant(c, nilK, localRegs[i])
		}
	}
	for i, reg := range localRegs {
		c.ReleaseRegister(reg)
		c.DeclareLocal(s.names[i], reg)
	}
}

func NewLocalStat(names []Name, values []ExpNode) (LocalStat, error) {
	return LocalStat{names: names, values: values}, nil
}
