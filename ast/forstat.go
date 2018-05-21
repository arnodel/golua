package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
	"github.com/arnodel/golua/token"
)

type ForStat struct {
	Location
	itervar Name
	start   ExpNode
	stop    ExpNode
	step    ExpNode
	body    BlockStat
}

func NewForStat(startTok, endTok *token.Token, itervar Name, params []ExpNode, body BlockStat) (*ForStat, error) {
	return &ForStat{
		Location: LocFromTokens(startTok, endTok),
		itervar:  itervar,
		start:    params[0],
		stop:     params[1],
		step:     params[2],
		body:     body,
	}, nil
}

func (s *ForStat) HWrite(w HWriter) {
	w.Writef("for %s", s.itervar)
	w.Indent()
	if s.start != nil {
		w.Next()
		w.Writef("start: ")
		s.start.HWrite(w)
	}
	if s.stop != nil {
		w.Next()
		w.Writef("stop: ")
		s.stop.HWrite(w)
	}
	if s.step != nil {
		w.Next()
		w.Writef("step: ")
		s.step.HWrite(w)
	}
	w.Next()
	s.body.HWrite(w)
	w.Dedent()
}

func (s ForStat) CompileStat(c *ir.Compiler) {
	startReg := c.GetFreeRegister()
	r := s.start.CompileExp(c, startReg)
	ir.EmitMoveNoLine(c, startReg, r)
	c.TakeRegister(startReg)

	stopReg := c.GetFreeRegister()
	r = s.stop.CompileExp(c, stopReg)
	ir.EmitMoveNoLine(c, stopReg, r)
	c.TakeRegister(stopReg)

	stepReg := c.GetFreeRegister()
	r = s.step.CompileExp(c, stepReg)
	ir.EmitMoveNoLine(c, stepReg, r)
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
	c.DeclareLocal(ir.Name(s.itervar.string), iterReg)
	s.body.CompileBlock(c)

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
