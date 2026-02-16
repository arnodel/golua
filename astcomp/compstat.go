package astcomp

import (
	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

const globalAttribError = "only <const> is allowed for global declarations"

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
	initRegs := make([]ir.Register, 4)
	c.compileExpList(s.Params, initRegs)
	fReg := initRegs[0]
	sReg := initRegs[1]
	varReg := initRegs[2]
	closeReg := initRegs[3]

	c.PushContext()
	c.PushCloseAction(closeReg) // Now closeReg is no longer needed
	c.DeclareLocal(loopFRegName, fReg)
	c.DeclareLocal(loopSRegName, sReg)
	c.DeclareLocal(loopVarRegName, varReg)

	loopLbl := c.GetNewLabel()
	must(c.EmitLabelNoLine(loopLbl))

	nameAttribs := make([]ast.NameAttrib, len(s.Vars))
	for i, name := range s.Vars {
		constDeclAttrib := ast.NewDeclAttrib(ast.Location{}, ast.ConstAttrib)
		nameAttribs[i] = ast.NewNameAttrib(name, &constDeclAttrib) // Loop variables are read-only in Lua 5.5
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
	endLbl := c.DeclareGotoLabelNoLine(breakLblName)
	c.emitInstr(s, ir.JumpIf{Cond: testReg, Label: endLbl})
	c.emitInstr(s, ir.Transform{Dst: varReg, Op: ops.OpId, Src: var1})
	c.compileBlock(s.Body)

	c.emitInstr(s, ir.Jump{Label: loopLbl})

	must(c.EmitGotoLabel(breakLblName))
	c.PopContext()

}

// ProcessForStat compiles a ForStat.
func (c *compiler) ProcessForStat(s ast.ForStat) {

	// Get register for current value of i and initialise it
	startReg := c.GetFreeRegister()
	r := c.compileExp(s.Start, startReg)
	ir.EmitMoveNoLine(c.CodeBuilder, startReg, r)
	c.TakeRegister(startReg)

	// Get register for the stop value and initialise it
	stopReg := c.GetFreeRegister()
	r = c.compileExp(s.Stop, stopReg)
	ir.EmitMoveNoLine(c.CodeBuilder, stopReg, r)
	c.TakeRegister(stopReg)

	// Get register for the step value and initialise it
	stepReg := c.GetFreeRegister()
	r = c.compileExp(s.Step, stepReg)
	ir.EmitMoveNoLine(c.CodeBuilder, stepReg, r)
	c.TakeRegister(stepReg)

	// Prepare the for loop
	c.emitInstr(s, ir.PrepForLoop{
		Start: startReg,
		Stop:  stopReg,
		Step:  stepReg,
	})

	c.PushContext()
	loopLbl := c.GetNewLabel()
	must(c.EmitLabelNoLine(loopLbl))
	endLbl := c.DeclareGotoLabelNoLine(breakLblName)

	// If startReg is nil, then there are no iterations in the loop
	c.EmitNoLine(ir.JumpIf{
		Cond:  startReg,
		Label: endLbl,
		Not:   true,
	})

	// Here compile the loop body
	c.PushContext()
	iterReg := c.GetFreeRegister()
	// We copy the loop variable because the body may change it
	// iter <- start
	ir.EmitMoveNoLine(c.CodeBuilder, iterReg, startReg)
	c.DeclareLocal(ir.Name(s.Var.Val), iterReg)
	c.MarkConstantReg(iterReg) // Loop variable is read-only in Lua 5.5
	c.compileBlock(s.Body)
	c.PopContext()

	//Advance the for loop
	c.emitInstr(s, ir.AdvForLoop{
		Start: startReg,
		Stop:  stopReg,
		Step:  stepReg,
	})
	// If startReg is not nil, it means the loop continues
	c.EmitNoLine(ir.JumpIf{
		Cond:  startReg,
		Label: loopLbl,
	})

	// break:
	must(c.EmitGotoLabel(breakLblName))
	c.PopContext()

	c.ReleaseRegister(startReg)
	c.ReleaseRegister(stopReg)
	c.ReleaseRegister(stepReg)
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
		must(c.EmitLabelNoLine(lbl))
		lbl = c.GetNewLabel()
		c.compileCond(s, lbl)
	}
	if s.Else != nil {
		c.emitInstr(s, ir.Jump{Label: endLbl}) // TODO: better location
		must(c.EmitLabelNoLine(lbl))
		c.CompileStat(s.Else)
	} else {
		must(c.EmitLabelNoLine(lbl))
	}
	must(c.EmitLabelNoLine(endLbl))
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
		nameAttrib := s.NameAttribs[i]
		c.DeclareLocal(ir.Name(nameAttrib.Name.Val), reg)
		attrib := nameAttrib.Attrib
		if attrib == nil {
			attrib = s.PrefixAttrib
		}
		if attrib != nil {
			switch attrib.Type {
			case ast.ConstAttrib:
				c.MarkConstantReg(reg)
			case ast.CloseAttrib:
				c.MarkConstantReg(reg)
				c.PushCloseAction(reg)
			default:
				panic(compilerBug{})
			}
		}
	}
}

// ProcessGlobalFunctionStat compiles a GlobalFunctionStat.
func (c *compiler) ProcessGlobalFunctionStat(s ast.GlobalFunctionStat) {
	checkGlobalName(s.Name)
	// First, declare the global (as mutable, since function values can be reassigned)
	c.DeclareGlobal(ir.Name(s.Name.Val), ir.MutableGlobal)

	// Compile the function and assign it to the global
	fReg := c.GetFreeRegister()
	c.compileExpInto(s.Function, fReg)
	c.TakeRegister(fReg)

	// Check that the global is not already defined, then assign via _ENV
	lvals := []ast.Var{globalVar(s.Name)}
	c.compileDefineAssignments(lvals, []ir.Register{fReg})
}

// ProcessGlobalStat compiles a GlobalStat.
func (c *compiler) ProcessGlobalStat(s ast.GlobalStat) {
	if s.PrefixAttrib != nil && s.PrefixAttrib.Type != ast.ConstAttrib {
		panic(Error{Where: s.PrefixAttrib, Message: globalAttribError})
	}

	// Compile the values BEFORE declaring the globals, so that RHS expressions
	// like "global a = a" correctly read any local 'a' from outer scope.
	var valueRegs []ir.Register
	if len(s.Values) > 0 {
		valueRegs = make([]ir.Register, len(s.NameAttribs))
		c.compileExpList(s.Values, valueRegs)
	}

	// Check that _ENV is not being declared as a global
	for _, nameAttrib := range s.NameAttribs {
		checkGlobalName(nameAttrib.Name)
	}

	// Now register the global declarations
	for _, nameAttrib := range s.NameAttribs {
		var declType ir.GlobalDeclType
		attrib := nameAttrib.Attrib
		if attrib == nil {
			attrib = s.PrefixAttrib
		}
		if attrib != nil {
			if attrib.Type != ast.ConstAttrib {
				panic(Error{Where: attrib, Message: globalAttribError})
			}
			declType = ir.ConstGlobal
		} else {
			declType = ir.MutableGlobal
		}
		c.DeclareGlobal(ir.Name(nameAttrib.Name.Val), declType)
	}

	// Handle value assignments (to globals via _ENV)
	if len(valueRegs) > 0 {
		lvals := make([]ast.Var, len(s.NameAttribs))
		for i, nameAttrib := range s.NameAttribs {
			lvals[i] = globalVar(nameAttrib.Name)
		}
		c.compileDefineAssignments(lvals, valueRegs)
	}
}

// ProcessGlobalWildcardStat compiles a GlobalWildcardStat.
func (c *compiler) ProcessGlobalWildcardStat(s ast.GlobalWildcardStat) {
	var declType ir.GlobalDeclType
	if s.Attrib != nil {
		if s.Attrib.Type != ast.ConstAttrib {
			panic(Error{Where: s.Attrib, Message: globalAttribError})
		}
		declType = ir.ConstGlobal
	} else {
		declType = ir.MutableGlobal
	}
	c.SetGlobalWildcard(declType)
}

// ProcessRepeatStat compiles a RepeatStat.
func (c *compiler) ProcessRepeatStat(s ast.RepeatStat) {
	c.PushContext()
	c.DeclareGotoLabelNoLine(breakLblName)

	loopLbl := c.GetNewLabel()
	must(c.EmitLabelNoLine(loopLbl))
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

	must(c.EmitGotoLabel(breakLblName))
	c.PopContext()
}

// ProcessWhileStat compiles a WhileStat.
func (c *compiler) ProcessWhileStat(s ast.WhileStat) {
	c.PushContext()
	stopLbl := c.DeclareGotoLabelNoLine(breakLblName)

	loopLbl := c.GetNewLabel()
	must(c.EmitLabelNoLine(loopLbl))

	c.compileCond(s.CondStat, stopLbl)

	c.emitInstr(s, ir.Jump{Label: loopLbl}) // TODO: better location

	must(c.EmitGotoLabel(breakLblName))
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
	noBackLabels := getLabels(c.CodeBuilder, s.Stats)
	truncLen := len(s.Stats)
	if complete && !noBackLabels && s.Return == nil {
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
		if fc, ok := c.getTailCall(s.Return); ok {
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

// Declares goto labels for the statements in order, stopping when encountering
// a local variable declaration.  Return true if the whole slice was processed
// (so no need to get back labels)
func getLabels(c *ir.CodeBuilder, statements []ast.Stat) bool {
	for _, stat := range statements {
		switch s := stat.(type) {
		case ast.LabelStat:
			_, err := c.DeclareUniqueGotoLabel(ir.Name(s.Name.Val), s.Name.StartPos().Line)
			if err != nil {
				panic(Error{
					Where:   s.Name,
					Message: err.Error(),
				})
			}
		case ast.LocalStat, ast.LocalFunctionStat:
			return false
		}
	}
	return true
}

// Process the statements in reverse order to declare "back labels".  Return the
// number of statements processed.
func getBackLabels(c *ir.CodeBuilder, statements []ast.Stat) int {
	count := 0
	for i := len(statements) - 1; i >= 0; i-- {
		switch s := statements[i].(type) {
		case ast.EmptyStat:
			// That doesn't count
		case ast.LabelStat:
			_, err := c.DeclareUniqueGotoLabel(ir.Name(s.Name.Val), s.Name.StartPos().Line)
			if err != nil {
				panic(Error{
					Where:   s.Name,
					Message: err.Error(),
				})
			}
		default:
			return count
		}
		count++
	}
	return count
}

func (c *compiler) getTailCall(rtn []ast.ExpNode) (ast.FunctionCall, bool) {
	if len(rtn) != 1 || c.HasPendingCloseActions() {
		return ast.FunctionCall{}, false
	}
	fc, ok := rtn[0].(ast.FunctionCall)
	return fc, ok
}

// checkGlobalName panics with a compile error if name is not valid for a
// global declaration.
func checkGlobalName(name ast.Name) {
	if name.Val == "_ENV" {
		panic(Error{Where: name, Message: "'_ENV' cannot be declared as a global variable"})
	}
}
