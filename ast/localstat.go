package ast

type LocalStat struct {
	Location
	Names  []Name
	Values []ExpNode
}

var _ Stat = LocalStat{}

func NewLocalStat(names []Name, values []ExpNode) LocalStat {
	loc := MergeLocations(names[0], names[len(names)-1])
	if len(values) > 0 {
		loc = MergeLocations(loc, values[len(values)-1])
	}
	return LocalStat{Location: loc, Names: names, Values: values}
}

// ProcessStat uses the given StatProcessor to process the receiver.
func (s LocalStat) ProcessStat(p StatProcessor) {
	p.ProcessLocalStat(s)
}

// HWrite prints a tree representation of the node.
func (s LocalStat) HWrite(w HWriter) {
	w.Writef("local")
	w.Indent()
	for i, name := range s.Names {
		w.Next()
		w.Writef("name_%d: %s", i, name)
	}
	for i, val := range s.Values {
		w.Next()
		w.Writef("val_%d: ", i)
		val.HWrite(w)
	}
	w.Dedent()
}
