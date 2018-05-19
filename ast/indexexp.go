package ast

import "github.com/arnodel/golua/ir"

type IndexExp struct {
	Location
	collection ExpNode
	index      ExpNode
}

func NewIndexExp(coll ExpNode, idx ExpNode) (IndexExp, error) {
	return IndexExp{
		collection: coll,
		index:      idx,
	}, nil
}

func (e IndexExp) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
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

func (e IndexExp) CompileAssign(c *ir.Compiler, src ir.Register) {
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
