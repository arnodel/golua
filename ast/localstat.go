package ast

// LocalStat is a statement node representing the declaration / definition of a
// list of local variables.
type LocalStat struct {
	Location
	Names  []Name
	Values []ExpNode
}

var _ Stat = LocalStat{}

// NewLocalStat returns a LocalStat instance defining the given names with the
// given values.
func NewLocalStat(names []Name, values []ExpNode) LocalStat {
	loc := MergeLocations(names[0], names[len(names)-1])
	if len(values) > 0 {
		loc = MergeLocations(loc, values[len(values)-1])
	}
	// Give a name to functions defined here if possible
	for i, v := range values {
		if i >= len(names) {
			break
		}
		f, ok := v.(Function)
		if ok && f.Name == "" {
			f.Name = names[i].Val
			values[i] = f
		}
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
