package ast

import "github.com/arnodel/golua/ir"

type LocalFunctionStat struct {
	Location
	Function
	name Name
}

func NewLocalFunctionStat(name Name, fx Function) LocalFunctionStat {
	fx.name = name.string
	return LocalFunctionStat{
		Location: MergeLocations(name, fx), // TODO: use "local" for location start
		Function: fx,
		name:     name,
	}
}

func (s LocalFunctionStat) HWrite(w HWriter) {
	w.Writef("local function ")
	s.name.HWrite(w)
	s.Function.HWrite(w)
}

func (s LocalFunctionStat) CompileStat(c *ir.Compiler) {
	fReg := c.GetFreeRegister()
	c.DeclareLocal(ir.Name(s.name.string), fReg)
	reg := s.Function.CompileExp(c, fReg)
	EmitMove(c, s, fReg, reg)

}
