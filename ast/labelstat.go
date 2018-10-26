package ast

import "github.com/arnodel/golua/ir"

type LabelStat struct {
	Location
	Name
}

func NewLabelStat(label Name) LabelStat {
	return LabelStat{Location: label.Location, Name: label}
}

func (s LabelStat) HWrite(w HWriter) {
	w.Writef("label %s", s.Name.Val)
}

func (s LabelStat) CompileStat(c *ir.Compiler) {
	c.EmitGotoLabel(ir.Name(s.Name.Val))
}
