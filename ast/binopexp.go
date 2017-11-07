package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

type operand struct {
	op      ops.Op
	operand ExpNode
}

type BinOp struct {
	left   ExpNode
	opType ops.Op
	right  []operand
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

func CompileLogicalOp(c *Compiler, b *BinOp, dst ir.Register, not bool) ir.Register {
	doneLbl := c.GetNewLabel()
	reg := b.left.CompileExp(c, dst)
	EmitMove(c, dst, reg)
	c.Emit(ir.JumpIf{Cond: dst, Label: doneLbl, Not: not})
	for i, r := range b.right {
		reg := r.operand.CompileExp(c, dst)
		EmitMove(c, dst, reg)
		if i < len(b.right) {
			c.Emit(ir.JumpIf{Cond: dst, Label: doneLbl, Not: not})
		}
	}
	c.EmitLabel(doneLbl)
	return dst
}

func (b *BinOp) CompileExp(c *Compiler, dst ir.Register) ir.Register {
	if b.opType == ops.OpAnd {
		return CompileLogicalOp(c, b, dst, true)
	}
	if b.opType == ops.OpOr {
		return CompileLogicalOp(c, b, dst, false)
	}
	lsrc := CompileExp(c, b.left)
	c.TakeRegister(lsrc)
	for _, r := range b.right {
		rsrc := CompileExp(c, r.operand)
		c.Emit(ir.Combine{
			Op:   r.op,
			Dst:  dst,
			Lsrc: lsrc,
			Rsrc: rsrc,
		})
		c.ReleaseRegister(lsrc)
		lsrc = dst
		c.TakeRegister(dst)
	}
	c.ReleaseRegister(dst)
	return dst
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

type UnOp struct {
	op      ops.Op
	operand ExpNode
}

func (u *UnOp) HWrite(w HWriter) {
	w.Writef("unop: %s", u.op)
	w.Indent()
	w.Next()
	u.operand.HWrite(w)
	w.Dedent()
}

func (u *UnOp) CompileExp(c *Compiler, dst ir.Register) ir.Register {
	c.Emit(ir.Transform{
		Op:  u.op,
		Dst: dst,
		Src: CompileExp(c, u.operand),
	})
	return dst
}

func NewUnOp(op ops.Op, exp ExpNode) (*UnOp, error) {
	return &UnOp{
		op:      op,
		operand: exp,
	}, nil
}

type IndexExp struct {
	collection ExpNode
	index      ExpNode
}

func (e IndexExp) CompileExp(c *Compiler, dst ir.Register) ir.Register {
	tReg := CompileExp(c, e.collection)
	c.TakeRegister(tReg)
	iReg := CompileExp(c, e.index)
	c.Emit(ir.Lookup{
		Dst:   dst,
		Table: tReg,
		Index: iReg,
	})
	c.ReleaseRegister(tReg)
	return dst
}

func (e IndexExp) CompileAssign(c *Compiler, src ir.Register) {
	c.TakeRegister(src)
	tReg := CompileExp(c, e.collection)
	c.TakeRegister(tReg)
	iReg := CompileExp(c, e.index)
	c.ReleaseRegister(src)
	c.ReleaseRegister(tReg)
	c.Emit(ir.SetIndex{
		Table: tReg,
		Index: iReg,
		Src:   src,
	})
}

func (e IndexExp) HWrite(w HWriter) {
	w.Writef("idx")
	w.Indent()
	w.Next()
	w.Writef("coll: ")
	e.collection.HWrite(w)
	w.Next()
	w.Writef("at: ")
	e.index.HWrite(w)
	w.Dedent()
}

func NewIndexExp(coll ExpNode, idx ExpNode) (IndexExp, error) {
	return IndexExp{
		collection: coll,
		index:      idx,
	}, nil
}
