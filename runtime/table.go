package runtime

import "errors"

// Table implements a Lua table.
type Table struct {

	// This table has the key-value pairs of the table.
	content map[Value]tableValue

	// This is the metatable.
	meta *Table

	// The first key in the table
	first Value

	// The border is a positive integer index n such that t[n] != nil but t[n+1]
	// == nil.  If there is none such, it should be 0.  It is expensive to keep
	// it up to date so it might not always be correct, in which case
	// borderState gives a clue to finding its true value.
	border Int

	// If borderState == borderOK, then border is correct.  Otherwise you need
	// to look up or down depending on its value (borderCheckUp or
	// borderCheckDown).
	borderState int64
}

// NewTable returns a new Table.
func NewTable() *Table {
	return &Table{content: make(map[Value]tableValue)}
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
	if x, ok := k.(Float); ok {
		if n, tp := x.ToInt(); tp == IsInt {
			k = n
		}
	}
	return t.content[k].value
}

// Set implements t[k] = v (doesn't check if k is nil).
func (t *Table) Set(k, v Value) {
	switch x := k.(type) {
	case Int:
		t.setInt(x, v)
		return
	case Float:
		if n, tp := x.ToInt(); tp == IsInt {
			t.setInt(n, v)
			return
		}
	}
	t.set(k, v)
}

// SetCheck implements t[k] = v, returns an error if k is nil.
func (t *Table) SetCheck(k, v Value) error {
	if IsNil(k) {
		return errors.New("table index is nil")
	}
	t.Set(k, v)
	return nil
}

// Len returns a length for t (see lua docs for details).
func (t *Table) Len() Int {
	switch t.borderState {
	case borderCheckDown:
		for t.border > 0 && t.content[t.border].value == nil {
			t.border--
		}
		t.borderState = borderOK
	case borderCheckUp:
		for t.content[t.border+1].value != nil {
			t.border++
		}
		t.borderState = borderOK
	}
	return t.border
}

// Next returns the key-value pair that comes after k in the table t.
func (t *Table) Next(k Value) (next Value, val Value) {
	if IsNil(k) {
		next = t.first
	} else {
		next = t.content[k].next
	}
	for next != nil {
		ntv := t.content[next]
		if val = ntv.value; val != nil {
			return
		}
		next = ntv.next
	}
	return
}

const (
	borderOK int64 = iota
	borderCheckUp
	borderCheckDown
)

type tableValue struct {
	value, next Value
}

func (t *Table) setInt(n Int, v Value) {
	switch {
	case n > t.border && v != nil:
		t.border = n
		t.borderState = borderCheckUp
	case v == nil && t.border > 0 && n == t.border:
		t.border--
		t.borderState = borderCheckDown
	}
	t.set(n, v)
}

func (t *Table) set(k Value, v Value) {
	tv, ok := t.content[k]
	if v == nil && !ok {
		return
	}
	tv.value = v
	if !ok {
		tv.next = t.first
		t.first = k
	}
	t.content[k] = tv
}
