package runtime

import (
	"errors"
	"math/bits"
)

// Table implements a Lua table.
type Table struct {
	newIndexCount int

	array []tableValue

	// This table has the key-value pairs of the table.
	assoc map[Value]tableValue

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
	return &Table{assoc: make(map[Value]tableValue)}
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
	return t.getTableValue(k).value
}

// Set implements t[k] = v (doesn't check if k is nil).
func (t *Table) Set(k, v Value) {
	if n, ok := ToIntNoString(k); ok {
		t.setInt(n, v)
	} else {
		t.set(k, v)
	}
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
		for t.border > 0 && t.getInt(t.border).value.IsNil() {
			t.border--
		}
		t.borderState = borderOK
	case borderCheckUp:
		for !t.getInt(t.border + 1).value.IsNil() {
			t.border++ // I don't know if that ever happens (can't test it!)
		}
		t.borderState = borderOK
	}
	return t.border
}

// Next returns the key-value pair that comes after k in the table t.
func (t *Table) Next(k Value) (next Value, val Value, ok bool) {
	var tv tableValue
	ok = true
	if k.IsNil() {
		next = t.first
		ok = true
	} else {
		tv = t.getTableValue(k)
		if tv.value.IsNil() {
			ok = false
			return
		}
		next = tv.next
	}
	// Because we may have removed entries by setting values to nil, we loop
	// until we find a non-nil value.
	for !next.IsNil() {
		tv = t.getTableValue(next)
		val = tv.value
		if !val.IsNil() {
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

func (t *Table) getTableValue(k Value) tableValue {
	if n, ok := ToIntNoString(k); ok {
		return t.getInt(n)
	}
	return t.assoc[k]
}

func (t *Table) getInt(n int64) tableValue {
	if 1 <= n && int(n) <= len(t.array) {
		return t.array[int(n-1)]
	}
	return t.assoc[IntValue(n)]
}

func (t *Table) setInt(n int64, v Value) {
	arrlen := len(t.array)
	switch {
	case n > t.border && !v.IsNil():
		t.border = n
		t.borderState = borderCheckUp
	case v.IsNil() && t.border > 0 && n == t.border:
		t.border--
		t.borderState = borderCheckDown
	}
	if 1 <= n && int(n) <= arrlen {
		if t.setArray(n, v) {
			t.newIndexCount++
		}

	} else {
		if t.set(IntValue(n), v) && n > 0 {
			t.newIndexCount++
		}
	}
	if t.newIndexCount*2 >= arrlen {
		// log.Printf("Rebalance! newIndexCount=%d, arrlen=%d", t.newIndexCount, arrlen)
		t.newIndexCount = 0
		t.rebalance()
	}
}

func (t *Table) set(k Value, v Value) (isNew bool) {
	tv, ok := t.assoc[k]
	if v.IsNil() && !ok {
		return false
	}
	tv.value = v
	if !ok {
		tv.next = t.first
		t.first = k
	}
	t.assoc[k] = tv
	return !ok
}

func (t *Table) setArray(n int64, v Value) (isNew bool) {
	tv := &t.array[int(n-1)]
	isNew = tv.value.IsNil()
	if v.IsNil() && isNew {
		return false
	}
	tv.value = v
	if isNew {
		tv.next = t.first
		t.first = IntValue(n)
	}
	return
}

func (t *Table) rebalance() {
	var byLen [64]uint64
	for i := range t.array {
		byLen[bits.Len(uint(i))]++
	}
	for k := range t.assoc {
		n, ok := k.TryInt()
		if ok && n > 0 {
			byLen[bits.Len(uint(n-1))]++
		}
	}
	acc := byLen[0]
	var arrlen uint64
	var l uint64
	for i, c := range byLen[1:] {
		acc += c
		// log.Printf("i = %d, acc = %d", i, acc)
		if acc>>uint64(i) != 0 {
			l = uint64(i) + 1
			// log.Printf("Bing! l = %d", l)
		}
	}
	if l > 0 {
		arrlen = 1 << l
	}
	// log.Printf("new arrlen=%d", arrlen)
	var array []tableValue
	if arrlen > 0 {
		array = make([]tableValue, arrlen)
	}
	for k, tv := range t.assoc {
		n, ok := k.TryInt()
		if ok && n > 0 && uint64(n) <= arrlen {
			array[n-1] = tv
			delete(t.assoc, k)
		}
	}
	for i, tv := range t.array {
		if uint64(i) < arrlen {
			array[i] = tv
		} else {
			t.assoc[IntValue(int64(i+1))] = tv
		}
	}
	t.array = array
}
