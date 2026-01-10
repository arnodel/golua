package ast

// GlobalWildcardStat is a statement node representing the wildcard global
// declaration forms: "global *" or "global<const> *"
type GlobalWildcardStat struct {
	Location
	IsConst bool
}

var _ Stat = GlobalWildcardStat{}

// NewGlobalWildcardStat returns a GlobalWildcardStat for "global *" or "global<const> *"
func NewGlobalWildcardStat(loc Location, isConst bool) GlobalWildcardStat {
	return GlobalWildcardStat{
		Location: loc,
		IsConst:  isConst,
	}
}

// ProcessStat uses the given StatProcessor to process the receiver.
func (s GlobalWildcardStat) ProcessStat(p StatProcessor) {
	p.ProcessGlobalWildcardStat(s)
}

// HWrite prints a tree representation of the node.
func (s GlobalWildcardStat) HWrite(w HWriter) {
	if s.IsConst {
		w.Writef("global<const> *")
	} else {
		w.Writef("global *")
	}
}
