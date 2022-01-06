package ast

// LocalStat is a statement node representing the declaration / definition of a
// list of local variables.
type LocalStat struct {
	Location
	NameAttribs []NameAttrib
	Values      []ExpNode
}

var _ Stat = LocalStat{}

// NewLocalStat returns a LocalStat instance defining the given names with the
// given values.
func NewLocalStat(nameAttribs []NameAttrib, values []ExpNode) LocalStat {
	loc := MergeLocations(nameAttribs[0], nameAttribs[len(nameAttribs)-1])
	if len(values) > 0 {
		loc = MergeLocations(loc, values[len(values)-1])
	}
	return LocalStat{Location: loc, NameAttribs: nameAttribs, Values: values}
}

// ProcessStat uses the given StatProcessor to process the receiver.
func (s LocalStat) ProcessStat(p StatProcessor) {
	p.ProcessLocalStat(s)
}

// HWrite prints a tree representation of the node.
func (s LocalStat) HWrite(w HWriter) {
	w.Writef("local")
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

type NameAttrib struct {
	Location
	Name   Name
	Attrib *Name
}

func NewNameAttrib(name Name, attrib *Name) NameAttrib {
	loc := name.Location
	if attrib != nil {
		loc = MergeLocations(loc, attrib)
	}
	return NameAttrib{
		Location: loc,
		Name:     name,
		Attrib:   attrib,
	}
}

func (na NameAttrib) IsConst() bool {
	return na.Attrib != nil && na.Attrib.Val == "const"
}
