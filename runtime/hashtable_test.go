package runtime

import "testing"

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
	if asz != 16 {
		t.Errorf("expected array size to be 16, got %d", asz)
	}
	alen := mt.array.getLen()
	if alen != 16 {
		t.Errorf("Expected array length to be 16, got %d", alen)
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
