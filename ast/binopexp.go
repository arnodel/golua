package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

type BinOp struct {
	Location
	left   ExpNode
	opType ops.Op
	right  []operand
}

func NewBinOp(left ExpNode, op ops.Op, right ExpNode) *BinOp {
	loc := MergeLocations(left, right)
	leftOp, ok := left.(*BinOp)
	opType := op & 0xFF
	if ok && leftOp.opType == opType {
		return &BinOp{
			Location: loc,
			left:     leftOp.left,
			opType:   opType,
			right:    append(leftOp.right, operand{op, right}),
		}
	}
	return &BinOp{
		Location: loc,
		left:     left.(ExpNode),
		opType:   opType,
		right:    []operand{{op, right}},
	}
}

func (b *BinOp) HWrite(w HWriter) {
	w.Writef("binop: %d", b.opType)
	w.Indent()
	w.Next()
	b.left.HWrite(w)
	for _, r := range b.right {
		w.Next()
		w.Writef("op: %s", r.op)
		w.Next()
		w.Writef("right: ")
		r.operand.HWrite(w)
	}
	w.Dedent()
}

func (b *BinOp) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	if b.opType == ops.OpAnd {
		return compileLogicalOp(c, b, dst, true)
	}
	if b.opType == ops.OpOr {
		return compileLogicalOp(c, b, dst, false)
	}
	lsrc := CompileExp(c, b.left)
	c.TakeRegister(lsrc)
	for _, r := range b.right {
		rsrc := CompileExp(c, r.operand)
		switch r.op {
		case ops.OpNeq:
			// x ~= y ==> ~(x = y)
			EmitInstr(c, b, ir.Combine{
				Op:   ops.OpEq,
				Dst:  dst,
				Lsrc: lsrc,
				Rsrc: rsrc,
			})
			EmitInstr(c, b, ir.Transform{
				Op:  ops.OpNot,
				Dst: dst,
				Src: dst,
			})
		case ops.OpGt:
			// x > y ==> y < x
			EmitInstr(c, b, ir.Combine{
				Op:   ops.OpLt,
				Dst:  dst,
				Lsrc: rsrc,
				Rsrc: lsrc,
			})
		case ops.OpGeq:
			// x >= y ==> y <= x
			EmitInstr(c, b, ir.Combine{
				Op:   ops.OpLeq,
				Dst:  dst,
				Lsrc: rsrc,
				Rsrc: rsrc,
			})
		default:
			EmitInstr(c, b, ir.Combine{
				Op:   r.op,
				Dst:  dst,
				Lsrc: lsrc,
				Rsrc: rsrc,
			})
		}
		c.ReleaseRegister(lsrc)
		lsrc = dst
		c.TakeRegister(dst)
	}
	c.ReleaseRegister(dst)
	return dst
}

type operand struct {
	op      ops.Op
	operand ExpNode
}

func compileLogicalOp(c *ir.Compiler, b *BinOp, dst ir.Register, not bool) ir.Register {
	doneLbl := c.GetNewLabel()
	reg := b.left.CompileExp(c, dst)
	EmitMove(c, b.left, dst, reg)
	EmitInstr(c, b.left, ir.JumpIf{Cond: dst, Label: doneLbl, Not: not})
	for i, r := range b.right {
		reg := r.operand.CompileExp(c, dst)
		EmitMove(c, r.operand, dst, reg)
		if i < len(b.right) {
			EmitInstr(c, r.operand, ir.JumpIf{Cond: dst, Label: doneLbl, Not: not})
		}
	}
	c.EmitLabel(doneLbl)
	return dst
}
