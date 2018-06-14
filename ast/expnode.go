package ast

import (
	"github.com/arnodel/golua/ir"
)

func CompileExp(c *ir.Compiler, e ExpNode) ir.Register {
	r1 := c.GetFreeRegister()
	r2 := e.CompileExp(c, r1)
	if r1 != r2 {
		return r2
	}
	return r1
}

func CompileExpList(c *ir.Compiler, exps []ExpNode, dstRegs []ir.Register) {
	commonCount := len(exps)
	if commonCount > len(dstRegs) {
		commonCount = len(dstRegs)
	}
	var tailExp TailExpNode
	doTailExp := false
	if len(dstRegs) > len(exps) && len(exps) > 0 {
		tailExp, doTailExp = exps[len(exps)-1].(TailExpNode)
		if doTailExp {
			commonCount--
		}
	}
	for i, exp := range exps[:commonCount] {
		dst := c.GetFreeRegister()
		reg := exp.CompileExp(c, dst)
		EmitMove(c, exp, dst, reg)
		c.TakeRegister(dst)
		dstRegs[i] = dst
	}
	for i := commonCount; i < len(dstRegs); i++ {
		dst := c.GetFreeRegister()
		c.TakeRegister(dst)
		dstRegs[i] = dst
	}
	if doTailExp {
		tailExp.CompileTailExp(c, dstRegs[commonCount:])
	} else if len(dstRegs) > len(exps) {
		nilK := ir.NilType{}
		for _, dst := range dstRegs[len(exps):] {
			EmitLoadConst(c, nil, nilK, dst)
		}
	}
}
