package ast

import "github.com/arnodel/golua/ir"

// Node is a node in the AST
type Node interface {
	HWrite(w HWriter)
}

// HWriter is an interface for printing nodes
type HWriter interface {
	Writef(string, ...interface{})
	Indent()
	Dedent()
	Next()
}

// Stat is a statement
type Stat interface {
	Node
	CompileStat(c *ir.Compiler)
}

// ExpNode is an expression
type ExpNode interface {
	Node
	CompileExp(*ir.Compiler, ir.Register) ir.Register
}

// Var is an l-value
type Var interface {
	ExpNode
	CompileAssign(*ir.Compiler, ir.Register)
}
