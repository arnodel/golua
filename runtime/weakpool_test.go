//go:build go1.24

package runtime

import (
	"runtime"
	"testing"

	"github.com/arnodel/golua/runtime/internal/luagc"
)

// simulateGC simulates the Go GC collecting the given entries by directly
// invoking the cleanup function. This avoids relying on non-deterministic
// real GC behavior for unit tests.
func simulateGC(entries ...*weakEntry) {
	for _, entry := range entries {
		weakEntryCleanup(entry)
	}
}

// getEntry returns the weakEntry for a value, for use in simulated GC.
func getEntry(p *WeakPool, v luagc.Value) *weakEntry {
	return p.entries[v.Key()]
}

func TestWeakPoolNoGC(t *testing.T) {
	p := NewWeakPool()

	t1 := NewTable()
	t2 := NewTable()
	u1 := NewUserData(1, nil)

	p.Mark(t1, luagc.Finalize)
	p.Mark(t2, luagc.Finalize|luagc.Release)
	p.Mark(u1, luagc.Release)
	// Re-mark t1 (should update, not duplicate)
	p.Mark(t1, luagc.Finalize)

	// No GC has happened, so nothing should be pending.
	if n := len(p.ExtractPendingFinalize()); n != 0 {
		t.Fatalf("Expected no pending finalize, got %d", n)
	}
	if n := len(p.ExtractPendingRelease()); n != 0 {
		t.Fatalf("Expected no pending release, got %d", n)
	}

	// ExtractAllMarkedFinalize should return t1 and t2 (both marked for finalize).
	mf := p.ExtractAllMarkedFinalize()
	if len(mf) != 2 {
		t.Fatalf("Expected 2 marked finalize, got %d", len(mf))
	}
	// Second call should return nothing.
	if n := len(p.ExtractAllMarkedFinalize()); n != 0 {
		t.Fatalf("Expected no marked finalize on second call, got %d", n)
	}

	// ExtractAllMarkedRelease should return t2 and u1 (both marked for release).
	mr := p.ExtractAllMarkedRelease()
	if len(mr) != 2 {
		t.Fatalf("Expected 2 marked release, got %d", len(mr))
	}
	// Second call should return nothing.
	if n := len(p.ExtractAllMarkedRelease()); n != 0 {
		t.Fatalf("Expected no marked release on second call, got %d", n)
	}
}

func TestWeakPoolFinalizeOnly(t *testing.T) {
	p := NewWeakPool()

	t1 := NewTable()
	t2 := NewTable()

	p.Mark(t1, luagc.Finalize)
	p.Mark(t2, luagc.Finalize)

	e1 := getEntry(p, t1)
	e2 := getEntry(p, t2)

	// Simulate GC collecting both values.
	simulateGC(e2, e1) // Reverse order to test sorting

	pf := p.ExtractPendingFinalize()
	if len(pf) != 2 {
		t.Fatalf("Expected 2 pending finalize, got %d", len(pf))
	}

	// Should be sorted by reverse mark order (t2 before t1).
	if pf[0].Key() != t2.Key() {
		t.Fatal("Expected t2 first in pending finalize (reverse mark order)")
	}
	if pf[1].Key() != t1.Key() {
		t.Fatal("Expected t1 second in pending finalize (reverse mark order)")
	}

	// No pending release (not marked for it).
	pr := p.ExtractPendingRelease()
	if len(pr) != 0 {
		t.Fatalf("Expected 0 pending release, got %d", len(pr))
	}
}

func TestWeakPoolReleaseOnly(t *testing.T) {
	p := NewWeakPool()

	u1 := NewUserData(1, nil)
	u2 := NewUserData(2, nil)

	p.Mark(u1, luagc.Release)
	p.Mark(u2, luagc.Release)

	e1 := getEntry(p, u1)
	e2 := getEntry(p, u2)

	simulateGC(e1, e2)

	// Nothing pending for finalize (not marked).
	pf := p.ExtractPendingFinalize()
	if len(pf) != 0 {
		t.Fatalf("Expected 0 pending finalize, got %d", len(pf))
	}

	// Both should be immediately available for release (weFinalized is
	// already set because they were not marked for finalization).
	pr := p.ExtractPendingRelease()
	if len(pr) != 2 {
		t.Fatalf("Expected 2 pending release, got %d", len(pr))
	}
}

func TestWeakPoolFinalizeBeforeRelease(t *testing.T) {
	// Values marked for both Finalize and Release should only be released
	// after finalization has been extracted.
	p := NewWeakPool()

	tbl := NewTable()
	p.Mark(tbl, luagc.Finalize|luagc.Release)
	entry := getEntry(p, tbl)

	simulateGC(entry)

	// Release should not yet be available (finalization hasn't been extracted).
	pr := p.ExtractPendingRelease()
	if len(pr) != 0 {
		t.Fatalf("Expected 0 pending release before finalize, got %d", len(pr))
	}

	// Extract finalization.
	pf := p.ExtractPendingFinalize()
	if len(pf) != 1 {
		t.Fatalf("Expected 1 pending finalize, got %d", len(pf))
	}

	// Now release should be available.
	pr = p.ExtractPendingRelease()
	if len(pr) != 1 {
		t.Fatalf("Expected 1 pending release after finalize, got %d", len(pr))
	}

	// Both lists should now be empty.
	if n := len(p.ExtractPendingFinalize()); n != 0 {
		t.Fatalf("Expected no more pending finalize, got %d", n)
	}
	if n := len(p.ExtractPendingRelease()); n != 0 {
		t.Fatalf("Expected no more pending release, got %d", n)
	}
}

func TestWeakPoolRemarkChangesFlags(t *testing.T) {
	p := NewWeakPool()

	tbl := NewTable()

	// First mark for finalize only.
	p.Mark(tbl, luagc.Finalize)
	entry := getEntry(p, tbl)
	if !entry.hasFlag(weReleased) {
		t.Fatal("Expected weReleased to be set (not marked for release)")
	}
	if entry.hasFlag(weFinalized) {
		t.Fatal("Expected weFinalized to be clear (marked for finalize)")
	}

	// Re-mark for release only.
	p.Mark(tbl, luagc.Release)
	if entry.hasFlag(weReleased) {
		t.Fatal("Expected weReleased to be clear (marked for release)")
	}
	if !entry.hasFlag(weFinalized) {
		t.Fatal("Expected weFinalized to be set (not marked for finalize)")
	}
}

func TestWeakPoolExtractAllMarkedFinalize(t *testing.T) {
	p := NewWeakPool()

	t1 := NewTable()
	t2 := NewTable()
	u1 := NewUserData(1, nil)

	p.Mark(t1, luagc.Finalize)
	p.Mark(t2, luagc.Finalize|luagc.Release)
	p.Mark(u1, luagc.Release) // Not marked for finalize

	// Simulate GC for t1 only (it goes to pendingFinalize).
	e1 := getEntry(p, t1)
	simulateGC(e1)

	// ExtractAllMarkedFinalize should return t2 (still in entries, not yet
	// finalized). t1 is already in pendingFinalize but that list is cleared.
	// u1 is already flagged as finalized (Release-only).
	mf := p.ExtractAllMarkedFinalize()
	if len(mf) != 1 {
		t.Fatalf("Expected 1 marked finalize, got %d", len(mf))
	}
	if mf[0].Key() != t2.Key() {
		t.Fatal("Expected t2 in marked finalize")
	}
}

func TestWeakPoolExtractAllMarkedRelease(t *testing.T) {
	p := NewWeakPool()

	t1 := NewTable()
	u1 := NewUserData(1, nil)

	p.Mark(t1, luagc.Finalize|luagc.Release)
	p.Mark(u1, luagc.Release)

	// Simulate GC for u1 (goes to pendingRelease since it's release-only).
	eu1 := getEntry(p, u1)
	simulateGC(eu1)

	// ExtractAllMarkedRelease should return both t1 (from entries) and u1
	// (from pendingRelease).
	mr := p.ExtractAllMarkedRelease()
	if len(mr) != 2 {
		t.Fatalf("Expected 2 marked release, got %d", len(mr))
	}
}

func TestWeakPoolMultipleValues(t *testing.T) {
	const n = 20
	p := NewWeakPool()

	tables := make([]*Table, n)
	entries := make([]*weakEntry, n)

	for i := range tables {
		tables[i] = NewTable()
		p.Mark(tables[i], luagc.Finalize)
		entries[i] = getEntry(p, tables[i])
	}

	// Simulate GC for all in reverse order.
	for i := n - 1; i >= 0; i-- {
		simulateGC(entries[i])
	}

	pf := p.ExtractPendingFinalize()
	if len(pf) != n {
		t.Fatalf("Expected %d pending finalize, got %d", n, len(pf))
	}

	// Should be sorted by reverse mark order (last marked first).
	for i, v := range pf {
		expected := n - 1 - i
		if v.Key() != tables[expected].Key() {
			t.Fatalf("At position %d: expected table %d", i, expected)
		}
	}
}

func TestWeakPoolGet(t *testing.T) {
	p := NewWeakPool()

	tbl := NewTable()
	p.Mark(tbl, luagc.Finalize)

	ref := p.Get(tbl)
	if ref == nil {
		t.Fatal("Expected non-nil WeakRef")
	}

	// While the value is alive, Value() should return it.
	v := ref.Value()
	if v == nil {
		t.Fatal("Expected WeakRef.Value() to be non-nil for live value")
	}
}

func TestWeakPoolRealGC(t *testing.T) {
	// This test uses real GC to verify end-to-end behavior.
	// It's less deterministic but validates the AddCleanup integration.
	p := NewWeakPool()

	// Create values in a function scope so they become unreachable.
	func() {
		tbl := NewTable()
		p.Mark(tbl, luagc.Finalize)
	}()

	// Force GC and give cleanups time to run.
	for i := 0; i < 5; i++ {
		runtime.GC()
	}

	pf := p.ExtractPendingFinalize()
	if len(pf) != 1 {
		t.Fatalf("Expected 1 pending finalize after real GC, got %d", len(pf))
	}
}

func TestWeakPoolRealGCFinalizeAndRelease(t *testing.T) {
	p := NewWeakPool()

	func() {
		ud := NewUserData(42, nil)
		p.Mark(ud, luagc.Finalize|luagc.Release)
	}()

	for i := 0; i < 5; i++ {
		runtime.GC()
	}

	// Should have pending finalize.
	pf := p.ExtractPendingFinalize()
	if len(pf) != 1 {
		t.Fatalf("Expected 1 pending finalize, got %d", len(pf))
	}

	// After extracting finalize, release should be available.
	pr := p.ExtractPendingRelease()
	if len(pr) != 1 {
		t.Fatalf("Expected 1 pending release, got %d", len(pr))
	}
}
