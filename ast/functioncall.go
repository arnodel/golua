package ast

// TODO: the FunctionCall / BFunctionCall distinction is awkward.  Find a better
// expression of the difference.

// A BFunctionCall is an expression node that represents a function call
// surrounded by brackets. This changes the semantics of the call in some
// situations (e.g. ellipsis, tail calls).
//
// For example:
//
//    return f(x)   -- this may return multiple values and will be subject to TCO
//    return (f(x)) -- this will only return one single value and will not be subject to TCO
type BFunctionCall struct {
	Location
	Target ExpNode
	Method Name
	Args   []ExpNode
}

var _ ExpNode = BFunctionCall{}

// A FunctionCall is an expression node that represents a function call.
type FunctionCall struct {
	*BFunctionCall
}

var _ ExpNode = FunctionCall{}
var _ TailExpNode = FunctionCall{}
var _ Stat = FunctionCall{}

// NewFunctionCall returns a FunctionCall instance representing the call of
// <target> with <args>.  The <method> arg should be non-nil if the call syntax
// included ":", e.g. stack:pop().
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

// ProcessTailExp uses the give TailExpProcessor to process the receiver.
func (f FunctionCall) ProcessTailExp(p TailExpProcessor) {
	p.ProcessFunctionCallTailExp(f)
}

// ProcessStat uses the given StatProcessor to process the receiver.
func (f FunctionCall) ProcessStat(p StatProcessor) {
	p.ProcessFunctionCallStat(f)
}

// InBrackets turns the receiver into a BFunctionCall.
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
