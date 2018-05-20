package ast

import "github.com/arnodel/golua/ir"

type LabelStat struct {
	Location
	Name
}

func NewLabelStat(label Name) (LabelStat, error) {
	return LabelStat{Location: label.Location, Name: label}, nil
}

func (s LabelStat) HWrite(w HWriter) {
	w.Writef("label %s", s.Name.string)
}

func (s LabelStat) CompileStat(c *ir.Compiler) {
	c.EmitGotoLabel(ir.Name(s.Name.string))
}
