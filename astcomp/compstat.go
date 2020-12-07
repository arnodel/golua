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
var _ ast.StatProcessor = (*Compiler)(nil)

// ProcessAssignStat compiles a AssignStat.
func (c *Compiler) ProcessAssignStat(s ast.AssignStat) {

	// Evaluate the right hand side
	resultRegs := make([]ir.Register, len(s.Dest))
	c.CompileExpList(s.Src, resultRegs)

	// Compile the lvalues and assignments
	c.CompileAssignments(s.Dest, resultRegs)
}

// ProcessBlockStat compiles a BlockStat.
func (c *Compiler) ProcessBlockStat(s ast.BlockStat) {
	c.PushContext()
	c.CompileBlock(s)
	c.PopContext()
}

// ProcessBreakStat compiles a BreakStat.
func (c *Compiler) ProcessBreakStat(s ast.BreakStat) {
	c.EmitJump(s, breakLblName)
}

// ProcessEmptyStat compiles a EmptyStat.
func (c *Compiler) ProcessEmptyStat(s ast.EmptyStat) {
	// Nothing to compile!
}

// ProcessForInStat compiles a ForInStat.
func (c *Compiler) ProcessForInStat(s ast.ForInStat) {
	initRegs := make([]ir.Register, 3)
	c.CompileExpList(s.Params, initRegs)
	fReg := initRegs[0]
	sReg := initRegs[1]
	varReg := initRegs[2]

	c.PushContext()
	c.DeclareLocal(loopFRegName, fReg)
	c.DeclareLocal(loopSRegName, sReg)
	c.DeclareLocal(loopVarRegName, varReg)

	loopLbl := c.GetNewLabel()
	c.EmitLabel(loopLbl)

	// TODO: better locations

	c.CompileStat(ast.LocalStat{
		Names: s.Vars,
		Values: []ast.ExpNode{ast.FunctionCall{BFunctionCall: &ast.BFunctionCall{
			Location: s.Location,
			Target:   ast.Name{Location: s.Location, Val: string(loopFRegName)},
			Args: []ast.ExpNode{
				ast.Name{Location: s.Location, Val: string(loopSRegName)},
				ast.Name{Location: s.Location, Val: string(loopVarRegName)},
			},
		}}},
	})
	var1, _ := c.GetRegister(ir.Name(s.Vars[0].Val))

	testReg := c.GetFreeRegister()
	c.EmitLoadConst(s, ir.NilType{}, testReg)
	c.EmitInstr(s, ir.Combine{
		Dst:  testReg,
		Op:   ops.OpEq,
		Lsrc: var1,
		Rsrc: testReg,
	})
	endLbl := c.DeclareGotoLabel(breakLblName)
	c.EmitInstr(s, ir.JumpIf{Cond: testReg, Label: endLbl})
	c.EmitInstr(s, ir.Transform{Dst: varReg, Op: ops.OpId, Src: var1})
	c.CompileBlock(s.Body)

	c.EmitInstr(s, ir.Jump{Label: loopLbl})

	c.EmitGotoLabel(breakLblName)
	c.PopContext()

}

// ProcessForStat compiles a ForStat.
func (c *Compiler) ProcessForStat(s ast.ForStat) {
	startReg := c.GetFreeRegister()
	r := c.CompileExp(s.Start, startReg)
	ir.EmitMoveNoLine(c.CodeBuilder, startReg, r)
	if !ast.IsNumber(s.Start) {
		c.EmitNoLine(ir.Transform{
			Dst: startReg,
			Src: startReg,
			Op:  ops.OpToNumber,
		})
	}
	c.TakeRegister(startReg)

	stopReg := c.GetFreeRegister()
	r = c.CompileExp(s.Stop, stopReg)
	ir.EmitMoveNoLine(c.CodeBuilder, stopReg, r)
	if !ast.IsNumber(s.Stop) {
		c.EmitNoLine(ir.Transform{
			Dst: stopReg,
			Src: stopReg,
			Op:  ops.OpToNumber,
		})
	}
	c.TakeRegister(stopReg)

	stepReg := c.GetFreeRegister()
	r = c.CompileExp(s.Step, stepReg)
	ir.EmitMoveNoLine(c.CodeBuilder, stepReg, r)
	if !ast.IsNumber(s.Step) {
		c.EmitNoLine(ir.Transform{
			Dst: stepReg,
			Src: stepReg,
			Op:  ops.OpToNumber,
		})
	}
	c.TakeRegister(stepReg)

	zReg := c.GetFreeRegister()
	c.TakeRegister(zReg)
	c.EmitLoadConst(nil, ir.Int(0), zReg)
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

	c.PushContext()
	iterReg := c.GetFreeRegister()
	ir.EmitMoveNoLine(c.CodeBuilder, iterReg, startReg)
	c.DeclareLocal(ir.Name(s.Var.Val), iterReg)
	c.CompileBlock(s.Body)
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
func (c *Compiler) ProcessFunctionCallStat(f ast.FunctionCall) {
	c.compileCall(*f.BFunctionCall, false)
	c.EmitInstr(f, ir.Receive{})
}

// ProcessGotoStat compiles a GotoStat.
func (c *Compiler) ProcessGotoStat(s ast.GotoStat) {
	c.EmitJump(s, ir.Name(s.Label.Val))
}

// ProcessIfStat compiles a IfStat.
func (c *Compiler) ProcessIfStat(s ast.IfStat) {
	endLbl := c.GetNewLabel()
	lbl := c.GetNewLabel()
	c.compileCond(s.If, lbl)
	for _, s := range s.ElseIfs {
		c.EmitInstr(s.Cond, ir.Jump{Label: endLbl}) // TODO: better location
		c.EmitLabel(lbl)
		lbl = c.GetNewLabel()
		c.compileCond(s, lbl)
	}
	if s.Else != nil {
		c.EmitInstr(s, ir.Jump{Label: endLbl}) // TODO: better location
		c.EmitLabel(lbl)
		c.CompileStat(s.Else)
	} else {
		c.EmitLabel(lbl)
	}
	c.EmitLabel(endLbl)
}

func (c *Compiler) compileCond(s ast.CondStat, lbl ir.Label) {
	condReg := c.CompileExpNoDestHint(s.Cond)
	c.EmitInstr(s.Cond, ir.JumpIf{Cond: condReg, Label: lbl, Not: true})
	c.CompileStat(s.Body)
}

// ProcessLabelStat compiles a LabelStat.
func (c *Compiler) ProcessLabelStat(s ast.LabelStat) {
	c.EmitGotoLabel(ir.Name(s.Name.Val))
}

// ProcessLocalFunctionStat compiles a LocalFunctionStat.
func (c *Compiler) ProcessLocalFunctionStat(s ast.LocalFunctionStat) {
	fReg := c.GetFreeRegister()
	c.DeclareLocal(ir.Name(s.Name.Val), fReg)
	c.CompileExpInto(s.Function, fReg)
}

// ProcessLocalStat compiles a LocalStat.
func (c *Compiler) ProcessLocalStat(s ast.LocalStat) {
	localRegs := make([]ir.Register, len(s.Names))
	c.CompileExpList(s.Values, localRegs)
	for i, reg := range localRegs {
		c.ReleaseRegister(reg)
		c.DeclareLocal(ir.Name(s.Names[i].Val), reg)
	}
}

// ProcessRepeatStat compiles a RepeatStat.
func (c *Compiler) ProcessRepeatStat(s ast.RepeatStat) {
	c.PushContext()
	c.DeclareGotoLabel(breakLblName)

	loopLbl := c.GetNewLabel()
	c.EmitLabel(loopLbl)
	pop := c.CompileBlockNoPop(s.Body)
	condReg := c.CompileExpNoDestHint(s.Cond)
	negReg := c.GetFreeRegister()
	c.EmitInstr(s.Cond, ir.Transform{
		Op:  ops.OpNot,
		Dst: negReg,
		Src: condReg,
	})
	pop()
	c.EmitInstr(s.Cond, ir.JumpIf{
		Cond:  negReg,
		Label: loopLbl,
	})

	c.EmitGotoLabel(breakLblName)
	c.PopContext()
}

// ProcessWhileStat compiles a WhileStat.
func (c *Compiler) ProcessWhileStat(s ast.WhileStat) {
	c.PushContext()
	stopLbl := c.DeclareGotoLabel(breakLblName)

	loopLbl := c.GetNewLabel()
	c.EmitLabel(loopLbl)

	c.compileCond(s.CondStat, stopLbl)

	c.EmitInstr(s, ir.Jump{Label: loopLbl}) // TODO: better location

	c.EmitGotoLabel(breakLblName)
	c.PopContext()
}

func (c *Compiler) CompileStat(s ast.Stat) {
	s.ProcessStat(c)
}

//
// Helper functions
//

func (c *Compiler) CompileBlock(s ast.BlockStat) {
	c.CompileBlockNoPop(s)()
}

func (c *Compiler) CompileBlockNoPop(s ast.BlockStat) func() {
	totalDepth := 0
	getLabels(c.CodeBuilder, s.Stats)
	truncLen := len(s.Stats) - getBackLabels(c.CodeBuilder, s.Stats)
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
			c.EmitInstr(loc, ir.Call{
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
		if lbl, ok := statements[i].(ast.LabelStat); ok {
			count++
			c.DeclareGotoLabel(ir.Name(lbl.Name.Val))
		} else {
			break
		}
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
