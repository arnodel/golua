package ast

// LocalStat is a statement node representing the declaration / definition of a
// list of local variables.
type LocalStat struct {
	Location
	PrefixAttrib *DeclAttrib
	NameAttribs  []NameAttrib
	Values       []ExpNode
}

var _ Stat = LocalStat{}

// NewLocalStat returns a LocalStat instance defining the given names with the
// given values.
func NewLocalStat(prefixAttrib *DeclAttrib, nameAttribs []NameAttrib, values []ExpNode) LocalStat {
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
	return LocalStat{
		Location:     loc,
		PrefixAttrib: prefixAttrib,
		NameAttribs:  nameAttribs,
		Values:       values,
	}
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

// A NameAttrib is a name introduced by a declaration, together with an
// optional attribute (in Lua 5.4 that is 'close' or 'const').
type NameAttrib struct {
	Location
	Name   Name
	Attrib *DeclAttrib // nil if no attribute
}

// NewNameAttrib returns a new NameAttrib for the given name and attrib.
func NewNameAttrib(name Name, attrib *DeclAttrib) NameAttrib {
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
