package astcomp

import (
	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

//
// Condition compilation
//

// compileCondJump emits code that jumps to lbl if the truth of e is jumpIf,
// and falls through otherwise.  A condition only decides where to go next, so
// unlike compiling e as a value, this needs no register to hold the result of
// "and", "or" or "not": "if a or b and c then" becomes
//
//	if a jump THEN; if not b jump ELSE; if not c jump ELSE; THEN: ...
func (c *compiler) compileCondJump(e ast.ExpNode, lbl ir.Label, jumpIf bool) {
	e.ProcessExp(condJumpCompiler{compiler: c, lbl: lbl, jumpIf: jumpIf})
}

// compileValueJump is compileCondJump for an operand of "and" or "or"
// computed as a value: when it jumps, the value of e is in dst.  When it
// falls through, the value isn't needed, and dst holds nothing in particular.
// So in "x = a and b or c", a false a jumps straight to c.
func (c *compiler) compileValueJump(e ast.ExpNode, dst ir.Register, lbl ir.Label, jumpIf bool) {
	e.ProcessExp(condJumpCompiler{compiler: c, lbl: lbl, jumpIf: jumpIf, keep: true, dst: dst})
}

// condJumpCompiler compiles an expression as a condition, for
// compileCondJump and compileValueJump.  Expressions it has no shortcut for
// are computed as a value, then tested.
type condJumpCompiler struct {
	*compiler
	lbl    ir.Label
	jumpIf bool
	keep   bool        // the jump to lbl carries the value in dst
	dst    ir.Register // if keep is true
}

var _ ast.ExpProcessor = condJumpCompiler{}

// ProcessUnOpExp compiles a UnOp: "not" swaps the jump's direction, unless
// its value is needed.
func (c condJumpCompiler) ProcessUnOpExp(u ast.UnOp) {
	if u.Op != ops.OpNot || c.keep {
		c.jumpOnValue(u)
		return
	}
	c.compileCondJump(u.Operand, c.lbl, !c.jumpIf)
}

// ProcessBinOpExp compiles a BinOp: "and" and "or" jump on each operand, and
// "x ~= y", which is "not (x == y)", puts the "not" into the jump unless its
// value is needed.
func (c condJumpCompiler) ProcessBinOpExp(b ast.BinOp) {
	switch {
	case b.OpType == ops.OpAnd || b.OpType == ops.OpOr:
		c.logicalJump(b)
	case len(b.Right) == 1 && b.Right[0].Op == ops.OpNeq && !c.keep:
		b.Right = []ast.Operation{{Op: ops.OpEq, Operand: b.Right[0].Operand}}
		r := c.compileExpNoDestHint(b)
		c.emitInstr(b, ir.JumpIf{Cond: r, Label: c.lbl, Not: c.jumpIf})
	default:
		c.jumpOnValue(b)
	}
}

// logicalJump compiles a chain of "and" or of "or".  In an "and" chain a
// false operand decides the result, in an "or" chain a true one.  If that is
// the result that jumps, every operand can jump to lbl.  Otherwise a deciding
// operand skips past the chain, and the last operand decides.  The value of
// the chain is the value of the operand that jumps to lbl, so those operands
// keep it; the ones that skip don't need to.
func (c condJumpCompiler) logicalJump(b ast.BinOp) {
	operands := logicalOperands(b)
	deciding := b.OpType == ops.OpOr
	if deciding == c.jumpIf {
		for _, e := range operands {
			e.ProcessExp(c)
		}
		return
	}
	skip := c.GetNewLabel()
	last := len(operands) - 1
	for _, e := range operands[:last] {
		c.compileCondJump(e, skip, deciding)
	}
	operands[last].ProcessExp(c)
	must(c.EmitLabelNoLine(skip))
}

// logicalOperands returns the operands of a chain of "and" or of "or".
func logicalOperands(b ast.BinOp) []ast.ExpNode {
	operands := []ast.ExpNode{b.Left}
	for _, r := range b.Right {
		operands = append(operands, r.Operand)
	}
	return operands
}

// Constants: the jump is unconditional, or there is none.  A constant that
// doesn't jump needs no value.

func (c condJumpCompiler) ProcesBoolExp(b ast.Bool)      { c.constant(b, b.Val) }
func (c condJumpCompiler) ProcessNilExp(n ast.Nil)       { c.constant(n, false) }
func (c condJumpCompiler) ProcessIntExp(n ast.Int)       { c.constant(n, true) }
func (c condJumpCompiler) ProcessFloatExp(f ast.Float)   { c.constant(f, true) }
func (c condJumpCompiler) ProcessStringExp(s ast.String) { c.constant(s, true) }

func (c condJumpCompiler) constant(e ast.ExpNode, truth bool) {
	if truth == c.jumpIf {
		if c.keep {
			c.compileExpInto(e, c.dst)
		}
		c.emitInstr(e, ir.Jump{Label: c.lbl})
	}
}

// The other expressions are computed as a value, then tested.

func (c condJumpCompiler) ProcessBFunctionCallExp(f ast.BFunctionCall) { c.jumpOnValue(f) }
func (c condJumpCompiler) ProcessEtcExp(e ast.Etc)                     { c.jumpOnValue(e) }
func (c condJumpCompiler) ProcessFunctionExp(f ast.Function)           { c.jumpOnValue(f) }
func (c condJumpCompiler) ProcessFunctionCallExp(f ast.FunctionCall)   { c.jumpOnValue(f) }
func (c condJumpCompiler) ProcessIndexExp(e ast.IndexExp)              { c.jumpOnValue(e) }
func (c condJumpCompiler) ProcessNameExp(n ast.Name)                   { c.jumpOnValue(n) }
func (c condJumpCompiler) ProcessTableConstructorExp(t ast.TableConstructor) {
	c.jumpOnValue(t)
}

func (c condJumpCompiler) jumpOnValue(e ast.ExpNode) {
	var r ir.Register
	if c.keep {
		r = c.dst
		c.compileExpInto(e, r)
	} else {
		r = c.compileExpNoDestHint(e)
	}
	c.emitInstr(e, ir.JumpIf{Cond: r, Label: c.lbl, Not: !c.jumpIf})
}
