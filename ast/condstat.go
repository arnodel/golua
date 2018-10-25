package ast

import "github.com/arnodel/golua/ir"

// CondStat is a conditional statement, used in e.g. if statements and while /
// repeat until loops.
type CondStat struct {
	Cond ExpNode
	Body BlockStat
}

func (s CondStat) HWrite(w HWriter) {
	s.Cond.HWrite(w)
	w.Next()
	w.Writef("body: ")
	s.Body.HWrite(w)
}

// CompileCond compiles a conditional statement.
func (s CondStat) CompileCond(c *ir.Compiler, lbl ir.Label) {
	condReg := CompileExp(c, s.Cond)
	EmitInstr(c, s.Cond, ir.JumpIf{Cond: condReg, Label: lbl, Not: true})
	s.Body.CompileStat(c)
}
