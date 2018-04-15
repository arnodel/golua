package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

type BinOp struct {
	left   ExpNode
	opType ops.Op
	right  []operand
}

func NewBinOp(left ExpNode, op ops.Op, right ExpNode) (*BinOp, error) {
	leftOp, ok := left.(*BinOp)
	opType := op & 0xFF
	if ok && leftOp.opType == opType {
		return &BinOp{
			left:   leftOp.left,
			opType: opType,
			right:  append(leftOp.right, operand{op, right}),
		}, nil
	}
	return &BinOp{
		left:   left.(ExpNode),
		opType: opType,
		right:  []operand{{op, right}},
	}, nil
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
			c.Emit(ir.Combine{
				Op:   ops.OpEq,
				Dst:  dst,
				Lsrc: lsrc,
				Rsrc: rsrc,
			})
			c.Emit(ir.Transform{
				Op:  ops.OpNot,
				Dst: dst,
				Src: dst,
			})
		case ops.OpGt:
			// x > y ==> y < x
			c.Emit(ir.Combine{
				Op:   ops.OpLt,
				Dst:  dst,
				Lsrc: rsrc,
				Rsrc: lsrc,
			})
		case ops.OpGeq:
			// x >= y ==> y <= x
			c.Emit(ir.Combine{
				Op:   ops.OpLeq,
				Dst:  dst,
				Lsrc: rsrc,
				Rsrc: rsrc,
			})
		default:
			c.Emit(ir.Combine{
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
	ir.EmitMove(c, dst, reg)
	c.Emit(ir.JumpIf{Cond: dst, Label: doneLbl, Not: not})
	for i, r := range b.right {
		reg := r.operand.CompileExp(c, dst)
		ir.EmitMove(c, dst, reg)
		if i < len(b.right) {
			c.Emit(ir.JumpIf{Cond: dst, Label: doneLbl, Not: not})
		}
	}
	c.EmitLabel(doneLbl)
	return dst
}
