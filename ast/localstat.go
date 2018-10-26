package ast

import "github.com/arnodel/golua/ir"

type LocalStat struct {
	Location
	Names  []Name
	Values []ExpNode
}

func NewLocalStat(names []Name, values []ExpNode) LocalStat {
	loc := MergeLocations(names[0], names[len(names)-1])
	if len(values) > 0 {
		loc = MergeLocations(loc, values[len(values)-1])
	}
	return LocalStat{Location: loc, Names: names, Values: values}
}

func (s LocalStat) HWrite(w HWriter) {
	w.Writef("local")
	w.Indent()
	for i, name := range s.Names {
		w.Next()
		w.Writef("name_%d: %s", i, name)
	}
	for i, val := range s.Values {
		w.Next()
		w.Writef("val_%d: ", i)
		val.HWrite(w)
	}
	w.Dedent()
}

func (s LocalStat) CompileStat(c *ir.Compiler) {
	localRegs := make([]ir.Register, len(s.Names))
	CompileExpList(c, s.Values, localRegs)
	for i, reg := range localRegs {
		c.ReleaseRegister(reg)
		c.DeclareLocal(ir.Name(s.Names[i].Val), reg)
	}
}
