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

// LocalAttrib is the type of a local name attrib
type LocalAttrib uint8

// Valid values for LocalAttrib
const (
	NoAttrib    LocalAttrib = iota
	ConstAttrib             // <const>, introduced in Lua 5.4
	CloseAttrib             // <close>, introduced in Lua 5.4
)

// A NameAttrib is a name introduce by a local definition, together with an
// optional attribute (in Lua 5.4 that is 'close' or 'const').
type NameAttrib struct {
	Location
	Name   Name
	Attrib LocalAttrib
}

// NewNameAttrib returns a new NameAttribe for the given name and attrib.
func NewNameAttrib(name Name, attribName *Name, attrib LocalAttrib) NameAttrib {
	loc := name.Location
	if attribName != nil {
		loc = MergeLocations(loc, attribName)

	}
	return NameAttrib{
		Location: loc,
		Name:     name,
		Attrib:   attrib,
	}
}
