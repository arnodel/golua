package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
	"github.com/arnodel/golua/token"
)

type ForInStat struct {
	Location
	itervars []Name
	params   []ExpNode
	body     BlockStat
}

func NewForInStat(startTok, endTok *token.Token, itervars []Name, params []ExpNode, body BlockStat) *ForInStat {
	return &ForInStat{
		Location: LocFromTokens(startTok, endTok),
		itervars: itervars,
		params:   params,
		body:     body,
	}
}

func (s *ForInStat) HWrite(w HWriter) {
	w.Writef("for in")
	w.Indent()
	for i, v := range s.itervars {
		w.Next()
		w.Writef("var_%d: ", i)
		v.HWrite(w)
	}
	for i, p := range s.params {
		w.Next()
		w.Writef("param_%d", i)
		p.HWrite(w)
	}
	w.Next()
	w.Writef("body: ")
	s.body.HWrite(w)
	w.Dedent()
}

func (s ForInStat) CompileStat(c *ir.Compiler) {
	initRegs := make([]ir.Register, 3)
	CompileExpList(c, s.params, initRegs)
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
		names: s.itervars,
		values: []ExpNode{FunctionCall{&BFunctionCall{
			Location: s.Location,
			target:   Name{Location: s.Location, string: "<f>"},
			args: []ExpNode{
				Name{Location: s.Location, string: "<s>"},
				Name{Location: s.Location, string: "<var>"},
			},
		}}},
	}.CompileStat(c)
	var1, _ := c.GetRegister(ir.Name(s.itervars[0].string))

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
	s.body.CompileBlock(c)

	EmitInstr(c, s, ir.Jump{Label: loopLbl})

	c.EmitGotoLabel(ir.Name("<break>"))
	c.PopContext()
}
