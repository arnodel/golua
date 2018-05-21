package ast

import "github.com/arnodel/golua/ir"

type CondStat struct {
	cond ExpNode
	body BlockStat
}

func (s CondStat) HWrite(w HWriter) {
	s.cond.HWrite(w)
	w.Next()
	w.Writef("body: ")
	s.body.HWrite(w)
}

func (s CondStat) CompileCond(c *ir.Compiler, lbl ir.Label) {
	condReg := CompileExp(c, s.cond)
	EmitInstr(c, s.cond, ir.JumpIf{Cond: condReg, Label: lbl, Not: true})
	s.body.CompileStat(c)
}
