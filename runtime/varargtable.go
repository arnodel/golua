package runtime

import "errors"

// varargN is the key used to store the element count in a vararg table.
var varargN = StringValue("n")

var errBadVarargN = errors.New("vararg table has no proper 'n'")

// maxVarargN is the maximum allowed value for a vararg table's n field.
// Matches reference Lua 5.5's limit (values >= 2^30 are rejected).
const maxVarargN = 1<<30 - 1

// newVarargTable creates a vararg table (Lua 5.5) from a slice.
// The table's array part references the slice directly, and t.n is set to the length.
func newVarargTable(values []Value) *Table {
	t := NewTableFromSlice(values)
	t.Set(varargN, IntValue(int64(len(values))))
	return t
}

// expandVarargs converts a vararg value (either a raw []Value slice or a
// vararg *Table) into a []Value slice. For a raw slice it returns it directly.
// For a table, it validates t.n and returns the elements t[1]..t[n].
// When the table's underlying array is large enough, it returns a sub-slice
// of that array to avoid allocation.
func expandVarargs(v Value) ([]Value, error) {
	tbl, ok := v.TryTable()
	if !ok {
		return v.AsArray(), nil
	}
	n, err := varargTableN(tbl)
	if err != nil {
		return nil, err
	}
	// Fast path: if the table's array part covers all n elements,
	// return a sub-slice directly (no allocation).
	if tbl.array != nil && int64(len(tbl.array.values)) >= n {
		return tbl.array.values[:n], nil
	}
	// Slow path: some elements are in the hash part, build a new slice.
	vals := make([]Value, n)
	for i := int64(1); i <= n; i++ {
		vals[i-1] = tbl.Get(IntValue(i))
	}
	return vals, nil
}

// varargTableN reads and validates the n field from a vararg table.
// Returns the count or an error if n is not a non-negative integer.
func varargTableN(tbl *Table) (int64, error) {
	n, ok := tbl.Get(varargN).TryInt()
	if !ok || n < 0 || n > maxVarargN {
		return 0, errBadVarargN
	}
	return n, nil
}
