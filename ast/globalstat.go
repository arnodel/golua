package ast

// GlobalStat is a statement node representing the declaration / definition of a
// list of global variables.
type GlobalStat struct {
	Location
	PrefixAttrib *DeclAttrib
	NameAttribs  []NameAttrib
	Values       []ExpNode
}

var _ Stat = GlobalStat{}

// NewGlobalStat returns a GlobalStat instance defining the given names with the
// given values.
func NewGlobalStat(prefixAttrib *DeclAttrib, nameAttribs []NameAttrib, values []ExpNode) GlobalStat {
	var loc Location = mergeLocationsOf(
		optLocation(prefixAttrib),
		sliceLocation(nameAttribs),
		sliceLocation(values),
	)
	// Give a name to functions defined here if possible
	for i, v := range values {
		if i >= len(nameAttribs) {
			break
		}
		f, ok := v.(Function)
		if ok && f.Name == "" {
			f.Name = nameAttribs[i].Name.Val
			values[i] = f
		}
	}
	return GlobalStat{
		Location:     loc,
		PrefixAttrib: prefixAttrib,
		NameAttribs:  nameAttribs,
		Values:       values,
	}
}

// ProcessStat uses the given StatProcessor to process the receiver.
func (s GlobalStat) ProcessStat(p StatProcessor) {
	p.ProcessGlobalStat(s)
}

// HWrite prints a tree representation of the node.
func (s GlobalStat) HWrite(w HWriter) {
	w.Writef("global")
	w.Indent()
	for i, nameAttrib := range s.NameAttribs {
		w.Next()
		w.Writef("name_%d: %s", i, nameAttrib)
	}
	for i, val := range s.Values {
		w.Next()
		w.Writef("val_%d: ", i)
		val.HWrite(w)
	}
	w.Dedent()
}
