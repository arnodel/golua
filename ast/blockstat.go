package ast

import "github.com/arnodel/golua/ir"

type BlockStat struct {
	statements   []Stat
	returnValues []ExpNode
}

func NewBlockStat(stats []Stat, rtn []ExpNode) (BlockStat, error) {
	return BlockStat{stats, rtn}, nil
}

func (s BlockStat) HWrite(w HWriter) {
	w.Writef("block")
	w.Indent()
	for _, stat := range s.statements {
		w.Next()
		stat.HWrite(w)
	}
	if s.returnValues != nil {
		w.Next()
		w.Writef("return")
		w.Indent()
		for _, val := range s.returnValues {
			w.Next()
			val.HWrite(w)
		}
		w.Dedent()
	}
	w.Dedent()
}

func getLabels(c *ir.Compiler, statements []Stat) {
	for _, stat := range statements {
		switch s := stat.(type) {
		case LabelStat:
			c.DeclareGotoLabel(ir.Name(s))
		case LocalStat, LocalFunctionStat:
			return
		}
	}
}

func getBackLabels(c *ir.Compiler, statements []Stat) int {
	count := 0
	for i := len(statements) - 1; i >= 0; i-- {
		if lbl, ok := statements[i].(LabelStat); ok {
			count++
			c.DeclareGotoLabel(ir.Name(lbl))
		} else {
			break
		}
	}
	return count
}

func (s BlockStat) CompileBlock(c *ir.Compiler) {
	totalDepth := 0
	getLabels(c, s.statements)
	truncLen := len(s.statements) - getBackLabels(c, s.statements)
	for i, stat := range s.statements {
		switch stat.(type) {
		case LocalStat, LocalFunctionStat:
			totalDepth++
			c.PushContext()
			getLabels(c, s.statements[i+1:truncLen])
		}
		stat.CompileStat(c)
	}
	for ; totalDepth > 0; totalDepth-- {
		c.PopContext()
	}
	if s.returnValues != nil {
		cont, ok := c.GetRegister(ir.Name(Name("<caller>")))
		if !ok {
			panic("Cannot return: no caller")
		}
		CallWithArgs(c, s.returnValues, cont)
	}
}

func (s BlockStat) CompileChunk() *ir.Compiler {
	c := ir.NewCompiler()
	f := Function{
		ParList: ParList{hasDots: true},
		body:    s,
	}
	pf := Function{
		ParList: ParList{params: []Name{"_ENV"}},
		body:    BlockStat{returnValues: []ExpNode{f}},
	}
	pf.CompileBody(c)
	return c
}

func (s BlockStat) CompileStat(c *ir.Compiler) {
	c.PushContext()
	s.CompileBlock(c)
	c.PopContext()
}
