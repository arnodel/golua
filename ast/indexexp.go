package ast

import "github.com/arnodel/golua/ir"

type IndexExp struct {
	Location
	collection ExpNode
	index      ExpNode
}

func NewIndexExp(coll ExpNode, idx ExpNode) (IndexExp, error) {
	return IndexExp{
		Location:   MergeLocations(coll, idx), // TODO: use the "]" for locaion end
		collection: coll,
		index:      idx,
	}, nil
}

func (e IndexExp) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	tReg := CompileExp(c, e.collection)
	c.TakeRegister(tReg)
	iReg := CompileExp(c, e.index)
	EmitInstr(c, e, ir.Lookup{
		Dst:   dst,
		Table: tReg,
		Index: iReg,
	})
	c.ReleaseRegister(tReg)
	return dst
}

func (e IndexExp) CompileAssign(c *ir.Compiler) Assign {
	tReg := c.GetFreeRegister()
	CompileExpInto(c, e.collection, tReg)
	c.TakeRegister(tReg)
	iReg := c.GetFreeRegister()
	CompileExpInto(c, e.index, iReg)
	c.TakeRegister(iReg)
	return func(src ir.Register) {
		c.ReleaseRegister(tReg)
		c.ReleaseRegister(iReg)
		EmitInstr(c, e, ir.SetIndex{
			Table: tReg,
			Index: iReg,
			Src:   src,
		})
	}
}

func (e IndexExp) FunctionName() string {
	if s, ok := e.index.(String); ok {
		return string(s.val)
	}
	return ""
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
