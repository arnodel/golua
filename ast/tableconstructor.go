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
	fields []TableField
}

func NewTableConstructor(opTok, clTok *token.Token, fields []TableField) (TableConstructor, error) {
	return TableConstructor{
		Location: LocFromTokens(opTok, clTok),
		fields:   fields,
	}, nil
}

func (c TableConstructor) HWrite(w HWriter) {
	w.Writef("table")
	w.Indent()
	for _, f := range c.fields {
		w.Next()
		w.Writef("key: ")
		f.key.HWrite(w)
		w.Next()
		w.Writef("value: ")
		f.value.HWrite(w)
	}
	w.Dedent()
}

func (t TableConstructor) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	EmitInstr(c, t, ir.MkTable{Dst: dst})
	c.TakeRegister(dst)
	currImplicitKey := int64(1)
	for _, field := range t.fields {
		valReg := CompileExp(c, field.value)
		c.TakeRegister(valReg)
		keyExp := field.key
		if _, ok := keyExp.(NoTableKey); ok {
			keyExp = Int{val: currImplicitKey}
			currImplicitKey++
		}
		keyReg := CompileExp(c, keyExp)
		EmitInstr(c, field.value, ir.SetIndex{
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
	key   ExpNode
	value ExpNode
}

func NewTableField(key ExpNode, value ExpNode) (TableField, error) {
	return TableField{
		Location: MergeLocations(key, value),
		key:      key,
		value:    value,
	}, nil
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
