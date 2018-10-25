package ast

import (
	"github.com/arnodel/golua/ir"
)

type BlockStat struct {
	Location
	Stats  []Stat
	Return []ExpNode
}

func NewBlockStat(stats []Stat, rtn []ExpNode) BlockStat {
	return BlockStat{
		// TODO: set Location
		Stats:  stats,
		Return: rtn,
	}
}

func (s BlockStat) HWrite(w HWriter) {
	w.Writef("block")
	w.Indent()
	for _, stat := range s.Stats {
		w.Next()
		stat.HWrite(w)
	}
	if s.Return != nil {
		w.Next()
		w.Writef("return")
		w.Indent()
		for _, val := range s.Return {
			w.Next()
			val.HWrite(w)
		}
		w.Dedent()
	}
	w.Dedent()
}

func tailCall(rtn []ExpNode) (FunctionCall, bool) {
	if len(rtn) != 1 {
		return FunctionCall{}, false
	}
	fc, ok := rtn[0].(FunctionCall)
	return fc, ok
}

func (s BlockStat) CompileBlock(c *ir.Compiler) {
	s.CompileBlockNoPop(c)()
}

func (s BlockStat) CompileBlockNoPop(c *ir.Compiler) func() {
	totalDepth := 0
	getLabels(c, s.Stats)
	truncLen := len(s.Stats) - getBackLabels(c, s.Stats)
	for i, stat := range s.Stats {
		switch stat.(type) {
		case LocalStat, LocalFunctionStat:
			totalDepth++
			c.PushContext()
			getLabels(c, s.Stats[i+1:truncLen])
		}
		stat.CompileStat(c)
	}
	if s.Return != nil {
		if fc, ok := tailCall(s.Return); ok {
			fc.CompileCall(c, true)
		} else {
			contReg, ok := c.GetRegister(ir.Name("<caller>"))
			if !ok {
				panic("Cannot return: no caller")
			}
			compilePushArgs(c, s.Return, contReg)
			var loc Locator
			if len(s.Return) > 0 {
				loc = s.Return[0]
			}
			EmitInstr(c, loc, ir.Call{Cont: contReg})
		}
	}
	return func() {
		for ; totalDepth > 0; totalDepth-- {
			c.PopContext()
		}
	}
}

func (s BlockStat) CompileChunk(source string) *ir.Compiler {
	pc := ir.NewCompiler(source)
	pc.DeclareLocal("_ENV", pc.GetFreeRegister())
	c := pc.NewChild()

	f := Function{
		ParList: ParList{HasDots: true},
		Body:    s,
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
			c.DeclareGotoLabel(ir.Name(s.Name.Val))
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
			c.DeclareGotoLabel(ir.Name(lbl.Name.Val))
		} else {
			break
		}
	}
	return count
}
