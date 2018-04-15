package ast

import "github.com/arnodel/golua/ir"

type EmptyStat struct{}

func NewEmptyStat() (EmptyStat, error) {
	return EmptyStat{}, nil
}

func (s EmptyStat) HWrite(w HWriter) {
	w.Writef("empty stat")
}

func (s EmptyStat) CompileStat(c *ir.Compiler) {
	// Nothing to compile!
}
