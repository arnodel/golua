package ast

type BFunctionCall struct {
	Location
	Target ExpNode
	Method Name
	Args   []ExpNode
}

var _ ExpNode = BFunctionCall{}

type FunctionCall struct {
	*BFunctionCall
}

var _ ExpNode = FunctionCall{}
var _ TailExpNode = FunctionCall{}
var _ Stat = FunctionCall{}

func NewFunctionCall(target ExpNode, method Name, args []ExpNode) FunctionCall {
	// TODO: fix this by creating an Args node
	loc := target.Locate()
	if len(args) > 0 {
		loc = MergeLocations(loc, args[len(args)-1])
	} else if method.Val != "" {
		loc = MergeLocations(loc, method)
	}
	return FunctionCall{&BFunctionCall{
		Location: loc,
		Target:   target,
		Method:   method,
		Args:     args,
	}}
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (f FunctionCall) ProcessExp(p ExpProcessor) {
	p.ProcessFunctionCallExp(f)
}

func (f FunctionCall) ProcessTailExp(p TailExpProcessor) {
	p.ProcessFunctionCallTailExp(f)
}

// ProcessStat uses the given StatProcessor to process the receiver.
func (f FunctionCall) ProcessStat(p StatProcessor) {
	p.ProcessFunctionCallStat(f)
}

func (f FunctionCall) InBrackets() *BFunctionCall {
	return f.BFunctionCall
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (f BFunctionCall) ProcessExp(p ExpProcessor) {
	p.ProcessBFunctionCallExp(f)
}

// HWrite prints a tree representation of the node.
func (f BFunctionCall) HWrite(w HWriter) {
	w.Writef("call")
	w.Indent()
	w.Next()
	w.Writef("target: ")
	// w.Indent()
	f.Target.HWrite(w)
	// w.Dedent()
	if f.Method.Val != "" {
		w.Next()
		w.Writef("method: %s", f.Method)
	}
	for i, arg := range f.Args {
		w.Next()
		w.Writef("arg_%d: ", i)
		arg.HWrite(w)
	}
	w.Dedent()
}
