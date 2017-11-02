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

func (b *BinOp) CompileExp(c *Compiler) ir.Register {
	lsrc := b.left.CompileExp(c)
	dst := c.NewRegister()
	var rsrc ir.Register
	for _, r := range b.right {
		rsrc = r.operand.CompileExp(c)
		c.Emit(ir.Combine{
			Op:   r.op,
			Dst:  dst,
			Lsrc: lsrc,
			Rsrc: rsrc,
		})
		lsrc = dst
	}
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

func (u *UnOp) CompileExp(c *Compiler) ir.Register {
	dst := c.NewRegister()
	c.Emit(ir.Transform{
		Op:  u.op,
		Dst: dst,
		Src: u.operand.CompileExp(c),
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

func (e IndexExp) CompileExp(c *Compiler) ir.Register {
	dst := c.NewRegister()
	c.Emit(ir.Lookup{
		Dst:   dst,
		Table: e.collection.CompileExp(c),
		Index: e.index.CompileExp(c),
	})
	return dst
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
