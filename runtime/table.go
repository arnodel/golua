package runtime

import (
	"unsafe"

	"github.com/arnodel/golua/runtime/internal/luagc"
)

// Table implements a Lua table.
type Table struct {
	// This is where the implementation details are.
	*mixedTable

	meta *Table
}

// NewTable returns a new Table.
func NewTable() *Table {
	return &Table{mixedTable: &mixedTable{}}
}

// NewTableFromSlice creates a table whose array part references the given slice.
// The slice is NOT copied - modifications to the table affect the underlying slice.
func NewTableFromSlice(values []Value) *Table {
	return &Table{
		mixedTable: &mixedTable{
			array: &array{
				values: values,
				len:    uintptr(len(values)),
			},
		},
	}
}

// NewTableWithCapacity creates a table with preallocated capacity.
// This is used for table.create (Lua 5.5) to avoid repeated reallocations.
// nseq: capacity hint for array part (sequence elements)
// nrec: capacity hint for hash part (record/key-value pairs)
func NewTableWithCapacity(nseq, nrec int) *Table {
	return &Table{
		mixedTable: newMixedTableWithCapacity(nseq, nrec),
	}
}

// Metatable returns the table's metatable.
func (t *Table) Metatable() *Table {
	return t.meta
}

// SetMetatable sets the table's metatable.
func (t *Table) SetMetatable(m *Table) {
	t.meta = m
}

var _ luagc.Value = (*Table)(nil)

func (t *Table) Key() luagc.Key {
	return unsafe.Pointer(t.mixedTable)
}

func (t *Table) Clone() luagc.Value {
	clone := new(Table)
	*clone = *t
	return clone
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

// Len returns a length for t (see lua docs for details).
func (t *Table) Len() int64 {
	return int64(t.mixedTable.len())
}

// Next returns the key-value pair that comes after k in the table t.
//   - If k is NilValue, the first key-value pair in the table t is returned.
//   - If k is the last key in the table t, a pair of NilValues is returned.
//   - If the table t is empty, the returned key-value pair is always a pair of NilValues, regardless of k.
//   - In all cases, ok is true if and only if k is either NilValue or a key present in the table t.
func (t *Table) Next(k Value) (next Value, val Value, ok bool) {
	return t.mixedTable.next(k)
}

// IsArray reports whether t contains an array part.
//
//   - IsArray returns true if and only if the array part of t contains at least one value.
//   - Keys stored only in the map part do not affect the result.
//   - An empty table always returns false.
func (t *Table) IsArray() bool {
	return t.mixedTable.array.itemCount() > 0
}

// IsMap reports whether t contains a map part.
//
//   - IsMap returns true if and only if the map part of t contains at least one key-value pair.
//   - Values stored only in the array part do not affect the result.
//   - An empty table always returns false.
func (t *Table) IsMap() bool {
	for _, s := range t.mixedTable.hashTable.slots {
		if !s.isEmpty() {
			return true
		}
	}
	return false
}

// IsMixed reports whether t contains both an array part and a map part.
//
//   - IsMixed returns true if and only if both IsArray and IsMap return true.
//   - An empty table always returns false.
func (t *Table) IsMixed() bool {
	return t.IsArray() && t.IsMap()
}

// IsArrayOnly reports whether t contains an array part and no map part.
//
//   - IsArrayOnly returns true if and only if IsArray returns true and IsMap returns false.
//   - An empty table always returns false.
func (t *Table) IsArrayOnly() bool {
	return t.IsArray() && !t.IsMap()
}

// IsMapOnly reports whether t contains a map part and no array part.
//
//   - IsMapOnly returns true if and only if IsArray returns false and IsMap returns true.
//   - An empty table always returns false.
func (t *Table) IsMapOnly() bool {
	return !t.IsArray() && t.IsMap()
}

// MapKeys returns the keys stored in the map part of t.
//
//   - Only keys stored in the map part are included.
//   - Keys stored in the array part are never included.
//   - If the map part of t is empty, MapKeys returns an empty slice.
//   - The order of the returned keys is unspecified and should not be relied upon.
func (t *Table) MapKeys() []Value {
	keys := make([]Value, 0)
	for _, s := range t.mixedTable.hashTable.slots {
		if !s.isEmpty() {
			keys = append(keys, s.key)
		}
	}
	return keys
}

// MapGoKeys returns the string keys stored in the map part of t.
//
// - Only keys stored in the map part are considered.
// - Only keys that can be converted to a Go string are included.
// - Map keys that cannot be converted to a Go string are silently ignored.
// - Keys stored in the array part are never included.
// - If the map part of t contains no string keys, MapGoKeys returns an empty slice.
// - The order of the returned keys is unspecified and should not be relied upon.
func (t *Table) MapGoKeys() []string {
	keys := make([]string, 0)
	for _, s := range t.mixedTable.hashTable.slots {
		if s.isEmpty() {
			continue
		}
		// only stringified key make sense
		str, ok := s.key.ToString()
		if !ok {
			// we silently ignore non-string keys
			continue
		}
		keys = append(keys, str)
	}
	return keys
}

// ArrayKeys returns the keys stored in the array part of t.
//
// - Only keys corresponding to non-nil values in the array part are included.
// - Keys stored in the map part are never included.
// - If the array part of t is empty, ArrayKeys returns an empty slice.
// - The order of the returned keys is unspecified and should not be relied upon.
func (t *Table) ArrayKeys() []Value {
	keys := make([]Value, 0)
	for i, s := range t.mixedTable.array.values {
		if s.IsNil() {
			continue
		}
		keys = append(keys, IntValue(int64(i)))
	}
	return keys
}

// ArrayGoKeys returns the integer keys stored in the array part of t.
//
// - Only keys corresponding to non-nil values in the array part are included.
// - Keys stored in the map part are never included.
// - If the array part of t is empty, ArrayGoKeys returns an empty slice.
// - The order of the returned keys is unspecified and should not be relied upon.
func (t *Table) ArrayGoKeys() []int64 {
	keys := make([]int64, 0)
	for i, s := range t.mixedTable.array.values {
		if s.IsNil() {
			continue
		}
		keys = append(keys, int64(i))
	}
	return keys
}
