package ast

type IndexExp struct {
	Location
	Coll ExpNode
	Idx  ExpNode
}

var _ Var = IndexExp{}

func NewIndexExp(coll ExpNode, idx ExpNode) IndexExp {
	return IndexExp{
		Location: MergeLocations(coll, idx), // TODO: use the "]" for locaion end
		Coll:     coll,
		Idx:      idx,
	}
}

func (e IndexExp) ProcessExp(p ExpProcessor) {
	p.ProcessIndexExp(e)
}

func (e IndexExp) ProcessVar(p VarProcessor) {
	p.ProcessIndexExpVar(e)
}

func (e IndexExp) FunctionName() string {
	if s, ok := e.Idx.(String); ok {
		return string(s.Val)
	}
	return ""
}

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
