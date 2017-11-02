package ast

import "github.com/arnodel/golua/ir"

type FunctionCall struct {
	target ExpNode
	method Name
	args   []ExpNode
}

func (f FunctionCall) HWrite(w HWriter) {
	w.Writef("call")
	w.Indent()
	w.Next()
	w.Writef("target: ")
	// w.Indent()
	f.target.HWrite(w)
	// w.Dedent()
	if f.method != "" {
		w.Next()
		w.Writef("method: %s", f.method)
	}
	for i, arg := range f.args {
		w.Next()
		w.Writef("arg_%d: ", i)
		arg.HWrite(w)
	}
	w.Dedent()
}

func (f FunctionCall) CompileExp(c *Compiler) ir.Register {
	// TODO
	return c.NewRegister()
}

func (f FunctionCall) CompileStat(c *Compiler) {
	// TODO
}

func NewFunctionCall(target ExpNode, method Name, args []ExpNode) (*FunctionCall, error) {
	return &FunctionCall{
		target: target,
		method: method,
		args:   args,
	}, nil
}
