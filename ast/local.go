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

func (s LocalStat) CompileStat(c *ir.Compiler) {
	localRegs := make([]ir.Register, len(s.names))
	CompileExpList(c, s.values, localRegs)
	for i, reg := range localRegs {
		c.ReleaseRegister(reg)
		c.DeclareLocal(ir.Name(s.names[i]), reg)
	}
}

func NewLocalStat(names []Name, values []ExpNode) (LocalStat, error) {
	return LocalStat{names: names, values: values}, nil
}
