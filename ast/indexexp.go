package ast

import "github.com/arnodel/golua/ir"

type IndexExp struct {
	Location
	Coll ExpNode
	Idx  ExpNode
}

func NewIndexExp(coll ExpNode, idx ExpNode) IndexExp {
	return IndexExp{
		Location: MergeLocations(coll, idx), // TODO: use the "]" for locaion end
		Coll:     coll,
		Idx:      idx,
	}
}

func (e IndexExp) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	tReg := CompileExp(c, e.Coll)
	c.TakeRegister(tReg)
	iReg := CompileExp(c, e.Idx)
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
	CompileExpInto(c, e.Coll, tReg)
	c.TakeRegister(tReg)
	iReg := c.GetFreeRegister()
	CompileExpInto(c, e.Idx, iReg)
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
	if s, ok := e.Idx.(String); ok {
		return string(s.Val)
	}
	return ""
}

func (e IndexExp) HWrite(w HWriter) {
	w.Writef("idx")
	w.Indent()
	w.Next()
	w.Writef("coll: ")
	e.Coll.HWrite(w)
	w.Next()
	w.Writef("at: ")
	e.Idx.HWrite(w)
	w.Dedent()
}
