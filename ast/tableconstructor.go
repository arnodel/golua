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

// HWrite prints a tree representation of the node.
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

// ProcessExp uses the given ExpProcessor to process the receiver.
func (c TableConstructor) ProcessExp(p ExpProcessor) {
	p.ProcessTableConstructorExp(c)
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

// HWrite prints a tree representation of the node.
func (k NoTableKey) HWrite(w HWriter) {
	w.Writef("<no key>")
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (k NoTableKey) ProcessExp(p ExpProcessor) {
	panic("nothing to process?")
}
