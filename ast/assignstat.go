package ast

import "github.com/arnodel/golua/ir"

type AssignStat struct {
	dst []Var
	src []ExpNode
}

func NewAssignStat(dst []Var, src []ExpNode) (AssignStat, error) {
	return AssignStat{
		dst: dst,
		src: src,
	}, nil
}

func (s AssignStat) HWrite(w HWriter) {
	w.Writef("assign")
	w.Indent()
	for i, v := range s.dst {
		w.Next()
		w.Writef("dst_%d: ", i)
		v.HWrite(w)
	}
	for i, n := range s.src {
		w.Next()
		w.Writef("src_%d: ", i)
		n.HWrite(w)
	}
	w.Dedent()
}

func (s AssignStat) CompileStat(c *ir.Compiler) {
	resultRegs := make([]ir.Register, len(s.dst))
	CompileExpList(c, s.src, resultRegs)
	for i, reg := range resultRegs {
		c.ReleaseRegister(reg)
		s.dst[i].CompileAssign(c, reg)
	}
}
