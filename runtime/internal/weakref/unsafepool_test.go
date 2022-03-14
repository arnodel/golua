package weakref

import (
	"sync"
	"testing"
)

// Simulate the Go runtime object management (GC, SetFinalizer).
type testCollector struct {
	mx      sync.Mutex
	pending map[interface{}]func(Value)
}

// SetFinalizer simulates runtime.SetFinalizer.
func (c *testCollector) SetFinalizer(obj interface{}, finalizer interface{}) {
	c.mx.Lock()
	defer c.mx.Unlock()
	if finalizer == nil {
		delete(c.pending, obj)
	} else {
		c.pending[obj] = finalizer.(func(Value))
	}
}

func (c *testCollector) getFinalizer(obj interface{}) func(Value) {
	c.mx.Lock()
	defer c.mx.Unlock()
	defer delete(c.pending, obj)
	return c.pending[obj]
}

// GC simulate runtime.GC.
func (c *testCollector) GC(objs ...Value) {
	for _, obj := range objs {
		f := c.getFinalizer(obj)
		if f != nil {
			f(obj)
		}
	}
}

func (c *testCollector) FinalizerCount() int {
	c.mx.Lock()
	defer c.mx.Unlock()
	return len(c.pending)
}

// Replace the Go runtime.SetFinalizer function with the testCollector version,
// for testing of UnsafePool.
func installTestCollector() *testCollector {
	c := &testCollector{
		pending: make(map[interface{}]func(Value)),
	}
	setFinalizer = c.SetFinalizer
	return c
}

func TestUnsafePoolFinalize(t *testing.T) {
	const nObjs = 100
	c := installTestCollector()
	p := NewUnsafePool()
	objs := make([]Value, nObjs)
	ws := make([]WeakRef, nObjs/2)

	// Create nObjs pointers to integers, each containing its index in the objs
	// slice.  Get a weakref to each one with an even index.
	for i := range objs {
		n := newIntPtr(i)
		objs[i] = n
		p.Mark(n, Finalize)
		if i%2 == 0 {
			ws[i/2] = p.Get(n)
			if ws[i/2].Value() != n {
				t.Fatalf("Expected %p, got %p", n, ws[i/2].Value())
			}
		}
	}

	// Add some extra that will be collected at the end
	for i := 0; i < nObjs; i++ {
		n := newIntPtr(i)
		p.Mark(n, Finalize)
	}

	// In the "GC thread", go collect all the objs in reverse index order.
	go func() {
		for j := nObjs - 1; j >= 0; j-- {
			c.GC(objs[j])
		}
	}()

	// We get the objs in batches, but sorted in the correct order.
	j := nObjs - 1
	for j >= 0 {
		for _, obj := range p.ExtractPendingFinalize() {
			n := getInt(obj)
			if n != j {
				t.Fatalf("Expected %d, got %d", j, n)
			}
			j -= 2
		}
	}

	// All weakrefs are still alive because the pool cleared their resurrected
	// and reinstalled a finalizer for them.
	for i, w := range ws {
		if w.Value() == nil {
			t.Fatalf("Expected ws[%d] to be non nil", i)
		}
	}

	// Collect twice because the weakrefs were revived
	c.GC(objs...)
	c.GC(objs...)

	// Now all revived objects are ready to be finalized
	j = nObjs - 2
	for _, obj := range p.ExtractPendingFinalize() {
		n := getInt(obj)
		if n != j {
			t.Fatalf("Expected %d, got %d", j, n)
		}
		j -= 2
	}
	if j >= 0 {
		t.Fatal("Missing pending finalize")
	}

	// Finalizers have run, now collecting will mark the weakrefs as dead.
	c.GC(objs...)
	for i, w := range ws {
		if w.Value() != nil {
			t.Fatalf("Expected ws[%d] to be nil", i)
		}
	}

	extraMarked := p.ExtractAllMarkedFinalize()
	if len(extraMarked) != nObjs {
		t.Fatalf("Expected %d extra, got %d", nObjs, len(extraMarked))
	}
	// Check the extra values can be finalized
	for i, obj := range extraMarked {
		n := getInt(obj)
		if n != nObjs-1-i {
			t.Fatalf("Expected extra %d, got %d", nObjs-1-i, n)
		}
	}
}

func TestUnsafePoolRelease(t *testing.T) {
	const nObjs = 100
	c := installTestCollector()
	p := NewUnsafePool()
	objs := make([]Value, nObjs)
	ws := make([]WeakRef, nObjs/2)

	// Create nObjs pointers to integers, each containing its index in the objs
	// slice.  Get a weakref to each one with an even index.
	for i := range objs {
		n := newIntPtr(i)
		objs[i] = n
		p.Mark(n, Release)
		if i%2 == 0 {
			ws[i/2] = p.Get(n)
			if ws[i/2].Value() != n {
				t.Fatalf("Expected %p, got %p", n, ws[i/2].Value())
			}
		}
	}

	// Add some extra that will be collected at the end
	for i := 0; i < nObjs; i++ {
		n := newIntPtr(i)
		p.Mark(n, Release)
	}

	// In the "GC thread", go collect all the objs in reverse index order.
	go func() {
		for j := nObjs - 1; j >= 0; j-- {
			c.GC(objs[j])
		}
	}()

	// We get the objs in batches, but sorted in the correct order.
	j := nObjs - 1
	for j >= 0 {
		for _, obj := range p.ExtractPendingRelease() {
			n := getInt(obj)
			if n != j {
				t.Fatalf("Expected %d, got %d", j, n)
			}
			j -= 2
		}
	}

	// All weakrefs are still alive because the pool cleared their resurrected
	// and reinstalled a finalizer for them.
	for i, w := range ws {
		if w.Value() == nil {
			t.Fatalf("Expected ws[%d] to be non nil", i)
		}
	}

	// Collect twice because the weakrefs were revived
	c.GC(objs...)
	c.GC(objs...)

	// Now all revived objects are ready to be finalized
	j = nObjs - 2
	for _, obj := range p.ExtractPendingRelease() {
		n := getInt(obj)
		if n != j {
			t.Fatalf("Expected %d, got %d", j, n)
		}
		j -= 2
	}
	if j >= 0 {
		t.Fatal("Missing pending release")
	}

	// Finalizers have run, now collecting will mark the weakrefs as dead.
	c.GC(objs...)
	for i, w := range ws {
		if w.Value() != nil {
			t.Fatalf("Expected ws[%d] to be nil", i)
		}
	}

	// Check the extra values can be released
	extraMarked := p.ExtractAllMarkedRelease()
	if len(extraMarked) != nObjs {
		t.Fatalf("Expected %d extra, got %d", nObjs, len(extraMarked))
	}
	for i, obj := range extraMarked {
		n := getInt(obj)
		if n != nObjs-1-i {
			t.Fatalf("Expected extra %d, got %d", nObjs-1-i, n)
		}
	}
}

func TestUnsafePoolFinalizeRelease(t *testing.T) {

	// We're testing how finalize + release interact.
	c := installTestCollector()
	p := NewUnsafePool()
	n := newIntPtr(1)
	p.Mark(n, Finalize|Release)
	w := p.Get(n)

	// After collecting n, it's ready for finalizing but not release yet.
	c.GC(n)
	pf := p.ExtractPendingFinalize()
	if len(pf) != 1 {
		t.Fatalf("Expected 1 pending finalize, got %d", len(pf))
	}
	pr := p.ExtractPendingRelease()
	if len(pr) != 0 {
		t.Fatalf("Expected 0 pending release, got %d", len(pr))
	}

	// We can revive n still
	v := w.Value()
	if v == nil {
		t.Fatal("Expected weakref to be non nil")
	}

	// This collection just cancels the reviving
	c.GC(n)
	pf = p.ExtractPendingFinalize()
	if len(pf) != 0 {
		t.Fatalf("Expected 0 pending finalize, got %d", len(pf))
	}
	pr = p.ExtractPendingRelease()
	if len(pr) != 0 {
		t.Fatalf("Expected 0 pending release, got %d", len(pr))
	}

	// After collection, the finalize is not re-run but the release is now run.
	c.GC(n)
	pf = p.ExtractPendingFinalize()
	if len(pf) != 0 {
		t.Fatalf("Expected 0 pending finalize, got %d", len(pf))
	}
	pr = p.ExtractPendingRelease()
	if len(pr) != 1 {
		t.Fatalf("Expected 1 pending release, got %d", len(pr))
	}
}

func newIntPtr(n int) *intVal {
	v := intVal(n)
	return &v
}

type intVal int

var _ Value = (*intVal)(nil)

func (v *intVal) Key() Key {
	return *v
}

func (v *intVal) Clone() Value {
	clone := new(intVal)
	*clone = *v
	return clone
}

func getInt(r Value) int {
	return int(*r.(*intVal))
}
