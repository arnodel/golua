package ast

// LabelStat is a statement node that represents a label.
type LabelStat struct {
	Location
	Name
}

var _ Stat = LabelStat{}

// NewLabelStat returns a LabelStat instance for the given label.
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
