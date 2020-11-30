package ast

// Locator can provide a location in code.  All AST nodes are locators.
type Locator interface {
	Locate() Location
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

// Var is an l-value, i.e. it can be assigned to.
type Var interface {
	ExpNode
	FunctionName() string
	ProcessVar(VarProcessor)
}

// A StatProcessor can process all implementations of Stat (i.e. statements).
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

// An ExpProcessor can process all implementations of ExpNode (i.e. expressions)
type ExpProcessor interface {
	ProcessBFunctionCallExp(BFunctionCall)
	ProcessBinOpExp(BinOp)
	ProcesBoolExp(Bool)
	ProcessEtcExp(Etc)
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

// A TailExpProcessor can process all implementations fo TailExpNode.
type TailExpProcessor interface {
	ProcessEtcTailExp(Etc)
	ProcessFunctionCallTailExp(FunctionCall)
}

// A VarProcessor can process all types of l-values.
type VarProcessor interface {
	ProcessIndexExpVar(IndexExp)
	ProcessNameVar(Name)
}
