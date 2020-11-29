package ast

import "github.com/arnodel/golua/ir"

type LabelStat struct {
	Location
	Name
}

var _ Stat = LabelStat{}

func NewLabelStat(label Name) LabelStat {
	return LabelStat{Location: label.Location, Name: label}
}

// ProcessStat uses the given StatProcessor to process the receiver.
func (s LabelStat) ProcessStat(p StatProcessor) {
	p.ProcessLabelStat(s)
}

// HWrite prints a tree representation of the node.
func (s LabelStat) HWrite(w HWriter) {
	w.Writef("label %s", s.Name.Val)
}

func (s LabelStat) CompileStat(c *ir.CodeBuilder) {
	c.EmitGotoLabel(ir.Name(s.Name.Val))
}
