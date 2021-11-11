package runtime

import (
	"errors"
)

// Table implements a Lua table.
type Table struct {
	mixedTable

	meta *Table
}

// NewTable returns a new Table.
func NewTable() *Table {
	return &Table{mixedTable: mixedTable{}}
}

// Metatable returns the table's metatable.
func (t *Table) Metatable() *Table {
	return t.meta
}

// SetMetatable sets the table's metatable.
func (t *Table) SetMetatable(m *Table) {
	t.meta = m
}

// Get returns t[k].
func (t *Table) Get(k Value) Value {
	return t.get(k)
}

// Set implements t[k] = v (doesn't check if k is nil).
func (t *Table) Set(k, v Value) uint64 {
	if v.IsNil() {
		t.mixedTable.remove(k)
		return 0
	}
	t.mixedTable.insert(k, v)
	return 16
}

// Reset implements t[k] = v only if t[k] was already non-nil.
func (t *Table) Reset(k, v Value) (wasSet bool) {
	if v.IsNil() {
		return t.mixedTable.remove(k)
	}
	return t.mixedTable.reset(k, v)
}

// SetCheck implements t[k] = v, returns an error if k is nil.
func (t *Table) SetCheck(k, v Value) error {
	if k.IsNil() {
		return errors.New("table index is nil")
	}
	t.Set(k, v)
	return nil
}

// Len returns a length for t (see lua docs for details).
func (t *Table) Len() int64 {
	return int64(t.mixedTable.len())
}

// Next returns the key-value pair that comes after k in the table t.
func (t *Table) Next(k Value) (next Value, val Value, ok bool) {
	return t.mixedTable.next(k)
}
