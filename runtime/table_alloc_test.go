package runtime

import (
	"runtime"
	"testing"
)

// TestTableSetChargesWhatItAllocates checks that the memory Table.Set
// reports tracks what Go actually allocates as a table grows, for both the
// array part and the hash part.
func TestTableSetChargesWhatItAllocates(t *testing.T) {
	cases := []struct {
		name string
		key  func(i int) Value
	}{
		{"array", func(i int) Value { return IntValue(int64(i)) }},
		{"hash", func(i int) Value { return IntValue(int64(2*i + 1)) }},
	}
	for _, c := range cases {
		for _, n := range []int{1000, 100000} {
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			tbl := NewTable()
			charged := TableHeaderSize
			for i := 1; i <= n; i++ {
				charged += tbl.Set(c.key(i), BoolValue(true))
			}
			runtime.ReadMemStats(&after)
			runtime.KeepAlive(tbl)
			actual := after.TotalAlloc - before.TotalAlloc
			ratio := float64(charged) / float64(actual)
			t.Logf("%s n=%d: charged %d, allocated %d (%.2f)", c.name, n, charged, actual, ratio)
			if ratio < 0.75 || ratio > 1.25 {
				t.Errorf("%s n=%d: charged %d bytes for %d allocated (ratio %.2f, want 0.75..1.25)", c.name, n, charged, actual, ratio)
			}
		}
	}
}

// TestTableSetChargesNothingWithoutGrowing checks that storing into a table
// that has room, or overwriting a key, is not charged.
func TestTableSetChargesNothingWithoutGrowing(t *testing.T) {
	tbl := NewTable()
	for i := int64(1); i <= 64; i++ {
		tbl.Set(IntValue(i), IntValue(i))
	}
	for i := int64(1); i <= 64; i++ {
		if n := tbl.Set(IntValue(i), IntValue(-i)); n != 0 {
			t.Fatalf("overwriting key %d charged %d bytes", i, n)
		}
	}
	if n := tbl.Set(StringValue("k"), NilValue); n != 0 {
		t.Fatalf("removing a key charged %d bytes", n)
	}
}
