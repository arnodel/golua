package ast

// GlobalWildcardStat is a statement node representing the wildcard global
// declaration forms: "global *" or "global<attrib> *"
type GlobalWildcardStat struct {
	Location
	Attrib *DeclAttrib // The optional attribute (nil if none)
}

var _ Stat = GlobalWildcardStat{}

// NewGlobalWildcardStat returns a GlobalWildcardStat for "global *" or "global<attrib> *"
func NewGlobalWildcardStat(loc Location, attrib *DeclAttrib) GlobalWildcardStat {
	return GlobalWildcardStat{
		Location: loc,
		Attrib:   attrib,
	}
}

// ProcessStat uses the given StatProcessor to process the receiver.
func (s GlobalWildcardStat) ProcessStat(p StatProcessor) {
	p.ProcessGlobalWildcardStat(s)
}

// HWrite prints a tree representation of the node.
func (s GlobalWildcardStat) HWrite(w HWriter) {
	w.Writef("global")
	if s.Attrib != nil {
		switch s.Attrib.Type {
		case ConstAttrib:
			w.Writef("<const>")
		case CloseAttrib:
			w.Writef("<close>")
		}
	}
	w.Writef(" *")
}
