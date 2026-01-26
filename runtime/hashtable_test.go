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

// TestMixedTableSize15WithLargeKeys tests the specific bug scenario:
// table.create(15) with keys all > 15 (so they go to hashtable).
// When the hashtable fills and grow() is called, it would decide to grow
// the array from 15 to 16, but no items could be moved (all keys > 16).
// The fix ensures the hashtable grows in this case.
func TestMixedTableSize15WithLargeKeys(t *testing.T) {
	mt := newMixedTableWithCapacity(15, 0)

	// Insert many keys that are all > 15, so they go to the hashtable
	for i := 50; i <= 100; i++ {
		mt.insert(v(i), v(i*10))
	}

	// Verify all values are present
	for i := 50; i <= 100; i++ {
		got := mt.get(v(i))
		if got != v(i*10) {
			t.Errorf("mt[%d] = %v, want %d", i, got, i*10)
		}
	}

	// Iterate to verify no issues
	count := 0
	key := NilValue
	for {
		key, _, _ = mt.next(key)
		if key.IsNil() {
			break
		}
		count++
		if count > 200 {
			t.Fatal("Iteration seems infinite")
		}
	}

	if count != 51 {
		t.Errorf("Expected 51 items, got %d", count)
	}
}

// TestMixedTableArbitraryCapacity verifies that arbitrary capacity hints
// are supported (not rounded to power of 2). The grow() logic handles
// non-power-of-2 array sizes correctly.
func TestMixedTableArbitraryCapacity(t *testing.T) {
	testCases := []int{1, 2, 3, 5, 7, 9, 15, 100, 1000}

	for _, hint := range testCases {
		mt := newMixedTableWithCapacity(hint, 0)
		if int(mt.array.size()) != hint {
			t.Errorf("Capacity hint %d: expected array size %d, got %d",
				hint, hint, mt.array.size())
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

// TestNonPowerOf2ArrayGrowth verifies that tables with non-power-of-2 array
// sizes don't unnecessarily grow when keys outside the array are added.
func TestNonPowerOf2ArrayGrowth(t *testing.T) {
	// Create table with array hint of 15 (not a power of 2)
	mt := newMixedTableWithCapacity(15, 0)

	// Verify initial array size is 15
	if mt.array.size() != 15 {
		t.Errorf("Expected initial array size 15, got %d", mt.array.size())
	}

	// Fill array with values 1-15
	for i := 1; i <= 15; i++ {
		mt.insert(v(i), v(i))
	}

	// Add a few keys that are way outside the array (sparse)
	// These should go to hashtable, not trigger array growth
	mt.insert(v(100), v(100))
	mt.insert(v(200), v(200))

	// Array should still be size 15 (no unnecessary growth)
	if mt.array.size() != 15 {
		t.Errorf("Array grew unnecessarily to %d, expected 15", mt.array.size())
	}

	// All values should be retrievable
	for i := 1; i <= 15; i++ {
		if mt.get(v(i)) != v(i) {
			t.Errorf("Lost value at index %d", i)
		}
	}
	if mt.get(v(100)) != v(100) {
		t.Error("Lost value at index 100")
	}
	if mt.get(v(200)) != v(200) {
		t.Error("Lost value at index 200")
	}
}

// TestNonPowerOf2ArrayDenseGrowth verifies that when enough dense keys are
// added to justify growth, the array grows appropriately.
func TestNonPowerOf2ArrayDenseGrowth(t *testing.T) {
	// Create table with array hint of 15
	mt := newMixedTableWithCapacity(15, 0)

	// Fill array
	for i := 1; i <= 15; i++ {
		mt.insert(v(i), v(i))
	}

	// Add dense keys 16-30 (enough to justify doubling)
	for i := 16; i <= 30; i++ {
		mt.insert(v(i), v(i))
	}

	// Array should have grown to accommodate dense keys
	// With relative bucketing, keys 16-30 are in bucket 1 for arrSize=15
	// After growth, array should be at least 30
	if mt.array.size() < 30 {
		t.Errorf("Array should have grown to at least 30, got %d", mt.array.size())
	}

	// All values should be retrievable
	for i := 1; i <= 30; i++ {
		if mt.get(v(i)) != v(i) {
			t.Errorf("Lost value at index %d", i)
		}
	}
}

