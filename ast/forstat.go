package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

type ForStat struct {
	Location
	itervar Name
	start   ExpNode
	stop    ExpNode
	step    ExpNode
	body    BlockStat
}

func NewForStat(itervar Name, params []ExpNode, body BlockStat) (*ForStat, error) {
	return &ForStat{
		itervar: itervar,
		start:   params[0],
		stop:    params[1],
		step:    params[2],
		body:    body,
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
	ir.EmitMove(c, startReg, r)
	c.TakeRegister(startReg)

	stopReg := c.GetFreeRegister()
	r = s.stop.CompileExp(c, stopReg)
	ir.EmitMove(c, stopReg, r)
	c.TakeRegister(stopReg)

	stepReg := c.GetFreeRegister()
	r = s.step.CompileExp(c, stepReg)
	ir.EmitMove(c, stepReg, r)
	c.TakeRegister(stepReg)

	zReg := c.GetFreeRegister()
	c.TakeRegister(zReg)
	c.Emit(ir.LoadConst{
		Dst:  zReg,
		Kidx: c.GetConstant(ir.Int(0)),
	})
	c.Emit(ir.Combine{
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
	c.Emit(ir.JumpIf{
		Cond:  zReg,
		Label: negStepLbl,
	})
	c.Emit(ir.Combine{
		Op:   ops.OpLt,
		Dst:  condReg,
		Lsrc: stopReg,
		Rsrc: startReg,
	})
	c.Emit(ir.JumpIf{
		Cond:  condReg,
		Label: endLbl,
	})
	c.Emit(ir.Jump{Label: bodyLbl})
	c.EmitLabel(negStepLbl)
	c.Emit(ir.Combine{
		Op:   ops.OpLt,
		Dst:  condReg,
		Lsrc: startReg,
		Rsrc: stopReg,
	})
	c.Emit(ir.JumpIf{
		Cond:  condReg,
		Label: endLbl,
	})
	c.EmitLabel(bodyLbl)

	iterReg := c.GetFreeRegister()
	ir.EmitMove(c, iterReg, startReg)
	c.DeclareLocal(ir.Name(s.itervar.string), iterReg)
	s.body.CompileBlock(c)

	c.Emit(ir.Combine{
		Op:   ops.OpAdd,
		Dst:  startReg,
		Lsrc: startReg,
		Rsrc: stepReg,
	})
	c.Emit(ir.Jump{Label: loopLbl})

	c.EmitGotoLabel(breakLblName)
	c.PopContext()

	c.ReleaseRegister(startReg)
	c.ReleaseRegister(stopReg)
	c.ReleaseRegister(stepReg)
	c.ReleaseRegister(zReg)
}
