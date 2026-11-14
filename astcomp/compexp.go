package astcomp

import (
	"fmt"

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
	if b.OpType == ops.OpAnd || b.OpType == ops.OpOr {
		c.compileLogicalOp(b)
		return
	}
	lsrc := c.compileExpNoDestHint(b.Left)
	for _, r := range b.Right {
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
		lsrc = c.dst
	}
}

// compileLogicalOp compiles a chain of "and" or of "or" as a value, with
// short-circuiting: an operand that decides the result (false in an "and"
// chain, true in an "or" chain) is the result, and jumps to the end with its
// value.  If none does, the last operand is the result.
func (c *expCompiler) compileLogicalOp(b ast.BinOp) {
	operands := logicalOperands(b)
	deciding := b.OpType == ops.OpOr
	doneLbl := c.GetNewLabel()
	last := len(operands) - 1
	for _, e := range operands[:last] {
		c.compileValueJump(e, c.dst, doneLbl, deciding)
	}
	c.compileExpInto(operands[last], c.dst)
	must(c.EmitLabelNoLine(doneLbl))
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
	if s, ok := e.Idx.(ast.String); ok {
		// A constant key (e.g. a field name) needs no register.
		c.emitInstr(e, ir.LookupK{
			Dst:   c.dst,
			Table: tReg,
			Kidx:  c.GetConstantIndex(ir.String(s.Val)),
		})
		return
	}
	iReg := c.compileExpNoDestHint(e.Idx)
	c.emitInstr(e, ir.Lookup{
		Dst:   c.dst,
		Table: tReg,
		Index: iReg,
	})
}

// ProcessNameExp compiles a NameExp.
func (c *expCompiler) ProcessNameExp(n ast.Name) {
	// Check if it's bound to a local name
	reg, ok := c.GetRegister(ir.Name(n.Val))
	if ok {
		c.dst = reg
		return
	}
	// This is a global variable - validate access is authorized
	declType := c.GetGlobalDeclType(ir.Name(n.Val))
	if !declType.IsAllowed() {
		panic(Error{
			Where:   n,
			Message: fmt.Sprintf("attempt to read undeclared global variable '%s'", n.Val),
		})
	}
	// Access the global via _ENV
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
	// The table is made with capacity for what the constructor lists: values
	// without a key go to its array part, the rest are counted as fields.
	// A last value that may be several (a call, "...") is not counted.
	var nvalues, nfields int
	for i, field := range t.Fields {
		if _, noKey := field.Key.(ast.NoTableKey); !noKey {
			nfields++
		} else if _, tail := field.Value.(ast.TailExpNode); !tail || i < len(t.Fields)-1 {
			nvalues++
		}
	}
	c.emitInstr(t, ir.MkTable{Dst: c.dst, NValues: nvalues, NFields: nfields})
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
		if noKey {
			keyExp = ast.Int{Val: uint64(currImplicitKey)}
			currImplicitKey++
		}
		if s, ok := keyExp.(ast.String); ok {
			c.emitInstr(field.Value, ir.SetIndexK{
				Table: c.dst,
				Kidx:  c.GetConstantIndex(ir.String(s.Val)),
				Src:   valReg,
			})
		} else {
			keyReg := c.compileExpNoDestHint(keyExp)
			c.emitInstr(field.Value, ir.SetIndex{
				Table: c.dst,
				Index: keyReg,
				Src:   valReg,
			})
		}
	}
}

// ProcessUnOpExp compiles a UnOpExp.
func (c *expCompiler) ProcessUnOpExp(u ast.UnOp) {
	c.emitInstr(u, ir.Transform{
		Op:  u.Op,
		Dst: c.dst,
		Src: c.compileExpNoDestHint(u.Operand),
	})
}

// constantValue returns the value of e if it is known at compile time: a
// literal, or a negated number literal (the parser reads "-1" as "-" applied
// to 1).  Integers wrap around as "-" does at run time.  A float zero is left
// to be negated at run time: the constant table can't tell 0.0 from -0.0.
func constantValue(e ast.ExpNode) (ir.Constant, bool) {
	switch x := e.(type) {
	case ast.Nil:
		return ir.NilType{}, true
	case ast.Bool:
		return ir.Bool(x.Val), true
	case ast.Int:
		return ir.Int(x.Val), true
	case ast.Float:
		return ir.Float(x.Val), true
	case ast.String:
		return ir.String(x.Val), true
	}
	if u, ok := asUnOp(e); ok && u.Op == ops.OpNeg {
		k, _ := constantValue(u.Operand)
		switch k := k.(type) {
		case ir.Int:
			return -k, true
		case ir.Float:
			return -k, k != 0
		}
	}
	return nil, false
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
	if k, ok := constantValue(e); ok {
		c.emitLoadConst(e, k, dst)
		return dst
	}
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
		dstRegs[i] = dst
	}
	for i := commonCount; i < len(dstRegs); i++ {
		dst := c.GetFreeRegister()
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
		etcReg := c.GetFreeRegister()
		c.DeclareLocal(ellipsisRegName, etcReg)
		c.emitInstr(f, ir.ReceiveEtc{Dst: recvRegs, Etc: etcReg})

		if f.VarargName != nil {
			// Create a table whose array part references the vararg data
			tableReg := c.GetFreeRegister()
			c.emitInstr(f, ir.MkVarargTable{Dst: tableReg, Etc: etcReg})
			c.DeclareLocal(ir.Name(f.VarargName.Val), tableReg)
			c.MarkConstantReg(tableReg)
		}
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
	self := noRegister
	if f.Method.Val != "" {
		// self is listed in the call, so it is read at the call: copy a
		// captured variable now (see stable).
		self = c.stable(f.Target, fReg)
		fReg = c.GetFreeRegister()
		c.emitInstr(f.Target, ir.LookupK{
			Dst:   fReg,
			Table: self,
			Kidx:  c.GetConstantIndex(ir.String(f.Method.Val)),
		})
	}
	// A call lists its arguments in its operand words and makes the callee's
	// frame itself, unless the last argument is an etc (a vararg or a call),
	// whose values must come after the others and whose number is not known,
	// or there are more arguments than fit: then the callee's continuation
	// is made first and everything is pushed into it one by one.
	if etc, listed := splitArgs(f.Args); listed {
		// The callee is read at the call, after the arguments: a captured
		// variable is copied first when an argument could reassign it (see
		// stable), and copy propagation reads through the copy when nothing
		// can (a fixed cell, see ir.markFixed).  With nothing to evaluate
		// in between, the call reads the variable itself.
		callee := fReg
		if len(f.Args) > 0 || self != noRegister {
			callee = c.stable(f.Target, fReg)
		}
		c.compileListedCall(f, callee, self, f.Args, tail)
	} else {
		contReg := c.GetFreeRegister()
		c.emitInstr(f, ir.MkCont{
			Dst:     contReg,
			Closure: fReg,
			Tail:    tail,
		})
		c.compilePushedCall(f, contReg, self, f.Args, etc, tail)
	}
}

// splitArgs says whether a call with these arguments lists them in its
// operand words (see compileCall), and returns the etc among them if there
// is one.
func splitArgs(args []ast.ExpNode) (etc ast.TailExpNode, listed bool) {
	if n := len(args); n > 0 {
		etc, _ = args[n-1].(ast.TailExpNode)
	}
	return etc, etc == nil && len(args) <= maxListedArgs
}

// noRegister stands for "no register" where one is optional.
const noRegister = ^ir.Register(0)

// maxListedArgs is the most arguments a call lists in its operand words.  The
// listed arguments are all live until the call, and a frame has 256
// registers; beyond this many they are pushed one by one as before.
const maxListedArgs = 32

// compileListedCall evaluates the arguments, then emits the call of callee
// with the argument registers listed in it (self first, if given): the call
// makes the callee's frame and pushes them itself.
func (c *compiler) compileListedCall(loc ast.Locator, callee, self ir.Register, args []ast.ExpNode, tail bool) {
	var argRegs []ir.Register
	if self != noRegister {
		argRegs = append(argRegs, self)
	}
	for _, arg := range args {
		argRegs = append(argRegs, c.stable(arg, c.compileExpNoDestHint(arg)))
	}
	c.emitInstr(loc, ir.Call{Cont: callee, Args: argRegs, Tail: tail})
}

// compilePushedCall pushes self (if given) and the arguments one by one into
// contReg, the callee's continuation, the etc last, then emits the call.
func (c *compiler) compilePushedCall(loc ast.Locator, contReg, self ir.Register, args []ast.ExpNode, etc ast.TailExpNode, tail bool) {
	if self != noRegister {
		c.emitInstr(loc, ir.Push{Cont: contReg, Item: self})
	}
	if etc != nil {
		args = args[:len(args)-1]
	}
	for _, arg := range args {
		c.emitInstr(arg, ir.Push{Cont: contReg, Item: c.compileExpNoDestHint(arg)})
	}
	if etc != nil {
		etcReg := c.compileEtcExp(etc, c.GetFreeRegister())
		c.emitInstr(etc, ir.Push{Cont: contReg, Item: etcReg, Etc: true})
	}
	c.emitInstr(loc, ir.Call{Cont: contReg, Tail: tail})
}

// stable returns a register holding the value reg has now, for a value that
// is read later than it is evaluated (an argument listed in a call is read at
// the call).  A captured variable (a cell) is copied into a temporary: a
// function called while evaluating a later argument could reassign it, and
// "f(x, g())" must pass the x it had before g ran.  Other registers cannot
// change in between, as Lua expressions do not assign to locals.
func (c *compiler) stable(e ast.Locator, reg ir.Register) ir.Register {
	if !c.IsCellReg(reg) {
		return reg
	}
	tmp := c.GetFreeRegister()
	c.emitMove(e, tmp, reg)
	return tmp
}

func globalVar(n ast.Name) ast.IndexExp {
	return ast.IndexExp{
		Location: n.Location,
		Coll:     ast.Name{Location: n.Location, Val: "_ENV"},
		Idx:      n.AstString(),
	}
}
