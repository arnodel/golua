package ast

import "github.com/arnodel/golua/ir"

type LabelStat Name

func NewLabelStat(label Name) (LabelStat, error) {
	return LabelStat(label), nil
}

func (s LabelStat) HWrite(w HWriter) {
	w.Writef("label %s", string(s))
}

func (s LabelStat) CompileStat(c *ir.Compiler) {
	c.EmitGotoLabel(ir.Name(s))
}
