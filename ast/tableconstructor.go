package ast

import "github.com/arnodel/golua/ir"

type TableConstructor []TableField

type NoTableKey struct{}

func (k NoTableKey) HWrite(w HWriter) {
	w.Writef("<no key>")
}

func (k NoTableKey) CompileExp(c *Compiler, dst ir.Register) ir.Register {
	panic("NoTableKey should not be compiled")
}

func (c TableConstructor) HWrite(w HWriter) {
	w.Writef("table")
	w.Indent()
	for _, f := range c {
		w.Next()
		w.Writef("key: ")
		f.key.HWrite(w)
		w.Next()
		w.Writef("value: ")
		f.value.HWrite(w)
	}
	w.Dedent()
}

func (t TableConstructor) CompileExp(c *Compiler, dst ir.Register) ir.Register {
	// TODO
	return dst
}

type TableField struct {
	key   ExpNode
	value ExpNode
}

func NewTableConstructor(fields []TableField) (TableConstructor, error) {
	return fields, nil
}

func NewTableField(key ExpNode, value ExpNode) (TableField, error) {
	return TableField{
		key:   key,
		value: value,
	}, nil
}
