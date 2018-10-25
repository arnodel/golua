package ast

import "github.com/arnodel/golua/ir"

type AssignStat struct {
	Location
	Dest []Var
	Src  []ExpNode
}

func NewAssignStat(dst []Var, src []ExpNode) AssignStat {
	return AssignStat{
		Location: MergeLocations(dst[0], src[len(src)-1]),
		Dest:     dst,
		Src:      src,
	}
}

func (s AssignStat) HWrite(w HWriter) {
	w.Writef("assign")
	w.Indent()
	for i, v := range s.Dest {
		w.Next()
		w.Writef("dst_%d: ", i)
		v.HWrite(w)
	}
	for i, n := range s.Src {
		w.Next()
		w.Writef("src_%d: ", i)
		n.HWrite(w)
	}
	w.Dedent()
}

func (s AssignStat) CompileStat(c *ir.Compiler) {

	// Evaluate the right hand side
	resultRegs := make([]ir.Register, len(s.Dest))
	CompileExpList(c, s.Src, resultRegs)

	// Compile the lvalues
	assigns := make([]Assign, len(s.Dest))
	for i, v := range s.Dest {
		assigns[i] = v.CompileAssign(c)
	}

	// Compile the assignments
	for i, reg := range resultRegs {
		c.ReleaseRegister(reg)
		assigns[i](reg)
	}
}
