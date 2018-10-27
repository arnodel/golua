package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
	"github.com/arnodel/golua/token"
)

type ForStat struct {
	Location
	Var   Name
	Start ExpNode
	Stop  ExpNode
	Step  ExpNode
	Body  BlockStat
}

func NewForStat(startTok, endTok *token.Token, itervar Name, params []ExpNode, body BlockStat) *ForStat {
	return &ForStat{
		Location: LocFromTokens(startTok, endTok),
		Var:      itervar,
		Start:    params[0],
		Stop:     params[1],
		Step:     params[2],
		Body:     body,
	}
}

func (s *ForStat) HWrite(w HWriter) {
	w.Writef("for %s", s.Var)
	w.Indent()
	if s.Start != nil {
		w.Next()
		w.Writef("Start: ")
		s.Start.HWrite(w)
	}
	if s.Stop != nil {
		w.Next()
		w.Writef("Stop: ")
		s.Stop.HWrite(w)
	}
	if s.Step != nil {
		w.Next()
		w.Writef("Step: ")
		s.Step.HWrite(w)
	}
	w.Next()
	s.Body.HWrite(w)
	w.Dedent()
}

func (s ForStat) CompileStat(c *ir.Compiler) {
	startReg := c.GetFreeRegister()
	r := s.Start.CompileExp(c, startReg)
	ir.EmitMoveNoLine(c, startReg, r)
	if !IsNumber(s.Start) {
		c.EmitNoLine(ir.Transform{
			Dst: startReg,
			Src: startReg,
			Op:  ops.OpToNumber,
		})
	}
	c.TakeRegister(startReg)

	stopReg := c.GetFreeRegister()
	r = s.Stop.CompileExp(c, stopReg)
	ir.EmitMoveNoLine(c, stopReg, r)
	if !IsNumber(s.Stop) {
		c.EmitNoLine(ir.Transform{
			Dst: stopReg,
			Src: stopReg,
			Op:  ops.OpToNumber,
		})
	}
	c.TakeRegister(stopReg)

	stepReg := c.GetFreeRegister()
	r = s.Step.CompileExp(c, stepReg)
	ir.EmitMoveNoLine(c, stepReg, r)
	if !IsNumber(s.Step) {
		c.EmitNoLine(ir.Transform{
			Dst: stepReg,
			Src: stepReg,
			Op:  ops.OpToNumber,
		})
	}
	c.TakeRegister(stepReg)

	zReg := c.GetFreeRegister()
	c.TakeRegister(zReg)
	c.EmitNoLine(ir.LoadConst{
		Dst:  zReg,
		Kidx: c.GetConstant(ir.Int(0)),
	})
	c.EmitNoLine(ir.Combine{
		Op:   ops.OpLt,
		Dst:  zReg,
		Lsrc: stepReg,
		Rsrc: zReg,
	})

	c.PushContext()

	loopLbl := c.GetNewLabel()
	c.EmitLabel(loopLbl)
	endLbl := c.DeclareGotoLabel(breakLblName)

	condReg := c.GetFreeRegister()
	negStepLbl := c.GetNewLabel()
	bodyLbl := c.GetNewLabel()
	c.EmitNoLine(ir.JumpIf{
		Cond:  zReg,
		Label: negStepLbl,
	})
	c.EmitNoLine(ir.Combine{
		Op:   ops.OpLt,
		Dst:  condReg,
		Lsrc: stopReg,
		Rsrc: startReg,
	})
	c.EmitNoLine(ir.JumpIf{
		Cond:  condReg,
		Label: endLbl,
	})
	c.EmitNoLine(ir.Jump{Label: bodyLbl})
	c.EmitLabel(negStepLbl)
	c.EmitNoLine(ir.Combine{
		Op:   ops.OpLt,
		Dst:  condReg,
		Lsrc: startReg,
		Rsrc: stopReg,
	})
	c.EmitNoLine(ir.JumpIf{
		Cond:  condReg,
		Label: endLbl,
	})
	c.EmitLabel(bodyLbl)

	iterReg := c.GetFreeRegister()
	ir.EmitMoveNoLine(c, iterReg, startReg)
	c.DeclareLocal(ir.Name(s.Var.Val), iterReg)
	s.Body.CompileBlock(c)

	c.EmitNoLine(ir.Combine{
		Op:   ops.OpAdd,
		Dst:  startReg,
		Lsrc: startReg,
		Rsrc: stepReg,
	})
	c.EmitNoLine(ir.Jump{Label: loopLbl})

	c.EmitGotoLabel(breakLblName)
	c.PopContext()

	c.ReleaseRegister(startReg)
	c.ReleaseRegister(stopReg)
	c.ReleaseRegister(stepReg)
	c.ReleaseRegister(zReg)
}
