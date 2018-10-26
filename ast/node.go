package ast

import (
	"github.com/arnodel/golua/token"

	"github.com/arnodel/golua/ir"
)

func EmitInstr(c *ir.Compiler, l Locator, instr ir.Instruction) {
	line := 0
	if l != nil {
		loc := l.Locate()
		if loc.start != nil {
			line = loc.start.Line
		}
	}
	c.Emit(instr, line)
}

func EmitLoadConst(c *ir.Compiler, l Locator, k ir.Constant, reg ir.Register) {
	line := 0
	if l != nil {
		loc := l.Locate()
		if loc.start != nil {
			line = loc.start.Line
		}
	}
	ir.EmitConstant(c, k, reg, line)
}

func EmitMove(c *ir.Compiler, l Locator, dst, src ir.Register) {
	if src == dst {
		return
	}
	line := 0
	if l != nil {
		loc := l.Locate()
		if loc.start != nil {
			line = loc.start.Line
		}
	}
	ir.EmitMove(c, dst, src, line)
}

type Locator interface {
	Locate() Location
}

type Location struct {
	start *token.Pos
	end   *token.Pos
}

func (l Location) StartPos() *token.Pos {
	return l.start
}

func (l Location) EndPos() *token.Pos {
	return l.end
}

func (l Location) Locate() Location {
	return l
}

func LocFromToken(tok *token.Token) Location {
	if tok == nil || tok.Pos.Offset < 0 {
		return Location{}
	}
	pos := tok.Pos
	return Location{&pos, &pos}
}

func LocFromTokens(t1, t2 *token.Token) Location {
	var p1, p2 *token.Pos
	if t1 != nil && t1.Pos.Offset >= 0 {
		p1 = new(token.Pos)
		*p1 = t1.Pos
	}
	if t2 != nil && t2.Pos.Offset >= 0 {
		p2 = new(token.Pos)
		*p2 = t2.Pos
	}
	return Location{p1, p2}
}

func MergeLocations(l1, l2 Locator) Location {
	l := l1.Locate()
	ll := l2.Locate()
	if ll.start != nil && (l.start == nil || l.start.Offset > ll.start.Offset) {
		l.start = ll.start
	}
	if ll.end != nil && (l.end == nil || l.end.Offset < ll.end.Offset) {
		l.end = ll.end
	}
	return l
}

// Node is a node in the AST
type Node interface {
	Locator
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

// TailExpNode is an expression which can be the tail of an exp list
type TailExpNode interface {
	Node
	CompileTailExp(*ir.Compiler, []ir.Register)
	CompileEtcExp(*ir.Compiler, ir.Register) ir.Register
}

// Var is an l-value
type Var interface {
	ExpNode
	CompileAssign(*ir.Compiler) Assign
	FunctionName() string
}

type Assign func(ir.Register)
