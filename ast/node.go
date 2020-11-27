package ast

import (
	"github.com/arnodel/golua/token"

	"github.com/arnodel/golua/ir"
)

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
	ProcessStat(StatProcessor)
}

// ExpNode is an expression
type ExpNode interface {
	Node
	ProcessExp(ExpProcessor)
}

// TailExpNode is an expression which can be the tail of an exp list
type TailExpNode interface {
	Node
	ProcessTailExp(TailExpProcessor)
}

// Var is an l-value
type Var interface {
	ExpNode
	FunctionName() string
	ProcessVar(VarProcessor)
}

type Assign func(ir.Register)

type StatProcessor interface {
	ProcessAssignStat(AssignStat)
	ProcessBlockStat(BlockStat)
	ProcessBreakStat(BreakStat)
	ProcessEmptyStat(EmptyStat)
	ProcessForInStat(ForInStat)
	ProcessForStat(ForStat)
	ProcessFunctionCallStat(FunctionCall)
	ProcessGotoStat(GotoStat)
	ProcessIfStat(IfStat)
	ProcessLabelStat(LabelStat)
	ProcessLocalFunctionStat(LocalFunctionStat)
	ProcessLocalStat(LocalStat)
	ProcessRepeatStat(RepeatStat)
	ProcessWhileStat(WhileStat)
}

type ExpProcessor interface {
	ProcessBFunctionCallExp(BFunctionCall)
	ProcessBinOpExp(BinOp)
	ProcesBoolExp(Bool)
	ProcessEtcExp(EtcType)
	ProcessFunctionExp(Function)
	ProcessFunctionCallExp(FunctionCall)
	ProcessIndexExp(IndexExp)
	ProcessNameExp(Name)
	ProcessNilExp(NilType)
	ProcessIntExp(Int)
	ProcessFloatExp(Float)
	ProcessStringExp(String)
	ProcessTableConstructorExp(TableConstructor)
	ProcessUnOpExp(UnOp)
}

type TailExpProcessor interface {
	ProcessEtcTailExp(EtcType)
	ProcessFunctionCallTailExp(FunctionCall)
}

type VarProcessor interface {
	ProcessIndexExpVar(IndexExp)
	ProcessNameVar(Name)
}
