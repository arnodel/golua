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
	border int64

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
	if x, ok := k.TryFloat(); ok {
		if n, tp := floatToInt(x); tp == IsInt {
			k = IntValue(n)
		}
	}
	return t.content[k].value
}

// Set implements t[k] = v (doesn't check if k is nil).
func (t *Table) Set(k, v Value) {
	switch k.Type() {
	case IntType:
		t.setInt(k.AsInt(), v)
		return
	case FloatType:
		if n, tp := floatToInt(k.AsFloat()); tp == IsInt {
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
func (t *Table) Len() int64 {
	switch t.borderState {
	case borderCheckDown:
		for t.border > 0 && t.content[IntValue(t.border)].value.IsNil() {
			t.border--
		}
		t.borderState = borderOK
	case borderCheckUp:
		for !t.content[IntValue(t.border+1)].value.IsNil() {
			t.border++ // I don't know if that ever happens (can't test it!)
		}
		t.borderState = borderOK
	}
	return t.border
}

// Next returns the key-value pair that comes after k in the table t.
func (t *Table) Next(k Value) (next Value, val Value, ok bool) {
	var tv tableValue
	if k.IsNil() {
		next = t.first
		ok = true
	} else {
		tv, ok = t.content[k]
		if !ok {
			return
		}
		next = tv.next
	}
	// Because we may have removed entries by setting values to nil, we loop
	// until we find a non-nil value.
	for !next.IsNil() {
		tv = t.content[next]
		if val = tv.value; !val.IsNil() {
			return
		}
		next = tv.next
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

func (t *Table) setInt(n int64, v Value) {
	switch {
	case n > t.border && !v.IsNil():
		t.border = n
		t.borderState = borderCheckUp
	case v.IsNil() && t.border > 0 && n == t.border:
		t.border--
		t.borderState = borderCheckDown
	}
	t.set(IntValue(n), v)
}

func (t *Table) set(k Value, v Value) {
	tv, ok := t.content[k]
	if v.IsNil() && !ok {
		return
	}
	tv.value = v
	if !ok {
		tv.next = t.first
		t.first = k
	}
	t.content[k] = tv
}
