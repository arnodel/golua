package astcomp

import (
	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

//
// Statement compilation
//

// Static check that no statement is overlooked.
var _ ast.StatProcessor = (*compiler)(nil)

// ProcessAssignStat compiles a AssignStat.
func (c *compiler) ProcessAssignStat(s ast.AssignStat) {

	// Evaluate the right hand side
	resultRegs := make([]ir.Register, len(s.Dest))
	c.compileExpList(s.Src, resultRegs)

	// Compile the lvalues and assignments
	c.compileAssignments(s.Dest, resultRegs)
}

// ProcessBlockStat compiles a BlockStat.
func (c *compiler) ProcessBlockStat(s ast.BlockStat) {
	c.PushContext()
	c.compileBlock(s)
	c.PopContext()
}

// ProcessBreakStat compiles a BreakStat.
func (c *compiler) ProcessBreakStat(s ast.BreakStat) {
	c.emitJump(s, breakLblName)
}

// ProcessEmptyStat compiles a EmptyStat.
func (c *compiler) ProcessEmptyStat(s ast.EmptyStat) {
	// Nothing to compile!
}

// ProcessForInStat compiles a ForInStat.
func (c *compiler) ProcessForInStat(s ast.ForInStat) {
	initRegs := make([]ir.Register, 3)
	c.compileExpList(s.Params, initRegs)
	fReg := initRegs[0]
	sReg := initRegs[1]
	varReg := initRegs[2]

	c.PushContext()
	c.DeclareLocal(loopFRegName, fReg)
	c.DeclareLocal(loopSRegName, sReg)
	c.DeclareLocal(loopVarRegName, varReg)

	loopLbl := c.GetNewLabel()
	c.EmitLabel(loopLbl)

	nameAttribs := make([]ast.NameAttrib, len(s.Vars))
	for i, name := range s.Vars {
		nameAttribs[i] = ast.NewNameAttrib(name, nil)
	}
	c.CompileStat(ast.LocalStat{
		NameAttribs: nameAttribs,
		Values: []ast.ExpNode{ast.FunctionCall{BFunctionCall: &ast.BFunctionCall{
			Location: s.Params[0].Locate(), // To report the line where the function is if it fails
			Target:   ast.Name{Location: s.Location, Val: string(loopFRegName)},
			Args: []ast.ExpNode{
				ast.Name{Location: s.Location, Val: string(loopSRegName)},
				ast.Name{Location: s.Location, Val: string(loopVarRegName)},
			},
		}}},
	})
	var1, _ := c.GetRegister(ir.Name(s.Vars[0].Val))

	testReg := c.GetFreeRegister()
	c.emitLoadConst(s, ir.NilType{}, testReg)
	c.emitInstr(s, ir.Combine{
		Dst:  testReg,
		Op:   ops.OpEq,
		Lsrc: var1,
		Rsrc: testReg,
	})
	endLbl := c.DeclareGotoLabel(breakLblName)
	c.emitInstr(s, ir.JumpIf{Cond: testReg, Label: endLbl})
	c.emitInstr(s, ir.Transform{Dst: varReg, Op: ops.OpId, Src: var1})
	c.compileBlock(s.Body)

	c.emitInstr(s, ir.Jump{Label: loopLbl})

	c.EmitGotoLabel(breakLblName)
	c.PopContext()

}

// ProcessForStat compiles a ForStat.
func (c *compiler) ProcessForStat(s ast.ForStat) {
	startReg := c.GetFreeRegister()
	r := c.compileExp(s.Start, startReg)
	ir.EmitMoveNoLine(c.CodeBuilder, startReg, r)
	if !ast.IsNumber(s.Start) {
		c.emitInstr(s.Start, ir.Transform{
			Dst: startReg,
			Src: startReg,
			Op:  ops.OpToNumber,
		})
	}
	c.TakeRegister(startReg)

	stopReg := c.GetFreeRegister()
	r = c.compileExp(s.Stop, stopReg)
	ir.EmitMoveNoLine(c.CodeBuilder, stopReg, r)
	if !ast.IsNumber(s.Stop) {
		c.emitInstr(s.Stop, ir.Transform{
			Dst: stopReg,
			Src: stopReg,
			Op:  ops.OpToNumber,
		})
	}
	c.TakeRegister(stopReg)

	stepReg := c.GetFreeRegister()
	r = c.compileExp(s.Step, stepReg)
	ir.EmitMoveNoLine(c.CodeBuilder, stepReg, r)
	if !ast.IsNumber(s.Step) {
		c.emitInstr(s.Step, ir.Transform{
			Dst: stepReg,
			Src: stepReg,
			Op:  ops.OpToNumber,
		})
	}
	c.TakeRegister(stepReg)

	zReg := c.GetFreeRegister()
	c.TakeRegister(zReg)
	c.emitLoadConst(nil, ir.Int(0), zReg)
	c.emitInstr(s, ir.Combine{
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

	c.PushContext()
	iterReg := c.GetFreeRegister()
	ir.EmitMoveNoLine(c.CodeBuilder, iterReg, startReg)
	c.DeclareLocal(ir.Name(s.Var.Val), iterReg)
	c.compileBlock(s.Body)
	c.PopContext()

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

// ProcessFunctionCallStat compiles a FunctionCallStat.
func (c *compiler) ProcessFunctionCallStat(f ast.FunctionCall) {
	c.compileCall(*f.BFunctionCall, false)
	c.emitInstr(f, ir.Receive{})
}

// ProcessGotoStat compiles a GotoStat.
func (c *compiler) ProcessGotoStat(s ast.GotoStat) {
	c.emitJump(s, ir.Name(s.Label.Val))
}

// ProcessIfStat compiles a IfStat.
func (c *compiler) ProcessIfStat(s ast.IfStat) {
	endLbl := c.GetNewLabel()
	lbl := c.GetNewLabel()
	c.compileCond(s.If, lbl)
	for _, s := range s.ElseIfs {
		c.emitInstr(s.Cond, ir.Jump{Label: endLbl}) // TODO: better location
		c.EmitLabel(lbl)
		lbl = c.GetNewLabel()
		c.compileCond(s, lbl)
	}
	if s.Else != nil {
		c.emitInstr(s, ir.Jump{Label: endLbl}) // TODO: better location
		c.EmitLabel(lbl)
		c.CompileStat(s.Else)
	} else {
		c.EmitLabel(lbl)
	}
	c.EmitLabel(endLbl)
}

func (c *compiler) compileCond(s ast.CondStat, lbl ir.Label) {
	condReg := c.compileExpNoDestHint(s.Cond)
	c.emitInstr(s.Cond, ir.JumpIf{Cond: condReg, Label: lbl, Not: true})
	c.CompileStat(s.Body)
}

// ProcessLabelStat compiles a LabelStat.
func (c *compiler) ProcessLabelStat(s ast.LabelStat) {
	if err := c.EmitGotoLabel(ir.Name(s.Name.Val)); err != nil {
		panic(Error{
			Where:   s,
			Message: err.Error(),
		})
	}
}

// ProcessLocalFunctionStat compiles a LocalFunctionStat.
func (c *compiler) ProcessLocalFunctionStat(s ast.LocalFunctionStat) {
	fReg := c.GetFreeRegister()
	c.DeclareLocal(ir.Name(s.Name.Val), fReg)
	c.compileExpInto(s.Function, fReg)
}

// ProcessLocalStat compiles a LocalStat.
func (c *compiler) ProcessLocalStat(s ast.LocalStat) {
	localRegs := make([]ir.Register, len(s.NameAttribs))
	c.compileExpList(s.Values, localRegs)
	for i, reg := range localRegs {
		c.ReleaseRegister(reg)
		c.DeclareLocal(ir.Name(s.NameAttribs[i].Name.Val), reg)
		if s.NameAttribs[i].IsConst() {
			c.MarkConstantReg(reg)
		}
	}
}

// ProcessRepeatStat compiles a RepeatStat.
func (c *compiler) ProcessRepeatStat(s ast.RepeatStat) {
	c.PushContext()
	c.DeclareGotoLabel(breakLblName)

	loopLbl := c.GetNewLabel()
	c.EmitLabel(loopLbl)
	pop := c.compileBlockNoPop(s.Body, false)
	condReg := c.compileExpNoDestHint(s.Cond)
	negReg := c.GetFreeRegister()
	c.emitInstr(s.Cond, ir.Transform{
		Op:  ops.OpNot,
		Dst: negReg,
		Src: condReg,
	})
	pop()
	c.emitInstr(s.Cond, ir.JumpIf{
		Cond:  negReg,
		Label: loopLbl,
	})

	c.EmitGotoLabel(breakLblName)
	c.PopContext()
}

// ProcessWhileStat compiles a WhileStat.
func (c *compiler) ProcessWhileStat(s ast.WhileStat) {
	c.PushContext()
	stopLbl := c.DeclareGotoLabel(breakLblName)

	loopLbl := c.GetNewLabel()
	c.EmitLabel(loopLbl)

	c.compileCond(s.CondStat, stopLbl)

	c.emitInstr(s, ir.Jump{Label: loopLbl}) // TODO: better location

	c.EmitGotoLabel(breakLblName)
	c.PopContext()
}

func (c *compiler) CompileStat(s ast.Stat) {
	s.ProcessStat(c)
}

//
// Helper functions
//

func (c *compiler) compileBlock(s ast.BlockStat) {
	c.compileBlockNoPop(s, true)()
}

func (c *compiler) compileBlockNoPop(s ast.BlockStat, complete bool) func() {
	totalDepth := 0
	getLabels(c.CodeBuilder, s.Stats)
	truncLen := len(s.Stats)
	if complete {
		truncLen -= getBackLabels(c.CodeBuilder, s.Stats)
	}
	for i, stat := range s.Stats {
		switch stat.(type) {
		case ast.LocalStat, ast.LocalFunctionStat:
			totalDepth++
			c.PushContext()
			getLabels(c.CodeBuilder, s.Stats[i+1:truncLen])
		}
		c.CompileStat(stat)
	}
	if s.Return != nil {
		if fc, ok := tailCall(s.Return); ok {
			c.compileCall(*fc.BFunctionCall, true)
		} else {
			contReg := c.getCallerReg()
			c.compilePushArgs(s.Return, contReg)
			var loc ast.Locator
			if len(s.Return) > 0 {
				loc = s.Return[0]
			}
			c.emitInstr(loc, ir.Call{
				Cont: contReg,
				Tail: true,
			})
		}
	}
	return func() {
		for ; totalDepth > 0; totalDepth-- {
			c.PopContext()
		}
	}
}

func getLabels(c *ir.CodeBuilder, statements []ast.Stat) {
	for _, stat := range statements {
		switch s := stat.(type) {
		case ast.LabelStat:
			c.DeclareGotoLabel(ir.Name(s.Name.Val))
		case ast.LocalStat, ast.LocalFunctionStat:
			return
		}
	}
}

func getBackLabels(c *ir.CodeBuilder, statements []ast.Stat) int {
	count := 0
	for i := len(statements) - 1; i >= 0; i-- {
		switch s := statements[i].(type) {
		case ast.EmptyStat:
			// That doesn't count
		case ast.LabelStat:
			c.DeclareGotoLabel(ir.Name(s.Name.Val))
		default:
			return count
		}
		count++
	}
	return count
}

func tailCall(rtn []ast.ExpNode) (ast.FunctionCall, bool) {
	if len(rtn) != 1 {
		return ast.FunctionCall{}, false
	}
	fc, ok := rtn[0].(ast.FunctionCall)
	return fc, ok
}
