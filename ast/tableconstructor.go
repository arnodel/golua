package ast

import (
	"github.com/arnodel/golua/token"
)

//
// TabelConstructor
//

// A TableConstructor is an expression node representing a table literal, e.g.
//
//    { "hello", 4.5, x = 2, [z] = true }
type TableConstructor struct {
	Location
	Fields []TableField
}

var _ ExpNode = TableConstructor{}

// NewTableConstructor returns a TableConstructor instance with the given fields.
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

// A TableField is a (key, value ) pair of expression nodes representing a field
// in a table literal.  There is a special key type called "NoTableKey" for
// fields that do not have a key.
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

// NoTableKey is a special expression node that can only be used as the Key of a
// TableField, meaning a field with no key.
type NoTableKey struct {
	Location
}

// HWrite prints a tree representation of the node.
func (k NoTableKey) HWrite(w HWriter) {
	w.Writef("<no key>")
}

// ProcessExp uses the given ExpProcessor to process the receiver.  It panics as
// this type is a placeholder that should not be processed as an expression.
func (k NoTableKey) ProcessExp(p ExpProcessor) {
	panic("nothing to process?")
}
