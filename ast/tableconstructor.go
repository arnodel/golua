package ast

import (
	"github.com/arnodel/golua/token"
)

//
// TabelConstructor
//

type TableConstructor struct {
	Location
	Fields []TableField
}

var _ ExpNode = TableConstructor{}

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

func (t TableConstructor) ProcessExp(p ExpProcessor) {
	p.ProcessTableConstructorExp(t)
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

func (k NoTableKey) ProcessExp(p ExpProcessor) {
	panic("nothing to process?")
}
