package ast

import "github.com/arnodel/golua/ir"

type BreakStat struct {
	Location
}

func NewBreakStat() (BreakStat, error) {
	return BreakStat{}, nil
}

func (s BreakStat) HWrite(w HWriter) {
	w.Writef("break")
}

func (s BreakStat) CompileStat(c *ir.Compiler) {
	lbl, ok := c.GetGotoLabel(breakLblName)
	if !ok {
		panic("Cannot break from here")
	}
	c.Emit(ir.Jump{Label: lbl})
}

var breakLblName = ir.Name("<break>")
