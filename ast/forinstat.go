package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

type ForInStat struct {
	Location
	itervars []Name
	params   []ExpNode
	body     BlockStat
}

func NewForInStat(itervars []Name, params []ExpNode, body BlockStat) (*ForInStat, error) {
	return &ForInStat{
		itervars: itervars,
		params:   params,
		body:     body,
	}, nil
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

	LocalStat{
		names: s.itervars,
		values: []ExpNode{&FunctionCall{
			target: Name{string: "<f>"},
			args:   []ExpNode{Name{string: "<s>"}, Name{string: "<var>"}},
		}},
	}.CompileStat(c)
	var1, _ := c.GetRegister(ir.Name(s.itervars[0].string))

	testReg := c.GetFreeRegister()
	ir.EmitConstant(c, ir.NilType{}, testReg)
	c.Emit(ir.Combine{
		Dst:  testReg,
		Op:   ops.OpEq,
		Lsrc: var1,
		Rsrc: testReg,
	})
	endLbl := c.DeclareGotoLabel(ir.Name("<break>"))
	c.Emit(ir.JumpIf{Cond: testReg, Label: endLbl})
	c.Emit(ir.Transform{Dst: varReg, Op: ops.OpId, Src: var1})
	s.body.CompileBlock(c)

	c.Emit(ir.Jump{Label: loopLbl})

	c.EmitGotoLabel(ir.Name("<break>"))
	c.PopContext()
}
