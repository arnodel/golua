package luagc

import (
	"reflect"
	"testing"
)

func TestClonePoolNoGC(t *testing.T) {
	// This checks that RefPool behaves at least like SafePool (no GC)
	p := NewClonePool()

	n1 := newIntPtr(1)
	n2 := newIntPtr(2)
	n3 := newIntPtr(3)

	p.Mark(n1, Finalize)
	p.Mark(n2, Finalize|Release)
	p.Mark(n3, Release)
	p.Mark(n1, Finalize)

	if n := len(p.ExtractPendingFinalize()); n != 0 {
		t.Fatalf("Expected no pending finalize, got %d", n)
	}
	if n := len(p.ExtractPendingRelease()); n != 0 {
		t.Fatalf("Expected no pending release, got %d", n)
	}
	mf := p.ExtractAllMarkedFinalize()
	if !reflect.DeepEqual(mf, []Value{n1, n2}) {
		t.Fatalf("Incorrect marked finalize: %+v", mf)
	}
	if n := len(p.ExtractAllMarkedFinalize()); n != 0 {
		t.Fatalf("Expected no marked finalize, got %d", n)
	}
	mr := p.ExtractAllMarkedRelease()
	if !reflect.DeepEqual(mr, []Value{n3, n2}) {
		t.Fatalf("Incorrect marked release: %+v", mf)
	}
	if n := len(p.ExtractAllMarkedRelease()); n != 0 {
		t.Fatalf("Expected no marked release, got %d", n)
	}

	if p.Get(n1) != nil {
		t.Fatalf("It should not be possible to get weak refs")
	}
}

func TestClonePoolGC(t *testing.T) {
	// This tests that GC can remove unreachable refs, but doesn't exercise the
	// fact that GC normally happens in a separate goroutine.  As all operations
	// on the pool are protected by a mutex, this should still be a valuable
	// test.
	c := installTestCollector()
	p := NewClonePool()

	n1 := newIntPtr(1)
	n2 := newIntPtr(2)
	n3 := newIntPtr(3)
	n4 := newIntPtr(4)

	p.Mark(n1, Finalize)
	p.Mark(n2, Finalize|Release)
	p.Mark(n3, Release)
	p.Mark(n1, Finalize)
	p.Mark(n4, Finalize|Release)
	p.Mark(n4, 0)

	c.GC(n2, n1, n3, n4)

	mf := p.ExtractPendingFinalize()
	if !reflect.DeepEqual(mf, []Value{n1, n2}) {
		t.Fatalf("GC1: Incorrect pending finalize: %+v", mf)
	}
	mr := p.ExtractPendingRelease()
	if !reflect.DeepEqual(mr, []Value{n3}) {
		t.Fatalf("GC1: Incorrect pending release: %+v", mr)
	}

	c.GC(mf...)
	c.GC(mr...)

	mf = p.ExtractPendingFinalize()
	if n := len(mf); n != 0 {
		t.Fatalf("GC2: Expected no pending finalize, got %d", n)
	}
	mr = p.ExtractPendingRelease()
	if !reflect.DeepEqual(mr, []Value{n2}) {
		t.Fatalf("GC2: Incorrect pending release: %+v", mf)
	}

	c.GC(mf...)
	c.GC(mr...)

	mf = p.ExtractPendingFinalize()
	if n := len(mf); n != 0 {
		t.Fatalf("GC3: Expected no pending finalize, got %d", n)
	}
	mr = p.ExtractPendingRelease()
	if n := len(mr); n != 0 {
		t.Fatalf("GC3: Expected no pending release, got %d", n)
	}

	p.Mark(n4, Release)

	mf = p.ExtractAllMarkedFinalize()
	if n := len(mf); n != 0 {
		t.Fatalf("GC3: Expected no marked finalize, got %d", n)
	}
	mr = p.ExtractAllMarkedRelease()
	if n := len(mr); n != 1 {
		t.Fatalf("GC3: Expected 1 marked release, got %d", n)
	}

	// The Go finalizer for n4 should run but find no pendingRef, so abort
	// early.  It's not easy to test for that.
	if c.FinalizerCount() != 1 {
		t.Fatalf("Expected one finalizer to remain before last GC")
	}
	c.GC(n4)
	if c.FinalizerCount() != 0 {
		t.Fatalf("Expected no finazlier to remain after last GC")
	}
	mr = p.ExtractPendingRelease()
	if n := len(mr); n != 0 {
		t.Fatalf("Last GC: Expected 0 marked release, got %d", n)
	}
}
