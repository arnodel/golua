package ast

// GlobalFunctionStat is a statement node that represents a global function
// definition, i.e. "global function Name() ...".
type GlobalFunctionStat struct {
	Location
	Function
	Name Name
}

var _ Stat = GlobalFunctionStat{}

// NewGlobalFunctionStat returns a GlobalFunctionStat instance for the given name
// and function definition.
func NewGlobalFunctionStat(name Name, fx Function) GlobalFunctionStat {
	fx.Name = name.Val
	return GlobalFunctionStat{
		Location: MergeLocations(name, fx), // TODO: use "global" for location start
		Function: fx,
		Name:     name,
	}
}

// ProcessStat uses the given StatProcessor to process the receiver.
func (s GlobalFunctionStat) ProcessStat(p StatProcessor) {
	p.ProcessGlobalFunctionStat(s)
}

// HWrite prints a tree representation of the node.
func (s GlobalFunctionStat) HWrite(w HWriter) {
	w.Writef("global function ")
	s.Name.HWrite(w)
	s.Function.HWrite(w)
}
