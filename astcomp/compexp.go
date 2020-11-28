package astcomp

import (
	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

//
// Expression compilation
//

type expCompiler struct {
	*Compiler
	dst ir.Register
}

var _ ast.ExpProcessor = (*expCompiler)(nil)

// ProcessBFunctionCallExp compiles a BFunctionCall
func (c *expCompiler) ProcessBFunctionCallExp(f ast.BFunctionCall) {
	c.compileCall(f, false)
	c.EmitInstr(f, ir.Receive{Dst: []ir.Register{c.dst}})
}

// ProcessBinOpExp compiles a BinOpExp.
func (c *expCompiler) ProcessBinOpExp(b ast.BinOp) {
	if b.OpType == ops.OpAnd {
		c.compileLogicalOp(b, true)
		return
	}
	if b.OpType == ops.OpOr {
		c.compileLogicalOp(b, false)
		return
	}
	lsrc := c.CompileExpNoDestHint(b.Left)
	for _, r := range b.Right {
		c.TakeRegister(lsrc)
		rsrc := c.CompileExpNoDestHint(r.Operand)
		switch r.Op {
		case ops.OpNeq:
			// x ~= y ==> ~(x = y)
			c.EmitInstr(b, ir.Combine{
				Op:   ops.OpEq,
				Dst:  c.dst,
				Lsrc: lsrc,
				Rsrc: rsrc,
			})
			c.EmitInstr(b, ir.Transform{
				Op:  ops.OpNot,
				Dst: c.dst,
				Src: c.dst,
			})
		case ops.OpGt:
			// x > y ==> y < x
			c.EmitInstr(b, ir.Combine{
				Op:   ops.OpLt,
				Dst:  c.dst,
				Lsrc: rsrc,
				Rsrc: lsrc,
			})
		case ops.OpGeq:
			// x >= y ==> y <= x
			c.EmitInstr(b, ir.Combine{
				Op:   ops.OpLeq,
				Dst:  c.dst,
				Lsrc: rsrc,
				Rsrc: lsrc,
			})
		default:
			c.EmitInstr(b, ir.Combine{
				Op:   r.Op,
				Dst:  c.dst,
				Lsrc: lsrc,
				Rsrc: rsrc,
			})
		}
		c.ReleaseRegister(lsrc)
		lsrc = c.dst
	}
	return
}

// This implements short-circuiting in logical expressions.
func (c *expCompiler) compileLogicalOp(b ast.BinOp, not bool) {
	doneLbl := c.GetNewLabel()
	c.CompileExpInto(b.Left, c.dst)
	c.EmitInstr(b.Left, ir.JumpIf{Cond: c.dst, Label: doneLbl, Not: not})
	for i, r := range b.Right {
		c.CompileExpInto(r.Operand, c.dst)
		if i < len(b.Right) {
			c.EmitInstr(r.Operand, ir.JumpIf{Cond: c.dst, Label: doneLbl, Not: not})
		}
	}
	c.EmitLabel(doneLbl)
}

// ProcesBoolExp compiles a oolExp.
func (c *expCompiler) ProcesBoolExp(b ast.Bool) {
	c.EmitLoadConst(b, ir.Bool(b.Val), c.dst)
}

// ProcessEtcExp compiles a EtcExp.
func (c *expCompiler) ProcessEtcExp(e ast.EtcType) {
	reg := c.getEllipsisReg()
	c.EmitInstr(e, ir.EtcLookup{Dst: c.dst, Etc: reg})
}

// ProcessFunctionExp compiles a Function.
func (c *expCompiler) ProcessFunctionExp(f ast.Function) {
	fc := c.NewChild(f.Name)
	fc.compileFunctionBody(f)
	kidx := c.GetConstant(fc.GetCode())
	c.EmitInstr(f, ir.MkClosure{
		Dst:      c.dst,
		Code:     kidx,
		Upvalues: fc.Upvalues(),
	})
}

// ProcessFunctionCallExp compiles a FunctionCall.
func (c *expCompiler) ProcessFunctionCallExp(f ast.FunctionCall) {
	c.ProcessBFunctionCallExp(*f.BFunctionCall)
}

// ProcessIndexExp compiles a IndexExp.
func (c *expCompiler) ProcessIndexExp(e ast.IndexExp) {
	tReg := c.CompileExpNoDestHint(e.Coll)
	c.TakeRegister(tReg)
	iReg := c.CompileExpNoDestHint(e.Idx)
	c.EmitInstr(e, ir.Lookup{
		Dst:   c.dst,
		Table: tReg,
		Index: iReg,
	})
	c.ReleaseRegister(tReg)
}

// ProcessNameExp compiles a NameExp.
func (c *expCompiler) ProcessNameExp(n ast.Name) {
	// Is it bound to a local name?
	reg, ok := c.GetRegister(ir.Name(n.Val))
	if ok {
		c.dst = reg
		return
	}
	// If not, try _ENV.
	c.CompileExp(globalVar(n))
}

// ProcessNilExp compiles a NilExp.
func (c *expCompiler) ProcessNilExp(n ast.NilType) {
	c.EmitLoadConst(n, ir.NilType{}, c.dst)
}

// ProcessIntExp compiles a IntExp.
func (c *expCompiler) ProcessIntExp(n ast.Int) {
	c.EmitLoadConst(n, ir.Int(n.Val), c.dst)
}

// ProcessFloatExp compiles a FloatExp.
func (c *expCompiler) ProcessFloatExp(f ast.Float) {
	c.EmitLoadConst(f, ir.Float(f.Val), c.dst)
}

// ProcessStringExp compiles a StringExp.
func (c *expCompiler) ProcessStringExp(s ast.String) {
	c.EmitLoadConst(s, ir.String(s.Val), c.dst)
}

// ProcessTableConstructorExp compiles a TableConstructorExp.
func (c *expCompiler) ProcessTableConstructorExp(t ast.TableConstructor) {
	c.EmitInstr(t, ir.MkTable{Dst: c.dst})
	c.TakeRegister(c.dst)
	currImplicitKey := 1
	for i, field := range t.Fields {
		keyExp := field.Key
		_, noKey := keyExp.(ast.NoTableKey)
		if i == len(t.Fields)-1 && noKey {
			tailExp, ok := field.Value.(ast.TailExpNode)
			if ok {
				etc := c.CompileEtcExp(tailExp, c.GetFreeRegister())
				c.EmitInstr(field.Value, ir.FillTable{
					Dst: c.dst,
					Idx: currImplicitKey,
					Etc: etc,
				})
				break
			}
		}
		valReg := c.CompileExpNoDestHint(field.Value)
		c.TakeRegister(valReg)
		if noKey {
			keyExp = ast.Int{Val: uint64(currImplicitKey)}
			currImplicitKey++
		}
		keyReg := c.CompileExpNoDestHint(keyExp)
		c.EmitInstr(field.Value, ir.SetIndex{
			Table: c.dst,
			Index: keyReg,
			Src:   valReg,
		})
		c.ReleaseRegister(valReg)
	}
	c.ReleaseRegister(c.dst)
}

// ProcessUnOpExp compiles a UnOpExp.
func (c *expCompiler) ProcessUnOpExp(u ast.UnOp) {
	c.EmitInstr(u, ir.Transform{
		Op:  u.Op,
		Dst: c.dst,
		Src: c.CompileExpNoDestHint(u.Operand),
	})
}

func (c *expCompiler) CompileExp(e ast.ExpNode) {
	e.ProcessExp(c)
}

type tailExpCompiler struct {
	*Compiler
	dsts []ir.Register
}

var _ ast.TailExpProcessor = tailExpCompiler{}

// ProcessEtcTailExp compiles an Etc tail expression.
func (c tailExpCompiler) ProcessEtcTailExp(e ast.EtcType) {
	reg := c.getEllipsisReg()
	for i, dst := range c.dsts {
		c.EmitInstr(e, ir.EtcLookup{
			Dst: dst,
			Etc: reg,
			Idx: i,
		})
	}
}

// ProcessFunctionCallTailExp compiles a FunctionCall tail expression.
func (c tailExpCompiler) ProcessFunctionCallTailExp(f ast.FunctionCall) {
	c.compileCall(*f.BFunctionCall, false)
	c.EmitInstr(f, ir.Receive{Dst: c.dsts})
}

func (c tailExpCompiler) CompileTailExp(e ast.TailExpNode) {
	e.ProcessTailExp(c)
}

type etcExpCompiler struct {
	*Compiler
	dst ir.Register
}

var _ ast.TailExpProcessor = (*etcExpCompiler)(nil)

func (c *etcExpCompiler) ProcessEtcTailExp(e ast.EtcType) {
	c.dst = c.getEllipsisReg()
}

func (c *etcExpCompiler) ProcessFunctionCallTailExp(f ast.FunctionCall) {
	c.compileCall(*f.BFunctionCall, false)
	c.EmitInstr(f, ir.ReceiveEtc{Etc: c.dst})
}

func (c *etcExpCompiler) CompileTailExp(e ast.TailExpNode) {
	e.ProcessTailExp(c)
}

//
// Helper functions
//

// CompileExp compiles the given expression into a register and returns it.
func (c *Compiler) CompileExp(e ast.ExpNode, dst ir.Register) ir.Register {
	ec := expCompiler{
		Compiler: c,
		dst:      dst,
	}
	ec.CompileExp(e)
	return ec.dst
}

// CompileExpNoDestHint compiles the given expression into any register (perhaps a
// new free one) and returns it.
func (c *Compiler) CompileExpNoDestHint(e ast.ExpNode) ir.Register {
	return c.CompileExp(e, c.GetFreeRegister())
}

// CompileExpInto compiles the given expression into the given register.
func (c *Compiler) CompileExpInto(e ast.ExpNode, dst ir.Register) {
	c.EmitMove(e, dst, c.CompileExp(e, dst))
}

// CompileTailExp compiles the given tail expression into the register slice.
func (c *Compiler) CompileTailExp(e ast.TailExpNode, dsts []ir.Register) {
	tailExpCompiler{
		Compiler: c,
		dsts:     dsts,
	}.CompileTailExp(e)
}

// CompileEtcExp compiles the given tail expression as an etc and returns the
// register the result lives in.
func (c *Compiler) CompileEtcExp(e ast.TailExpNode, dst ir.Register) ir.Register {
	ec := etcExpCompiler{
		Compiler: c,
		dst:      dst,
	}
	ec.CompileTailExp(e)
	return ec.dst
}

// CompileExpList compiles the given expressions into free registers, which are
// recorded into dstRegs (hence exps and dstRegs must have the same length).
// Those registers are taken so need to be released by the caller when no longer
// needed.
func (c *Compiler) CompileExpList(exps []ast.ExpNode, dstRegs []ir.Register) {
	commonCount := len(exps)
	if commonCount > len(dstRegs) {
		commonCount = len(dstRegs)
	}
	var tailExp ast.TailExpNode
	doTailExp := false
	if len(dstRegs) > len(exps) && len(exps) > 0 {
		tailExp, doTailExp = exps[len(exps)-1].(ast.TailExpNode)
		if doTailExp {
			commonCount--
		}
	}
	for i, exp := range exps[:commonCount] {
		dst := c.GetFreeRegister()
		c.CompileExpInto(exp, dst)
		c.TakeRegister(dst)
		dstRegs[i] = dst
	}
	for i := commonCount; i < len(dstRegs); i++ {
		dst := c.GetFreeRegister()
		c.TakeRegister(dst)
		dstRegs[i] = dst
	}
	if doTailExp {
		c.CompileTailExp(tailExp, dstRegs[commonCount:])
	} else if len(dstRegs) > len(exps) {
		nilK := ir.NilType{}
		for _, dst := range dstRegs[len(exps):] {
			c.EmitLoadConst(nil, nilK, dst)
		}
	}
}

func (c *Compiler) compileFunctionBody(f ast.Function) {
	recvRegs := make([]ir.Register, len(f.Params))
	callerReg := c.GetFreeRegister()
	c.DeclareLocal(callerRegName, callerReg)
	for i, p := range f.Params {
		reg := c.GetFreeRegister()
		c.DeclareLocal(ir.Name(p.Val), reg)
		recvRegs[i] = reg
	}
	if !f.HasDots {
		c.EmitInstr(f, ir.Receive{Dst: recvRegs})
	} else {
		reg := c.GetFreeRegister()
		c.DeclareLocal(ellipsisRegName, reg)
		c.EmitInstr(f, ir.ReceiveEtc{Dst: recvRegs, Etc: reg})
	}

	// Need to make sure there is a return instruction emitted at the
	// end.
	body := f.Body
	if body.Return == nil {
		body.Return = []ast.ExpNode{}
	}
	c.CompileBlock(body)

}

func (c *Compiler) getEllipsisReg() ir.Register {
	reg, ok := c.GetRegister(ellipsisRegName)
	if !ok {
		panic("... not defined")
	}
	return reg
}

func (c *Compiler) getCallerReg() ir.Register {
	reg, ok := c.GetRegister(callerRegName)
	if !ok {
		panic("no caller register")
	}
	return reg
}

func (c *Compiler) compileCall(f ast.BFunctionCall, tail bool) {
	fReg := c.CompileExpNoDestHint(f.Target)
	var contReg ir.Register
	if f.Method.Val != "" {
		self := fReg
		c.TakeRegister(self)
		fReg = c.GetFreeRegister()
		mReg := c.GetFreeRegister()
		c.EmitLoadConst(f.Method, ir.String(f.Method.Val), mReg)
		c.EmitInstr(f.Target, ir.Lookup{
			Dst:   fReg,
			Table: self,
			Index: mReg,
		})
		contReg = c.GetFreeRegister()
		c.EmitInstr(f, ir.MkCont{
			Dst:     contReg,
			Closure: fReg,
			Tail:    tail,
		})
		c.EmitInstr(f, ir.Push{
			Cont: fReg,
			Item: self,
		})
		c.ReleaseRegister(self)
	} else {
		contReg = c.GetFreeRegister()
		c.EmitInstr(f, ir.MkCont{
			Dst:     contReg,
			Closure: fReg,
			Tail:    tail,
		})
	}
	c.compilePushArgs(f.Args, contReg)
	c.EmitInstr(f, ir.Call{Cont: contReg})
}

// TODO: move this to somewhere better
func (c *Compiler) compilePushArgs(args []ast.ExpNode, contReg ir.Register) {
	c.TakeRegister(contReg)
	for i, arg := range args {
		var argReg ir.Register
		var isTailArg bool
		if i == len(args)-1 {
			var tailArg ast.TailExpNode
			tailArg, isTailArg = arg.(ast.TailExpNode)
			if isTailArg {
				argReg = c.CompileEtcExp(tailArg, c.GetFreeRegister())
			}
		}
		if !isTailArg {
			argReg = c.CompileExpNoDestHint(arg)
		}
		c.EmitInstr(arg, ir.Push{
			Cont: contReg,
			Item: argReg,
			Etc:  isTailArg,
		})
	}
	c.ReleaseRegister(contReg)
}

func globalVar(n ast.Name) ast.IndexExp {
	return ast.IndexExp{
		Location: n.Location,
		Coll:     ast.Name{Location: n.Location, Val: "_ENV"},
		Idx:      n.AstString(),
	}
}
