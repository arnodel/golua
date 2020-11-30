package ast

// An IndexExp is an expression node representing indexing, i.e. "Coll[Index]".
type IndexExp struct {
	Location
	Coll ExpNode
	Idx  ExpNode
}

var _ Var = IndexExp{}

// NewIndexExp returns an IndexExp instance for the given collection and index.
func NewIndexExp(coll ExpNode, idx ExpNode) IndexExp {
	return IndexExp{
		Location: MergeLocations(coll, idx), // TODO: use the "]" for locaion end
		Coll:     coll,
		Idx:      idx,
	}
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (e IndexExp) ProcessExp(p ExpProcessor) {
	p.ProcessIndexExp(e)
}

// ProcessVar uses the given VarProcessor to process the receiver.
func (e IndexExp) ProcessVar(p VarProcessor) {
	p.ProcessIndexExpVar(e)
}

// FunctionName returns the function name associated with this expression (i.e.
// if it is part of a function definition statement), or an empty string if
// there is no sensible name.
func (e IndexExp) FunctionName() string {
	if s, ok := e.Idx.(String); ok {
		return string(s.Val)
	}
	return ""
}

// HWrite prints a tree representation of the node.
func (e IndexExp) HWrite(w HWriter) {
	w.Writef("idx")
	w.Indent()
	w.Next()
	w.Writef("coll: ")
	e.Coll.HWrite(w)
	w.Next()
	w.Writef("at: ")
	e.Idx.HWrite(w)
	w.Dedent()
}
