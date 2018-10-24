package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

//
// TabelConstructor
//

type TableConstructor struct {
	Location
	Fields []TableField
}

func NewTableConstructor(opTok, clTok *token.Token, fields []TableField) TableConstructor {
	return TableConstructor{
		Location: LocFromTokens(opTok, clTok),
		Fields:   fields,
	}
}

func (c TableConstructor) HWrite(w HWriter) {
	w.Writef("table")
	w.Indent()
	for _, f := range c.Fields {
		w.Next()
		w.Writef("key: ")
		f.Key.HWrite(w)
		w.Next()
		w.Writef("value: ")
		f.Value.HWrite(w)
	}
	w.Dedent()
}

func (t TableConstructor) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	EmitInstr(c, t, ir.MkTable{Dst: dst})
	c.TakeRegister(dst)
	currImplicitKey := 1
	for i, field := range t.Fields {
		keyExp := field.Key
		_, noKey := keyExp.(NoTableKey)
		if i == len(t.Fields)-1 && noKey {
			tailExp, ok := field.Value.(TailExpNode)
			if ok {
				etc := tailExp.CompileEtcExp(c, c.GetFreeRegister())
				EmitInstr(c, field.Value, ir.FillTable{
					Dst: dst,
					Idx: currImplicitKey,
					Etc: etc,
				})
				break
			}
		}
		valReg := CompileExp(c, field.Value)
		c.TakeRegister(valReg)
		if _, ok := keyExp.(NoTableKey); ok {
			keyExp = Int{val: uint64(currImplicitKey)}
			currImplicitKey++
		}
		keyReg := CompileExp(c, keyExp)
		EmitInstr(c, field.Value, ir.SetIndex{
			Table: dst,
			Index: keyReg,
			Src:   valReg,
		})
		c.ReleaseRegister(valReg)
	}
	c.ReleaseRegister(dst)
	return dst
}

//
// TableField
//

type TableField struct {
	Location
	Key   ExpNode
	Value ExpNode
}

func NewTableField(key ExpNode, value ExpNode) TableField {
	return TableField{
		Location: MergeLocations(key, value),
		Key:      key,
		Value:    value,
	}
}

//
// NoTableKey
//

type NoTableKey struct {
	Location
}

func (k NoTableKey) HWrite(w HWriter) {
	w.Writef("<no key>")
}

func (k NoTableKey) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	panic("NoTableKey should not be compiled")
}
