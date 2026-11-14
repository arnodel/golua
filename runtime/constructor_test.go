package runtime

import (
	"io"
	"testing"

	"github.com/arnodel/golua/code"
)

// A table constructor makes its table with capacity for the values and fields
// it lists, so the table does not grow while the constructor fills it.
func TestConstructorCapacity(t *testing.T) {
	r := New(io.Discard)
	for _, c := range []struct {
		src             string
		nvalues, nslots int
	}{
		{"return {}", 0, 0},
		{"return {1, 2, 3}", 3, 0},
		{"return {x = 1, y = 2, z = 3}", 0, 4},
		{"return {x = 1, y = 2, z = 3, w = 4}", 0, 4},
		{"return {10, 20, x = 1}", 2, 1},
	} {
		tbl := runChunk(t, r, c.src).AsTable()
		if n := int(tbl.array.size()); n != c.nvalues {
			t.Errorf("%s: array part of %d values, want %d", c.src, n, c.nvalues)
		}
		n := 0
		if tbl.hashTable != nil {
			n = len(tbl.hashTable.slots)
		}
		if n != c.nslots {
			t.Errorf("%s: hash part of %d slots, want %d", c.src, n, c.nslots)
		}
	}
}

// The capacity is in the opcode that makes the table.
func TestConstructorOpcode(t *testing.T) {
	r := New(io.Discard)
	for _, c := range []struct {
		src              string
		nvalues, nfields int
	}{
		{"local t = {}", 0, 0},
		{"local t = {1, 2, 3}", 3, 0},
		{"local t = {x = 1, y = 2}", 0, 2},
		{"local t = {1, x = 2, 3, [4] = 5}", 2, 2},
		// A last value that may be several is not counted; one that is not
		// last gives one value.
		{"local t = {print()}", 0, 0},
		{"local t = {1, print()}", 1, 0},
		{"local t = {print(), 2}", 2, 0},
		{"local t = {...}", 0, 0},
		// More than the opcode can say is not asked for.
		{"local t = {1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20}", code.MaxTableCapacity, 0},
	} {
		found := false
		for _, op := range compileMain(t, r, c.src) {
			if !op.HasType1() && op.TypePfx() == code.Type4Pfx && !op.HasType4a() && op.GetUnOpK() == code.OpTable {
				nvalues, nfields := op.GetL().TableCapacity()
				if nvalues != c.nvalues || nfields != c.nfields {
					t.Errorf("%s: capacity for %d values and %d fields, want %d and %d", c.src, nvalues, nfields, c.nvalues, c.nfields)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: no opcode makes a table", c.src)
		}
	}
}
