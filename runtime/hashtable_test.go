package runtime

import (
	"testing"
)

func v(i interface{}) Value {
	return AsValue(i)
}

func TestHashTable(t *testing.T) {
	var ht *hashTable
	if !ht.full() {
		t.Error("Expected nil hashTable to be full")
	}
	ht = ht.grow()
	if ht.full() {
		t.Error("Expected growed hashTable not to be full")
	}
	ht.set(v("hello"), v(123))
	if ht.find(v("hello")) != v(123) {
		t.Error("expected hello => 123")
	}
	if !ht.full() {
		t.Error("should be full")
	}
}

// TestMixedTableWithNonPowerOf2Capacity tests that tables created with
// non-power-of-2 capacity hints work correctly. The capacity should be
// rounded up to a power of 2 to ensure consistent behavior with the
// grow() logic which uses power-of-2 bucket classification.
func TestMixedTableWithNonPowerOf2Capacity(t *testing.T) {
	// Create table with non-power-of-2 capacity (e.g., 5)
	mt := newMixedTableWithCapacity(5, 0)

	// Insert some values
	for i := 1; i <= 5; i++ {
		mt.insert(v(i), v(i*10))
	}

	// Verify values
	for i := 1; i <= 5; i++ {
		if mt.get(v(i)) != v(i*10) {
			t.Errorf("Expected mt[%d] = %d, got %v", i, i*10, mt.get(v(i)))
		}
	}

	// Verify the array length was updated correctly
	if mt.array.getLen() != 5 {
		t.Errorf("Expected array len 5, got %d", mt.array.getLen())
	}

	// Now modify and iterate - this would panic before the fix
	mt.insert(v(6), v(60))
	mt.insert(v(7), v(70))

	if mt.array.getLen() != 7 {
		t.Errorf("Expected array len 7, got %d", mt.array.getLen())
	}

	// Iterate with next() to verify no panic
	// Note: next() returns (key, val, ok) where ok indicates whether the iteration
	// can continue. Iteration is done when key is NilValue.
	count := 0
	key := NilValue
	for {
		var val Value
		key, val, _ = mt.next(key)
		if key.IsNil() {
			break
		}
		if val.IsNil() {
			t.Errorf("Expected non-nil value during iteration, key=%v", key)
		}
		count++
		if count > 100 {
			t.Fatal("Iteration seems infinite")
		}
	}

	if count != 7 {
		t.Errorf("Expected 7 items, got %d", count)
	}
}

// TestMixedTableGrowWithCapacityHint tests that tables created with capacity
// hints can grow correctly when more elements are inserted than the hint.
func TestMixedTableGrowWithCapacityHint(t *testing.T) {
	// Create table with capacity hint 5 (rounds to 8)
	mt := newMixedTableWithCapacity(5, 0)

	// Fill up the array beyond the original hint to force grow
	for i := 1; i <= 10; i++ {
		mt.insert(v(i), v(i*10))
	}

	// Verify all values are correct
	for i := 1; i <= 10; i++ {
		got := mt.get(v(i))
		if got != v(i*10) {
			t.Errorf("After grow: mt[%d] = %v, want %d", i, got, i*10)
		}
	}

	// The array should have grown to accommodate all elements
	if mt.array.getLen() != 10 {
		t.Errorf("Expected array len 10, got %d", mt.array.getLen())
	}

	// Iterate to verify no issues after grow
	count := 0
	key := NilValue
	for {
		key, _, _ = mt.next(key)
		if key.IsNil() {
			break
		}
		count++
		if count > 100 {
			t.Fatal("Iteration seems infinite after grow")
		}
	}

	if count != 10 {
		t.Errorf("Expected 10 items after grow, got %d", count)
	}
}

// TestMixedTableCapacityRoundedToPowerOf2 verifies that capacity hints
// are rounded up to power of 2.
func TestMixedTableCapacityRoundedToPowerOf2(t *testing.T) {
	testCases := []struct {
		hint     int
		expected int
	}{
		{1, 1},
		{2, 2},
		{3, 4},
		{5, 8},
		{7, 8},
		{9, 16},
		{100, 128},
		{1000, 1024},
	}

	for _, tc := range testCases {
		mt := newMixedTableWithCapacity(tc.hint, 0)
		if int(mt.array.size()) != tc.expected {
			t.Errorf("Capacity hint %d: expected array size %d, got %d",
				tc.hint, tc.expected, mt.array.size())
		}
	}
}

func TestMixedTable1To20(t *testing.T) {
	var mt = new(mixedTable)
	const n = 20
	for i := 1; i <= n; i++ {
		mt.insert(v(i), v(i+2))
	}
	for i := 1; i <= n; i++ {
		mti := mt.get(v(i))
		if mti != v(i+2) {
			t.Errorf("Expected mt[%d] to be %d, got %v", i, i+2, mti)
		}
	}
	asz := mt.array.size()
	if asz != 32 {
		t.Errorf("expected array size to be 32, got %d", asz)
	}
	alen := mt.array.getLen()
	if alen != 20 {
		t.Errorf("Expected array length to be 20, got %d", alen)
	}
	mtlen := mt.len()
	if mtlen != 20 {
		t.Errorf("Expected mt to have length 20, got %d", mtlen)
	}
}

func TestMixedTableRemove(t *testing.T) {
	mt := new(mixedTable)
	for i := 1; i <= 5; i++ {
		mt.insert(v(i), v(i))
	}
	mt.remove(v(5))
	mt.remove(v(4))
	alen := mt.array.getLen()
	if alen != 3 {
		t.Errorf("Expected array length to be 3, got %d", alen)
	}
	mtlen := mt.len()
	if mtlen != 3 {
		t.Errorf("Expected mt length to be 3, got %d", mtlen)
	}
}

func Test_calculateArraySize(t *testing.T) {
	type args struct {
		idxCountByLen *[uintptrLen]uintptr
	}
	var counts = func(cs ...uintptr) *[uintptrLen]uintptr {
		counts := new([uintptrLen]uintptr)
		for i, c := range cs {
			counts[i] = c
		}
		return counts
	}

	tests := []struct {
		name string
		args args
		want uintptr
	}{
		{
			name: "empty",
			args: args{
				idxCountByLen: counts(),
			},
			want: 0,
		},
		{
			name: "0",
			args: args{
				idxCountByLen: counts(1),
			},
			want: 1,
		},
		{
			name: "0 1",
			args: args{
				idxCountByLen: counts(1, 1),
			},
			want: 2,
		},
		{
			name: "1",
			args: args{
				idxCountByLen: counts(0, 1),
			},
			want: 2,
		},
		{
			name: "0 1 2",
			args: args{
				idxCountByLen: counts(1, 1, 1),
			},
			want: 4,
		},
		{
			name: "0 1 2 3",
			args: args{
				idxCountByLen: counts(1, 1, 2),
			},
			want: 4,
		},
		{
			name: "1 2 3 4",
			args: args{
				idxCountByLen: counts(0, 1, 2, 1),
			},
			want: 8,
		},

		{
			name: "0..15",
			args: args{
				idxCountByLen: counts(1, 1, 2, 4, 8),
			},
			want: 16,
		},
		{
			name: "0..16",
			args: args{
				idxCountByLen: counts(1, 1, 2, 4, 8, 1),
			},
			want: 32,
		},
		{
			name: "3..18",
			args: args{
				idxCountByLen: counts(0, 0, 1, 4, 8, 3),
			},
			want: 32,
		},
		{
			name: "10..49",
			args: args{
				idxCountByLen: counts(0, 0, 0, 0, 6, 16, 18),
			},
			want: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateArraySize(tt.args.idxCountByLen); got != tt.want {
				t.Errorf("calculateArraySize() = %v, want %v", got, tt.want)
			}
		})
	}
}
