package ast

import (
	"github.com/arnodel/golua/token"

	"github.com/arnodel/golua/ir"
)

type Location struct {
	start token.Pos
	end   token.Pos
}

func (l Location) StartPos() token.Pos {
	return l.start
}

func (l Location) EndPos() token.Pos {
	return l.end
}

// Node is a node in the AST
type Node interface {
	HWrite(w HWriter)
	StartPos() token.Pos
	EndPos() token.Pos
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
