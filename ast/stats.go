package ast

import (
	"github.com/arnodel/golua/ir"
)

type Stat interface {
	Node
	CompileStat(c *ir.Compiler)
}

func CompileExpList(c *ir.Compiler, exps []ExpNode, dstRegs []ir.Register) {
	commonCount := len(exps)
	if commonCount > len(dstRegs) {
		commonCount = len(dstRegs)
	}
	var fCall FunctionCall
	doFCall := false
	if len(exps) < len(dstRegs) && len(exps) > 0 {
		fCall, doFCall = exps[len(exps)-1].(FunctionCall)
		if doFCall {
			commonCount--
		}
	}
	for i, exp := range exps[:commonCount] {
		dst := c.GetFreeRegister()
		reg := exp.CompileExp(c, dst)
		ir.EmitMove(c, dst, reg)
		c.TakeRegister(dst)
		dstRegs[i] = dst
	}
	for i := commonCount; i < len(dstRegs); i++ {
		dst := c.GetFreeRegister()
		c.TakeRegister(dst)
		dstRegs[i] = dst
	}
	if doFCall {
		fCall.CompileCall(c)
		c.Emit(ir.Receive{Dst: dstRegs[commonCount:]})
	} else if len(dstRegs) > len(exps) {
		nilK := ir.NilType{}
		for _, dst := range dstRegs[len(exps):] {
			ir.EmitConstant(c, nilK, dst)
		}
	}
}

type CondStat struct {
	cond ExpNode
	body BlockStat
}

func (s CondStat) HWrite(w HWriter) {
	s.cond.HWrite(w)
	w.Next()
	w.Writef("body: ")
	s.body.HWrite(w)
}

func (s CondStat) CompileCond(c *ir.Compiler, lbl ir.Label) {
	condReg := CompileExp(c, s.cond)
	c.Emit(ir.JumpIf{Cond: condReg, Label: lbl, Not: true})
	s.body.CompileStat(c)
}
