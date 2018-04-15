package ast

import "github.com/arnodel/golua/ir"

type GotoStat struct {
	label Name
}

func NewGotoStat(lbl Name) (GotoStat, error) {
	return GotoStat{label: lbl}, nil
}

func (s GotoStat) HWrite(w HWriter) {
	w.Writef("goto %s", s.label)
}

func (s GotoStat) CompileStat(c *ir.Compiler) {
	lbl, ok := c.GetGotoLabel(ir.Name(s.label))
	if !ok {
		panic("Undefined label for goto")
	}
	c.Emit(ir.Jump{Label: lbl})
}
