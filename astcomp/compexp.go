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
	*compiler
	dst ir.Register
}

var _ ast.ExpProcessor = (*expCompiler)(nil)

// ProcessBFunctionCallExp compiles a BFunctionCall
func (c *expCompiler) ProcessBFunctionCallExp(f ast.BFunctionCall) {
	c.compileCall(f, false)
	c.emitInstr(f, ir.Receive{Dst: []ir.Register{c.dst}})
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
	lsrc := c.compileExpNoDestHint(b.Left)
	for _, r := range b.Right {
		c.TakeRegister(lsrc)
		rsrc := c.compileExpNoDestHint(r.Operand)
		switch r.Op {
		case ops.OpNeq:
			// x ~= y ==> ~(x = y)
			c.emitInstr(b, ir.Combine{
				Op:   ops.OpEq,
				Dst:  c.dst,
				Lsrc: lsrc,
				Rsrc: rsrc,
			})
			c.emitInstr(b, ir.Transform{
				Op:  ops.OpNot,
				Dst: c.dst,
				Src: c.dst,
			})
		case ops.OpGt:
			// x > y ==> y < x
			c.emitInstr(b, ir.Combine{
				Op:   ops.OpLt,
				Dst:  c.dst,
				Lsrc: rsrc,
				Rsrc: lsrc,
			})
		case ops.OpGeq:
			// x >= y ==> y <= x
			c.emitInstr(b, ir.Combine{
				Op:   ops.OpLeq,
				Dst:  c.dst,
				Lsrc: rsrc,
				Rsrc: lsrc,
			})
		default:
			c.emitInstr(b, ir.Combine{
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
	c.compileExpInto(b.Left, c.dst)
	c.emitInstr(b.Left, ir.JumpIf{Cond: c.dst, Label: doneLbl, Not: not})
	for i, r := range b.Right {
		c.compileExpInto(r.Operand, c.dst)
		if i < len(b.Right) {
			c.emitInstr(r.Operand, ir.JumpIf{Cond: c.dst, Label: doneLbl, Not: not})
		}
	}
	c.EmitLabel(doneLbl)
}

// ProcesBoolExp compiles a oolExp.
func (c *expCompiler) ProcesBoolExp(b ast.Bool) {
	c.emitLoadConst(b, ir.Bool(b.Val), c.dst)
}

// ProcessEtcExp compiles a EtcExp.
func (c *expCompiler) ProcessEtcExp(e ast.Etc) {
	reg := c.getEllipsisReg()
	c.emitInstr(e, ir.EtcLookup{Dst: c.dst, Etc: reg})
}

// ProcessFunctionExp compiles a Function.
func (c *expCompiler) ProcessFunctionExp(f ast.Function) {
	fc := c.NewChild(f.Name)
	fc.compileFunctionBody(f)
	kidx, upvalues := fc.Close()
	c.emitInstr(f, ir.MkClosure{
		Dst:      c.dst,
		Code:     kidx,
		Upvalues: upvalues,
	})
}

// ProcessFunctionCallExp compiles a FunctionCall.
func (c *expCompiler) ProcessFunctionCallExp(f ast.FunctionCall) {
	c.ProcessBFunctionCallExp(*f.BFunctionCall)
}

// ProcessIndexExp compiles a IndexExp.
func (c *expCompiler) ProcessIndexExp(e ast.IndexExp) {
	tReg := c.compileExpNoDestHint(e.Coll)
	c.TakeRegister(tReg)
	iReg := c.compileExpNoDestHint(e.Idx)
	c.emitInstr(e, ir.Lookup{
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
func (c *expCompiler) ProcessNilExp(n ast.Nil) {
	c.emitLoadConst(n, ir.NilType{}, c.dst)
}

// ProcessIntExp compiles a IntExp.
func (c *expCompiler) ProcessIntExp(n ast.Int) {
	c.emitLoadConst(n, ir.Int(n.Val), c.dst)
}

// ProcessFloatExp compiles a FloatExp.
func (c *expCompiler) ProcessFloatExp(f ast.Float) {
	c.emitLoadConst(f, ir.Float(f.Val), c.dst)
}

// ProcessStringExp compiles a StringExp.
func (c *expCompiler) ProcessStringExp(s ast.String) {
	c.emitLoadConst(s, ir.String(s.Val), c.dst)
}

// ProcessTableConstructorExp compiles a TableConstructorExp.
func (c *expCompiler) ProcessTableConstructorExp(t ast.TableConstructor) {
	c.emitInstr(t, ir.MkTable{Dst: c.dst})
	c.TakeRegister(c.dst)
	currImplicitKey := 1
	for i, field := range t.Fields {
		keyExp := field.Key
		_, noKey := keyExp.(ast.NoTableKey)
		if i == len(t.Fields)-1 && noKey {
			tailExp, ok := field.Value.(ast.TailExpNode)
			if ok {
				etc := c.compileEtcExp(tailExp, c.GetFreeRegister())
				c.emitInstr(field.Value, ir.FillTable{
					Dst: c.dst,
					Idx: currImplicitKey,
					Etc: etc,
				})
				break
			}
		}
		valReg := c.compileExpNoDestHint(field.Value)
		c.TakeRegister(valReg)
		if noKey {
			keyExp = ast.Int{Val: uint64(currImplicitKey)}
			currImplicitKey++
		}
		keyReg := c.compileExpNoDestHint(keyExp)
		c.emitInstr(field.Value, ir.SetIndex{
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
	c.emitInstr(u, ir.Transform{
		Op:  u.Op,
		Dst: c.dst,
		Src: c.compileExpNoDestHint(u.Operand),
	})
}

func (c *expCompiler) CompileExp(e ast.ExpNode) {
	e.ProcessExp(c)
}

//
// Tail Expression compilation
//

type tailExpCompiler struct {
	*compiler
	dsts []ir.Register
}

var _ ast.TailExpProcessor = tailExpCompiler{}

// ProcessEtcTailExp compiles an Etc tail expression.
func (c tailExpCompiler) ProcessEtcTailExp(e ast.Etc) {
	reg := c.getEllipsisReg()
	for i, dst := range c.dsts {
		c.emitInstr(e, ir.EtcLookup{
			Dst: dst,
			Etc: reg,
			Idx: i,
		})
	}
}

// ProcessFunctionCallTailExp compiles a FunctionCall tail expression.
func (c tailExpCompiler) ProcessFunctionCallTailExp(f ast.FunctionCall) {
	c.compileCall(*f.BFunctionCall, false)
	c.emitInstr(f, ir.Receive{Dst: c.dsts})
}

func (c tailExpCompiler) CompileTailExp(e ast.TailExpNode) {
	e.ProcessTailExp(c)
}

//
// Etc Expression compilation
//

type etcExpCompiler struct {
	*compiler
	dst ir.Register
}

var _ ast.TailExpProcessor = (*etcExpCompiler)(nil)

func (c *etcExpCompiler) ProcessEtcTailExp(e ast.Etc) {
	c.dst = c.getEllipsisReg()
}

func (c *etcExpCompiler) ProcessFunctionCallTailExp(f ast.FunctionCall) {
	c.compileCall(*f.BFunctionCall, false)
	c.emitInstr(f, ir.ReceiveEtc{Etc: c.dst})
}

func (c *etcExpCompiler) CompileTailExp(e ast.TailExpNode) {
	e.ProcessTailExp(c)
}

//
// Helper functions
//

// compileExp compiles the given expression into a register and returns it.
func (c *compiler) compileExp(e ast.ExpNode, dst ir.Register) ir.Register {
	ec := expCompiler{
		compiler: c,
		dst:      dst,
	}
	ec.CompileExp(e)
	return ec.dst
}

// compileExpNoDestHint compiles the given expression into any register (perhaps a
// new free one) and returns it.
func (c *compiler) compileExpNoDestHint(e ast.ExpNode) ir.Register {
	return c.compileExp(e, c.GetFreeRegister())
}

// compileExpInto compiles the given expression into the given register.
func (c *compiler) compileExpInto(e ast.ExpNode, dst ir.Register) {
	c.emitMove(e, dst, c.compileExp(e, dst))
}

// compileTailExp compiles the given tail expression into the register slice.
func (c *compiler) compileTailExp(e ast.TailExpNode, dsts []ir.Register) {
	tailExpCompiler{
		compiler: c,
		dsts:     dsts,
	}.CompileTailExp(e)
}

// compileEtcExp compiles the given tail expression as an etc and returns the
// register the result lives in.
func (c *compiler) compileEtcExp(e ast.TailExpNode, dst ir.Register) ir.Register {
	ec := etcExpCompiler{
		compiler: c,
		dst:      dst,
	}
	ec.CompileTailExp(e)
	return ec.dst
}

// compileExpList compiles the given expressions into free registers, which are
// recorded into dstRegs (hence exps and dstRegs must have the same length).
// Those registers are taken so need to be released by the caller when no longer
// needed.
func (c *compiler) compileExpList(exps []ast.ExpNode, dstRegs []ir.Register) {
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
		c.compileExpInto(exp, dst)
		c.TakeRegister(dst)
		dstRegs[i] = dst
	}
	for i := commonCount; i < len(dstRegs); i++ {
		dst := c.GetFreeRegister()
		c.TakeRegister(dst)
		dstRegs[i] = dst
	}
	if doTailExp {
		c.compileTailExp(tailExp, dstRegs[commonCount:])
	} else if len(dstRegs) > len(exps) {
		nilK := ir.NilType{}
		for _, dst := range dstRegs[len(exps):] {
			c.emitLoadConst(nil, nilK, dst)
		}
	}
}

func (c *compiler) compileFunctionBody(f ast.Function) {
	recvRegs := make([]ir.Register, len(f.Params))
	callerReg := c.GetFreeRegister()
	c.DeclareLocal(callerRegName, callerReg)
	for i, p := range f.Params {
		reg := c.GetFreeRegister()
		c.DeclareLocal(ir.Name(p.Val), reg)
		recvRegs[i] = reg
	}
	if !f.HasDots {
		c.emitInstr(f, ir.Receive{Dst: recvRegs})
	} else {
		reg := c.GetFreeRegister()
		c.DeclareLocal(ellipsisRegName, reg)
		c.emitInstr(f, ir.ReceiveEtc{Dst: recvRegs, Etc: reg})
	}

	// Need to make sure there is a return instruction emitted at the
	// end.
	body := f.Body
	if body.Return == nil {
		body.Return = []ast.ExpNode{}
	}
	c.compileBlock(body)

}

func (c *compiler) getEllipsisReg() ir.Register {
	reg, ok := c.GetRegister(ellipsisRegName)
	if !ok {
		panic("... not defined")
	}
	return reg
}

func (c *compiler) getCallerReg() ir.Register {
	reg, ok := c.GetRegister(callerRegName)
	if !ok {
		panic("no caller register")
	}
	return reg
}

func (c *compiler) compileCall(f ast.BFunctionCall, tail bool) {
	fReg := c.compileExpNoDestHint(f.Target)
	var contReg ir.Register
	if f.Method.Val != "" {
		self := fReg
		c.TakeRegister(self)
		fReg = c.GetFreeRegister()
		mReg := c.GetFreeRegister()
		c.emitLoadConst(f.Method, ir.String(f.Method.Val), mReg)
		c.emitInstr(f.Target, ir.Lookup{
			Dst:   fReg,
			Table: self,
			Index: mReg,
		})
		contReg = c.GetFreeRegister()
		c.emitInstr(f, ir.MkCont{
			Dst:     contReg,
			Closure: fReg,
			Tail:    tail,
		})
		c.emitInstr(f, ir.Push{
			Cont: fReg,
			Item: self,
		})
		c.ReleaseRegister(self)
	} else {
		contReg = c.GetFreeRegister()
		c.emitInstr(f, ir.MkCont{
			Dst:     contReg,
			Closure: fReg,
			Tail:    tail,
		})
	}
	c.compilePushArgs(f.Args, contReg)
	c.emitInstr(f, ir.Call{
		Cont: contReg,
		Tail: tail,
	})
}

// TODO: move this to somewhere better
func (c *compiler) compilePushArgs(args []ast.ExpNode, contReg ir.Register) {
	c.TakeRegister(contReg)
	for i, arg := range args {
		var argReg ir.Register
		var isTailArg bool
		if i == len(args)-1 {
			var tailArg ast.TailExpNode
			tailArg, isTailArg = arg.(ast.TailExpNode)
			if isTailArg {
				argReg = c.compileEtcExp(tailArg, c.GetFreeRegister())
			}
		}
		if !isTailArg {
			argReg = c.compileExpNoDestHint(arg)
		}
		c.emitInstr(arg, ir.Push{
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
