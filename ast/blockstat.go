package ast

import "github.com/arnodel/golua/ir"

type BlockStat struct {
	Location
	statements   []Stat
	returnValues []ExpNode
}

func NewBlockStat(stats []Stat, rtn []ExpNode) (BlockStat, error) {
	return BlockStat{
		// TODO: set Location
		statements:   stats,
		returnValues: rtn,
	}, nil
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
	if s.returnValues != nil {
		cont, ok := c.GetRegister(ir.Name("<caller>"))
		if !ok {
			panic("Cannot return: no caller")
		}
		CallWithArgs(c, s.returnValues, cont)
	}
	for ; totalDepth > 0; totalDepth-- {
		c.PopContext()
	}
}

func (s BlockStat) CompileChunk(source string) *ir.Compiler {
	pc := ir.NewCompiler(source)
	pc.DeclareLocal("_ENV", pc.GetFreeRegister())
	c := pc.NewChild()

	f := Function{
		ParList: ParList{hasDots: true},
		body:    s,
	}
	f.CompileBody(c)
	return c
}

func (s BlockStat) CompileStat(c *ir.Compiler) {
	c.PushContext()
	s.CompileBlock(c)
	c.PopContext()
}

func getLabels(c *ir.Compiler, statements []Stat) {
	for _, stat := range statements {
		switch s := stat.(type) {
		case LabelStat:
			c.DeclareGotoLabel(ir.Name(s.Name.string))
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
			c.DeclareGotoLabel(ir.Name(lbl.Name.string))
		} else {
			break
		}
	}
	return count
}
