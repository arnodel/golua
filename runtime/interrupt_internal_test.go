//go:build !noquotas
// +build !noquotas

package runtime

import (
	"bytes"
	"testing"
)

// TestInterruptIsUninstalledWhenItsContextEnds checks that an Interrupt does
// not keep a reference to, or raise the flag of, a runtime after the context
// it was installed in has finished: otherwise one triggered interrupt of a
// finished context would put every later CPU requirement on the slow path.
func TestInterruptIsUninstalledWhenItsContextEnds(t *testing.T) {
	var out bytes.Buffer
	r := New(&out)
	defer r.Close(nil)
	intr := NewInterrupt()
	if _, err := r.MainThread().CallContext(RuntimeContextDef{Interrupt: intr}, func() error {
		intr.mu.Lock()
		n := len(intr.installed)
		intr.mu.Unlock()
		if n != 1 {
			t.Errorf("installed in %d runtimes while the context runs, want 1", n)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	intr.mu.Lock()
	n := len(intr.installed)
	intr.mu.Unlock()
	if n != 0 {
		t.Fatalf("still installed in %d runtimes after the context ended", n)
	}
	intr.Trigger("late")
	if r.pending.Load() {
		t.Fatal("triggering an interrupt of a finished context raised the runtime's pending flag")
	}
}

// TestPendingClearedWhenNoInstalledInterruptTriggered checks the flag of a
// runtime goes back down once its context has looked at its interrupts and
// found none triggered.
func TestPendingClearedWhenNoInstalledInterruptTriggered(t *testing.T) {
	var out bytes.Buffer
	r := New(&out)
	defer r.Close(nil)
	r.pending.Store(true)
	// Not in a limited context: the slow path runs because the flag is up.
	r.RequireCPU(1)
	if r.pending.Load() {
		t.Fatal("pending flag still up after RequireCPU found no triggered interrupt")
	}
}
