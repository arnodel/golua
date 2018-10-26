package ast

import "github.com/arnodel/golua/ir"

type LocalFunctionStat struct {
	Location
	Function
	Name Name
}

func NewLocalFunctionStat(name Name, fx Function) LocalFunctionStat {
	fx.Name = name.Val
	return LocalFunctionStat{
		Location: MergeLocations(name, fx), // TODO: use "local" for location start
		Function: fx,
		Name:     name,
	}
}

func (s LocalFunctionStat) HWrite(w HWriter) {
	w.Writef("local function ")
	s.Name.HWrite(w)
	s.Function.HWrite(w)
}

func (s LocalFunctionStat) CompileStat(c *ir.Compiler) {
	fReg := c.GetFreeRegister()
	c.DeclareLocal(ir.Name(s.Name.Val), fReg)
	reg := s.Function.CompileExp(c, fReg)
	EmitMove(c, s, fReg, reg)

}
