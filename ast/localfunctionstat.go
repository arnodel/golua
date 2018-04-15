package ast

import "github.com/arnodel/golua/ir"

type LocalFunctionStat struct {
	Function
	name Name
}

func NewLocalFunctionStat(name Name, fx Function) (LocalFunctionStat, error) {
	return LocalFunctionStat{Function: fx, name: name}, nil
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
