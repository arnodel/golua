package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
	"github.com/arnodel/golua/token"
)

type ForInStat struct {
	Location
	Vars   []Name
	Params []ExpNode
	Body   BlockStat
}

func NewForInStat(startTok, endTok *token.Token, itervars []Name, params []ExpNode, body BlockStat) *ForInStat {
	return &ForInStat{
		Location: LocFromTokens(startTok, endTok),
		Vars:     itervars,
		Params:   params,
		Body:     body,
	}
}

func (s *ForInStat) HWrite(w HWriter) {
	w.Writef("for in")
	w.Indent()
	for i, v := range s.Vars {
		w.Next()
		w.Writef("var_%d: ", i)
		v.HWrite(w)
	}
	for i, p := range s.Params {
		w.Next()
		w.Writef("param_%d", i)
		p.HWrite(w)
	}
	w.Next()
	w.Writef("Body: ")
	s.Body.HWrite(w)
	w.Dedent()
}

func (s ForInStat) CompileStat(c *ir.Compiler) {
	initRegs := make([]ir.Register, 3)
	CompileExpList(c, s.Params, initRegs)
	fReg := initRegs[0]
	sReg := initRegs[1]
	varReg := initRegs[2]

	c.PushContext()
	c.DeclareLocal(ir.Name("<f>"), fReg)
	c.DeclareLocal(ir.Name("<s>"), sReg)
	c.DeclareLocal(ir.Name("<var>"), varReg)

	loopLbl := c.GetNewLabel()
	c.EmitLabel(loopLbl)

	// TODO: better locations

	LocalStat{
		Names: s.Vars,
		Values: []ExpNode{FunctionCall{&BFunctionCall{
			Location: s.Location,
			target:   Name{Location: s.Location, Val: "<f>"},
			args: []ExpNode{
				Name{Location: s.Location, Val: "<s>"},
				Name{Location: s.Location, Val: "<var>"},
			},
		}}},
	}.CompileStat(c)
	var1, _ := c.GetRegister(ir.Name(s.Vars[0].Val))

	testReg := c.GetFreeRegister()
	EmitLoadConst(c, s, ir.NilType{}, testReg)
	EmitInstr(c, s, ir.Combine{
		Dst:  testReg,
		Op:   ops.OpEq,
		Lsrc: var1,
		Rsrc: testReg,
	})
	endLbl := c.DeclareGotoLabel(ir.Name("<break>"))
	EmitInstr(c, s, ir.JumpIf{Cond: testReg, Label: endLbl})
	EmitInstr(c, s, ir.Transform{Dst: varReg, Op: ops.OpId, Src: var1})
	s.Body.CompileBlock(c)

	EmitInstr(c, s, ir.Jump{Label: loopLbl})

	c.EmitGotoLabel(ir.Name("<break>"))
	c.PopContext()
}
