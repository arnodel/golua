package runtime_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arnodel/golua/lib"
	rt "github.com/arnodel/golua/runtime"
)

// requireQuotas skips tests that need interrupts to take effect: with the
// noquotas build tag they are accepted and ignored.
func requireQuotas(t *testing.T) {
	t.Helper()
	if !rt.QuotasAvailable {
		t.Skip("interrupts are ignored in the noquotas build")
	}
}

// runInterrupted runs src in a context with the interrupt installed and no
// other limit, triggers the interrupt from another goroutine after a short
// delay, and returns the context's error and how long the run took.
func runInterrupted(t *testing.T, src string) (rt.RuntimeContext, error, time.Duration) {
	ctx, err, elapsed, _ := runInterruptedOutput(t, src)
	return ctx, err, elapsed
}

func runInterruptedOutput(t *testing.T, src string) (rt.RuntimeContext, error, time.Duration, string) {
	t.Helper()
	var out bytes.Buffer
	r := rt.New(&out)
	defer lib.LoadAll(r)()
	chunk, err := r.CompileAndLoadLuaChunk("test", []byte(src), rt.TableValue(r.GlobalEnv()))
	if err != nil {
		t.Fatal(err)
	}
	intr := rt.NewInterrupt()
	timer := time.AfterFunc(50*time.Millisecond, func() { intr.Trigger("interrupted by test") })
	defer timer.Stop()
	start := time.Now()
	thread := r.MainThread()
	ctx, err := thread.CallContext(rt.RuntimeContextDef{Interrupt: intr}, func() error {
		_, err := rt.Call1(thread, rt.FunctionValue(chunk))
		return err
	})
	return ctx, err, time.Since(start), out.String()
}

func TestInterruptStopsAnInfiniteLoop(t *testing.T) {
	requireQuotas(t)
	ctx, err, elapsed := runInterrupted(t, `while true do end`)
	if err == nil || !strings.Contains(err.Error(), "interrupted by test") {
		t.Fatalf("err = %v, want the interrupt reason", err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("took %v to stop", elapsed)
	}
	if ctx.Status() != rt.StatusKilled {
		t.Fatalf("status = %v, want killed", ctx.Status())
	}
}

func TestInterruptPassesThroughPcall(t *testing.T) {
	requireQuotas(t)
	// Each pcall swallows the termination of its own context; the interrupt
	// stays set, so each enclosing context terminates in turn.
	_, err, elapsed, out := runInterruptedOutput(t, `
		local function spin() while true do end end
		pcall(function()
			pcall(function()
				pcall(spin)
				print("level 3 returned")
				spin()
			end)
			print("level 2 returned")
			spin()
		end)
		print("level 1 returned")
		spin()`)
	if err == nil || !strings.Contains(err.Error(), "interrupted by test") {
		t.Fatalf("err = %v, want the interrupt reason", err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("took %v to stop", elapsed)
	}
	if out != "" {
		t.Fatalf("code ran in an enclosing context after the interrupt: %q", out)
	}
}

func TestInterruptOnlyAffectsItsContext(t *testing.T) {
	requireQuotas(t)
	// An already-triggered interrupt installed in an inner context ends that
	// context; the outer context carries on.
	var out bytes.Buffer
	r := rt.New(&out)
	defer lib.LoadAll(r)()
	thread := r.MainThread()
	intr := rt.NewInterrupt()
	intr.Trigger("stop inner")
	inner, innerErr := thread.CallContext(rt.RuntimeContextDef{Interrupt: intr}, func() error {
		chunk, _ := r.CompileAndLoadLuaChunk("inner", []byte(`while true do end`), rt.TableValue(r.GlobalEnv()))
		_, err := rt.Call1(thread, rt.FunctionValue(chunk))
		return err
	})
	if innerErr == nil || !strings.Contains(innerErr.Error(), "stop inner") {
		t.Fatalf("inner err = %v", innerErr)
	}
	if inner.Status() != rt.StatusKilled {
		t.Fatalf("inner status = %v", inner.Status())
	}
	chunk, _ := r.CompileAndLoadLuaChunk("outer", []byte(`local n = 0 for i = 1, 1000 do n = n + i end return n`), rt.TableValue(r.GlobalEnv()))
	v, err := rt.Call1(thread, rt.FunctionValue(chunk))
	if err != nil || v.AsInt() != 500500 {
		t.Fatalf("outer = %v, %v", v, err)
	}
}

func TestInterruptTriggerKeepsFirstReason(t *testing.T) {
	intr := rt.NewInterrupt()
	if _, ok := intr.Triggered(); ok {
		t.Fatal("new interrupt is triggered")
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			intr.Trigger("first")
		}()
	}
	wg.Wait()
	intr.Trigger("second")
	if reason, ok := intr.Triggered(); !ok || reason != "first" {
		t.Fatalf("Triggered() = %q, %v", reason, ok)
	}
}

func TestInterruptTriggerDefaultReason(t *testing.T) {
	intr := rt.NewInterrupt()
	intr.Trigger("")
	if reason, ok := intr.Triggered(); !ok || reason != "interrupted" {
		t.Fatalf("Triggered() = %q, %v", reason, ok)
	}
}

func TestInterruptNestedContextsBothApply(t *testing.T) {
	// A context pushed inside one that has an interrupt gets an interrupt of
	// its own.  Triggering the outer one must stop the inner context, and the
	// outer one with it: no code runs in the outer context after the inner one
	// ends.
	requireQuotas(t)
	var out bytes.Buffer
	r := rt.New(&out)
	defer lib.LoadAll(r)()
	spin, err := r.CompileAndLoadLuaChunk("spin", []byte(`while true do end`), rt.TableValue(r.GlobalEnv()))
	if err != nil {
		t.Fatal(err)
	}
	thread := r.MainThread()
	outer, inner := rt.NewInterrupt(), rt.NewInterrupt()
	timer := time.AfterFunc(50*time.Millisecond, func() { outer.Trigger("outer interrupt") })
	defer timer.Stop()
	start := time.Now()
	reachedAfterInner := false
	ctx, outerErr := thread.CallContext(rt.RuntimeContextDef{Interrupt: outer}, func() error {
		_, _ = thread.CallContext(rt.RuntimeContextDef{Interrupt: inner}, func() error {
			_, err := rt.Call1(thread, rt.FunctionValue(spin))
			return err
		})
		reachedAfterInner = true
		_, err := rt.Call1(thread, rt.FunctionValue(spin))
		return err
	})
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("took %v to stop", elapsed)
	}
	if outerErr == nil || !strings.Contains(outerErr.Error(), "outer interrupt") {
		t.Fatalf("outer err = %v, want the outer interrupt's reason", outerErr)
	}
	if ctx.Status() != rt.StatusKilled {
		t.Fatalf("outer status = %v, want killed", ctx.Status())
	}
	if reachedAfterInner {
		t.Fatal("code ran in the outer context after the inner one was stopped by the outer interrupt")
	}
	if _, ok := inner.Triggered(); ok {
		t.Fatal("the inner interrupt was triggered")
	}
}

func TestInterruptInnerLeavesOuterRunning(t *testing.T) {
	// The reverse: triggering the inner interrupt ends only the inner context.
	requireQuotas(t)
	var out bytes.Buffer
	r := rt.New(&out)
	defer lib.LoadAll(r)()
	spin, err := r.CompileAndLoadLuaChunk("spin", []byte(`while true do end`), rt.TableValue(r.GlobalEnv()))
	if err != nil {
		t.Fatal(err)
	}
	thread := r.MainThread()
	outer, inner := rt.NewInterrupt(), rt.NewInterrupt()
	timer := time.AfterFunc(50*time.Millisecond, func() { inner.Trigger("inner interrupt") })
	defer timer.Stop()
	var innerErr error
	_, outerErr := thread.CallContext(rt.RuntimeContextDef{Interrupt: outer}, func() error {
		_, innerErr = thread.CallContext(rt.RuntimeContextDef{Interrupt: inner}, func() error {
			_, err := rt.Call1(thread, rt.FunctionValue(spin))
			return err
		})
		return nil
	})
	if innerErr == nil || !strings.Contains(innerErr.Error(), "inner interrupt") {
		t.Fatalf("inner err = %v, want the inner interrupt's reason", innerErr)
	}
	if outerErr != nil {
		t.Fatalf("outer err = %v, want the outer context to carry on", outerErr)
	}
	if _, ok := outer.Triggered(); ok {
		t.Fatal("the outer interrupt was triggered")
	}
}

func TestInterruptInstalledInSeveralRuntimes(t *testing.T) {
	// One Interrupt, installed in contexts of two runtimes, stops both.
	requireQuotas(t)
	intr := rt.NewInterrupt()
	timer := time.AfterFunc(50*time.Millisecond, func() { intr.Trigger("shared interrupt") })
	defer timer.Stop()
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			var out bytes.Buffer
			r := rt.New(&out)
			defer lib.LoadAll(r)()
			spin, err := r.CompileAndLoadLuaChunk("spin", []byte(`while true do end`), rt.TableValue(r.GlobalEnv()))
			if err != nil {
				errs <- err
				return
			}
			thread := r.MainThread()
			_, err = thread.CallContext(rt.RuntimeContextDef{Interrupt: intr}, func() error {
				_, err := rt.Call1(thread, rt.FunctionValue(spin))
				return err
			})
			errs <- err
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case err := <-errs:
			if err == nil || !strings.Contains(err.Error(), "shared interrupt") {
				t.Fatalf("err = %v, want the interrupt reason", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("a runtime did not stop")
		}
	}
}

func TestInterruptNoQuotasIgnored(t *testing.T) {
	// With the noquotas build tag an Interrupt is accepted and ignored.
	if rt.QuotasAvailable {
		t.Skip("quotas are available")
	}
	var out bytes.Buffer
	r := rt.New(&out)
	defer lib.LoadAll(r)()
	chunk, err := r.CompileAndLoadLuaChunk("test", []byte(`local n = 0 for i = 1, 100000 do n = n + i end return n`), rt.TableValue(r.GlobalEnv()))
	if err != nil {
		t.Fatal(err)
	}
	intr := rt.NewInterrupt()
	intr.Trigger("ignored")
	thread := r.MainThread()
	var v rt.Value
	_, err = thread.CallContext(rt.RuntimeContextDef{Interrupt: intr}, func() error {
		var err error
		v, err = rt.Call1(thread, rt.FunctionValue(chunk))
		return err
	})
	if err != nil || v.AsInt() != 5000050000 {
		t.Fatalf("result = %v, %v", v, err)
	}
}
