package weakref

import (
	"reflect"
	"testing"
)

func TestSafePool(t *testing.T) {
	p := NewSafePool()
	n1 := newIntPtr(1)
	n2 := newIntPtr(2)
	n3 := newIntPtr(3)
	p.Mark(n1, Finalize)
	p.Mark(n2, Finalize|Release)
	p.Mark(n3, Release)
	p.Mark(n1, Finalize)
	w := p.Get(n1)
	if w.Value() == nil {
		t.Fatalf("Expected weakref to be non-nil")
	}
	if n := len(p.ExtractPendingFinalize()); n != 0 {
		t.Fatalf("Expected no pending finalize, got %d", n)
	}
	if n := len(p.ExtractPendingRelease()); n != 0 {
		t.Fatalf("Expected no pending release, got %d", n)
	}
	mf := p.ExtractAllMarkedFinalize()
	if !reflect.DeepEqual(mf, []interface{}{n1, n2}) {
		t.Fatalf("Incorrect marked finalize: %+v", mf)
	}
	if n := len(p.ExtractAllMarkedFinalize()); n != 0 {
		t.Fatalf("Expected no marked finalize, got %d", n)
	}
	mr := p.ExtractAllMarkedRelease()
	if !reflect.DeepEqual(mr, []interface{}{n3, n2}) {
		t.Fatalf("Incorrect marked release: %+v", mf)
	}
	if n := len(p.ExtractAllMarkedRelease()); n != 0 {
		t.Fatalf("Expected no marked release, got %d", n)
	}
}
